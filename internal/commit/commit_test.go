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
