package so

import (
	"errors"
	"fmt"
	"os"
)

// ErrInvalidSuffix is returned when the requested rename suffix contains
// forbidden characters (whitespace, `:`, `.`, `@`).
var ErrInvalidSuffix = errors.New("invalid suffix")

// RenameCurrentWindow renames the window containing $TMUX_PANE in session
// from "<agent>@<old>" to "<agent>@<suffix>", deduping against existing
// window names if needed.
func RenameCurrentWindow(tx *Tmux, session, suffix string) error {
	paneID := os.Getenv("TMUX_PANE")
	if paneID == "" {
		return errors.New("rename: must be called from inside a tmux pane (TMUX_PANE unset)")
	}
	if !IsValidSuffix(suffix) {
		return fmt.Errorf("rename: %w: %q (no whitespace, `:`, `.`, or `@`)",
			ErrInvalidSuffix, suffix)
	}
	if tx == nil {
		return errors.New("rename: internal: tmux is nil")
	}
	current, err := tx.DisplayMessage(paneID, "#W")
	if err != nil {
		return fmt.Errorf("rename: read current window name: %w", err)
	}
	agent, _, ok := ParseWindowName(current)
	if !ok {
		agent = "agent"
	}
	wantBase := agent + "@" + suffix
	wins, err := tx.ListWindows(session)
	if err != nil {
		return fmt.Errorf("rename: list windows: %w", err)
	}
	// Exclude the current window from the dedup list.
	filtered := make([]string, 0, len(wins))
	for _, w := range wins {
		if w != current {
			filtered = append(filtered, w)
		}
	}
	newName := DedupName(wantBase, filtered)
	if err := tx.RenameWindow(paneID, newName); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
