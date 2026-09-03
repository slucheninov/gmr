package main

import (
	"errors"
	"strings"
	"testing"
)

func TestAskStayOnBranch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  bool
	}{
		{"s\n", true},
		{"stay\n", true},
		{"y\n", true},
		{"\n", false},
		{"m\n", false},
		{"anything\n", false},
	}
	for _, tt := range cases {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := askStayOnBranch(strings.NewReader(tt.input), "feature-x", "main")
			if got != tt.want {
				t.Errorf("askStayOnBranch(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseGmrArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		want    gmrOptions
		wantErr error
	}{
		{
			name:    "empty",
			args:    nil,
			want:    gmrOptions{},
			wantErr: nil,
		},
		{
			name: "message only",
			args: []string{"-m"},
			want: gmrOptions{messageOnly: true},
		},
		{
			name: "long flags",
			args: []string{"--message", "--stay", "my/branch"},
			want: gmrOptions{messageOnly: true, stayOnBranch: true, branchArg: "my/branch"},
		},
		{
			name: "stay short",
			args: []string{"-s"},
			want: gmrOptions{stayOnBranch: true},
		},
		{
			name:    "help",
			args:    []string{"-h"},
			wantErr: errShowHelp,
		},
		{
			name:    "version",
			args:    []string{"--version"},
			wantErr: errShowVersion,
		},
		{
			name:    "unknown flag",
			args:    []string{"-q"},
			wantErr: errors.New("unknown option: -q"),
		},
		{
			name:    "double branch",
			args:    []string{"a", "b"},
			wantErr: errors.New("unexpected argument: b"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseGmrArgs(tt.args)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) && (err == nil || err.Error() != tt.wantErr.Error()) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGmrArgs: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBuildProviders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		order string
		want  string
	}{
		{"", "Gemini,Claude,OpenAI"},
		{"openai", "OpenAI"},
		{"claude,openai", "Claude,OpenAI"},
		{"openai,gemini", "OpenAI,Gemini"},
		{" OpenAI , Claude ", "OpenAI,Claude"},
		{"anthropic", "Claude"},
		{"bogus,openai", "OpenAI"},
		{"bogus", ""},
	}
	for _, tt := range tests {
		t.Run("order="+tt.order, func(t *testing.T) {
			t.Parallel()
			ps := buildProviders(tt.order)
			var ns []string
			for _, p := range ps {
				ns = append(ns, p.Name())
			}
			got := strings.Join(ns, ",")
			if got != tt.want {
				t.Errorf("buildProviders(%q) = %q, want %q", tt.order, got, tt.want)
			}
		})
	}
}

func TestResolveBranch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		current      string
		main         string
		arg          string
		wantBranch   string
		wantExisting bool
		wantErr      bool
	}{
		{name: "base creates derived branch", current: "main", main: "main"},
		{name: "base uses branch argument", current: "main", main: "main", arg: "feat/new", wantBranch: "feat/new"},
		{name: "existing feature branch", current: "feat/existing", main: "main", wantBranch: "feat/existing", wantExisting: true},
		{name: "matching argument on feature", current: "feat/existing", main: "main", arg: "feat/existing", wantBranch: "feat/existing", wantExisting: true},
		{name: "conflicting argument", current: "feat/existing", main: "main", arg: "feat/other", wantErr: true},
		{name: "detached head", current: "", main: "main", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotBranch, gotExisting, err := resolveBranch(tt.current, tt.main, tt.arg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotBranch != tt.wantBranch || gotExisting != tt.wantExisting {
				t.Errorf("resolveBranch() = (%q, %v), want (%q, %v)", gotBranch, gotExisting, tt.wantBranch, tt.wantExisting)
			}
		})
	}
}
