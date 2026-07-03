package commit

import (
	"regexp"
	"strings"
	"unicode"
)

// conventionalRe matches "type(scope)!: description" or "type!: description" or "type: description"
var conventionalRe = regexp.MustCompile(`^([a-zA-Z]+)(?:\([^)]*\))?!?:\s*(.+)$`)

// BranchName derives a branch name like "fix-detect" from a commit title.
// Rules:
//   - Input is the commit message title (first line). Whitespace is trimmed.
//   - If it matches conventional-commit form type(scope)!: description — word1 = type
//     (lowercase, scope and ! stripped), word2 = first word of description.
//   - Otherwise fallback: take the first two usable words of the title.
//   - Each word is normalized: lowercased, keep only [a-z0-9] runes.
//     Words that become empty after normalization are skipped.
//   - If only one usable word exists, return just that word.
//   - If zero usable words, return "" (caller falls back to timestamp).
func BranchName(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}

	var words []string

	if m := conventionalRe.FindStringSubmatch(title); m != nil {
		// m[1] = type, m[2] = description
		typeWord := normalize(m[1])
		// split description and take first word
		descWords := strings.Fields(m[2])
		var descWord string
		for _, w := range descWords {
			if n := normalize(w); n != "" {
				descWord = n
				break
			}
		}
		if typeWord != "" {
			words = append(words, typeWord)
		}
		if descWord != "" {
			words = append(words, descWord)
		}
	} else {
		// fallback: first two usable words of title
		fields := strings.FieldsFunc(title, func(r rune) bool {
			return unicode.IsSpace(r)
		})
		for _, f := range fields {
			if n := normalize(f); n != "" {
				words = append(words, n)
				if len(words) == 2 {
					break
				}
			}
		}
	}

	return strings.Join(words, "-")
}

// normalize lowercases a word and keeps only [a-z0-9] runes.
func normalize(w string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(w) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
