package so

import (
	"errors"
	"os/exec"
	"testing"
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
