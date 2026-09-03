package commit

import "testing"

func TestTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"feat: add x", "feat: add x"},
		{"feat: add x\n\nbody", "feat: add x"},
		{"feat: add x\r\n\r\nbody", "feat: add x"},
	}
	for _, c := range cases {
		if got := Title(c.in); got != c.want {
			t.Errorf("Title(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBody(t *testing.T) {
	cases := []struct{ in, want string }{
		{"feat: add x", ""},
		{"feat: add x\n\n", ""},
		{"feat: add x\n\nbody line", "body line"},
		{"feat: add x\n\n- one\n- two\n", "- one\n- two"},
	}
	for _, c := range cases {
		if got := Body(c.in); got != c.want {
			t.Errorf("Body(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMRDescription_WithBody(t *testing.T) {
	in := "feat: add x\n\n- bullet one\n- bullet two"
	got := MRDescription(in)
	want := "- bullet one\n- bullet two\n"
	if got != want {
		t.Errorf("MRDescription = %q, want %q", got, want)
	}
}

func TestMRDescription_NoBody(t *testing.T) {
	got := MRDescription("feat: add x")
	want := "## Summary\n\nfeat: add x\n"
	if got != want {
		t.Errorf("MRDescription = %q, want %q", got, want)
	}
}

func TestHumanize(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"feat prefix", "feat: add retries", "Add retries"},
		{"fix prefix", "fix: crash on empty diff", "Crash on empty diff"},
		{"chore prefix", "chore: bump deps", "Bump deps"},
		{"feat with scope", "feat(auth): add login", "Add login"},
		{"feat with bang", "feat!: breaking change", "Breaking change"},
		{"feat with scope and bang", "feat(auth)!: breaking change", "Breaking change"},
		{"uppercase type", "FIX: closed bug", "Closed bug"},
		{"no prefix", "add retries", "Add retries"},
		{"non-type prefix preserved", "Note: check this", "Note: check this"},
		{"non-type prefix preserved warn", "WARN: x", "WARN: x"},
		{"already capitalized", "Add feature", "Add feature"},
		{"body preserved", "feat: add x\n\n- one\n- two", "Add x\n\n- one\n- two"},
		{"crlf body preserved", "feat: add x\r\n\r\n- one\r\n- two", "Add x\r\n\r\n- one\r\n- two"},
		{"empty after strip", "feat:", "feat:"},
		{"empty string", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Humanize(c.in); got != c.want {
				t.Errorf("Humanize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
