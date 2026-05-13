package so

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// SendPrompt delivers prompt to a target pane. target can be:
//   - a pane id like "%42" (stable across renames — preferred form)
//   - a full tmux target like "so:cursor@new"
//   - a bare window name like "cursor@new" (resolved within session)
//
// SendPrompt waits for the target pane to be idle (visible content stable
// for ~1.5s) before pasting, which covers both freshly-spawned agents
// still processing their briefing and agents mid-response from a prior task.
// Empty prompts are rejected.
func SendPrompt(tx *Tmux, session, target, prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return errors.New("send: prompt is empty")
	}
	resolved, window, err := resolveTarget(tx, session, target)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	// Coarse pre-check: if the window is still pre-rename, give the TUI a
	// moment to come up before we even start polling.
	if _, suffix, ok := ParseWindowName(window); ok && (suffix == "new" || strings.HasPrefix(suffix, "new-")) {
		time.Sleep(2 * time.Second)
	}

	// Wait for the pane to be idle so we don't paste mid-stream.
	if err := waitForPaneIdle(tx, resolved, 1500*time.Millisecond, 30*time.Second); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	bufName := fmt.Sprintf("so-%d", os.Getpid())
	if err := tx.PasteText(resolved, bufName, prompt); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return tx.SendEnter(resolved)
}

// resolveTarget normalizes a user-supplied target into a tmux-compatible
// reference and a best-effort current window name.
func resolveTarget(tx *Tmux, session, target string) (resolved, window string, err error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", errors.New("empty target")
	}
	if strings.HasPrefix(target, "%") {
		win, err := tx.WindowNameFromPaneID(target)
		if err != nil {
			return "", "", fmt.Errorf("resolve pane id %q: %w", target, err)
		}
		return target, win, nil
	}
	if strings.Contains(target, ":") {
		parts := strings.SplitN(target, ":", 2)
		return target, parts[1], nil
	}
	exists, err := tx.WindowExists(session, target)
	if err != nil {
		return "", "", err
	}
	if !exists {
		wins, _ := tx.ListWindows(session)
		return "", "", fmt.Errorf("window %q not found (active: %s)",
			target, strings.Join(wins, ", "))
	}
	return session + ":" + target, target, nil
}

// waitForPaneIdle polls target's visible pane contents and returns once
// the contents have been unchanged for idleDuration. Caps at maxWait and
// returns nil (not an error) at the cap — we'd rather deliver than fail.
func waitForPaneIdle(tx *Tmux, target string, idleDuration, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	const poll = 500 * time.Millisecond
	var last string
	lastChange := time.Now()
	for time.Now().Before(deadline) {
		cur, err := tx.CapturePane(target)
		if err != nil {
			return fmt.Errorf("capture-pane: %w", err)
		}
		if cur != last {
			last = cur
			lastChange = time.Now()
		} else if time.Since(lastChange) >= idleDuration {
			return nil
		}
		time.Sleep(poll)
	}
	return nil
}

// PromptFromArgsOrStdin returns prompt content from positional arg or
// stdin. Returns an error if both are missing/empty.
func PromptFromArgsOrStdin(args []string, stdin io.Reader) (string, error) {
	if len(args) > 0 && args[0] != "-" && args[0] != "" {
		return args[0], nil
	}
	b, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	if strings.TrimSpace(string(b)) == "" {
		return "", errors.New("no prompt provided (pass as arg or via stdin)")
	}
	return string(b), nil
}
