package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/slucheninov/gmr/internal/git"
	"github.com/slucheninov/gmr/internal/ui"
)

// runCommitPush implements `gmr -c`: it commits working-tree changes to the
// current branch with a generated message and pushes the branch to origin.
// It never creates a branch or opens an MR/PR, and does not need gh/glab.
// With a clean working tree it skips AI generation and only pushes commits
// that are not on origin yet. The commit prompt behaves like `gmr -m`: the
// message is always printed to stdout. With working-tree changes and
// non-TTY stdin, nothing is committed or pushed.
func runCommitPush(r git.Runner, mainBranch string) error {
	branch, err := git.CurrentBranch(r)
	if err != nil {
		return err
	}
	if branch == "" {
		return errors.New("detached HEAD is not supported. Check out a branch first")
	}
	if branch == mainBranch {
		ui.Warn("You are on the base branch '%s': changes will be pushed to it directly, without an MR/PR", branch)
	}

	hasChanges, err := git.HasChanges(r)
	if err != nil {
		return err
	}
	if !hasChanges {
		return pushUnpushed(r, branch, mainBranch)
	}

	if !hasAPIKey() {
		return errors.New("no API key set. Export GEMINI_API_KEY, ANTHROPIC_API_KEY, or OPENAI_API_KEY")
	}
	msg, err := obtainCommitMessage(r)
	if err != nil {
		return err
	}

	action := actionPrintOnly
	if stdinIsTTY() {
		action = promptCommitChoice(os.Stdin, fmt.Sprintf("Commit and push to '%s'?", branch))
	}

	return actOnCommitPushChoice(r, action, msg, branch, mainBranch)
}

// actOnCommitPushChoice prints msg to stdout and, unless action is
// actionPrintOnly, commits it to branch and pushes branch to origin.
func actOnCommitPushChoice(r git.Runner, action commitAction, msg, branch, mainBranch string) error {
	committed, err := commitWithChoice(r, action, msg, branch)
	if err != nil {
		return err
	}

	if !committed {
		ui.OK("Commit message generated (not committed, not pushed)")
		ui.Log("Changes are staged; unstage with: git reset")
		return nil
	}

	return pushBranch(r, branch, mainBranch)
}

// pushUnpushed handles `gmr -c` with a clean working tree: it pushes branch
// when origin does not have it yet or it has commits ahead of origin, and
// reports an error when there is nothing to push.
func pushUnpushed(r git.Runner, branch, mainBranch string) error {
	if !git.RemoteBranchExists(r, branch) {
		ui.Log("No working-tree changes; '%s' is not on origin yet", branch)
		return pushBranch(r, branch, mainBranch)
	}
	n, err := git.UnpushedCount(r, branch)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("nothing to commit or push: '%s' is up to date with origin", branch)
	}
	ui.Log("No working-tree changes; pushing %d unpushed commit(s) on '%s'", n, branch)
	return pushBranch(r, branch, mainBranch)
}

// pushBranch pushes branch to origin (setting upstream) and, for a feature
// branch, hints how to open an MR/PR for it.
func pushBranch(r git.Runner, branch, mainBranch string) error {
	ui.Log("Pushing to origin/%s...", branch)
	if err := git.Push(r, branch); err != nil {
		return err
	}
	ui.OK("Done! Pushed to origin/%s", branch)
	if branch != mainBranch {
		ui.Log("To open an MR/PR for this branch, run: gmr")
	}
	return nil
}
