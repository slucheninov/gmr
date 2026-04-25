package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner abstracts running git commands; it exists so tests can substitute it.
type Runner interface {
	Run(args ...string) (string, error)
	RunInteractive(args ...string) error
}

type execRunner struct{}

// NewRunner returns the default Runner that shells out to the `git` binary.
func NewRunner() Runner { return execRunner{} }

func (execRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return strings.TrimSpace(stdout.String()), fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (execRunner) RunInteractive(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// IsRepo returns nil if the current directory is inside a git work tree.
func IsRepo(r Runner) error {
	out, err := r.Run("rev-parse", "--is-inside-work-tree")
	if err != nil || out != "true" {
		return errors.New("not inside a git repository")
	}
	return nil
}

// CurrentBranch returns the active branch name.
func CurrentBranch(r Runner) (string, error) {
	return r.Run("branch", "--show-current")
}

// RemoteURL returns the URL of the given remote.
func RemoteURL(r Runner, remote string) (string, error) {
	return r.Run("remote", "get-url", remote)
}

// HasChanges reports whether there are staged or unstaged changes.
func HasChanges(r Runner) (bool, error) {
	if _, err := r.Run("diff", "--quiet"); err != nil {
		return true, nil
	}
	if _, err := r.Run("diff", "--cached", "--quiet"); err != nil {
		return true, nil
	}
	return false, nil
}

// StageAll runs `git add -A`.
func StageAll(r Runner) error {
	_, err := r.Run("add", "-A")
	return err
}

// CachedDiffStat returns `git diff --cached --stat`.
func CachedDiffStat(r Runner) (string, error) {
	return r.Run("diff", "--cached", "--stat")
}

// CachedDiff returns `git diff --cached`.
func CachedDiff(r Runner) (string, error) {
	return r.Run("diff", "--cached")
}

// DetectMainBranch resolves the base branch name. It honours the GMR_MAIN_BRANCH
// override, otherwise reads origin/HEAD, falling back to main/master.
func DetectMainBranch(r Runner) string {
	if override := strings.TrimSpace(os.Getenv("GMR_MAIN_BRANCH")); override != "" {
		return override
	}
	if out, err := r.Run("symbolic-ref", "-q", "refs/remotes/origin/HEAD"); err == nil && out != "" {
		return strings.TrimPrefix(out, "refs/remotes/origin/")
	}
	if _, err := r.Run("show-ref", "--verify", "--quiet", "refs/heads/main"); err == nil {
		return "main"
	}
	if _, err := r.Run("show-ref", "--verify", "--quiet", "refs/heads/master"); err == nil {
		return "master"
	}
	return "master"
}

// Checkout switches to the given branch, creating it if create is true.
func Checkout(r Runner, branch string, create bool) error {
	args := []string{"checkout"}
	if create {
		args = append(args, "-b")
	}
	args = append(args, branch)
	_, err := r.Run(args...)
	return err
}

// Commit creates a commit with the given message.
func Commit(r Runner, msg string) error {
	_, err := r.Run("commit", "-m", msg)
	return err
}

// Push pushes the branch to origin with -u.
func Push(r Runner, branch string) error {
	_, err := r.Run("push", "-u", "origin", branch)
	return err
}

// Pull runs `git pull --quiet`.
func Pull(r Runner) error {
	_, err := r.Run("pull", "--quiet")
	return err
}

// LimitLines returns the first n lines of s; if s already has <= n lines it is
// returned unchanged. The returned string keeps the original trailing newline
// semantics of those n lines.
func LimitLines(s string, n int) (string, bool) {
	if n <= 0 {
		return "", true
	}
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			count++
			if count == n {
				return s[:i+1], true
			}
		}
	}
	return s, false
}
