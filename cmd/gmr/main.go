package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/slucheninov/gmr/internal/ai"
	"github.com/slucheninov/gmr/internal/commit"
	"github.com/slucheninov/gmr/internal/git"
	"github.com/slucheninov/gmr/internal/platform"
	"github.com/slucheninov/gmr/internal/ui"
	"github.com/slucheninov/gmr/internal/version"
)

const helpText = `gmr — Git Merge Request / Pull Request automation

Usage: gmr [-m] [branch-name]

Creates a branch, generates an AI commit message (Gemini → Claude → OpenAI → manual),
commits staged changes, and opens a GitLab MR or GitHub PR (auto-detected).

If branch-name is omitted, generates: auto/YYYYMMDD-HHMMSS

Options:
  -h, --help      Show this help
  -m, --message   Generate commit message only (no commit, branch, or MR/PR)
  -v, --version   Show version

Environment variables:
  GEMINI_API_KEY       Google Gemini API key
  ANTHROPIC_API_KEY    Anthropic Claude API key
  OPENAI_API_KEY       OpenAI API key
  GMR_MAIN_BRANCH      Base branch (default: auto-detect from origin/HEAD)
  GMR_GEMINI_MODEL     Gemini model (default: gemini-flash-latest)
  GMR_ANTHROPIC_MODEL  Claude model (default: claude-sonnet-4-20250514)
  GMR_OPENAI_MODEL     OpenAI model (default: gpt-4o-mini)
  GMR_MAX_DIFF         Max diff lines for AI (default: 500)
`

func main() {
	args := os.Args[1:]
	messageOnly := false
	branchArg := ""

	for _, a := range args {
		switch a {
		case "-h", "--help":
			fmt.Print(helpText)
			return
		case "-v", "--version":
			fmt.Printf("gmr %s\n", version.Version)
			return
		case "-m", "--message":
			messageOnly = true
		default:
			if strings.HasPrefix(a, "-") {
				ui.Errf("unknown option: %s", a)
			}
			if branchArg != "" {
				ui.Errf("unexpected argument: %s", a)
			}
			branchArg = a
		}
	}

	if err := run(messageOnly, branchArg); err != nil {
		ui.Errf("%s", err.Error())
	}
}

func run(messageOnly bool, branchArg string) error {
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("OPENAI_API_KEY") == "" {
		return errors.New("no API key set. Export GEMINI_API_KEY, ANTHROPIC_API_KEY, or OPENAI_API_KEY")
	}

	r := git.NewRunner()

	if err := git.IsRepo(r); err != nil {
		return err
	}

	mainBranch := git.DetectMainBranch(r)

	var (
		plat        platform.Kind
		remoteURL   string
		branchName  string
		gitlabPath  string
	)

	if !messageOnly {
		var err error
		remoteURL, err = git.RemoteURL(r, "origin")
		if err != nil {
			return errors.New("no 'origin' remote found")
		}
		plat, err = platform.Detect(remoteURL)
		if err != nil {
			return err
		}
		ui.Log("Platform: %s", plat)

		if plat == platform.GitLab {
			if _, err := exec.LookPath("glab"); err != nil {
				return errors.New("glab is not installed. Install: https://gitlab.com/gitlab-org/cli")
			}
			if err := runQuiet("glab", "auth", "status"); err != nil {
				return errors.New("glab is not authenticated for GitLab API. Run: glab auth login")
			}
			gitlabPath, err = platform.GitLabProjectPath(remoteURL)
			if err != nil {
				return err
			}
		} else {
			if _, err := exec.LookPath("gh"); err != nil {
				return errors.New("gh is not installed. Install: https://cli.github.com")
			}
			if err := runQuiet("gh", "auth", "status"); err != nil {
				return errors.New("gh is not authenticated for GitHub API. Run: gh auth login")
			}
		}

		current, err := git.CurrentBranch(r)
		if err != nil {
			return err
		}
		if current != mainBranch {
			return fmt.Errorf("current branch is '%s', not '%s'. Switch to %s first", current, mainBranch, mainBranch)
		}
	}

	hasChanges, err := git.HasChanges(r)
	if err != nil {
		return err
	}
	if !hasChanges {
		return errors.New("no changes to commit. Make some changes first")
	}

	if !messageOnly {
		branchName = branchArg
		if branchName == "" {
			branchName = "auto/" + time.Now().Format("20060102-150405")
		}
		ui.Log("Branch: %s", branchName)
	}

	commitMsg, err := generateCommitMessage(r)
	if err != nil {
		return err
	}
	if commitMsg == "" {
		return errors.New("commit message is empty. Aborted")
	}

	if messageOnly {
		fmt.Println(commitMsg)
		ui.OK("Commit message generated (not committed)")
		return nil
	}

	ui.Log("Creating branch '%s'...", branchName)
	if err := git.Checkout(r, branchName, true); err != nil {
		return err
	}

	cleanup := func() {
		current, _ := git.CurrentBranch(r)
		if current != mainBranch {
			ui.Warn("Returning to %s...", mainBranch)
			_ = git.Checkout(r, mainBranch, false)
		}
	}
	defer cleanup()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		if _, ok := <-sigCh; ok {
			cleanup()
			os.Exit(130)
		}
	}()

	ui.Log("Committing...")
	if err := git.Commit(r, commitMsg); err != nil {
		return err
	}

	mrTitle := commit.Title(commitMsg)
	mrDescription := commit.MRDescription(commitMsg)

	if plat == platform.GitLab {
		if err := git.Push(r, branchName); err != nil {
			return err
		}
		ui.Log("Creating Merge Request...")
		args := []string{
			"mr", "create",
			"-R", gitlabPath,
			"--source-branch", branchName,
			"--target-branch", mainBranch,
			"--title", mrTitle,
			"--yes",
			"--remove-source-branch",
			"--squash-before-merge",
			"--description", mrDescription,
		}
		c := exec.Command("glab", args...)
		c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := c.Run(); err != nil {
			return err
		}
	} else {
		ui.Log("Creating Pull Request...")
		if err := git.Push(r, branchName); err != nil {
			return err
		}
		c := exec.Command("gh", "pr", "create", "--fill")
		c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := c.Run(); err != nil {
			return err
		}
		ui.Log("Enabling auto-merge (squash)...")
		out, err := exec.Command("gh", "pr", "merge", "--auto", "--squash").CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "Auto merge is not allowed") {
				ui.Warn("Auto-merge is disabled for this repository — skipping.")
				fmt.Fprintln(os.Stderr, "  Hint: enable it in Settings → General → 'Allow auto-merge'.")
			} else {
				fmt.Fprint(os.Stderr, string(out))
				ui.Warn("Failed to enable auto-merge — skipping.")
			}
		}
	}

	ui.Log("Switching back to %s...", mainBranch)
	if err := git.Checkout(r, mainBranch, false); err != nil {
		return err
	}
	if err := git.Pull(r); err != nil {
		ui.Warn("git pull failed: %s", err)
	}

	if plat == platform.GitLab {
		ui.OK("Done! MR created, you are on %s", mainBranch)
	} else {
		ui.OK("Done! PR created, you are on %s", mainBranch)
	}
	return nil
}

func generateCommitMessage(r git.Runner) (string, error) {
	if err := git.StageAll(r); err != nil {
		return "", err
	}
	stat, err := git.CachedDiffStat(r)
	if err != nil {
		return "", err
	}
	full, err := git.CachedDiff(r)
	if err != nil {
		return "", err
	}

	maxDiffLines := 500
	if v := os.Getenv("GMR_MAX_DIFF"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxDiffLines = n
		}
	}

	limited, truncated := git.LimitLines(full, maxDiffLines)
	diffContent := stat + "\n---\n" + limited
	if truncated {
		diffContent += fmt.Sprintf("\n... (diff truncated at %d lines)", maxDiffLines)
	}

	providers := []ai.Provider{
		ai.NewGemini(os.Getenv("GEMINI_API_KEY"), os.Getenv("GMR_GEMINI_MODEL")),
		ai.NewClaude(os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("GMR_ANTHROPIC_MODEL")),
		ai.NewOpenAI(os.Getenv("OPENAI_API_KEY"), os.Getenv("GMR_OPENAI_MODEL")),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var msg string
	for _, p := range providers {
		out, err := p.Generate(ctx, diffContent)
		if err == nil && out != "" {
			msg = out
			break
		}
	}

	if msg == "" {
		ui.Warn("All APIs unavailable. Enter commit message manually:")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line), nil
	}

	fmt.Fprintln(ui.Out)
	fmt.Fprintln(ui.Out, ui.Highlight("Generated commit message:"))
	ui.Banner()
	fmt.Fprintln(ui.Out, msg)
	ui.Banner()
	fmt.Fprintln(ui.Out)

	fmt.Fprint(ui.Out, ui.Prompt("Accept? [Y/n/e(edit)]: "))
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "n":
		fmt.Fprint(ui.Out, "Enter your commit message: ")
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line), nil
	case "e":
		edited, err := editInEditor(msg)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(edited), nil
	}
	return msg, nil
}

func editInEditor(initial string) (string, error) {
	tmp, err := os.CreateTemp("", "gmr-commit-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(initial); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	c := exec.Command(editor, tmp.Name())
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return "", err
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func runQuiet(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr = nil, nil
	return c.Run()
}
