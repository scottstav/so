package so

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
)

// ErrTmuxUnavailable is returned when the tmux binary cannot be invoked.
var ErrTmuxUnavailable = errors.New("tmux unavailable")

// Tmux is a thin wrapper around the tmux CLI. The zero value uses the
// default tmux server. Tests may set Socket to an isolated path.
type Tmux struct {
	TmuxBin string // defaults to "tmux"
	Socket  string // if set, passes `-S <socket>`
}

// DefaultTmux returns a Tmux configured to use the system tmux.
func DefaultTmux() *Tmux { return &Tmux{} }

func (t *Tmux) bin() string {
	if t.TmuxBin == "" {
		return "tmux"
	}
	return t.TmuxBin
}

func (t *Tmux) cmd(args ...string) *exec.Cmd {
	full := []string{}
	if t.Socket != "" {
		full = append(full, "-S", t.Socket)
	}
	full = append(full, args...)
	return exec.Command(t.bin(), full...)
}

func (t *Tmux) run(args ...string) (string, error) {
	c := t.cmd(args...)
	var out, errBuf bytes.Buffer
	c.Stdout = &out
	c.Stderr = &errBuf
	if err := c.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			// Command couldn't start at all (binary missing, etc.).
			return "", fmt.Errorf("%w: %v", ErrTmuxUnavailable, err)
		}
		return "", fmt.Errorf("tmux %s: %w (stderr: %s)",
			strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}

// SessionExists returns true if the named session exists.
func (t *Tmux) SessionExists(name string) (bool, error) {
	c := t.cmd("has-session", "-t", name)
	c.Stderr = io.Discard
	err := c.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// tmux ran and returned non-zero — session simply doesn't exist.
		return false, nil
	}
	// Anything else means tmux itself couldn't run.
	return false, fmt.Errorf("%w: %v", ErrTmuxUnavailable, err)
}

// NewSession creates a detached session with tmux's default first window.
// Used in tests for setting up a baseline server; production code should
// prefer NewSessionWithWindow to avoid an extra default window.
func (t *Tmux) NewSession(name string) error {
	_, err := t.run("new-session", "-d", "-s", name)
	return err
}

// NewSessionWithWindow creates a detached session whose first window is
// named windowName and (if cmd is non-empty) runs cmd. Use this from
// launch logic so the session doesn't end up with a stray default window.
func (t *Tmux) NewSessionWithWindow(session, windowName, cmd string) error {
	args := []string{"new-session", "-d", "-s", session, "-n", windowName}
	if cmd != "" {
		args = append(args, cmd)
	}
	_, err := t.run(args...)
	return err
}

// NewWindow creates a window in session. If cmd is non-empty, it runs as
// the window's initial command.
func (t *Tmux) NewWindow(session, name, cmd string) error {
	args := []string{"new-window", "-d", "-t", session, "-n", name}
	if cmd != "" {
		args = append(args, cmd)
	}
	_, err := t.run(args...)
	return err
}

// ListWindows returns the window names in a session.
func (t *Tmux) ListWindows(session string) ([]string, error) {
	out, err := t.run("list-windows", "-t", session, "-F", "#W")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}

// WindowExists returns true if session:name is a real window.
func (t *Tmux) WindowExists(session, name string) (bool, error) {
	wins, err := t.ListWindows(session)
	if err != nil {
		return false, err
	}
	return slices.Contains(wins, name), nil
}

// RenameWindow renames the window at target to newName.
func (t *Tmux) RenameWindow(target, newName string) error {
	_, err := t.run("rename-window", "-t", target, newName)
	return err
}

// SelectWindow brings target into focus in its session.
func (t *Tmux) SelectWindow(target string) error {
	_, err := t.run("select-window", "-t", target)
	return err
}

// SwitchClient switches the current client to the named session.
func (t *Tmux) SwitchClient(session string) error {
	_, err := t.run("switch-client", "-t", session)
	return err
}

// DisplayMessage returns the result of tmux display-message -p of the
// given format string, optionally for a specific target.
func (t *Tmux) DisplayMessage(target, format string) (string, error) {
	args := []string{"display-message", "-p"}
	if target != "" {
		args = append(args, "-t", target)
	}
	args = append(args, format)
	out, err := t.run(args...)
	return strings.TrimRight(out, "\n"), err
}

// PaneID returns the immutable pane id (e.g. "%42") for the given target.
// Pane ids survive window renames, making them the stable routing address.
func (t *Tmux) PaneID(target string) (string, error) {
	return t.DisplayMessage(target, "#{pane_id}")
}

// WindowNameFromPaneID returns the current window name for a pane id.
func (t *Tmux) WindowNameFromPaneID(paneID string) (string, error) {
	return t.DisplayMessage(paneID, "#W")
}

// CapturePane returns the current visible pane contents (no scrollback).
func (t *Tmux) CapturePane(target string) (string, error) {
	return t.run("capture-pane", "-p", "-t", target)
}

// PasteText loads text into a tmux buffer, pastes it into target, then
// deletes the buffer. The trailing Enter is NOT sent — call SendEnter
// separately if you want submission.
func (t *Tmux) PasteText(target, bufferName, text string) error {
	c := t.cmd("load-buffer", "-b", bufferName, "-")
	c.Stdin = strings.NewReader(text)
	var errBuf bytes.Buffer
	c.Stderr = &errBuf
	if err := c.Run(); err != nil {
		return fmt.Errorf("load-buffer: %w (stderr: %s)", err, strings.TrimSpace(errBuf.String()))
	}
	if _, err := t.run("paste-buffer", "-b", bufferName, "-t", target); err != nil {
		return err
	}
	if _, err := t.run("delete-buffer", "-b", bufferName); err != nil {
		return err
	}
	return nil
}

// SendEnter sends a single Enter keystroke to target.
func (t *Tmux) SendEnter(target string) error {
	_, err := t.run("send-keys", "-t", target, "Enter")
	return err
}
