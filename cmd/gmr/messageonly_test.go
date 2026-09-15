package main

import (
	"strings"
	"testing"

	"github.com/slucheninov/gmr/internal/platform"
)

func TestParseMessageOnlyChoice(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  msgOnlyAction
	}{
		{"", actionCommit},
		{"\n", actionCommit},
		{"y\n", actionCommit},
		{"Y\n", actionCommit},
		{"yes\n", actionCommit},
		{"  yes  \n", actionCommit},
		{"n\n", actionPrintOnly},
		{"N\n", actionPrintOnly},
		{"no\n", actionPrintOnly},
		{"e\n", actionEdit},
		{"E\n", actionEdit},
		{"edit\n", actionEdit},
		{"anything\n", actionPrintOnly},
		{"q\n", actionPrintOnly},
	}
	for _, tt := range cases {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := parseMessageOnlyChoice(tt.input)
			if got != tt.want {
				t.Errorf("parseMessageOnlyChoice(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPromptMessageOnlyChoice(t *testing.T) {
	t.Parallel()
	got := promptMessageOnlyChoice(strings.NewReader("e\n"), "feature-x")
	if got != actionEdit {
		t.Errorf("promptMessageOnlyChoice() = %v, want actionEdit", got)
	}
}

func TestNextStepCommands(t *testing.T) {
	t.Parallel()

	t.Run("detached HEAD returns nil", func(t *testing.T) {
		t.Parallel()
		got := nextStepCommands(platform.GitHub, "", "", "main", "auto-20260101-000000")
		if got != nil {
			t.Errorf("nextStepCommands() = %v, want nil", got)
		}
	})

	t.Run("github feature branch", func(t *testing.T) {
		t.Parallel()
		got := nextStepCommands(platform.GitHub, "", "feat/thing", "main", "auto-20260101-000000")
		want := []string{
			"git push -u origin feat/thing",
			"gh pr create --fill --base main --head feat/thing",
			"gmr  # pushes and opens MR/PR for this branch's commits",
		}
		assertEqualCmds(t, got, want)
	})

	t.Run("gitlab feature branch", func(t *testing.T) {
		t.Parallel()
		got := nextStepCommands(platform.GitLab, "group/project", "feat/thing", "main", "auto-20260101-000000")
		want := []string{
			"git push -u origin feat/thing",
			"glab mr create -R group/project --source-branch feat/thing --target-branch main --fill --yes --remove-source-branch --squash-before-merge",
			"gmr  # pushes and opens MR/PR for this branch's commits",
		}
		assertEqualCmds(t, got, want)
	})

	t.Run("unknown platform feature branch skips create command", func(t *testing.T) {
		t.Parallel()
		got := nextStepCommands("", "", "feat/thing", "main", "auto-20260101-000000")
		want := []string{
			"git push -u origin feat/thing",
			"gmr  # pushes and opens MR/PR for this branch's commits",
		}
		assertEqualCmds(t, got, want)
	})

	t.Run("github main branch", func(t *testing.T) {
		t.Parallel()
		got := nextStepCommands(platform.GitHub, "", "main", "main", "fix-thing")
		want := []string{
			"git push origin main",
			"git switch -c fix-thing && git push -u origin fix-thing",
			"gh pr create --fill --base main --head fix-thing",
			"# Note: local main still has this commit; reset later with: git switch main && git reset --hard origin/main",
		}
		assertEqualCmds(t, got, want)
	})

	t.Run("gitlab main branch", func(t *testing.T) {
		t.Parallel()
		got := nextStepCommands(platform.GitLab, "group/project", "main", "main", "fix-thing")
		want := []string{
			"git push origin main",
			"git switch -c fix-thing && git push -u origin fix-thing",
			"glab mr create -R group/project --source-branch fix-thing --target-branch main --fill --yes --remove-source-branch --squash-before-merge",
			"# Note: local main still has this commit; reset later with: git switch main && git reset --hard origin/main",
		}
		assertEqualCmds(t, got, want)
	})

	t.Run("branch name needing quoting", func(t *testing.T) {
		t.Parallel()
		got := nextStepCommands(platform.GitHub, "", "feat/weird name", "main", "auto-20260101-000000")
		want := []string{
			"git push -u origin 'feat/weird name'",
			"gh pr create --fill --base main --head 'feat/weird name'",
			"gmr  # pushes and opens MR/PR for this branch's commits",
		}
		assertEqualCmds(t, got, want)
	})
}

func TestShellQuote(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"", "''"},
		{"feat/add-thing", "feat/add-thing"},
		{"fix-detect2", "fix-detect2"},
		{"my.branch_name", "my.branch_name"},
		{"has space", "'has space'"},
		{"has'quote", `'has'\''quote'`},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got := shellQuote(tt.in)
			if got != tt.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func assertEqualCmds(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d commands, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("cmd[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
