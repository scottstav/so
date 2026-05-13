package so

import (
	"os"
	"strings"
	"testing"
)

func TestLaunch_UnknownAgent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	opts := LaunchOpts{Agent: "nonexistent"}
	_, err := Launch(nil, "test-base", opts)
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention unknown agent name; got %v", err)
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("error should list known agents (e.g. claude); got %v", err)
	}
}

func TestLaunch_CreatesWindowAndDedupes(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}

	// Override agents.conf with a fake agent so we don't need claude installed.
	if err := os.WriteFile(AgentsConfPath(), []byte("fake=cat\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := LaunchOpts{
		Agent:        "fake",
		SkipBriefing: true,
		SkipFocus:    true,
	}
	r1, err := Launch(tx, "test-base", opts)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if !strings.HasSuffix(r1.Target, ":fake@new") {
		t.Errorf("got target %q, want suffix :fake@new", r1.Target)
	}
	if r1.PaneID == "" || !strings.HasPrefix(r1.PaneID, "%") {
		t.Errorf("got PaneID %q, want non-empty starting with %%", r1.PaneID)
	}
	r2, err := Launch(tx, "test-base", opts)
	if err != nil {
		t.Fatalf("Launch (2nd): %v", err)
	}
	if !strings.HasSuffix(r2.Target, ":fake@new-2") {
		t.Errorf("got target %q, want suffix :fake@new-2", r2.Target)
	}
	if r2.PaneID == r1.PaneID {
		t.Errorf("expected distinct pane ids, both got %q", r1.PaneID)
	}
}
