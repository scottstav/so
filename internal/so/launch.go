package so

import (
	"fmt"
	"strings"
	"time"
)

// LaunchOpts controls how an agent is launched.
type LaunchOpts struct {
	Agent string // required: key into the agents registry

	// SkipBriefing skips post-launch briefing injection. Used in tests
	// where the spawned command is not an interactive TUI.
	SkipBriefing bool

	// SkipFocus skips switch-client / select-window after launch.
	SkipFocus bool

	// BriefingDelay is how long to wait after spawning before injecting
	// the briefing. Defaults to 2s.
	BriefingDelay time.Duration
}

// Launch spawns the given agent in a new window of session and (unless
// suppressed by opts) injects the briefing and switches focus. Returns
// the new window's tmux target (e.g. "so:claude@new").
func Launch(tx *Tmux, session string, opts LaunchOpts) (string, error) {
	if err := EnsureDefaults(); err != nil {
		return "", fmt.Errorf("launch: %w", err)
	}
	agents, err := LoadAgents()
	if err != nil {
		return "", fmt.Errorf("launch: %w", err)
	}
	cmd, ok := agents[opts.Agent]
	if !ok {
		return "", fmt.Errorf("launch: unknown agent %q (known: %s)",
			opts.Agent, strings.Join(AgentNames(agents), ", "))
	}
	if tx == nil {
		return "", fmt.Errorf("launch: internal: tmux is nil")
	}

	exists, err := tx.SessionExists(session)
	if err != nil {
		return "", fmt.Errorf("launch: %w", err)
	}
	var wins []string
	if exists {
		wins, err = tx.ListWindows(session)
		if err != nil {
			return "", fmt.Errorf("launch: %w", err)
		}
	}
	winName := DedupName(opts.Agent+"@new", wins)

	if !exists {
		if err := tx.NewSessionWithWindow(session, winName, cmd); err != nil {
			return "", fmt.Errorf("launch: create session: %w", err)
		}
	} else {
		if err := tx.NewWindow(session, winName, cmd); err != nil {
			return "", fmt.Errorf("launch: new-window: %w", err)
		}
	}
	target := session + ":" + winName

	if !opts.SkipBriefing {
		delay := opts.BriefingDelay
		if delay == 0 {
			delay = 2 * time.Second
		}
		time.Sleep(delay)
		text, err := LoadBriefing()
		if err != nil {
			return target, fmt.Errorf("launch: load briefing: %w", err)
		}
		bufName := fmt.Sprintf("so-brief-%d-%d", time.Now().UnixNano(), len(winName))
		if err := tx.PasteText(target, bufName, text); err != nil {
			return target, fmt.Errorf("launch: paste briefing: %w", err)
		}
		if err := tx.SendEnter(target); err != nil {
			return target, fmt.Errorf("launch: submit briefing: %w", err)
		}
	}

	if !opts.SkipFocus {
		// switch-client only works if a client is attached; errors are
		// non-fatal (the window still exists).
		_ = tx.SwitchClient(session)
		_ = tx.SelectWindow(target)
	}

	return target, nil
}
