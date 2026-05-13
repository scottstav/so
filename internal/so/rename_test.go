package so

import (
	"errors"
	"testing"
)

func TestRename_NoTmuxPaneEnv(t *testing.T) {
	t.Setenv("TMUX_PANE", "")
	err := RenameCurrentWindow(nil, "test-base", "auth-bug")
	if err == nil {
		t.Fatal("expected error when TMUX_PANE is unset")
	}
}

func TestRename_RejectsInvalidSuffix(t *testing.T) {
	t.Setenv("TMUX_PANE", "%1")
	err := RenameCurrentWindow(nil, "test-base", "has space")
	if err == nil {
		t.Fatal("expected error for invalid suffix")
	}
	if !errors.Is(err, ErrInvalidSuffix) {
		t.Errorf("expected ErrInvalidSuffix, got %v", err)
	}
}

func TestRename_KeepsAgentPrefix_Integration(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()

	if err := tx.NewWindow("test-base", "claude@new", ""); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	paneID, err := tx.DisplayMessage("test-base:claude@new", "#{pane_id}")
	if err != nil {
		t.Fatalf("DisplayMessage: %v", err)
	}
	t.Setenv("TMUX_PANE", paneID)

	if err := RenameCurrentWindow(tx, "test-base", "auth-bug"); err != nil {
		t.Fatalf("RenameCurrentWindow: %v", err)
	}
	wins, _ := tx.ListWindows("test-base")
	for _, w := range wins {
		if w == "claude@auth-bug" {
			return
		}
	}
	t.Fatalf("expected claude@auth-bug in %v", wins)
}
