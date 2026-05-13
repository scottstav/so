package so

import (
	"strings"
	"testing"
	"time"
)

func TestSend_RejectsEmptyPrompt(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()
	if err := tx.NewWindow("test-base", "claude@auth-bug", ""); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	err := SendPrompt(tx, "test-base", "claude@auth-bug", "")
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
}

func TestSend_RejectsMissingTarget(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()
	err := SendPrompt(tx, "test-base", "nope@nope", "hi")
	if err == nil {
		t.Fatal("expected error for missing window")
	}
	if !strings.Contains(err.Error(), "nope@nope") {
		t.Errorf("error should mention window name; got %v", err)
	}
}

func TestSend_DeliversToExistingWindow(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()
	if err := tx.NewWindow("test-base", "claude@auth-bug", "cat"); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	if err := SendPrompt(tx, "test-base", "claude@auth-bug", "hello there"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	out, err := tx.CapturePane("test-base:claude@auth-bug")
	if err != nil {
		t.Fatalf("capture-pane: %v", err)
	}
	if !strings.Contains(out, "hello there") {
		t.Errorf("expected 'hello there' in capture; got %q", out)
	}
}

func TestSend_AcceptsPaneID(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()
	if err := tx.NewWindow("test-base", "claude@auth-bug", "cat"); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	paneID, err := tx.PaneID("test-base:claude@auth-bug")
	if err != nil {
		t.Fatalf("PaneID: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Rename the window — pane id should still route correctly.
	if err := tx.RenameWindow("test-base:claude@auth-bug", "claude@renamed"); err != nil {
		t.Fatalf("RenameWindow: %v", err)
	}

	if err := SendPrompt(tx, "test-base", paneID, "hello-via-pane-id"); err != nil {
		t.Fatalf("SendPrompt via pane id: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	out, err := tx.CapturePane(paneID)
	if err != nil {
		t.Fatalf("capture-pane: %v", err)
	}
	if !strings.Contains(out, "hello-via-pane-id") {
		t.Errorf("expected 'hello-via-pane-id' in capture; got %q", out)
	}
}
