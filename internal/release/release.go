// Package release derives the next semver tag and parses AI release output.
package release

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Bump is a semantic-version increment level.
type Bump int

const (
	// Patch is a backwards-compatible bug fix release.
	Patch Bump = iota
	// Minor is a backwards-compatible feature release.
	Minor
	// Major is a breaking-change release.
	Major
)

// String returns "patch", "minor" or "major".
func (b Bump) String() string {
	switch b {
	case Major:
		return "major"
	case Minor:
		return "minor"
	default:
		return "patch"
	}
}

// ParseBump parses "patch"/"minor"/"major" case-insensitively.
func ParseBump(s string) (Bump, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "patch":
		return Patch, true
	case "minor":
		return Minor, true
	case "major":
		return Major, true
	default:
		return Patch, false
	}
}

// Version is a parsed semantic version.
type Version struct{ Major, Minor, Patch int }

// String formats the version as "MAJOR.MINOR.PATCH" (no prefix).
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Bumped returns v incremented at level b (minor/major zero the lower parts).
func (v Version) Bumped(b Bump) Version {
	switch b {
	case Major:
		return Version{Major: v.Major + 1}
	case Minor:
		return Version{Major: v.Major, Minor: v.Minor + 1}
	default:
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	}
}

// compare returns -1, 0 or 1 as v is numerically less than, equal to, or
// greater than o.
func (v Version) compare(o Version) int {
	if v.Major != o.Major {
		return signOf(v.Major - o.Major)
	}
	if v.Minor != o.Minor {
		return signOf(v.Minor - o.Minor)
	}
	return signOf(v.Patch - o.Patch)
}

func signOf(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	default:
		return 0
	}
}

// tagPattern matches an optional non-numeric prefix followed by exactly
// MAJOR.MINOR.PATCH, each a plain (unsigned) digit run, with nothing else
// trailing (no pre-release/build suffix, no extra dotted parts).
var tagPattern = regexp.MustCompile(`^([^0-9]*)([0-9]+)\.([0-9]+)\.([0-9]+)$`)

// ParseTag splits a tag like "v1.2.3" into its prefix ("v") and version.
// ok is false for anything that is not <optional prefix>MAJOR.MINOR.PATCH.
func ParseTag(tag string) (prefix string, v Version, ok bool) {
	m := tagPattern.FindStringSubmatch(tag)
	if m == nil {
		return "", Version{}, false
	}
	major, errMajor := strconv.Atoi(m[2])
	minor, errMinor := strconv.Atoi(m[3])
	patch, errPatch := strconv.Atoi(m[4])
	if errMajor != nil || errMinor != nil || errPatch != nil {
		return "", Version{}, false
	}
	return m[1], Version{Major: major, Minor: minor, Patch: patch}, true
}

// Latest returns the highest semver tag from the list, its prefix and version.
// ok is false when the list contains no parseable semver tag. On a tie in
// version, the first matching tag encountered wins.
func Latest(tags []string) (tag, prefix string, v Version, ok bool) {
	for _, t := range tags {
		p, parsed, valid := ParseTag(t)
		if !valid {
			continue
		}
		if !ok || parsed.compare(v) > 0 {
			tag, prefix, v, ok = t, p, parsed, true
		}
	}
	return tag, prefix, v, ok
}

// NextTag returns the tag to create. With no existing semver tag it returns
// defaultPrefix+"0.0.1" regardless of b; otherwise the highest tag bumped at b,
// keeping that tag's prefix.
func NextTag(tags []string, b Bump, defaultPrefix string) string {
	_, prefix, v, ok := Latest(tags)
	if !ok {
		return defaultPrefix + "0.0.1"
	}
	return prefix + v.Bumped(b).String()
}

// bumpLinePattern matches a "BUMP: <level>" line, tolerating extra spaces
// around the colon and level.
var bumpLinePattern = regexp.MustCompile(`(?i)^bump:\s*(patch|minor|major)\s*$`)

// fencePattern matches a leading markdown code-fence line, e.g. "```" or "```text".
var fencePattern = regexp.MustCompile("^```[a-zA-Z0-9]*$")

// ParseAIResponse extracts the bump and the release notes from a provider
// reply shaped like "BUMP: minor\n---\n<notes>". When the reply does not follow
// that shape it returns Patch and the whole trimmed reply as the notes.
func ParseAIResponse(out string) (Bump, string) {
	text := strings.ReplaceAll(out, "\r\n", "\n")
	text = stripCodeFence(text)
	fallback := strings.TrimSpace(text)

	lines := strings.Split(text, "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) {
		return Patch, fallback
	}

	m := bumpLinePattern.FindStringSubmatch(strings.TrimSpace(lines[i]))
	if m == nil {
		return Patch, fallback
	}
	bump, _ := ParseBump(m[1])

	j := i + 1
	for j < len(lines) && strings.TrimSpace(lines[j]) != "---" {
		j++
	}
	if j >= len(lines) {
		return Patch, fallback
	}

	notes := strings.TrimSpace(strings.Join(lines[j+1:], "\n"))
	return bump, notes
}

// stripCodeFence removes a leading ```/```lang fence line and a matching
// trailing ``` fence line, if present, ignoring surrounding blank lines.
func stripCodeFence(text string) string {
	lines := strings.Split(text, "\n")
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines) - 1
	for end >= 0 && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if start > end {
		return text
	}

	newStart, newEnd := start, end
	changed := false
	if fencePattern.MatchString(strings.TrimSpace(lines[start])) {
		newStart++
		changed = true
	}
	if newEnd > newStart && strings.TrimSpace(lines[end]) == "```" {
		newEnd--
		changed = true
	}
	if !changed {
		return text
	}
	return strings.Join(lines[newStart:newEnd+1], "\n")
}
