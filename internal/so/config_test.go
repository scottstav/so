package so

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDir_RespectsXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	got := ConfigDir()
	want := filepath.Join(tmp, "so")
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_DefaultsToHomeConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tmp)
	got := ConfigDir()
	want := filepath.Join(tmp, ".config", "so")
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestEnsureDefaults_WritesMissingFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}
	agents := filepath.Join(tmp, "so", "agents.conf")
	briefing := filepath.Join(tmp, "so", "briefing.md")
	for _, p := range []string{agents, briefing} {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
		if len(b) == 0 {
			t.Fatalf("expected %s to be non-empty", p)
		}
	}
	if !strings.Contains(string(mustRead(t, agents)), "claude=claude") {
		t.Fatalf("agents.conf missing default claude entry")
	}
}

func TestEnsureDefaults_PreservesExisting(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir := filepath.Join(tmp, "so")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "claude=my-custom-claude\n"
	if err := os.WriteFile(filepath.Join(dir, "agents.conf"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDefaults(); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}
	got := string(mustRead(t, filepath.Join(dir, "agents.conf")))
	if got != custom {
		t.Fatalf("EnsureDefaults clobbered existing config; got %q want %q", got, custom)
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return b
}
