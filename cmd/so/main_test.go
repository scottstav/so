package main

import (
	"os/exec"
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
