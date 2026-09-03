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
		"symbolic-ref -q refs/remotes/origin/HEAD":    {err: errors.New("no head")},
		"show-ref --verify --quiet refs/heads/main":   {err: errors.New("no")},
		"show-ref --verify --quiet refs/heads/master": {out: ""},
	}}
	if got := DetectMainBranch(r); got != "master" {
		t.Errorf("fallback master: got %q", got)
	}
}

func TestBranchExists(t *testing.T) {
	// exists locally
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"rev-parse --verify --quiet refs/heads/fix-detect": {out: "abc123"},
	}}
	if !BranchExists(r, "fix-detect") {
		t.Error("expected branch to exist locally")
	}

	// exists on origin
	r2 := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"rev-parse --verify --quiet refs/heads/fix-detect":          {err: errors.New("not found")},
		"rev-parse --verify --quiet refs/remotes/origin/fix-detect": {out: "def456"},
	}}
	if !BranchExists(r2, "fix-detect") {
		t.Error("expected branch to exist on origin")
	}

	// does not exist
	r3 := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"rev-parse --verify --quiet refs/heads/fix-detect":          {err: errors.New("not found")},
		"rev-parse --verify --quiet refs/remotes/origin/fix-detect": {err: errors.New("not found")},
	}}
	if BranchExists(r3, "fix-detect") {
		t.Error("expected branch to not exist")
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

func TestHasCommitsSince(t *testing.T) {
	tests := []struct {
		name    string
		out     string
		runErr  error
		want    bool
		wantErr bool
	}{
		{name: "ahead", out: "2", want: true},
		{name: "not ahead", out: "0"},
		{name: "git error", runErr: errors.New("bad revision"), wantErr: true},
		{name: "invalid count", out: "many", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fakeRunner{responses: map[string]struct {
				out string
				err error
			}{
				"rev-list --count main..HEAD": {out: tt.out, err: tt.runErr},
			}}
			got, err := HasCommitsSince(r, "main")
			if (err != nil) != tt.wantErr {
				t.Fatalf("HasCommitsSince() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("HasCommitsSince() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTags(t *testing.T) {
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"tag --list": {out: "v0.1.0\nv0.2.0\n\nv0.3.0"},
	}}
	got, err := Tags(r)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"v0.1.0", "v0.2.0", "v0.3.0"}
	if len(got) != len(want) {
		t.Fatalf("Tags() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Tags()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTags_Empty(t *testing.T) {
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"tag --list": {out: ""},
	}}
	got, err := Tags(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("Tags() = %v, want empty", got)
	}
}

func TestLatestTag_ReturnsEmptyOnError(t *testing.T) {
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"describe --tags --abbrev=0": {err: errors.New("no tags found")},
	}}
	got, err := LatestTag(r)
	if err != nil {
		t.Fatalf("LatestTag() error = %v, want nil", err)
	}
	if got != "" {
		t.Errorf("LatestTag() = %q, want empty string", got)
	}
}

func TestLatestTag_Found(t *testing.T) {
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"describe --tags --abbrev=0": {out: "v1.2.3"},
	}}
	got, err := LatestTag(r)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.2.3" {
		t.Errorf("LatestTag() = %q, want v1.2.3", got)
	}
}

func TestLogRange_WithFrom(t *testing.T) {
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"log v1.0.0..HEAD --no-merges --pretty=format:- %s%n%b": {out: "- fix bug\n- add feature\n"},
	}}
	got, err := LogRange(r, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "- fix bug\n- add feature" {
		t.Errorf("LogRange() = %q", got)
	}
}

func TestLogRange_NoFrom(t *testing.T) {
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"log --no-merges --pretty=format:- %s%n%b": {out: "- initial commit"},
	}}
	got, err := LogRange(r, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "- initial commit" {
		t.Errorf("LogRange() = %q", got)
	}
}

func TestLastCommitMessage(t *testing.T) {
	r := &fakeRunner{responses: map[string]struct {
		out string
		err error
	}{
		"log -1 --pretty=%B": {out: "feat: existing branch\n\nDetails"},
	}}
	got, err := LastCommitMessage(r)
	if err != nil {
		t.Fatal(err)
	}
	if got != "feat: existing branch\n\nDetails" {
		t.Errorf("LastCommitMessage() = %q", got)
	}
}
