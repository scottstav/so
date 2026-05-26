package so

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestScrubSoVars(t *testing.T) {
	in := []string{
		"PATH=/usr/bin",
		"SO_SESSION=so-personal",
		"HOME=/home/x",
		"SO_AGENTS_CONF=/home/x/.config/so/personal-agents.conf",
		"FOO=SO_SESSION=not-a-real-prefix",
	}
	got := scrubSoVars(in)
	for _, e := range got {
		if strings.HasPrefix(e, "SO_SESSION=") || strings.HasPrefix(e, "SO_AGENTS_CONF=") {
			t.Errorf("scrubSoVars left a so-scoped var: %q", e)
		}
	}
	want := []string{
		"PATH=/usr/bin",
		"HOME=/home/x",
		"FOO=SO_SESSION=not-a-real-prefix", // value containing the key is kept
	}
	if len(got) != len(want) {
		t.Fatalf("scrubSoVars returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("scrubSoVars[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestNewSession_NoSoVarLeakToGlobalEnv pins down the core account-separation
// fix: the tmux server seeds its GLOBAL environment from the process that
// first starts it. When `so` is invoked by the personal launcher (which
// exports SO_SESSION / SO_AGENTS_CONF) and happens to start the server, those
// personal values must NOT become the server-global default — otherwise every
// later session, including the default/work one, inherits the personal agent
// registry and spawns children into the wrong account.
func TestNewSession_NoSoVarLeakToGlobalEnv(t *testing.T) {
	requireTmux(t)
	t.Setenv("SO_SESSION", "so-personal")
	t.Setenv("SO_AGENTS_CONF", "/home/x/.config/so/personal-agents.conf")

	sock := t.TempDir() + "/sock"
	tx := &Tmux{Socket: sock}
	defer func() { _ = exec.Command("tmux", "-S", sock, "kill-server").Run() }()

	// NewSession starts the server; its global env is seeded here.
	if err := tx.NewSession("test-base"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	out, err := tx.run("show-environment", "-g")
	if err != nil {
		t.Fatalf("show-environment -g: %v", err)
	}
	if strings.Contains(out, "SO_AGENTS_CONF") {
		t.Errorf("SO_AGENTS_CONF leaked into tmux server-global env:\n%s", out)
	}
	if strings.Contains(out, "SO_SESSION") {
		t.Errorf("SO_SESSION leaked into tmux server-global env:\n%s", out)
	}
}

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

// TestTmux_SessionExists_NoPrefixMatch pins down that SessionExists
// requires an exact match — tmux's default `-t` does prefix matching,
// which previously caused queries for "so" to succeed when only
// "so-personal" was running (and downstream new-window calls would
// then land in the wrong session).
func TestTmux_SessionExists_NoPrefixMatch(t *testing.T) {
	requireTmux(t)
	sock := t.TempDir() + "/sock"
	tx := &Tmux{Socket: sock}
	defer func() { _ = exec.Command("tmux", "-S", sock, "kill-server").Run() }()

	if err := tx.NewSession("so-personal"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	exists, err := tx.SessionExists("so")
	if err != nil {
		t.Fatalf("SessionExists: %v", err)
	}
	if exists {
		t.Fatal(`expected "so" NOT to exist when only "so-personal" is running ` +
			`(tmux prefix match leak — see cc-picker bug)`)
	}

	exists, err = tx.SessionExists("so-personal")
	if err != nil {
		t.Fatalf("SessionExists: %v", err)
	}
	if !exists {
		t.Fatal(`expected exact "so-personal" lookup to succeed`)
	}
}

// TestTmux_ListWindows_NoPrefixMatch ensures ListWindows also requires
// an exact session match — same bug class as SessionExists.
func TestTmux_ListWindows_NoPrefixMatch(t *testing.T) {
	requireTmux(t)
	sock := t.TempDir() + "/sock"
	tx := &Tmux{Socket: sock}
	defer func() { _ = exec.Command("tmux", "-S", sock, "kill-server").Run() }()

	if err := tx.NewSession("so-personal"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := tx.NewWindow("so-personal", "claude@personal", ""); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}

	// Listing "so" must NOT return "so-personal"'s windows.
	if _, err := tx.ListWindows("so"); err == nil {
		t.Fatal(`expected ListWindows("so") to error when only "so-personal" exists`)
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

func TestTmux_NewWindowAt_SetsStartDir(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()

	cwd := t.TempDir()
	out := cwd + "/pwd.txt"
	// Spawn a shell that records its cwd, then sleeps so the pane stays
	// alive long enough for `display-message` lookups if we ever want them.
	cmd := "pwd > " + out + "; sleep 5"
	if err := tx.NewWindowAt("test-base", "claude@new", cwd, "bash -lc "+shellQuote(cmd)); err != nil {
		t.Fatalf("NewWindowAt: %v", err)
	}
	var got []byte
	for i := 0; i < 40; i++ {
		got, _ = os.ReadFile(out)
		if len(got) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	want := cwd + "\n"
	if string(got) != want {
		// On macOS /tmp is symlinked to /private/tmp; tolerate that.
		if strings.TrimSpace(string(got)) != strings.TrimSpace(cwd) {
			t.Errorf("new window pwd = %q, want %q", got, want)
		}
	}
}

func TestTmux_NewSessionWithWindowAt_SetsStartDir(t *testing.T) {
	requireTmux(t)
	sock := t.TempDir() + "/sock"
	tx := &Tmux{Socket: sock}
	defer func() { _ = exec.Command("tmux", "-S", sock, "kill-server").Run() }()

	cwd := t.TempDir()
	out := cwd + "/pwd.txt"
	cmd := "pwd > " + out + "; sleep 5"
	if err := tx.NewSessionWithWindowAt("fresh", "claude@new", cwd, "bash -lc "+shellQuote(cmd)); err != nil {
		t.Fatalf("NewSessionWithWindowAt: %v", err)
	}
	var got []byte
	for i := 0; i < 40; i++ {
		got, _ = os.ReadFile(out)
		if len(got) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if strings.TrimSpace(string(got)) != strings.TrimSpace(cwd) {
		t.Errorf("new session window pwd = %q, want %q", got, cwd)
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
