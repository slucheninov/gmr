// Package commit contains helpers to derive MR/PR metadata from a commit message.
package commit

import "strings"

// Title returns the first line of a commit message.
func Title(msg string) string {
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		return strings.TrimRight(msg[:i], "\r")
	}
	return msg
}

// Body returns everything after the first line, with surrounding whitespace
// removed. It is empty when the commit message has no body.
func Body(msg string) string {
	i := strings.IndexByte(msg, '\n')
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(msg[i+1:])
}

// MRDescription mirrors the Bash build_mr_description: when the commit has a
// body it is returned verbatim, otherwise a short "## Summary" description is
// generated from the title.
func MRDescription(msg string) string {
	if body := Body(msg); body != "" {
		return body + "\n"
	}
	return "## Summary\n\n" + Title(msg) + "\n"
}
