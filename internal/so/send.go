package so

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// SendPrompt delivers prompt to window in session. If the target window
// is still named "<agent>@new" (or @new-N pre-rename) it waits ~2s for
// TUI warmup before pasting. Empty prompts are rejected.
func SendPrompt(tx *Tmux, session, window, prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return errors.New("send: prompt is empty")
	}
	exists, err := tx.WindowExists(session, window)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	if !exists {
		wins, _ := tx.ListWindows(session)
		return fmt.Errorf("send: window %q not found (active: %s)",
			window, strings.Join(wins, ", "))
	}
	if _, suffix, ok := ParseWindowName(window); ok && (suffix == "new" || strings.HasPrefix(suffix, "new-")) {
		// Window has not yet been renamed by the agent — wait for warmup.
		time.Sleep(2 * time.Second)
	}

	target := session + ":" + window
	bufName := fmt.Sprintf("so-%d", os.Getpid())
	if err := tx.PasteText(target, bufName, prompt); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return tx.SendEnter(target)
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
