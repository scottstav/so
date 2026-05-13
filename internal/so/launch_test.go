package so

import (
	"os"
	"strings"
	"testing"
	"time"
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

func TestLaunch_PassesExtraArgs(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	// Write a tiny script that records its args and then sleeps so the
	// pane stays alive long enough to capture.
	scriptDir := t.TempDir()
	scriptPath := scriptDir + "/recorder"
	outPath := scriptDir + "/args.txt"
	script := "#!/bin/bash\nprintf '%s\\n' \"$@\" > " + outPath + "\nsleep 10\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(AgentsConfPath(), []byte("rec="+scriptPath+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Launch(tx, "test-base", LaunchOpts{
		Agent:        "rec",
		ExtraArgs:    []string{"--resume", "hello world"},
		SkipBriefing: true,
		SkipFocus:    true,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Poll for the args file (script writes it as soon as it starts).
	var got []byte
	for i := 0; i < 30; i++ {
		got, _ = os.ReadFile(outPath)
		if len(got) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	want := "--resume\nhello world\n"
	if string(got) != want {
		t.Errorf("recorder got %q, want %q", got, want)
	}
}
