package main

import (
	"errors"
	"testing"
)

func TestParseGmrArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		args   []string
		want   gmrOptions
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
