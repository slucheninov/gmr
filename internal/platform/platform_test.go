package platform

import "testing"

func TestDetect(t *testing.T) {
	cases := []struct {
		url  string
		want Kind
		err  bool
	}{
		{"git@github.com:foo/bar.git", GitHub, false},
		{"https://github.com/foo/bar.git", GitHub, false},
		{"git@gitlab.com:group/proj.git", GitLab, false},
		{"https://gitlab.com/group/proj.git", GitLab, false},
		{"https://gitlab.example.com/group/proj.git", GitLab, false},
		{"https://bitbucket.org/foo/bar.git", "", true},
	}
	for _, c := range cases {
		got, err := Detect(c.url)
		if c.err {
			if err == nil {
				t.Errorf("Detect(%q): expected error, got %s", c.url, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Detect(%q): unexpected error: %v", c.url, err)
			continue
		}
		if got != c.want {
			t.Errorf("Detect(%q) = %s, want %s", c.url, got, c.want)
		}
	}
}

func TestGitLabProjectPath(t *testing.T) {
	cases := []struct {
		url  string
		want string
		err  bool
	}{
		{"git@gitlab.com:group/proj.git", "group/proj", false},
		{"https://gitlab.com/group/proj.git", "group/proj", false},
		{"https://gitlab.com/group/sub/proj.git", "group/sub/proj", false},
		{"https://gitlab.com/group/proj", "group/proj", false},
		{"git@github.com:foo/bar.git", "", true},
	}
	for _, c := range cases {
		got, err := GitLabProjectPath(c.url)
		if c.err {
			if err == nil {
				t.Errorf("GitLabProjectPath(%q): expected error, got %s", c.url, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("GitLabProjectPath(%q): unexpected error: %v", c.url, err)
			continue
		}
		if got != c.want {
			t.Errorf("GitLabProjectPath(%q) = %s, want %s", c.url, got, c.want)
		}
	}
}
