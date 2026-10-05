package main

import (
	"errors"
	"strings"
	"testing"
)

func TestPushUnpushed(t *testing.T) {
	t.Parallel()
	const showRef = "show-ref --verify --quiet refs/remotes/origin/feat/x"
	tests := []struct {
		name      string
		responses map[string]fakeGitResponse
		wantCalls []string
		wantErr   bool
	}{
		{
			name: "branch not on origin is pushed",
			responses: map[string]fakeGitResponse{
				showRef:                 {err: errors.New("not found")},
				"push -u origin feat/x": {},
			},
			wantCalls: []string{showRef, "push -u origin feat/x"},
		},
		{
			name: "unpushed commits are pushed",
			responses: map[string]fakeGitResponse{
				showRef:                                  {},
				"rev-list --count origin/feat/x..feat/x": {out: "2"},
				"push -u origin feat/x":                  {},
			},
			wantCalls: []string{showRef, showRef, "rev-list --count origin/feat/x..feat/x", "push -u origin feat/x"},
		},
		{
			name: "up to date is an error",
			responses: map[string]fakeGitResponse{
				showRef:                                  {},
				"rev-list --count origin/feat/x..feat/x": {out: "0"},
			},
			wantCalls: []string{showRef, showRef, "rev-list --count origin/feat/x..feat/x"},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &fakeGitRunner{responses: tt.responses}
			err := pushUnpushed(r, "feat/x", "main")
			if (err != nil) != tt.wantErr {
				t.Fatalf("pushUnpushed() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertEqualCmds(t, r.calls, tt.wantCalls)
		})
	}
}

func TestRunCommitPush_EarlyExits(t *testing.T) {
	t.Parallel()

	t.Run("detached HEAD", func(t *testing.T) {
		t.Parallel()
		r := &fakeGitRunner{responses: map[string]fakeGitResponse{
			"branch --show-current": {out: ""},
		}}
		err := runCommitPush(r, "main")
		if err == nil || !strings.Contains(err.Error(), "detached HEAD") {
			t.Fatalf("runCommitPush() error = %v, want detached HEAD error", err)
		}
	})

	t.Run("clean tree up to date", func(t *testing.T) {
		t.Parallel()
		r := &fakeGitRunner{responses: map[string]fakeGitResponse{
			"branch --show-current":                              {out: "main"},
			"status --porcelain":                                 {out: ""},
			"show-ref --verify --quiet refs/remotes/origin/main": {},
			"rev-list --count origin/main..main":                 {out: "0"},
		}}
		err := runCommitPush(r, "main")
		if err == nil || !strings.Contains(err.Error(), "nothing to commit or push") {
			t.Fatalf("runCommitPush() error = %v, want nothing-to-push error", err)
		}
	})

	t.Run("clean tree with unpushed commits on base", func(t *testing.T) {
		t.Parallel()
		r := &fakeGitRunner{responses: map[string]fakeGitResponse{
			"branch --show-current":                              {out: "main"},
			"status --porcelain":                                 {out: ""},
			"show-ref --verify --quiet refs/remotes/origin/main": {},
			"rev-list --count origin/main..main":                 {out: "1"},
			"push -u origin main":                                {},
		}}
		if err := runCommitPush(r, "main"); err != nil {
			t.Fatalf("runCommitPush() error = %v", err)
		}
		if last := r.calls[len(r.calls)-1]; last != "push -u origin main" {
			t.Errorf("last git call = %q, want push", last)
		}
	})
}
