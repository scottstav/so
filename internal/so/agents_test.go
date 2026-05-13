package so

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAgents_Valid(t *testing.T) {
	input := `# comment
claude=claude
cursor=cursor-agent

# another comment
codex=codex
`
	got, err := parseAgents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseAgents: %v", err)
	}
	want := map[string]string{
		"claude": "claude",
		"cursor": "cursor-agent",
		"codex":  "codex",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("got[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseAgents_RejectsMalformed(t *testing.T) {
	cases := []string{
		"noequalsign\n",
		"=missingkey\n",
		"key=\n",
	}
	for _, in := range cases {
		if _, err := parseAgents(strings.NewReader(in)); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestLoadAgents_ReadsFromConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	agents, err := LoadAgents()
	if err != nil {
		t.Fatalf("LoadAgents: %v", err)
	}
	if agents["claude"] != "claude" {
		t.Errorf("got %q, want %q", agents["claude"], "claude")
	}
	if agents["cursor"] != "cursor-agent" {
		t.Errorf("got %q, want %q", agents["cursor"], "cursor-agent")
	}
	if _, err := os.Stat(filepath.Join(tmp, "so", "agents.conf")); err != nil {
		t.Fatal(err)
	}
}
