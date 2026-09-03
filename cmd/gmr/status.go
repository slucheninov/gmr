package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/slucheninov/gmr/internal/ci"
	"github.com/slucheninov/gmr/internal/git"
	"github.com/slucheninov/gmr/internal/platform"
	"github.com/slucheninov/gmr/internal/ui"
)

const statusHelp = `gmr status — show CI/CD pipeline status for a branch or tag

Usage: gmr status [options] [ref]

Shows the most recent pipeline/workflow runs for the current branch (or the
given ref) and, when no ref is given, for the latest semver tag as well.

Options:
  -h, --help     Show this help
      --limit N  Number of recent runs to show per ref (default: 3, clamped 1-20)

Exit status: 0 when the newest run of every inspected ref passed (or none
exist), 1 when the newest run of any inspected ref failed. This makes
'gmr status' usable in scripts.
`

var errStatusShowHelp = errors.New("show status help")

type statusOptions struct {
	limit int
	ref   string
}

// parseStatusArgs parses the arguments to `gmr status`.
func parseStatusArgs(args []string) (statusOptions, error) {
	o := statusOptions{limit: 3}
	var positional []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			return statusOptions{}, errStatusShowHelp
		case a == "--limit":
			if i+1 >= len(args) {
				return statusOptions{}, errors.New("--limit requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return statusOptions{}, fmt.Errorf("invalid --limit value: %q", args[i])
			}
			o.limit = clampStatusLimit(n)
		case strings.HasPrefix(a, "--limit="):
			v := strings.TrimPrefix(a, "--limit=")
			n, err := strconv.Atoi(v)
			if err != nil {
				return statusOptions{}, fmt.Errorf("invalid --limit value: %q", v)
			}
			o.limit = clampStatusLimit(n)
		case strings.HasPrefix(a, "-"):
			return statusOptions{}, fmt.Errorf("unknown option: %s", a)
		default:
			positional = append(positional, a)
		}
	}

	if len(positional) > 1 {
		return statusOptions{}, fmt.Errorf("unexpected argument: %s", positional[1])
	}
	if len(positional) == 1 {
		o.ref = positional[0]
	}

	return o, nil
}

// clampStatusLimit clamps n to the [1, 20] range accepted by `gmr status --limit`.
func clampStatusLimit(n int) int {
	if n < 1 {
		return 1
	}
	if n > 20 {
		return 20
	}
	return n
}

// statusMain is the entry point for `gmr status`, dispatched from main().
func statusMain(args []string) {
	opts, err := parseStatusArgs(args)
	if err != nil {
		if errors.Is(err, errStatusShowHelp) {
			fmt.Print(statusHelp)
			return
		}
		ui.Errf("%s", err.Error())
		return
	}

	failed, err := runStatus(opts)
	if err != nil {
		ui.Errf("%s", err.Error())
		return
	}
	if failed {
		os.Exit(1)
	}
}

// statusTarget is one ref (branch or tag) to report on.
type statusTarget struct {
	label string // "Branch" or "Tag"
	ref   string
}

// runStatus implements `gmr status`. It returns (true, nil) when the newest
// run of any inspected ref failed, so the caller can exit(1) without gmr
// printing a red "Error:" line for what is really just bad CI news.
func runStatus(opts statusOptions) (bool, error) {
	r := git.NewRunner()
	if err := git.IsRepo(r); err != nil {
		return false, err
	}

	pc, err := detectPlatform(r)
	if err != nil {
		return false, err
	}

	var targets []statusTarget
	if opts.ref != "" {
		targets = append(targets, statusTarget{label: "Branch", ref: opts.ref})
	} else {
		current, err := git.CurrentBranch(r)
		if err != nil {
			return false, err
		}
		if current == "" {
			return false, errors.New("detached HEAD is not supported. Check out a branch first")
		}
		targets = append(targets, statusTarget{label: "Branch", ref: current})

		if latest, err := git.LatestTag(r); err == nil && latest != "" && latest != current {
			targets = append(targets, statusTarget{label: "Tag", ref: latest})
		}
	}

	ciRunner := ci.NewRunner()
	anyFailed := false

	for i, t := range targets {
		if i > 0 {
			fmt.Fprintln(ui.Out)
		}

		var runs []ci.Run
		var fetchErr error
		if pc.kind == platform.GitLab {
			runs, fetchErr = ci.GitLabPipelines(ciRunner, pc.gitlabPath, t.ref, opts.limit)
		} else {
			runs, fetchErr = ci.GitHubRuns(ciRunner, t.ref, opts.limit)
		}
		if fetchErr != nil {
			ui.Warn("Failed to fetch runs for %s %s: %s", strings.ToLower(t.label), t.ref, fetchErr)
			continue
		}

		if len(runs) > 0 {
			newest := &runs[0]
			var jobs []ci.Job
			var jobsErr error
			if pc.kind == platform.GitLab {
				jobs, jobsErr = ci.GitLabJobs(ciRunner, pc.gitlabPath, newest.ID)
			} else {
				jobs, jobsErr = ci.GitHubJobs(ciRunner, newest.ID)
			}
			if jobsErr != nil {
				ui.Warn("Failed to fetch jobs for %s %s: %s", strings.ToLower(t.label), t.ref, jobsErr)
			} else {
				newest.Jobs = jobs
			}
		}

		renderRuns(ui.Out, t.label, t.ref, runs)
		fmt.Fprintln(ui.Out, verdict(t.label, t.ref, runs))

		if isNewestFailed(runs) {
			anyFailed = true
		}
	}

	return anyFailed, nil
}

// isNewestFailed reports whether the newest (first) run is done and not OK.
func isNewestFailed(runs []ci.Run) bool {
	if len(runs) == 0 {
		return false
	}
	return runs[0].State.Done() && !runs[0].State.OK()
}

// renderRuns writes a human-readable listing of runs for one ref to w. Job
// lines are printed only for the newest (first) run.
func renderRuns(w io.Writer, label, ref string, runs []ci.Run) {
	fmt.Fprintf(w, "%s %s\n", label, ref)
	if len(runs) == 0 {
		fmt.Fprintln(w, "  no pipelines found")
		return
	}

	now := time.Now()
	for i, run := range runs {
		fields := []string{glyph(run.State), runName(run), string(run.State)}
		if age := relAge(run.Created, now); age != "" {
			fields = append(fields, age)
		}
		if run.URL != "" {
			fields = append(fields, run.URL)
		}
		fmt.Fprintf(w, "  %s\n", strings.Join(fields, "  "))

		if i == 0 {
			for _, j := range run.Jobs {
				fmt.Fprintf(w, "      %s %s\n", glyph(j.State), jobLabel(j))
			}
		}
	}
}

// verdict renders the plain-language "did it pass" line for one ref.
func verdict(label, ref string, runs []ci.Run) string {
	prefix := fmt.Sprintf("%s %s", label, ref)
	if len(runs) == 0 {
		return prefix + ": no pipelines found"
	}

	newest := runs[0]
	if !newest.State.Done() {
		return prefix + ": still running"
	}
	if newest.State.OK() {
		return prefix + ": all pipelines passed"
	}

	var failedJobs []string
	for _, j := range newest.Jobs {
		if j.State.Done() && !j.State.OK() {
			failedJobs = append(failedJobs, jobLabel(j))
		}
	}
	if len(failedJobs) > 0 {
		return fmt.Sprintf("%s: FAILED (%s)", prefix, strings.Join(failedJobs, ", "))
	}
	return prefix + ": FAILED"
}

// runName returns the run's display name, falling back to its ID.
func runName(r ci.Run) string {
	if r.Name != "" {
		return r.Name
	}
	return r.ID
}

// jobLabel returns "stage/name" when Stage is set (GitLab), else just Name.
func jobLabel(j ci.Job) string {
	if j.Stage != "" {
		return j.Stage + "/" + j.Name
	}
	return j.Name
}

// glyph returns a colored one-character status indicator for s.
func glyph(s ci.State) string {
	switch s {
	case ci.Success, ci.Skipped:
		return ui.Colorize(ui.ColorGreen, "✓")
	case ci.Failed:
		return ui.Colorize(ui.ColorRed, "✗")
	case ci.Running:
		return ui.Colorize(ui.ColorYellow, "●")
	case ci.Pending:
		return ui.Colorize(ui.ColorYellow, "○")
	default: // Canceled, Unknown
		return "–"
	}
}

// relAge renders t relative to now as "just now", "3m ago", "2h ago", "5d ago",
// or "" when t is the zero value.
func relAge(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	}
}
