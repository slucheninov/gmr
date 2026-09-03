package ci

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type call struct {
	name string
	args []string
}

type fakeRunner struct {
	out   string
	err   error
	calls []call
}

func (f *fakeRunner) Run(name string, args ...string) (string, error) {
	f.calls = append(f.calls, call{name: name, args: append([]string{}, args...)})
	return f.out, f.err
}

func joinArgs(args []string) string {
	return strings.Join(args, " ")
}

// ---- State ----

func TestState_Done(t *testing.T) {
	cases := []struct {
		state State
		want  bool
	}{
		{Success, true},
		{Failed, true},
		{Canceled, true},
		{Skipped, true},
		{Running, false},
		{Pending, false},
		{Unknown, false},
	}
	for _, c := range cases {
		if got := c.state.Done(); got != c.want {
			t.Errorf("State(%q).Done() = %v, want %v", c.state, got, c.want)
		}
	}
}

func TestState_OK(t *testing.T) {
	cases := []struct {
		state State
		want  bool
	}{
		{Success, true},
		{Skipped, true},
		{Failed, false},
		{Running, false},
		{Pending, false},
		{Canceled, false},
		{Unknown, false},
	}
	for _, c := range cases {
		if got := c.state.OK(); got != c.want {
			t.Errorf("State(%q).OK() = %v, want %v", c.state, got, c.want)
		}
	}
}

// ---- state mapping tables ----

func TestGithubState(t *testing.T) {
	cases := []struct {
		status     string
		conclusion string
		want       State
	}{
		{"completed", "success", Success},
		{"completed", "failure", Failed},
		{"completed", "cancelled", Canceled},
		{"completed", "skipped", Skipped},
		{"completed", "timed_out", Failed},
		{"completed", "startup_failure", Failed},
		{"completed", "action_required", Failed},
		{"completed", "neutral", Unknown},
		{"queued", "", Pending},
		{"waiting", "", Pending},
		{"requested", "", Pending},
		{"pending", "", Pending},
		{"in_progress", "", Running},
		{"weird_status", "", Unknown},
	}
	for _, c := range cases {
		if got := githubState(c.status, c.conclusion); got != c.want {
			t.Errorf("githubState(%q, %q) = %q, want %q", c.status, c.conclusion, got, c.want)
		}
	}
}

func TestGitlabState(t *testing.T) {
	cases := []struct {
		status string
		want   State
	}{
		{"success", Success},
		{"failed", Failed},
		{"running", Running},
		{"pending", Pending},
		{"created", Pending},
		{"waiting_for_resource", Pending},
		{"preparing", Pending},
		{"scheduled", Pending},
		{"manual", Pending},
		{"canceled", Canceled},
		{"canceling", Canceled},
		{"skipped", Skipped},
		{"something_else", Unknown},
	}
	for _, c := range cases {
		if got := gitlabState(c.status); got != c.want {
			t.Errorf("gitlabState(%q) = %q, want %q", c.status, got, c.want)
		}
	}
}

// ---- GitHubRuns argv ----

func TestGitHubRuns_ArgvNoRef(t *testing.T) {
	r := &fakeRunner{out: "[]"}
	if _, err := GitHubRuns(r, "", 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(r.calls))
	}
	got := r.calls[0]
	if got.name != "gh" {
		t.Errorf("name = %q, want gh", got.name)
	}
	want := "run list --limit 10 --json databaseId,displayTitle,workflowName,headBranch,status,conclusion,url,createdAt"
	if joinArgs(got.args) != want {
		t.Errorf("args = %q, want %q", joinArgs(got.args), want)
	}
}

func TestGitHubRuns_ArgvWithRef(t *testing.T) {
	r := &fakeRunner{out: "[]"}
	if _, err := GitHubRuns(r, "feature/x", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "run list --limit 5 --json databaseId,displayTitle,workflowName,headBranch,status,conclusion,url,createdAt --branch feature/x"
	if joinArgs(r.calls[0].args) != want {
		t.Errorf("args = %q, want %q", joinArgs(r.calls[0].args), want)
	}
}

func TestGitHubRuns_LimitClamp(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "1"},
		{-5, "1"},
		{1, "1"},
		{100, "100"},
		{101, "100"},
		{9999, "100"},
	}
	for _, c := range cases {
		r := &fakeRunner{out: "[]"}
		if _, err := GitHubRuns(r, "", c.in); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		args := r.calls[0].args
		var got string
		for i, a := range args {
			if a == "--limit" && i+1 < len(args) {
				got = args[i+1]
			}
		}
		if got != c.want {
			t.Errorf("limit(%d): got %q, want %q", c.in, got, c.want)
		}
	}
}

// ---- GitHubRuns happy path ----

func TestGitHubRuns_Decode(t *testing.T) {
	out := `[
		{"databaseId":123456789,"displayTitle":"a title","workflowName":"CI","headBranch":"main","status":"completed","conclusion":"success","url":"https://github.com/o/r/actions/runs/123456789","createdAt":"2026-01-02T03:04:05Z"},
		{"databaseId":42,"displayTitle":"fallback title","workflowName":"","headBranch":"dev","status":"in_progress","conclusion":"","url":"https://github.com/o/r/actions/runs/42","createdAt":"2026-01-01T00:00:00Z"}
	]`
	r := &fakeRunner{out: out}
	runs, err := GitHubRuns(r, "", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}

	run0 := runs[0]
	if run0.ID != "123456789" {
		t.Errorf("run0.ID = %q, want 123456789", run0.ID)
	}
	if run0.Name != "CI" {
		t.Errorf("run0.Name = %q, want CI", run0.Name)
	}
	if run0.Ref != "main" {
		t.Errorf("run0.Ref = %q, want main", run0.Ref)
	}
	if run0.State != Success {
		t.Errorf("run0.State = %q, want success", run0.State)
	}
	if run0.URL != "https://github.com/o/r/actions/runs/123456789" {
		t.Errorf("run0.URL = %q", run0.URL)
	}
	wantTime, _ := time.Parse(time.RFC3339, "2026-01-02T03:04:05Z")
	if !run0.Created.Equal(wantTime) {
		t.Errorf("run0.Created = %v, want %v", run0.Created, wantTime)
	}

	run1 := runs[1]
	if run1.ID != "42" {
		t.Errorf("run1.ID = %q, want 42", run1.ID)
	}
	if run1.Name != "fallback title" {
		t.Errorf("run1.Name = %q, want fallback title (from displayTitle)", run1.Name)
	}
	if run1.State != Running {
		t.Errorf("run1.State = %q, want running", run1.State)
	}
}

func TestGitHubJobs_Decode(t *testing.T) {
	out := `{"jobs":[{"name":"build","status":"completed","conclusion":"success"},{"name":"test","status":"completed","conclusion":"failure"}]}`
	r := &fakeRunner{out: out}
	jobs, err := GitHubJobs(r, "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].Name != "build" || jobs[0].State != Success || jobs[0].Stage != "" {
		t.Errorf("jobs[0] = %+v", jobs[0])
	}
	if jobs[1].Name != "test" || jobs[1].State != Failed {
		t.Errorf("jobs[1] = %+v", jobs[1])
	}
	wantArgs := "run view 123 --json jobs"
	if joinArgs(r.calls[0].args) != wantArgs {
		t.Errorf("args = %q, want %q", joinArgs(r.calls[0].args), wantArgs)
	}
}

// ---- GitLabPipelines argv / URL encoding ----

func TestGitLabPipelines_ArgvEncodingAndRef(t *testing.T) {
	r := &fakeRunner{out: "[]"}
	if _, err := GitLabPipelines(r, "group/sub/project", "main", 15); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(r.calls))
	}
	got := r.calls[0]
	if got.name != "glab" {
		t.Errorf("name = %q, want glab", got.name)
	}
	if len(got.args) != 2 || got.args[0] != "api" {
		t.Fatalf("args = %v, want [api <path>]", got.args)
	}
	path := got.args[1]
	if !strings.Contains(path, "group%2Fsub%2Fproject") {
		t.Errorf("path %q does not contain encoded project path with %%2F", path)
	}
	if !strings.Contains(path, "ref=main") {
		t.Errorf("path %q does not contain ref=main", path)
	}
	if !strings.Contains(path, "per_page=15") {
		t.Errorf("path %q does not contain per_page=15", path)
	}
	if strings.Contains(path, "group/sub/project") {
		t.Errorf("path %q still contains unescaped slashes in project path", path)
	}
}

func TestGitLabPipelines_ArgvNoRef(t *testing.T) {
	r := &fakeRunner{out: "[]"}
	if _, err := GitLabPipelines(r, "group/project", "", 30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := r.calls[0].args[1]
	if strings.Contains(path, "ref=") {
		t.Errorf("path %q should not contain ref= when ref is empty", path)
	}
	want := "projects/group%2Fproject/pipelines?per_page=30"
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestGitLabPipelines_Decode(t *testing.T) {
	out := `[
		{"id":987,"status":"success","ref":"main","web_url":"https://gitlab.com/g/p/-/pipelines/987","created_at":"2026-02-03T04:05:06Z","source":"push"},
		{"id":10,"status":"failed","ref":"dev","web_url":"https://gitlab.com/g/p/-/pipelines/10","created_at":"2026-02-01T00:00:00Z","source":"schedule"}
	]`
	r := &fakeRunner{out: out}
	runs, err := GitLabPipelines(r, "g/p", "", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	run0 := runs[0]
	if run0.ID != "987" {
		t.Errorf("run0.ID = %q, want 987", run0.ID)
	}
	if run0.Name != "push" {
		t.Errorf("run0.Name = %q, want push", run0.Name)
	}
	if run0.Ref != "main" {
		t.Errorf("run0.Ref = %q, want main", run0.Ref)
	}
	if run0.State != Success {
		t.Errorf("run0.State = %q, want success", run0.State)
	}
	if run0.URL != "https://gitlab.com/g/p/-/pipelines/987" {
		t.Errorf("run0.URL = %q", run0.URL)
	}
	wantTime, _ := time.Parse(time.RFC3339, "2026-02-03T04:05:06Z")
	if !run0.Created.Equal(wantTime) {
		t.Errorf("run0.Created = %v, want %v", run0.Created, wantTime)
	}
	if runs[1].State != Failed {
		t.Errorf("run1.State = %q, want failed", runs[1].State)
	}
}

func TestGitLabJobs_Decode(t *testing.T) {
	out := `[{"name":"build","stage":"build","status":"success"},{"name":"deploy","stage":"deploy","status":"canceled"}]`
	r := &fakeRunner{out: out}
	jobs, err := GitLabJobs(r, "g/p", "987")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].Name != "build" || jobs[0].Stage != "build" || jobs[0].State != Success {
		t.Errorf("jobs[0] = %+v", jobs[0])
	}
	if jobs[1].Name != "deploy" || jobs[1].Stage != "deploy" || jobs[1].State != Canceled {
		t.Errorf("jobs[1] = %+v", jobs[1])
	}
	wantPath := "projects/g%2Fp/pipelines/987/jobs"
	if r.calls[0].args[1] != wantPath {
		t.Errorf("path = %q, want %q", r.calls[0].args[1], wantPath)
	}
}

// ---- empty results ----

func TestGitHubRuns_Empty(t *testing.T) {
	r := &fakeRunner{out: "[]"}
	runs, err := GitHubRuns(r, "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected empty slice, got %v", runs)
	}
}

func TestGitLabPipelines_Empty(t *testing.T) {
	r := &fakeRunner{out: "[]"}
	runs, err := GitLabPipelines(r, "g/p", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected empty slice, got %v", runs)
	}
}

// ---- error handling ----

func TestGitHubRuns_NonZeroExit(t *testing.T) {
	r := &fakeRunner{out: "gh: authentication required", err: errors.New("exit status 1")}
	_, err := GitHubRuns(r, "", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "authentication required") {
		t.Errorf("error %q does not contain CLI output", err.Error())
	}
}

func TestGitLabPipelines_NonZeroExit(t *testing.T) {
	r := &fakeRunner{out: "401 Unauthorized", err: errors.New("exit status 1")}
	_, err := GitLabPipelines(r, "g/p", "", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401 Unauthorized") {
		t.Errorf("error %q does not contain CLI output", err.Error())
	}
}

func TestGitHubRuns_NonZeroExit_TruncatesLongOutput(t *testing.T) {
	longOut := strings.Repeat("x", 1000)
	r := &fakeRunner{out: longOut, err: errors.New("exit status 1")}
	_, err := GitHubRuns(r, "", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Count(err.Error(), "x") > 400 {
		t.Errorf("snippet not truncated to ~400 bytes, error contains %d x's: %q", strings.Count(err.Error(), "x"), err.Error())
	}
	if len(err.Error()) >= len(longOut) {
		t.Errorf("error message (%d bytes) not shorter than raw output (%d bytes), truncation not applied", len(err.Error()), len(longOut))
	}
}

func TestGitHubRuns_InvalidJSON(t *testing.T) {
	r := &fakeRunner{out: "not json at all"}
	_, err := GitHubRuns(r, "", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh") {
		t.Errorf("error %q should mention gh CLI", err.Error())
	}
}

func TestGitLabPipelines_InvalidJSON(t *testing.T) {
	r := &fakeRunner{out: "<html>not json</html>"}
	_, err := GitLabPipelines(r, "g/p", "", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "glab") {
		t.Errorf("error %q should mention glab CLI", err.Error())
	}
}

// ---- whitespace / BOM tolerance ----

func TestGitHubRuns_WhitespaceAndBOM(t *testing.T) {
	out := "\ufeff  \n[{\"databaseId\":1,\"displayTitle\":\"t\",\"workflowName\":\"w\",\"headBranch\":\"main\",\"status\":\"completed\",\"conclusion\":\"success\",\"url\":\"u\",\"createdAt\":\"2026-01-01T00:00:00Z\"}]\n\t  "
	r := &fakeRunner{out: out}
	runs, err := GitHubRuns(r, "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "1" {
		t.Errorf("runs = %+v", runs)
	}
}

// ---- regression: databaseId as JSON number ----

func TestGitHubRuns_DatabaseIDIsNumber(t *testing.T) {
	out := `[{"databaseId":9876543210,"displayTitle":"t","workflowName":"w","headBranch":"main","status":"completed","conclusion":"success","url":"u","createdAt":"2026-01-01T00:00:00Z"}]`
	r := &fakeRunner{out: out}
	runs, err := GitHubRuns(r, "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].ID != "9876543210" {
		t.Errorf("ID = %q, want 9876543210", runs[0].ID)
	}
}
