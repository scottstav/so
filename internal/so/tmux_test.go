package so

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available; skipping integration test")
	}
}

func withFreshSession(t *testing.T) (*Tmux, func()) {
	t.Helper()
	requireTmux(t)
	sock := t.TempDir() + "/sock"
	tx := &Tmux{Socket: sock}
	if err := tx.NewSession("test-base"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	teardown := func() {
		_ = exec.Command("tmux", "-S", sock, "kill-server").Run()
	}
	return tx, teardown
}

func TestTmux_SessionLifecycle(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()

	exists, err := tx.SessionExists("test-base")
	if err != nil {
		t.Fatalf("SessionExists: %v", err)
	}
	if !exists {
		t.Fatal("expected test-base session to exist")
	}

	exists, err = tx.SessionExists("nope")
	if err != nil {
		t.Fatalf("SessionExists: %v", err)
	}
	if exists {
		t.Fatal("expected nope session NOT to exist")
	}
}

func TestTmux_NewWindowAndList(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()

	if err := tx.NewWindow("test-base", "claude@new", ""); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	wins, err := tx.ListWindows("test-base")
	if err != nil {
		t.Fatalf("ListWindows: %v", err)
	}
	found := false
	for _, w := range wins {
		if w == "claude@new" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected claude@new in %v", wins)
	}
}

func TestTmux_RenameWindow(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()

	if err := tx.NewWindow("test-base", "claude@new", ""); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	if err := tx.RenameWindow("test-base:claude@new", "claude@auth-bug"); err != nil {
		t.Fatalf("RenameWindow: %v", err)
	}
	wins, err := tx.ListWindows("test-base")
	if err != nil {
		t.Fatalf("ListWindows: %v", err)
	}
	for _, w := range wins {
		if w == "claude@new" {
			t.Fatal("expected claude@new to be gone after rename")
		}
	}
}

func TestTmux_WindowExists(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()

	if err := tx.NewWindow("test-base", "claude@new", ""); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}

	ok, err := tx.WindowExists("test-base", "claude@new")
	if err != nil {
		t.Fatalf("WindowExists: %v", err)
	}
	if !ok {
		t.Fatal("expected claude@new to exist")
	}

	ok, err = tx.WindowExists("test-base", "nope@nope")
	if err != nil {
		t.Fatalf("WindowExists: %v", err)
	}
	if ok {
		t.Fatal("expected nope@nope NOT to exist")
	}
}

func TestTmux_NewWindowAt_SetsStartDir(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()

	cwd := t.TempDir()
	out := cwd + "/pwd.txt"
	// Spawn a shell that records its cwd, then sleeps so the pane stays
	// alive long enough for `display-message` lookups if we ever want them.
	cmd := "pwd > " + out + "; sleep 5"
	if err := tx.NewWindowAt("test-base", "claude@new", cwd, "bash -lc "+shellQuote(cmd)); err != nil {
		t.Fatalf("NewWindowAt: %v", err)
	}
	var got []byte
	for i := 0; i < 40; i++ {
		got, _ = os.ReadFile(out)
		if len(got) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	want := cwd + "\n"
	if string(got) != want {
		// On macOS /tmp is symlinked to /private/tmp; tolerate that.
		if strings.TrimSpace(string(got)) != strings.TrimSpace(cwd) {
			t.Errorf("new window pwd = %q, want %q", got, want)
		}
	}
}

func TestTmux_NewSessionWithWindowAt_SetsStartDir(t *testing.T) {
	requireTmux(t)
	sock := t.TempDir() + "/sock"
	tx := &Tmux{Socket: sock}
	defer func() { _ = exec.Command("tmux", "-S", sock, "kill-server").Run() }()

	cwd := t.TempDir()
	out := cwd + "/pwd.txt"
	cmd := "pwd > " + out + "; sleep 5"
	if err := tx.NewSessionWithWindowAt("fresh", "claude@new", cwd, "bash -lc "+shellQuote(cmd)); err != nil {
		t.Fatalf("NewSessionWithWindowAt: %v", err)
	}
	var got []byte
	for i := 0; i < 40; i++ {
		got, _ = os.ReadFile(out)
		if len(got) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if strings.TrimSpace(string(got)) != strings.TrimSpace(cwd) {
		t.Errorf("new session window pwd = %q, want %q", got, cwd)
	}
}

func TestTmux_NoTmuxError(t *testing.T) {
	tx := &Tmux{TmuxBin: "/nonexistent/tmux"}
	_, err := tx.SessionExists("foo")
	if err == nil {
		t.Fatal("expected error when tmux binary is missing")
	}
	if !errors.Is(err, ErrTmuxUnavailable) && err.Error() == "" {
		t.Fatalf("unexpected error: %v", err)
	}
}
