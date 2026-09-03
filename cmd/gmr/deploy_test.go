package main

import (
	"errors"
	"testing"

	"github.com/slucheninov/gmr/internal/release"
)

func TestParseDeployArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		want    deployOptions
		wantErr string
	}{
		{
			name: "no args",
			args: nil,
			want: deployOptions{},
		},
		{
			name: "patch",
			args: []string{"--patch"},
			want: deployOptions{bump: release.Patch, bumpForced: true},
		},
		{
			name: "minor",
			args: []string{"--minor"},
			want: deployOptions{bump: release.Minor, bumpForced: true},
		},
		{
			name: "major",
			args: []string{"--major"},
			want: deployOptions{bump: release.Major, bumpForced: true},
		},
		{
			name: "no-release",
			args: []string{"--no-release"},
			want: deployOptions{noRelease: true},
		},
		{
			name: "yes short",
			args: []string{"-y"},
			want: deployOptions{yes: true},
		},
		{
			name: "yes long",
			args: []string{"--yes"},
			want: deployOptions{yes: true},
		},
		{
			name: "explicit tag",
			args: []string{"v1.2.3"},
			want: deployOptions{explicitTag: "v1.2.3"},
		},
		{
			name: "explicit tag no prefix",
			args: []string{"1.2.3"},
			want: deployOptions{explicitTag: "1.2.3"},
		},
		{
			name: "flags and tag combined",
			args: []string{"--minor", "-y", "v2.0.0"},
			want: deployOptions{bump: release.Minor, bumpForced: true, yes: true, explicitTag: "v2.0.0"},
		},
		{
			name:    "invalid explicit tag",
			args:    []string{"not-a-tag"},
			wantErr: `invalid tag "not-a-tag": expected <prefix>MAJOR.MINOR.PATCH (e.g. v1.2.3)`,
		},
		{
			name:    "two bump flags",
			args:    []string{"--patch", "--minor"},
			wantErr: "only one of --patch, --minor, --major may be given",
		},
		{
			name:    "three bump flags",
			args:    []string{"--patch", "--minor", "--major"},
			wantErr: "only one of --patch, --minor, --major may be given",
		},
		{
			name:    "unknown flag",
			args:    []string{"--bogus"},
			wantErr: "unknown option: --bogus",
		},
		{
			name:    "too many positional args",
			args:    []string{"v1.0.0", "v2.0.0"},
			wantErr: "unexpected argument: v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDeployArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseDeployArgs(%v) error = %v, want %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDeployArgs(%v): unexpected error: %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("parseDeployArgs(%v) = %#v, want %#v", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseDeployArgsHelp(t *testing.T) {
	t.Parallel()
	_, err := parseDeployArgs([]string{"-h"})
	if !errors.Is(err, errDeployShowHelp) {
		t.Fatalf("parseDeployArgs([-h]) error = %v, want errDeployShowHelp", err)
	}
	_, err = parseDeployArgs([]string{"--help"})
	if !errors.Is(err, errDeployShowHelp) {
		t.Fatalf("parseDeployArgs([--help]) error = %v, want errDeployShowHelp", err)
	}
}

func TestIsYes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"YES\n", true},
		{" yes \n", true},
		{"\n", false},
		{"n\n", false},
		{"no\n", false},
		{"anything\n", false},
	}
	for _, tt := range cases {
		if got := isYes(tt.in); got != tt.want {
			t.Errorf("isYes(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
