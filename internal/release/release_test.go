package release

import "testing"

func TestBumpString(t *testing.T) {
	cases := []struct {
		b    Bump
		want string
	}{
		{Patch, "patch"},
		{Minor, "minor"},
		{Major, "major"},
	}
	for _, tt := range cases {
		if got := tt.b.String(); got != tt.want {
			t.Errorf("Bump(%d).String() = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestParseBump(t *testing.T) {
	cases := []struct {
		in     string
		want   Bump
		wantOK bool
	}{
		{"patch", Patch, true},
		{"minor", Minor, true},
		{"major", Major, true},
		{"PATCH", Patch, true},
		{"Minor", Minor, true},
		{"  major  ", Major, true},
		{"bogus", Patch, false},
		{"", Patch, false},
	}
	for _, tt := range cases {
		got, ok := ParseBump(tt.in)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("ParseBump(%q) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestVersionString(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	if got := v.String(); got != "1.2.3" {
		t.Errorf("Version.String() = %q, want 1.2.3", got)
	}
}

func TestVersionBumped(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	cases := []struct {
		b    Bump
		want Version
	}{
		{Patch, Version{1, 2, 4}},
		{Minor, Version{1, 3, 0}},
		{Major, Version{2, 0, 0}},
	}
	for _, tt := range cases {
		if got := v.Bumped(tt.b); got != tt.want {
			t.Errorf("Bumped(%v) = %v, want %v", tt.b, got, tt.want)
		}
	}
}

func TestParseTag_Valid(t *testing.T) {
	cases := []struct {
		tag        string
		wantPrefix string
		wantV      Version
	}{
		{"v1.2.3", "v", Version{1, 2, 3}},
		{"1.2.3", "", Version{1, 2, 3}},
		{"release-1.2.3", "release-", Version{1, 2, 3}},
		{"v01.2.3", "v", Version{1, 2, 3}},
		{"v0.10.0", "v", Version{0, 10, 0}},
	}
	for _, tt := range cases {
		prefix, v, ok := ParseTag(tt.tag)
		if !ok {
			t.Errorf("ParseTag(%q): ok = false, want true", tt.tag)
			continue
		}
		if prefix != tt.wantPrefix || v != tt.wantV {
			t.Errorf("ParseTag(%q) = (%q, %v), want (%q, %v)", tt.tag, prefix, v, tt.wantPrefix, tt.wantV)
		}
	}
}

func TestParseTag_Invalid(t *testing.T) {
	cases := []string{
		"v1.2",
		"v1.2.3.4",
		"v1.2.3-rc1",
		"abc",
		"",
		"v1.2.-3",
		"v1.+2.3",
	}
	for _, tag := range cases {
		if _, _, ok := ParseTag(tag); ok {
			t.Errorf("ParseTag(%q): ok = true, want false", tag)
		}
	}
}

func TestLatest(t *testing.T) {
	tags := []string{"v0.9.0", "v0.10.0", "not-a-tag", "v0.2.0"}
	tag, prefix, v, ok := Latest(tags)
	if !ok {
		t.Fatal("Latest: ok = false, want true")
	}
	if tag != "v0.10.0" || prefix != "v" || v != (Version{0, 10, 0}) {
		t.Errorf("Latest() = (%q, %q, %v), want (v0.10.0, v, {0 10 0})", tag, prefix, v)
	}
}

func TestLatest_TieKeepsFirstSeen(t *testing.T) {
	tags := []string{"v1.0.0", "release-1.0.0"}
	tag, prefix, _, ok := Latest(tags)
	if !ok {
		t.Fatal("Latest: ok = false, want true")
	}
	if tag != "v1.0.0" || prefix != "v" {
		t.Errorf("Latest() = (%q, %q), want (v1.0.0, v)", tag, prefix)
	}
}

func TestLatest_NoParseableTags(t *testing.T) {
	_, _, _, ok := Latest([]string{"abc", "v1.2", ""})
	if ok {
		t.Error("Latest: ok = true, want false")
	}
}

func TestNextTag(t *testing.T) {
	cases := []struct {
		name          string
		tags          []string
		b             Bump
		defaultPrefix string
		want          string
	}{
		{"no tags", nil, Minor, "v", "v0.0.1"},
		{"garbage only", []string{"abc", "v1.2"}, Major, "v", "v0.0.1"},
		{"empty default prefix", nil, Patch, "", "0.0.1"},
		{"patch bump", []string{"v1.2.3"}, Patch, "v", "v1.2.4"},
		{"minor bump", []string{"v1.2.3"}, Minor, "v", "v1.3.0"},
		{"major bump keeps prefix", []string{"release-1.2.3"}, Major, "v", "release-2.0.0"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := NextTag(tt.tags, tt.b, tt.defaultPrefix); got != tt.want {
				t.Errorf("NextTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAIResponse_WellFormed(t *testing.T) {
	in := "BUMP: minor\n---\nSummary line.\n\n### Added\n- new thing\n"
	b, notes := ParseAIResponse(in)
	if b != Minor {
		t.Errorf("bump = %v, want Minor", b)
	}
	want := "Summary line.\n\n### Added\n- new thing"
	if notes != want {
		t.Errorf("notes = %q, want %q", notes, want)
	}
}

func TestParseAIResponse_ExtraSpacesAndCase(t *testing.T) {
	in := "bump:  Minor  \n---\nnotes here"
	b, notes := ParseAIResponse(in)
	if b != Minor {
		t.Errorf("bump = %v, want Minor", b)
	}
	if notes != "notes here" {
		t.Errorf("notes = %q", notes)
	}
}

func TestParseAIResponse_CRLF(t *testing.T) {
	in := "BUMP: major\r\n---\r\nsome notes\r\nmore notes"
	b, notes := ParseAIResponse(in)
	if b != Major {
		t.Errorf("bump = %v, want Major", b)
	}
	if notes != "some notes\nmore notes" {
		t.Errorf("notes = %q", notes)
	}
}

func TestParseAIResponse_LeadingBlankLines(t *testing.T) {
	in := "\n\nBUMP: patch\n---\nfix stuff"
	b, notes := ParseAIResponse(in)
	if b != Patch {
		t.Errorf("bump = %v, want Patch", b)
	}
	if notes != "fix stuff" {
		t.Errorf("notes = %q", notes)
	}
}

func TestParseAIResponse_MarkdownFence(t *testing.T) {
	in := "```text\nBUMP: minor\n---\nfenced notes\n```"
	b, notes := ParseAIResponse(in)
	if b != Minor {
		t.Errorf("bump = %v, want Minor", b)
	}
	if notes != "fenced notes" {
		t.Errorf("notes = %q", notes)
	}
}

func TestParseAIResponse_PlainFence(t *testing.T) {
	in := "```\nBUMP: major\n---\nnotes\n```"
	b, notes := ParseAIResponse(in)
	if b != Major {
		t.Errorf("bump = %v, want Major", b)
	}
	if notes != "notes" {
		t.Errorf("notes = %q", notes)
	}
}

func TestParseAIResponse_MissingSeparatorFallsBack(t *testing.T) {
	in := "BUMP: minor\nno separator here, just text"
	b, notes := ParseAIResponse(in)
	if b != Patch {
		t.Errorf("bump = %v, want Patch (fallback)", b)
	}
	if notes != in {
		t.Errorf("notes = %q, want whole trimmed reply %q", notes, in)
	}
}

func TestParseAIResponse_NoBumpLineFallsBack(t *testing.T) {
	in := "Just some free-form text with no bump marker."
	b, notes := ParseAIResponse(in)
	if b != Patch {
		t.Errorf("bump = %v, want Patch (fallback)", b)
	}
	if notes != in {
		t.Errorf("notes = %q, want %q", notes, in)
	}
}

func TestParseAIResponse_Empty(t *testing.T) {
	b, notes := ParseAIResponse("")
	if b != Patch {
		t.Errorf("bump = %v, want Patch", b)
	}
	if notes != "" {
		t.Errorf("notes = %q, want empty", notes)
	}
}
