package main

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func buildBin(t *testing.T) string {
	t.Helper()
	bin := t.TempDir() + "/so"
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func TestCLI_HelpFlag(t *testing.T) {
	bin := buildBin(t)
	out, err := exec.Command(bin, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("so --help: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Scott's orchestrator") {
		t.Errorf("expected help banner; got: %s", out)
	}
}

func TestCLI_NoArgs(t *testing.T) {
	bin := buildBin(t)
	out, err := exec.Command(bin).CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit; got success with: %s", out)
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Errorf("expected usage on no-args; got: %s", out)
	}
}

func TestParseLaunchFlags(t *testing.T) {
	tests := []struct {
		name      string
		in        []string
		want      launchFlags
		wantArgs  []string
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "no flags",
			in:       []string{"--resume"},
			want:     launchFlags{},
			wantArgs: []string{"--resume"},
		},
		{
			name:     "empty",
			in:       nil,
			want:     launchFlags{},
			wantArgs: nil,
		},
		{
			name:     "no-attach only",
			in:       []string{"--no-attach"},
			want:     launchFlags{NoAttach: true},
			wantArgs: nil,
		},
		{
			name:     "cwd short",
			in:       []string{"-C", "/tmp/foo"},
			want:     launchFlags{Cwd: "/tmp/foo"},
			wantArgs: nil,
		},
		{
			name:     "cwd long",
			in:       []string{"--cwd", "/tmp/foo"},
			want:     launchFlags{Cwd: "/tmp/foo"},
			wantArgs: nil,
		},
		{
			name:     "cwd long with equals",
			in:       []string{"--cwd=/tmp/foo"},
			want:     launchFlags{Cwd: "/tmp/foo"},
			wantArgs: nil,
		},
		{
			name:     "both flags then double-dash then agent args",
			in:       []string{"--no-attach", "-C", "/tmp/foo", "--", "--resume", "hi"},
			want:     launchFlags{Cwd: "/tmp/foo", NoAttach: true},
			wantArgs: []string{"--resume", "hi"},
		},
		{
			name:     "flag then unknown arg stops parsing",
			in:       []string{"--no-attach", "--resume"},
			want:     launchFlags{NoAttach: true},
			wantArgs: []string{"--resume"},
		},
		{
			name:     "double-dash with no extras",
			in:       []string{"--no-attach", "--"},
			want:     launchFlags{NoAttach: true},
			wantArgs: []string{},
		},
		{
			name:      "cwd missing argument",
			in:        []string{"-C"},
			wantErr:   true,
			errSubstr: "-C requires a directory argument",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, gotArgs, err := parseLaunchFlags(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.errSubstr != "" && !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("flags = %+v, want %+v", got, tc.want)
			}
			if !reflect.DeepEqual(gotArgs, tc.wantArgs) {
				t.Errorf("args = %#v, want %#v", gotArgs, tc.wantArgs)
			}
		})
	}
}
