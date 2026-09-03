package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/slucheninov/gmr/internal/ci"
	"github.com/slucheninov/gmr/internal/ui"
)

func TestParseStatusArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		want    statusOptions
		wantErr string
	}{
		{
			name: "defaults",
			args: nil,
			want: statusOptions{limit: 3},
		},
		{
			name: "limit flag",
			args: []string{"--limit", "5"},
			want: statusOptions{limit: 5},
		},
		{
			name: "limit flag equals form",
			args: []string{"--limit=7"},
			want: statusOptions{limit: 7},
		},
		{
			name: "limit clamped low",
			args: []string{"--limit", "0"},
			want: statusOptions{limit: 1},
		},
		{
			name: "limit clamped negative",
			args: []string{"--limit", "-5"},
			want: statusOptions{limit: 1},
		},
		{
			name: "limit clamped high",
			args: []string{"--limit", "999"},
			want: statusOptions{limit: 20},
		},
		{
			name: "positional ref",
			args: []string{"my-branch"},
			want: statusOptions{limit: 3, ref: "my-branch"},
		},
		{
			name: "limit and ref",
			args: []string{"--limit", "2", "v1.0.0"},
			want: statusOptions{limit: 2, ref: "v1.0.0"},
		},
		{
			name:    "bad limit value",
			args:    []string{"--limit", "abc"},
			wantErr: `invalid --limit value: "abc"`,
		},
		{
			name:    "bad limit equals value",
			args:    []string{"--limit=abc"},
			wantErr: `invalid --limit value: "abc"`,
		},
		{
			name:    "limit missing value",
			args:    []string{"--limit"},
			wantErr: "--limit requires a value",
		},
		{
			name:    "unknown flag",
			args:    []string{"--bogus"},
			wantErr: "unknown option: --bogus",
		},
		{
			name:    "too many positional args",
			args:    []string{"branch-a", "branch-b"},
			wantErr: "unexpected argument: branch-b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseStatusArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseStatusArgs(%v) error = %v, want %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseStatusArgs(%v): unexpected error: %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("parseStatusArgs(%v) = %#v, want %#v", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseStatusArgsHelp(t *testing.T) {
	t.Parallel()
	_, err := parseStatusArgs([]string{"-h"})
	if !errors.Is(err, errStatusShowHelp) {
		t.Fatalf("parseStatusArgs([-h]) error = %v, want errStatusShowHelp", err)
	}
	_, err = parseStatusArgs([]string{"--help"})
	if !errors.Is(err, errStatusShowHelp) {
		t.Fatalf("parseStatusArgs([--help]) error = %v, want errStatusShowHelp", err)
	}
}

// withPlainUI forces ui.Out to a buffer (never a terminal) for the duration
// of the test, so ui.Colorize never emits ANSI escapes regardless of the
// environment's NO_COLOR setting or whether stderr happens to be a TTY.
func withPlainUI(t *testing.T) *bytes.Buffer {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	orig := ui.Out
	ui.Out = &buf
	t.Cleanup(func() { ui.Out = orig })
	return &buf
}

func TestGlyph(t *testing.T) {
	withPlainUI(t)
	tests := []struct {
		state ci.State
		want  string
	}{
		{ci.Success, "✓"},
		{ci.Skipped, "✓"},
		{ci.Failed, "✗"},
		{ci.Running, "●"},
		{ci.Pending, "○"},
		{ci.Canceled, "–"},
		{ci.Unknown, "–"},
	}
	for _, tt := range tests {
		if got := glyph(tt.state); got != tt.want {
			t.Errorf("glyph(%s) = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestRelAge(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"zero", time.Time{}, ""},
		{"just now", now.Add(-10 * time.Second), "just now"},
		{"minutes", now.Add(-3 * time.Minute), "3m ago"},
		{"just under an hour", now.Add(-59 * time.Minute), "59m ago"},
		{"hours", now.Add(-2 * time.Hour), "2h ago"},
		{"just under a day", now.Add(-23 * time.Hour), "23h ago"},
		{"days", now.Add(-5 * 24 * time.Hour), "5d ago"},
		{"future clamps to just now", now.Add(1 * time.Minute), "just now"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relAge(tt.t, now); got != tt.want {
				t.Errorf("relAge(%v, now) = %q, want %q", tt.t, got, tt.want)
			}
		})
	}
}

func TestVerdict(t *testing.T) {
	withPlainUI(t)
	tests := []struct {
		name  string
		label string
		ref   string
		runs  []ci.Run
		want  string
	}{
		{
			name:  "no runs",
			label: "Branch",
			ref:   "master",
			runs:  nil,
			want:  "Branch master: no pipelines found",
		},
		{
			name:  "still running",
			label: "Tag",
			ref:   "v0.9.0",
			runs:  []ci.Run{{State: ci.Running}},
			want:  "Tag v0.9.0: still running",
		},
		{
			name:  "pending counts as running",
			label: "Branch",
			ref:   "feat",
			runs:  []ci.Run{{State: ci.Pending}},
			want:  "Branch feat: still running",
		},
		{
			name:  "all passed",
			label: "Branch",
			ref:   "master",
			runs:  []ci.Run{{State: ci.Success}},
			want:  "Branch master: all pipelines passed",
		},
		{
			name:  "skipped counts as passed",
			label: "Branch",
			ref:   "master",
			runs:  []ci.Run{{State: ci.Skipped}},
			want:  "Branch master: all pipelines passed",
		},
		{
			name:  "failed with no job detail",
			label: "Branch",
			ref:   "master",
			runs:  []ci.Run{{State: ci.Failed}},
			want:  "Branch master: FAILED",
		},
		{
			name:  "failed naming jobs",
			label: "Branch",
			ref:   "master",
			runs: []ci.Run{{
				State: ci.Failed,
				Jobs: []ci.Job{
					{Name: "build", State: ci.Success},
					{Name: "test", State: ci.Failed},
					{Stage: "deploy", Name: "prod", State: ci.Failed},
				},
			}},
			want: "Branch master: FAILED (test, deploy/prod)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verdict(tt.label, tt.ref, tt.runs); got != tt.want {
				t.Errorf("verdict() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsNewestFailed(t *testing.T) {
	tests := []struct {
		name string
		runs []ci.Run
		want bool
	}{
		{"no runs", nil, false},
		{"success", []ci.Run{{State: ci.Success}}, false},
		{"running", []ci.Run{{State: ci.Running}}, false},
		{"failed", []ci.Run{{State: ci.Failed}}, true},
		{"canceled counts as failed", []ci.Run{{State: ci.Canceled}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewestFailed(tt.runs); got != tt.want {
				t.Errorf("isNewestFailed(%v) = %v, want %v", tt.runs, got, tt.want)
			}
		})
	}
}

func TestRenderRuns(t *testing.T) {
	buf := withPlainUI(t)

	created := time.Now().Add(-2 * time.Hour)
	runs := []ci.Run{
		{
			Name:    "release",
			State:   ci.Success,
			URL:     "https://example.com/run/1",
			Created: created,
			Jobs: []ci.Job{
				{Name: "build", State: ci.Success},
				{Stage: "test", Name: "unit", State: ci.Failed},
			},
		},
		{
			Name:  "release",
			State: ci.Failed,
			URL:   "https://example.com/run/2",
		},
	}

	renderRuns(buf, "Branch", "master", runs)
	out := buf.String()

	if !strings.Contains(out, "Branch master") {
		t.Errorf("output missing header, got:\n%s", out)
	}
	if strings.ContainsRune(out, '\033') {
		t.Errorf("output contains ANSI escapes with NO_COLOR set, got:\n%q", out)
	}
	if !strings.Contains(out, "✓") || !strings.Contains(out, "✗") {
		t.Errorf("output missing expected glyphs, got:\n%s", out)
	}
	if !strings.Contains(out, "release") {
		t.Errorf("output missing run name, got:\n%s", out)
	}
	if !strings.Contains(out, "https://example.com/run/1") {
		t.Errorf("output missing run URL, got:\n%s", out)
	}
	if !strings.Contains(out, "2h ago") {
		t.Errorf("output missing relative age, got:\n%s", out)
	}
	if !strings.Contains(out, "      ✓ build") {
		t.Errorf("output missing indented job line for newest run, got:\n%s", out)
	}
	if !strings.Contains(out, "      ✗ test/unit") {
		t.Errorf("output missing indented job line with stage prefix, got:\n%s", out)
	}
	// The second (older) run's jobs must not be rendered.
	if strings.Count(out, "\n") != 5 {
		t.Errorf("expected 5 lines (header + 2 runs + 2 job lines), got %d in:\n%s", strings.Count(out, "\n"), out)
	}
}

func TestRenderRunsNoRuns(t *testing.T) {
	buf := withPlainUI(t)
	renderRuns(buf, "Tag", "v1.0.0", nil)
	out := buf.String()
	if !strings.Contains(out, "Tag v1.0.0") || !strings.Contains(out, "no pipelines found") {
		t.Errorf("renderRuns with no runs = %q", out)
	}
}

func TestJobLabel(t *testing.T) {
	tests := []struct {
		job  ci.Job
		want string
	}{
		{ci.Job{Name: "build"}, "build"},
		{ci.Job{Stage: "test", Name: "unit"}, "test/unit"},
	}
	for _, tt := range tests {
		if got := jobLabel(tt.job); got != tt.want {
			t.Errorf("jobLabel(%+v) = %q, want %q", tt.job, got, tt.want)
		}
	}
}
