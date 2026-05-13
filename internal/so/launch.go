package so

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// LaunchOpts controls how an agent is launched.
type LaunchOpts struct {
	Agent string // required: key into the agents registry

	// ExtraArgs are appended to the agent's command line, shell-quoted.
	// E.g. ExtraArgs=[]string{"--resume"} with agent "claude" produces
	// the command "claude --resume" inside the new tmux window.
	ExtraArgs []string

	// SkipBriefing skips post-launch briefing injection. Used in tests
	// where the spawned command is not an interactive TUI.
	SkipBriefing bool

	// SkipFocus skips switch-client / select-window after launch.
	SkipFocus bool

	// BriefingDelay is how long to wait after spawning before injecting
	// the briefing. Defaults to 2s.
	BriefingDelay time.Duration
}

// LaunchResult describes a freshly-spawned agent.
type LaunchResult struct {
	PaneID string // stable id (e.g. "%42"); preferred routing target
	Target string // human-readable target (e.g. "so:claude@new")
	Window string // window name (e.g. "claude@new")
}

// Launch spawns the given agent in a new window of session and (unless
// suppressed by opts) injects the briefing and switches focus.
func Launch(tx *Tmux, session string, opts LaunchOpts) (LaunchResult, error) {
	var zero LaunchResult
	if err := EnsureDefaults(); err != nil {
		return zero, fmt.Errorf("launch: %w", err)
	}
	agents, err := LoadAgents()
	if err != nil {
		return zero, fmt.Errorf("launch: %w", err)
	}
	baseCmd, ok := agents[opts.Agent]
	if !ok {
		return zero, fmt.Errorf("launch: unknown agent %q (known: %s)",
			opts.Agent, strings.Join(AgentNames(agents), ", "))
	}
	if tx == nil {
		return zero, fmt.Errorf("launch: internal: tmux is nil")
	}
	cmd := buildAgentCommand(baseCmd, opts.ExtraArgs)

	exists, err := tx.SessionExists(session)
	if err != nil {
		return zero, fmt.Errorf("launch: %w", err)
	}
	var wins []string
	if exists {
		wins, err = tx.ListWindows(session)
		if err != nil {
			return zero, fmt.Errorf("launch: %w", err)
		}
	}
	winName := DedupName(opts.Agent+"@new", wins)

	// Propagate the resolved session and any SO_AGENTS_CONF override into
	// the new pane so agents that spawn peers from this pane (block 4)
	// stay in the same so-world instead of leaking back to the default.
	env := []string{"SO_SESSION=" + session}
	if v := os.Getenv("SO_AGENTS_CONF"); v != "" {
		env = append(env, "SO_AGENTS_CONF="+v)
	}

	if !exists {
		if err := tx.NewSessionWithWindow(session, winName, cmd, env...); err != nil {
			return zero, fmt.Errorf("launch: create session: %w", err)
		}
	} else {
		if err := tx.NewWindow(session, winName, cmd, env...); err != nil {
			return zero, fmt.Errorf("launch: new-window: %w", err)
		}
	}
	target := session + ":" + winName

	paneID, err := tx.PaneID(target)
	if err != nil {
		return zero, fmt.Errorf("launch: resolve pane id: %w", err)
	}
	result := LaunchResult{PaneID: paneID, Target: target, Window: winName}

	if !opts.SkipBriefing {
		delay := opts.BriefingDelay
		if delay == 0 {
			delay = 2 * time.Second
		}
		time.Sleep(delay)
		text, err := LoadBriefing()
		if err != nil {
			return result, fmt.Errorf("launch: load briefing: %w", err)
		}
		bufName := fmt.Sprintf("so-brief-%d-%d", time.Now().UnixNano(), len(winName))
		if err := tx.PasteText(paneID, bufName, text); err != nil {
			return result, fmt.Errorf("launch: paste briefing: %w", err)
		}
		if err := tx.SendEnter(paneID); err != nil {
			return result, fmt.Errorf("launch: submit briefing: %w", err)
		}
	}

	if !opts.SkipFocus {
		_ = tx.SwitchClient(session)
		_ = tx.SelectWindow(target)
	}

	return result, nil
}
