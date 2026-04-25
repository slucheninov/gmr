package platform

import (
	"fmt"
	"regexp"
	"strings"
)

// Kind identifies a hosting platform.
type Kind string

const (
	GitHub Kind = "github"
	GitLab Kind = "gitlab"
)

// Detect returns the platform inferred from a git remote URL.
func Detect(remoteURL string) (Kind, error) {
	switch {
	case strings.Contains(remoteURL, "github.com"):
		return GitHub, nil
	case strings.Contains(remoteURL, "gitlab"):
		return GitLab, nil
	default:
		return "", fmt.Errorf("unknown platform for remote: %s (expected github.com or gitlab)", remoteURL)
	}
}

var (
	gitlabSSHRe   = regexp.MustCompile(`^git@gitlab\.com:(.+?)(?:\.git)?$`)
	gitlabHTTPSRe = regexp.MustCompile(`^https://gitlab\.com/(.+?)(?:\.git)?/?$`)
)

// GitLabProjectPath extracts the canonical "group/project" path from a GitLab
// remote URL.
func GitLabProjectPath(remoteURL string) (string, error) {
	if m := gitlabSSHRe.FindStringSubmatch(remoteURL); m != nil {
		return m[1], nil
	}
	if m := gitlabHTTPSRe.FindStringSubmatch(remoteURL); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("cannot parse GitLab project path from remote: %s", remoteURL)
}
