package git

import (
	"errors"
	"strings"
	"testing"
)

type fakeRunner struct {
	responses map[string]struct {
		out string
		err error
	}
	calls []string
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	r, ok := f.responses[key]
	if !ok {
		return "", errors.New("unexpected call: " + key)
	}
	return r.out, r.err
}

func (f *fakeRunner) RunInteractive(args ...string) error { return nil }

func TestLimitLines(t *testing.T) {
	in := "a\nb\nc\nd\n"
	got, truncated := LimitLines(in, 2)
	if got != "a\nb\n" || !truncated {
		t.Errorf("LimitLines truncated case: got %q (truncated=%v)", got, truncated)
	}
	got, truncated = LimitLines(in, 10)
	if got != in || truncated {
		t.Errorf("LimitLines no-truncate: got %q (truncated=%v)", got, truncated)
	}
	got, _ = LimitLines("", 10)
	if got != "" {
		t.Errorf("LimitLines empty: got %q", got)
	}
}

func TestDetectMainBranch_Override(t *testing.T) {
	t.Setenv("GMR_MAIN_BRANCH", "develop")
	if got := DetectMainBranch(&fakeRunner{}); got != "develop" {
		t.Errorf("override: got %q, want develop", got)
	}
}

func TestDetectMainBranch_OriginHEAD(t *testing.T) {
	t.Setenv("GMR_MAIN_BRANCH", "")
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"symbolic-ref -q refs/remotes/origin/HEAD": {out: "refs/remotes/origin/main"},
	}}
	if got := DetectMainBranch(r); got != "main" {
		t.Errorf("origin/HEAD: got %q, want main", got)
	}
}

func TestDetectMainBranch_FallbackMaster(t *testing.T) {
	t.Setenv("GMR_MAIN_BRANCH", "")
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"symbolic-ref -q refs/remotes/origin/HEAD":   {err: errors.New("no head")},
		"show-ref --verify --quiet refs/heads/main":  {err: errors.New("no")},
		"show-ref --verify --quiet refs/heads/master": {out: ""},
	}}
	if got := DetectMainBranch(r); got != "master" {
		t.Errorf("fallback master: got %q", got)
	}
}

func TestHasChanges(t *testing.T) {
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"status --porcelain": {out: " M README.md"},
	}}
	yes, err := HasChanges(r)
	if err != nil || !yes {
		t.Errorf("expected changes detected; got yes=%v err=%v", yes, err)
	}

	r = &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"status --porcelain": {out: "?? examples/docker/etcd-cluster/"},
	}}
	yes, err = HasChanges(r)
	if err != nil || !yes {
		t.Errorf("expected untracked changes detected; got yes=%v err=%v", yes, err)
	}

	r = &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"status --porcelain": {out: ""},
	}}
	yes, err = HasChanges(r)
	if err != nil || yes {
		t.Errorf("expected no changes; got yes=%v err=%v", yes, err)
	}
}
