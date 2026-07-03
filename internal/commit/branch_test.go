package commit

import "testing"

func TestBranchName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// conventional commit with scope
		{"fix(git): detect untracked files", "fix-detect"},
		// conventional commit without scope
		{"feat: add base URL overrides", "feat-add"},
		// conventional commit with !
		{"feat!: breaking change now", "feat-breaking"},
		// conventional commit with scope and !
		{"chore(deps)!: bump version to 2.0", "chore-bump"},
		// plain title, no conventional prefix
		{"Update installation docs", "update-installation"},
		// punctuation stripping
		{"fix: 'quoted' stuff", "fix-quoted"},
		// backtick stripping
		{"feat: `code` thing", "feat-code"},
		// Cyrillic only (no ASCII survives)
		{"фікс: оновлення", ""},
		// empty string
		{"", ""},
		// single word
		{"Refactor", "refactor"},
		// single word conventional (description word is empty after normalize)
		{"chore: Привіт", "chore"},
		// whitespace only
		{"   ", ""},
	}
	for _, c := range cases {
		got := BranchName(c.in)
		if got != c.want {
			t.Errorf("BranchName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
