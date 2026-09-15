package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/slucheninov/gmr/internal/git"
	"github.com/slucheninov/gmr/internal/platform"
	"github.com/slucheninov/gmr/internal/ui"
)

// msgOnlyAction is the user's choice at the `gmr -m` commit prompt.
type msgOnlyAction int

const (
	actionPrintOnly msgOnlyAction = iota
	actionCommit
	actionEdit
)

// parseMessageOnlyChoice interprets the raw line read from stdin at the
// `gmr -m` commit prompt. "y"/"yes"/empty commits to the current branch,
// "e"/"edit" opens $EDITOR first and then commits, and everything else
// (including "n"/"no") is the safe default: print the message only, no git
// actions.
func parseMessageOnlyChoice(input string) msgOnlyAction {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "", "y", "yes":
		return actionCommit
	case "e", "edit":
		return actionEdit
	default:
		return actionPrintOnly
	}
}

// promptMessageOnlyChoice writes the commit prompt for branch to ui.Out and
// reads the answer from in, returning the parsed choice.
func promptMessageOnlyChoice(in io.Reader, branch string) msgOnlyAction {
	fmt.Fprint(ui.Out, ui.Prompt(fmt.Sprintf("Commit to '%s'? [Y/n/e(edit)] (n = print only): ", branch)))
	reader := bufio.NewReader(in)
	line, _ := reader.ReadString('\n')
	return parseMessageOnlyChoice(line)
}

// runMessageOnly implements `gmr -m`: it generates a commit message (falling
// back to manual entry if every AI provider fails), then asks whether to
// commit it to the current branch, print it only, or edit it first. It never
// creates a branch, pushes, or opens an MR/PR. The message is always printed
// to stdout so `gmr -m | ...` stays pipe-friendly; everything else goes to
// stderr via the ui package. When stdin is not a TTY, it never prompts and
// behaves as if "n" (print only) was chosen, so scripted/piped use is safe.
func runMessageOnly(r git.Runner, mainBranch string) error {
	msg, ok, err := generateRawMessage(r)
	if err != nil {
		return err
	}
	if !ok {
		ui.Warn("All APIs unavailable. Enter commit message manually:")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(line)
	} else {
		printGeneratedMessage(msg)
	}
	if msg == "" {
		return errors.New("commit message is empty. Aborted")
	}

	current, err := git.CurrentBranch(r)
	if err != nil {
		return err
	}

	action := actionPrintOnly
	if stdinIsTTY() {
		action = promptMessageOnlyChoice(os.Stdin, current)
	}

	return actOnMessageOnlyChoice(r, action, msg, current, mainBranch)
}

// actOnMessageOnlyChoice performs the git action for action, prints msg to
// stdout in every case, and prints a "Next steps" hint block after a
// successful commit.
func actOnMessageOnlyChoice(r git.Runner, action msgOnlyAction, msg, current, mainBranch string) error {
	if action == actionEdit {
		edited, err := editInEditor(msg)
		if err != nil {
			return err
		}
		edited = strings.TrimSpace(edited)
		if edited == "" {
			return errors.New("commit message is empty. Aborted")
		}
		msg = edited
		action = actionCommit
	}

	fmt.Println(msg)

	if action != actionCommit {
		ui.OK("Commit message generated (not committed)")
		ui.Log("Changes are staged; unstage with: git reset")
		return nil
	}

	if err := git.Commit(r, msg); err != nil {
		return err
	}
	ui.OK("Committed to '%s'", current)
	printNextSteps(r, current, mainBranch)
	return nil
}

// printNextSteps prints copy-pasteable commands (to ui.Out) for pushing the
// just-created commit and opening an MR/PR. The platform is detected from the
// `origin` remote URL without requiring gh/glab to be installed or
// authenticated; when there is no origin remote or its host is unrecognized,
// only remote-agnostic commands are printed.
func printNextSteps(r git.Runner, branch, mainBranch string) {
	ui.Log("Next steps:")

	if branch == "" {
		ui.Warn("Detached HEAD — push commands skipped.")
		return
	}

	remoteURL, err := git.RemoteURL(r, "origin")
	if err != nil {
		ui.Warn("No 'origin' remote configured — add one, then run 'gmr' to open an MR/PR.")
		return
	}

	var kind platform.Kind
	var gitlabPath string
	if k, err := platform.Detect(remoteURL); err == nil {
		kind = k
		if k == platform.GitLab {
			gitlabPath, _ = platform.GitLabProjectPath(remoteURL)
		}
	}

	for _, c := range nextStepCommands(kind, gitlabPath, branch, mainBranch) {
		fmt.Fprintf(ui.Out, "  %s\n", c)
	}
}

// nextStepCommands returns the copy-pasteable shell commands to push the
// current commit and open an MR/PR, given the detected platform. kind == ""
// means the platform is unknown or there is nothing to detect it from; in
// that case only remote-agnostic commands are returned. branch == "" is
// detached HEAD and returns nil — the caller is expected to warn separately.
func nextStepCommands(kind platform.Kind, gitlabPath, branch, mainBranch string) []string {
	if branch == "" {
		return nil
	}

	if branch != mainBranch {
		cmds := []string{"git push -u origin " + shellQuote(branch)}
		cmds = append(cmds, createCommand(kind, gitlabPath, branch, mainBranch)...)
		cmds = append(cmds, "gmr  # pushes and opens MR/PR for this branch's commits")
		return cmds
	}

	// On the main branch: push it, then run gmr — it moves the unpushed
	// commit(s) to a new branch, opens the MR/PR, and resets local main to
	// origin/main.
	return []string{
		"git push origin " + shellQuote(mainBranch),
		"gmr  # moves the commit to a new branch and opens MR/PR",
	}
}

// createCommand returns the gh/glab MR/PR-creation command for the given
// platform and head branch, or nil when the platform is unknown.
func createCommand(kind platform.Kind, gitlabPath, head, mainBranch string) []string {
	switch kind {
	case platform.GitHub:
		return []string{fmt.Sprintf("gh pr create --fill --base %s --head %s", shellQuote(mainBranch), shellQuote(head))}
	case platform.GitLab:
		return []string{fmt.Sprintf("glab mr create -R %s --source-branch %s --target-branch %s --fill --yes --remove-source-branch --squash-before-merge", shellQuote(gitlabPath), shellQuote(head), shellQuote(mainBranch))}
	default:
		return nil
	}
}

// shellQuote wraps s in single quotes if it contains characters that would
// need escaping in a POSIX shell; otherwise s is returned unchanged.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '/' || r == '.':
		default:
			return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
		}
	}
	return s
}
