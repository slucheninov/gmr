package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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

// HasChanges reports whether there are staged, unstaged, or untracked changes.
func HasChanges(r Runner) (bool, error) {
	out, err := r.Run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// HasCommitsSince reports whether HEAD contains commits that are not in base.
func HasCommitsSince(r Runner, base string) (bool, error) {
	out, err := r.Run("rev-list", "--count", base+"..HEAD")
	if err != nil {
		return false, err
	}
	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return false, fmt.Errorf("invalid commit count %q: %w", out, err)
	}
	return count > 0, nil
}

// LastCommitMessage returns the full message of the commit at HEAD.
func LastCommitMessage(r Runner) (string, error) {
	return r.Run("log", "-1", "--pretty=%B")
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

// BranchExists reports whether name exists locally or on origin.
func BranchExists(r Runner, name string) bool {
	if _, err := r.Run("rev-parse", "--verify", "--quiet", "refs/heads/"+name); err == nil {
		return true
	}
	_, err := r.Run("rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+name)
	return err == nil
}

// FetchTags fetches tags from origin so local tag state matches the remote.
func FetchTags(r Runner) error {
	_, err := r.Run("fetch", "--tags", "--force", "--quiet")
	return err
}

// Tags returns all local tag names.
func Tags(r Runner) ([]string, error) {
	out, err := r.Run("tag", "--list")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var tags []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

// TagExists reports whether a tag exists locally.
func TagExists(r Runner, name string) bool {
	_, err := r.Run("rev-parse", "-q", "--verify", "refs/tags/"+name)
	return err == nil
}

// LatestTag returns the most recent reachable tag, or "" when there is none.
func LatestTag(r Runner) (string, error) {
	out, err := r.Run("describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", nil
	}
	return out, nil
}

// CreateTag creates an annotated tag with the given message.
func CreateTag(r Runner, name, msg string) error {
	_, err := r.Run("tag", "-a", name, "-m", msg)
	return err
}

// DeleteLocalTag removes a local tag (used to roll back a failed push).
func DeleteLocalTag(r Runner, name string) error {
	_, err := r.Run("tag", "-d", name)
	return err
}

// PushTag pushes a single tag to origin.
func PushTag(r Runner, name string) error {
	_, err := r.Run("push", "origin", name)
	return err
}

// LogRange returns "- subject" lines with bodies for commits in from..HEAD.
// When from is empty the whole history is used.
func LogRange(r Runner, from string) (string, error) {
	rangeArg := ""
	if from != "" {
		rangeArg = from + "..HEAD"
	}
	args := []string{"log"}
	if rangeArg != "" {
		args = append(args, rangeArg)
	}
	args = append(args, "--no-merges", "--pretty=format:- %s%n%b")
	out, err := r.Run(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
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
