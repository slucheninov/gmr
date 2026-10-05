package main

import (
	"errors"
	"strings"
)

// fakeGitRunner is a git.Runner that answers from a fixed response table and
// records every call. Unknown calls fail, so tests catch unexpected git usage.
type fakeGitRunner struct {
	responses map[string]fakeGitResponse
	calls     []string
}

type fakeGitResponse struct {
	out string
	err error
}

func (f *fakeGitRunner) Run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	r, ok := f.responses[key]
	if !ok {
		return "", errors.New("unexpected call: " + key)
	}
	return r.out, r.err
}

func (f *fakeGitRunner) RunInteractive(args ...string) error {
	_, err := f.Run(args...)
	return err
}
