// Package commit contains helpers to derive MR/PR metadata from a commit message.
package commit

import (
	"regexp"
	"strings"
	"unicode"
)

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

// conventionalPrefixRe matches a leading Conventional Commits type prefix,
// e.g. "feat:", "fix(scope):", "feat!:", "feat(scope)!:".
var conventionalPrefixRe = regexp.MustCompile(`(?i)^(feat|fix|chore|docs|style|refactor|perf|test|tests|build|ci|revert)(\([^)]*\))?!?:\s*`)

// Humanize strips a leading Conventional Commits prefix from the commit title
// and capitalizes the first letter, leaving the body untouched. It is a no-op
// when the title has no known type prefix.
func Humanize(msg string) string {
	i := strings.IndexByte(msg, '\n')
	title := msg
	rest := ""
	if i >= 0 {
		title = msg[:i]
		rest = msg[i:]
	}
	cr := ""
	if strings.HasSuffix(title, "\r") {
		title = title[:len(title)-1]
		cr = "\r"
	}

	stripped := conventionalPrefixRe.ReplaceAllString(title, "")
	if stripped != title && stripped == "" {
		// The whole title was just a type prefix (e.g. "feat:"); nothing left
		// to humanize, so leave the message untouched.
		return msg
	}
	return capitalizeFirst(stripped) + cr + rest
}

// capitalizeFirst uppercases the first rune of s if it is a lowercase letter;
// any other first rune (already uppercase, digit, punctuation, ...) is left alone.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if unicode.IsLower(r[0]) {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
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
