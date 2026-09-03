// Package ci reports CI pipeline status from the GitHub and GitLab CLIs.
package ci

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// State is a normalized pipeline or job state.
type State string

const (
	Success  State = "success"
	Failed   State = "failed"
	Running  State = "running"
	Pending  State = "pending"
	Canceled State = "canceled"
	Skipped  State = "skipped"
	Unknown  State = "unknown"
)

// Done reports whether the state is terminal (no longer changing).
func (s State) Done() bool {
	switch s {
	case Success, Failed, Canceled, Skipped:
		return true
	default:
		return false
	}
}

// OK reports whether the state is a success (or skipped, which does not fail a run).
func (s State) OK() bool {
	return s == Success || s == Skipped
}

// Job is a single job inside a run.
type Job struct {
	Name  string
	Stage string // GitLab stage; empty for GitHub
	State State
}

// Run is one pipeline (GitLab) or workflow run (GitHub).
type Run struct {
	ID      string
	Name    string // workflow name / pipeline source
	Ref     string // branch or tag it ran on
	State   State
	URL     string
	Created time.Time
	Jobs    []Job // filled only by the *Jobs calls
}

// Runner executes an external CLI. Tests substitute a fake.
type Runner interface {
	Run(name string, args ...string) (string, error)
}

type execRunner struct{}

// NewRunner returns a Runner backed by os/exec that returns combined output.
func NewRunner() Runner { return execRunner{} }

func (execRunner) Run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// clampLimit clamps n to the [1, 100] range accepted by the GitHub/GitLab
// list endpoints.
func clampLimit(n int) int {
	if n <= 0 {
		return 1
	}
	if n > 100 {
		return 100
	}
	return n
}

// clean strips a leading UTF-8 BOM and surrounding whitespace from CLI output.
func clean(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\ufeff")
	return strings.TrimSpace(s)
}

// truncate shortens s to at most n bytes, for embedding untrusted CLI output
// in error messages without flooding the terminal.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// runCLI runs cliName with args, cleans the output, and turns a non-zero exit
// into an error that includes a bounded snippet of the CLI's own output.
func runCLI(r Runner, cliName string, args ...string) (string, error) {
	out, err := r.Run(cliName, args...)
	cleaned := clean(out)
	if err != nil {
		snippet := truncate(cleaned, 400)
		if snippet == "" {
			snippet = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", cliName, strings.Join(args, " "), snippet)
	}
	return cleaned, nil
}

// decodeJSON unmarshals data into v, wrapping decode failures with the CLI
// name and a short snippet of what was actually received.
func decodeJSON(cliName, data string, v interface{}) error {
	if err := json.Unmarshal([]byte(data), v); err != nil {
		return fmt.Errorf("%s: invalid JSON output: %w (got %q)", cliName, err, truncate(data, 200))
	}
	return nil
}

// ghRun mirrors the fields requested from `gh run list --json`.
type ghRun struct {
	DatabaseID   json.Number `json:"databaseId"`
	DisplayTitle string      `json:"displayTitle"`
	WorkflowName string      `json:"workflowName"`
	HeadBranch   string      `json:"headBranch"`
	Status       string      `json:"status"`
	Conclusion   string      `json:"conclusion"`
	URL          string      `json:"url"`
	CreatedAt    string      `json:"createdAt"`
}

// githubState normalizes a `gh` run's status/conclusion pair into a State.
func githubState(status, conclusion string) State {
	switch status {
	case "completed":
		switch conclusion {
		case "success":
			return Success
		case "failure":
			return Failed
		case "cancelled":
			return Canceled
		case "skipped":
			return Skipped
		case "timed_out", "startup_failure", "action_required":
			return Failed
		default:
			return Unknown
		}
	case "queued", "waiting", "requested", "pending":
		return Pending
	case "in_progress":
		return Running
	default:
		return Unknown
	}
}

// GitHubRuns returns the most recent workflow runs for ref, newest first.
// ref may be a branch or a tag name; an empty ref means all refs.
func GitHubRuns(r Runner, ref string, limit int) ([]Run, error) {
	limit = clampLimit(limit)
	args := []string{
		"run", "list",
		"--limit", strconv.Itoa(limit),
		"--json", "databaseId,displayTitle,workflowName,headBranch,status,conclusion,url,createdAt",
	}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	out, err := runCLI(r, "gh", args...)
	if err != nil {
		return nil, err
	}
	var raw []ghRun
	if err := decodeJSON("gh", out, &raw); err != nil {
		return nil, err
	}
	runs := make([]Run, 0, len(raw))
	for _, gr := range raw {
		name := gr.WorkflowName
		if name == "" {
			name = gr.DisplayTitle
		}
		created, _ := time.Parse(time.RFC3339, gr.CreatedAt)
		runs = append(runs, Run{
			ID:      gr.DatabaseID.String(),
			Name:    name,
			Ref:     gr.HeadBranch,
			State:   githubState(gr.Status, gr.Conclusion),
			URL:     gr.URL,
			Created: created,
		})
	}
	return runs, nil
}

// ghJobsResponse mirrors `gh run view <id> --json jobs`.
type ghJobsResponse struct {
	Jobs []ghJob `json:"jobs"`
}

type ghJob struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// GitHubJobs returns the jobs of one workflow run.
func GitHubJobs(r Runner, runID string) ([]Job, error) {
	out, err := runCLI(r, "gh", "run", "view", runID, "--json", "jobs")
	if err != nil {
		return nil, err
	}
	var resp ghJobsResponse
	if err := decodeJSON("gh", out, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		jobs = append(jobs, Job{
			Name:  j.Name,
			State: githubState(j.Status, j.Conclusion),
		})
	}
	return jobs, nil
}

// glPipeline mirrors one element of the GitLab pipelines API response.
type glPipeline struct {
	ID        json.Number `json:"id"`
	Status    string      `json:"status"`
	Ref       string      `json:"ref"`
	WebURL    string      `json:"web_url"`
	CreatedAt string      `json:"created_at"`
	Source    string      `json:"source"`
}

// gitlabState normalizes a GitLab pipeline/job status string into a State.
func gitlabState(status string) State {
	switch status {
	case "success":
		return Success
	case "failed":
		return Failed
	case "running":
		return Running
	case "pending", "created", "waiting_for_resource", "preparing", "scheduled", "manual":
		return Pending
	case "canceled", "canceling":
		return Canceled
	case "skipped":
		return Skipped
	default:
		return Unknown
	}
}

// encodeProjectPath percent-encodes a "group/project" path for use as a path
// segment in the GitLab REST API, turning "/" into "%2F" as the API requires.
func encodeProjectPath(projectPath string) string {
	return url.QueryEscape(projectPath)
}

// gitlabPipelinesPath builds the `projects/<id>/pipelines?...` API path.
func gitlabPipelinesPath(projectPath, ref string, limit int) string {
	var b strings.Builder
	b.WriteString("projects/")
	b.WriteString(encodeProjectPath(projectPath))
	b.WriteString("/pipelines?")
	if ref != "" {
		b.WriteString("ref=")
		b.WriteString(url.QueryEscape(ref))
		b.WriteString("&")
	}
	b.WriteString("per_page=")
	b.WriteString(strconv.Itoa(limit))
	return b.String()
}

// GitLabPipelines returns the most recent pipelines for ref, newest first.
// projectPath is the "group/project" path.
func GitLabPipelines(r Runner, projectPath, ref string, limit int) ([]Run, error) {
	limit = clampLimit(limit)
	out, err := runCLI(r, "glab", "api", gitlabPipelinesPath(projectPath, ref, limit))
	if err != nil {
		return nil, err
	}
	var raw []glPipeline
	if err := decodeJSON("glab", out, &raw); err != nil {
		return nil, err
	}
	runs := make([]Run, 0, len(raw))
	for _, p := range raw {
		created, _ := time.Parse(time.RFC3339, p.CreatedAt)
		runs = append(runs, Run{
			ID:      p.ID.String(),
			Name:    p.Source,
			Ref:     p.Ref,
			State:   gitlabState(p.Status),
			URL:     p.WebURL,
			Created: created,
		})
	}
	return runs, nil
}

// glJob mirrors one element of the GitLab pipeline jobs API response.
type glJob struct {
	Name   string `json:"name"`
	Stage  string `json:"stage"`
	Status string `json:"status"`
}

// GitLabJobs returns the jobs of one pipeline.
func GitLabJobs(r Runner, projectPath, pipelineID string) ([]Job, error) {
	path := fmt.Sprintf("projects/%s/pipelines/%s/jobs", encodeProjectPath(projectPath), pipelineID)
	out, err := runCLI(r, "glab", "api", path)
	if err != nil {
		return nil, err
	}
	var raw []glJob
	if err := decodeJSON("glab", out, &raw); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(raw))
	for _, j := range raw {
		jobs = append(jobs, Job{
			Name:  j.Name,
			Stage: j.Stage,
			State: gitlabState(j.Status),
		})
	}
	return jobs, nil
}
