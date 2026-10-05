package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/slucheninov/gmr/internal/git"
	"github.com/slucheninov/gmr/internal/ui"
)

// commitAction is the user's choice at the `gmr -m` or `gmr -c` prompt.
type commitAction int

const (
	actionPrintOnly commitAction = iota
	actionCommit
	actionEdit
)

// parseCommitChoice interprets the raw line read from stdin at the
// `gmr -m` or `gmr -c` commit prompt. "y"/"yes"/empty commits to the current branch,
// "e"/"edit" opens $EDITOR first and then commits, and everything else
// (including "n"/"no") is the safe default: print the message only, no git
// actions.
func parseCommitChoice(input string) commitAction {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "", "y", "yes":
		return actionCommit
	case "e", "edit":
		return actionEdit
	default:
		return actionPrintOnly
	}
}

// promptCommitChoice writes question followed by the
// "[Y/n/e(edit)] (n = print only)" choices to ui.Out and reads the answer
// from in, returning the parsed choice. It backs both the `gmr -m` and the
// `gmr -c` prompts. A read that ends without a full answer line (EOF, e.g.
// stdin is /dev/null or the user pressed Ctrl+D) is never taken as consent
// and yields actionPrintOnly.
func promptCommitChoice(in io.Reader, question string) commitAction {
	fmt.Fprint(ui.Out, ui.Prompt(question+" [Y/n/e(edit)] (n = print only): "))
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(ui.Out)
		return actionPrintOnly
	}
	return parseCommitChoice(line)
}

// obtainCommitMessage stages all changes and generates a commit message via
// the AI provider chain, printing it to ui.Out. If every provider fails, it
// falls back to reading a message from stdin. An empty result is an error.
func obtainCommitMessage(r git.Runner) (string, error) {
	msg, ok, err := generateRawMessage(r)
	if err != nil {
		return "", err
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
		return "", errors.New("commit message is empty. Aborted")
	}
	return msg, nil
}

// applyEditChoice resolves actionEdit by opening msg in $EDITOR and turning
// the choice into actionCommit with the edited message. Other actions are
// returned unchanged.
func applyEditChoice(action commitAction, msg string) (commitAction, string, error) {
	if action != actionEdit {
		return action, msg, nil
	}
	edited, err := editInEditor(msg)
	if err != nil {
		return action, "", err
	}
	edited = strings.TrimSpace(edited)
	if edited == "" {
		return action, "", errors.New("commit message is empty. Aborted")
	}
	return actionCommit, edited, nil
}

// commitWithChoice applies the optional edit, prints the final message to
// stdout, and commits only when requested. Callers handle their own next steps.
func commitWithChoice(r git.Runner, action commitAction, msg, branch string) (bool, error) {
	action, msg, err := applyEditChoice(action, msg)
	if err != nil {
		return false, err
	}

	fmt.Println(msg)
	if action != actionCommit {
		return false, nil
	}
	if err := git.Commit(r, msg); err != nil {
		return false, err
	}
	ui.OK("Committed to '%s'", branch)
	return true, nil
}
