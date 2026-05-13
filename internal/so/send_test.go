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

	out, err := tx.run("capture-pane", "-p", "-t", "test-base:claude@auth-bug")
	if err != nil {
		t.Fatalf("capture-pane: %v", err)
	}
	if !strings.Contains(out, "hello there") {
		t.Errorf("expected 'hello there' in capture; got %q", out)
	}
}
