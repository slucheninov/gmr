package main

import (
	"bytes"
	"errors"
	"os"

	"github.com/slucheninov/gmr/internal/git"
	"github.com/slucheninov/gmr/internal/ui"

	"strings"
	"testing"
)

func TestParseCommitChoice(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  commitAction
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
			got := parseCommitChoice(tt.input)
			if got != tt.want {
				t.Errorf("parseCommitChoice(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPromptCommitChoice(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  commitAction
	}{
		{"edit", "e\n", actionEdit},
		{"enter commits", "\n", actionCommit},
		// EOF without an answer (stdin is /dev/null, Ctrl+D at the prompt)
		// must never be taken as consent.
		{"EOF only", "", actionPrintOnly},
		{"answer without newline then EOF", "y", actionPrintOnly},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := promptCommitChoice(strings.NewReader(tt.input), "Commit to 'feature-x'?")
			if got != tt.want {
				t.Errorf("promptCommitChoice(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// These tests replace process-wide streams and EDITOR, so they must stay serial.
func TestCommitChoiceWorkflows(t *testing.T) {
	failure := errors.New("command failed")
	modes := []struct {
		name          string
		run           func(git.Runner, commitAction, string, string, string) error
		afterCommit   []string
		successText   string
		printOnlyText string
	}{
		{"message only", actOnMessageOnlyChoice, []string{"remote get-url origin"}, "Next steps:", "not committed)"},
		{"commit and push", actOnCommitPushChoice, []string{"push -u origin feat/x"}, "Pushed to origin/feat/x", "not committed, not pushed)"},
	}
	cases := []struct {
		name        string
		action      commitAction
		msg         string
		editor      string
		commitErr   error
		wantErr     error
		wantErrText string
		wantOutput  string
		wantCommit  bool
		wantSuccess bool
	}{
		{name: "print only", action: actionPrintOnly, msg: "Fix it", wantOutput: "Fix it\n"},
		{name: "commit", action: actionCommit, msg: "Fix it", wantOutput: "Fix it\n", wantCommit: true, wantSuccess: true},
		{name: "edit trims message", action: actionEdit, msg: "  Fix it\n", editor: "true", wantOutput: "Fix it\n", wantCommit: true, wantSuccess: true},
		{name: "empty edit aborts", action: actionEdit, msg: " \n", editor: "true", wantErrText: "commit message is empty"},
		{name: "editor failure aborts", action: actionEdit, msg: "Fix it", editor: "false", wantErrText: "exit status"},
		{name: "commit failure stops next steps", action: actionCommit, msg: "Fix it", commitErr: failure, wantErr: failure, wantOutput: "Fix it\n", wantCommit: true},
	}
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					if tc.editor != "" {
						t.Setenv("EDITOR", tc.editor)
					}
					stdout, err := os.CreateTemp(t.TempDir(), "stdout")
					if err != nil {
						t.Fatal(err)
					}
					var logs bytes.Buffer
					oldStdout, oldOut := os.Stdout, ui.Out
					os.Stdout, ui.Out = stdout, &logs
					t.Cleanup(func() { os.Stdout, ui.Out = oldStdout, oldOut; stdout.Close() })

					r := &fakeGitRunner{responses: map[string]fakeGitResponse{
						"commit -m Fix it":      {err: tc.commitErr},
						"push -u origin feat/x": {},
						"remote get-url origin": {out: "git@github.com:owner/repo.git"},
					}}
					err = mode.run(r, tc.action, tc.msg, "feat/x", "main")
					switch {
					case tc.wantErr != nil:
						if !errors.Is(err, tc.wantErr) {
							t.Fatalf("error = %v, want %v", err, tc.wantErr)
						}
					case tc.wantErrText != "":
						if err == nil || !strings.Contains(err.Error(), tc.wantErrText) {
							t.Fatalf("error = %v, want %q", err, tc.wantErrText)
						}
					case err != nil:
						t.Fatal(err)
					}
					var wantCalls []string
					if tc.wantCommit {
						wantCalls = append(wantCalls, "commit -m Fix it")
					}
					if tc.wantSuccess {
						wantCalls = append(wantCalls, mode.afterCommit...)
					}
					assertEqualCmds(t, r.calls, wantCalls)
					output, err := os.ReadFile(stdout.Name())
					if err != nil {
						t.Fatal(err)
					}
					if string(output) != tc.wantOutput {
						t.Errorf("stdout = %q, want %q", output, tc.wantOutput)
					}
					if got := strings.Contains(logs.String(), mode.successText); got != tc.wantSuccess {
						t.Errorf("success hint present = %v, want %v; logs: %s", got, tc.wantSuccess, logs.String())
					}
					if tc.action == actionPrintOnly && !strings.Contains(logs.String(), mode.printOnlyText) {
						t.Errorf("missing print-only status %q: %s", mode.printOnlyText, logs.String())
					}
				})
			}
		})
	}
}

func TestCommitPushFailure(t *testing.T) {
	t.Parallel()
	failure := errors.New("push rejected")
	r := &fakeGitRunner{responses: map[string]fakeGitResponse{
		"commit -m Fix it":      {},
		"push -u origin feat/x": {err: failure},
	}}
	if err := actOnCommitPushChoice(r, actionCommit, "Fix it", "feat/x", "main"); !errors.Is(err, failure) {
		t.Fatalf("error = %v, want %v", err, failure)
	}
	assertEqualCmds(t, r.calls, []string{"commit -m Fix it", "push -u origin feat/x"})
}
