# `so` orchestrator v0 implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `so` v0 CLI per `docs/superpowers/specs/2026-05-13-so-orchestrator-design.md` — a small Go binary that launches agents in a shared tmux session and feeds them prompts.

**Architecture:** One Go module (`github.com/scottstav/so`), one CLI binary (`so`). Subcommand dispatch in `cmd/so/main.go` delegates to functions in `internal/so/`. All tmux interaction is shelled out via `os/exec`. Config and briefing live in `~/.config/so/`, with embedded defaults that bootstrap on first run.

**Tech Stack:** Go 1.24+, stdlib only (no Cobra, no third-party deps), `tmux` CLI as the only runtime dependency.

---

## File structure

```
cmd/so/main.go                 # CLI entry + subcommand dispatch
internal/so/config.go          # ~/.config/so dir, embedded defaults, bootstrap
internal/so/agents.go          # agents.conf parser
internal/so/briefing.go        # briefing.md loader
internal/so/window.go          # window-name parsing + dedup
internal/so/tmux.go            # thin wrapper over `tmux ...` via os/exec
internal/so/send.go            # `so send` impl
internal/so/rename.go          # `so rename` impl
internal/so/ls.go              # `so ls` impl
internal/so/launch.go          # `so <agent>` impl (launch + brief)
internal/so/*_test.go          # unit tests per file
internal/so/defaults/agents.conf   # embedded default
internal/so/defaults/briefing.md   # embedded default
```

One Go package (`so`) for all logic to keep it browseable. `cmd/so/main.go` is the only consumer.

---

## Task 1: Project scaffold + skeleton main.go

**Files:**
- Create: `cmd/so/main.go`
- Modify: `go.mod` (already exists, no change needed)

- [ ] **Step 1: Write skeleton `cmd/so/main.go`**

```go
package main

import (
	"fmt"
	"os"
)

const sessionName = "so"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "send":
		os.Exit(runSend(args))
	case "rename":
		os.Exit(runRename(args))
	case "ls":
		os.Exit(runLs(args))
	case "-h", "--help", "help":
		printUsage()
		os.Exit(0)
	default:
		// Treat as `so <agent> [name]` — launch.
		os.Exit(runLaunch(cmd, args))
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `so — Scott's orchestrator

Usage:
  so <agent> [name]       launch an agent in a new tmux window
  so send <window> <msg>  feed a prompt to a running agent
  so rename <word>        rename the calling window's task suffix
  so ls                   list active agent windows

Config: ~/.config/so/agents.conf and ~/.config/so/briefing.md`)
}

// Stubs — real impls land in later tasks.
func runSend(args []string) int   { fmt.Fprintln(os.Stderr, "send: not implemented"); return 1 }
func runRename(args []string) int { fmt.Fprintln(os.Stderr, "rename: not implemented"); return 1 }
func runLs(args []string) int     { fmt.Fprintln(os.Stderr, "ls: not implemented"); return 1 }
func runLaunch(agent string, args []string) int {
	fmt.Fprintln(os.Stderr, "launch: not implemented")
	return 1
}
```

- [ ] **Step 2: Verify it builds and prints usage**

Run: `cd ~/projects/so && go build ./... && ./so`
Expected: stderr shows the usage, exit code 2.

Run: `./so --help`
Expected: stderr shows the usage, exit code 0.

- [ ] **Step 3: Commit**

```bash
cd ~/projects/so
git add cmd/so/main.go
git commit -m "scaffold: cmd/so/main.go with subcommand dispatch stubs"
```

---

## Task 2: Config directory + embedded defaults

**Files:**
- Create: `internal/so/config.go`
- Create: `internal/so/config_test.go`
- Create: `internal/so/defaults/agents.conf`
- Create: `internal/so/defaults/briefing.md`

- [ ] **Step 1: Write the embedded default files**

Create `internal/so/defaults/agents.conf`:

```
# Agent registry. Format: <name>=<command>
# Adding an agent: add a line. No code change required.
claude=claude
cursor=cursor-agent
```

Create `internal/so/defaults/briefing.md`:

```
You are running inside `so`, a small orchestrator for agent tmux sessions.

Your environment:
- tmux session: `so`
- your window:  <agent>@new  (rename it after your first real task)

After your first real task, pick one hyphen-word describing what you're
doing, and rename your window:

  so rename auth-bug          # window becomes claude@auth-bug

To send a prompt to another agent's session:

  so send <window> "your prompt"
  so send cursor@auth-bug "please review my diff in this repo"

To list active sessions and what they're working on:

  so ls

To spawn a new agent:

  so claude         # or cursor, etc.

When you spawn an agent to do a task for you, follow up with `so send`
to give it the task. In that task, tell the new agent how to route
results back to you (typically `so send` to your own window).

---

These commands are AVAILABLE to you, not REQUIRED. Most tasks don't need
them. Don't spawn agents or send to other sessions unless the user asks
or the work genuinely calls for cross-agent collaboration. Default to
doing the job yourself.
```

- [ ] **Step 2: Write failing test for config dir + bootstrap**

Create `internal/so/config_test.go`:

```go
package so

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDir_RespectsXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	got := ConfigDir()
	want := filepath.Join(tmp, "so")
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_DefaultsToHomeConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tmp)
	got := ConfigDir()
	want := filepath.Join(tmp, ".config", "so")
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestEnsureDefaults_WritesMissingFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}
	agents := filepath.Join(tmp, "so", "agents.conf")
	briefing := filepath.Join(tmp, "so", "briefing.md")
	for _, p := range []string{agents, briefing} {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
		if len(b) == 0 {
			t.Fatalf("expected %s to be non-empty", p)
		}
	}
	if !strings.Contains(string(mustRead(t, agents)), "claude=claude") {
		t.Fatalf("agents.conf missing default claude entry")
	}
}

func TestEnsureDefaults_PreservesExisting(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir := filepath.Join(tmp, "so")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "claude=my-custom-claude\n"
	if err := os.WriteFile(filepath.Join(dir, "agents.conf"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDefaults(); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}
	got := string(mustRead(t, filepath.Join(dir, "agents.conf")))
	if got != custom {
		t.Fatalf("EnsureDefaults clobbered existing config; got %q want %q", got, custom)
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return b
}
```

Run: `cd ~/projects/so && go test ./internal/so/...`
Expected: FAIL — `package so` doesn't exist yet.

- [ ] **Step 3: Implement config.go**

Create `internal/so/config.go`:

```go
package so

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed defaults/agents.conf defaults/briefing.md
var defaultsFS embed.FS

// ConfigDir returns the directory holding so's config files. Honors
// XDG_CONFIG_HOME, falls back to $HOME/.config/so.
func ConfigDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "so")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "so")
}

// AgentsConfPath returns the path to agents.conf.
func AgentsConfPath() string { return filepath.Join(ConfigDir(), "agents.conf") }

// BriefingPath returns the path to briefing.md.
func BriefingPath() string { return filepath.Join(ConfigDir(), "briefing.md") }

// EnsureDefaults writes the default agents.conf and briefing.md if they
// don't already exist. Existing files are never overwritten.
func EnsureDefaults() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if err := copyDefaultIfMissing("defaults/agents.conf", AgentsConfPath()); err != nil {
		return err
	}
	if err := copyDefaultIfMissing("defaults/briefing.md", BriefingPath()); err != nil {
		return err
	}
	return nil
}

func copyDefaultIfMissing(embedPath, dstPath string) error {
	if _, err := os.Stat(dstPath); err == nil {
		return nil // already there
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", dstPath, err)
	}
	b, err := defaultsFS.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", embedPath, err)
	}
	if err := os.WriteFile(dstPath, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd ~/projects/so && go test ./internal/so/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/so
git add internal/so/config.go internal/so/config_test.go internal/so/defaults/
git commit -m "config: ConfigDir + EnsureDefaults with embedded defaults"
```

---

## Task 3: Agent registry parsing

**Files:**
- Create: `internal/so/agents.go`
- Create: `internal/so/agents_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/so/agents_test.go`:

```go
package so

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAgents_Valid(t *testing.T) {
	input := `# comment
claude=claude
cursor=cursor-agent

# another comment
codex=codex
`
	got, err := parseAgents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseAgents: %v", err)
	}
	want := map[string]string{
		"claude": "claude",
		"cursor": "cursor-agent",
		"codex":  "codex",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("got[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseAgents_RejectsMalformed(t *testing.T) {
	cases := []string{
		"noequalsign\n",
		"=missingkey\n",
		"key=\n",
	}
	for _, in := range cases {
		if _, err := parseAgents(strings.NewReader(in)); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestLoadAgents_ReadsFromConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	agents, err := LoadAgents()
	if err != nil {
		t.Fatalf("LoadAgents: %v", err)
	}
	if agents["claude"] != "claude" {
		t.Errorf("got %q, want %q", agents["claude"], "claude")
	}
	if agents["cursor"] != "cursor-agent" {
		t.Errorf("got %q, want %q", agents["cursor"], "cursor-agent")
	}
	// Verify file was actually placed
	if _, err := os.Stat(filepath.Join(tmp, "so", "agents.conf")); err != nil {
		t.Fatal(err)
	}
}
```

Run: `cd ~/projects/so && go test ./internal/so/... -run Agents`
Expected: FAIL — `parseAgents` and `LoadAgents` undefined.

- [ ] **Step 2: Implement agents.go**

Create `internal/so/agents.go`:

```go
package so

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// LoadAgents reads agents.conf from ConfigDir() and returns the registry.
func LoadAgents() (map[string]string, error) {
	f, err := os.Open(AgentsConfPath())
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", AgentsConfPath(), err)
	}
	defer f.Close()
	return parseAgents(f)
}

func parseAgents(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	scan := bufio.NewScanner(r)
	lineno := 0
	for scan.Scan() {
		lineno++
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("line %d: expected key=value, got %q", lineno, line)
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if key == "" || val == "" {
			return nil, fmt.Errorf("line %d: empty key or value in %q", lineno, line)
		}
		out[key] = val
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// AgentNames returns the sorted list of registered agent names — useful
// for error messages and help text.
func AgentNames(reg map[string]string) []string {
	names := make([]string, 0, len(reg))
	for k := range reg {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd ~/projects/so && go test ./internal/so/...`
Expected: PASS (all tests, including Task 2's).

- [ ] **Step 4: Commit**

```bash
cd ~/projects/so
git add internal/so/agents.go internal/so/agents_test.go
git commit -m "agents: LoadAgents reads ~/.config/so/agents.conf"
```

---

## Task 4: Window name parsing + dedup

**Files:**
- Create: `internal/so/window.go`
- Create: `internal/so/window_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/so/window_test.go`:

```go
package so

import (
	"testing"
)

func TestParseWindowName(t *testing.T) {
	cases := []struct {
		in            string
		wantAgent     string
		wantSuffix    string
		wantOK        bool
	}{
		{"claude@new", "claude", "new", true},
		{"cursor@auth-bug", "cursor", "auth-bug", true},
		{"claude@new-2", "claude", "new-2", true},
		{"no-at-sign", "", "", false},
		{"@onlyrhs", "", "", false},
		{"onlyat@", "", "", false},
	}
	for _, tc := range cases {
		agent, suffix, ok := ParseWindowName(tc.in)
		if ok != tc.wantOK || agent != tc.wantAgent || suffix != tc.wantSuffix {
			t.Errorf("ParseWindowName(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.in, agent, suffix, ok, tc.wantAgent, tc.wantSuffix, tc.wantOK)
		}
	}
}

func TestDedupName(t *testing.T) {
	cases := []struct {
		base     string
		existing []string
		want     string
	}{
		{"claude@new", nil, "claude@new"},
		{"claude@new", []string{"cursor@new"}, "claude@new"},
		{"claude@new", []string{"claude@new"}, "claude@new-2"},
		{"claude@new", []string{"claude@new", "claude@new-2"}, "claude@new-3"},
		{"claude@new", []string{"claude@new", "claude@new-3"}, "claude@new-2"},
	}
	for _, tc := range cases {
		got := DedupName(tc.base, tc.existing)
		if got != tc.want {
			t.Errorf("DedupName(%q, %v) = %q, want %q",
				tc.base, tc.existing, got, tc.want)
		}
	}
}

func TestIsValidSuffix(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"new", true},
		{"auth-bug", true},
		{"a_b_c", true},
		{"has space", false},
		{"has:colon", false},
		{"has.dot", false},
		{"", false},
		{"has@at", false},
	}
	for _, tc := range cases {
		if got := IsValidSuffix(tc.in); got != tc.want {
			t.Errorf("IsValidSuffix(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
```

Run: `cd ~/projects/so && go test ./internal/so/... -run Window`
Expected: FAIL — functions undefined.

- [ ] **Step 2: Implement window.go**

Create `internal/so/window.go`:

```go
package so

import (
	"fmt"
	"strings"
)

// ParseWindowName splits a window name "<agent>@<suffix>" into its parts.
// Returns (agent, suffix, true) on success; ("", "", false) otherwise.
func ParseWindowName(name string) (agent, suffix string, ok bool) {
	at := strings.IndexByte(name, '@')
	if at <= 0 || at == len(name)-1 {
		return "", "", false
	}
	return name[:at], name[at+1:], true
}

// DedupName returns base if it's not in existing; otherwise appends -2, -3, ...
// until it finds an unused name. existing may be in any order.
func DedupName(base string, existing []string) string {
	set := map[string]struct{}{}
	for _, e := range existing {
		set[e] = struct{}{}
	}
	if _, taken := set[base]; !taken {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if _, taken := set[candidate]; !taken {
			return candidate
		}
	}
}

// IsValidSuffix returns true if s is a legal window-name suffix:
// non-empty, no whitespace, no `:`, `.`, or `@`.
func IsValidSuffix(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == ':' || r == '.' || r == '@' {
			return false
		}
	}
	return true
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd ~/projects/so && go test ./internal/so/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
cd ~/projects/so
git add internal/so/window.go internal/so/window_test.go
git commit -m "window: ParseWindowName, DedupName, IsValidSuffix"
```

---

## Task 5: tmux wrapper

**Files:**
- Create: `internal/so/tmux.go`
- Create: `internal/so/tmux_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/so/tmux_test.go`:

```go
package so

import (
	"errors"
	"os/exec"
	"testing"
)

// requireTmux skips the test if tmux is not available in PATH.
func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available; skipping integration test")
	}
}

// withFreshSession starts an isolated tmux server with a unique socket
// path so concurrent tests don't collide. Returns a Tmux pointed at that
// socket and a teardown func.
func withFreshSession(t *testing.T) (*Tmux, func()) {
	t.Helper()
	requireTmux(t)
	sock := t.TempDir() + "/sock"
	tx := &Tmux{Socket: sock}
	// Create a baseline session so the server stays alive.
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

// Sanity: error type for missing tmux binary surfaces helpfully.
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
```

Run: `cd ~/projects/so && go test ./internal/so/... -run Tmux`
Expected: FAIL — `Tmux` type undefined.

- [ ] **Step 2: Implement tmux.go**

Create `internal/so/tmux.go`:

```go
package so

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
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
		var pathErr *exec.Error
		if errors.As(err, &pathErr) {
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
	if err := c.Run(); err != nil {
		var pathErr *exec.Error
		if errors.As(err, &pathErr) {
			return false, fmt.Errorf("%w: %v", ErrTmuxUnavailable, err)
		}
		// tmux has-session returns non-zero when the session doesn't exist.
		return false, nil
	}
	return true, nil
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
	for _, w := range wins {
		if w == name {
			return true, nil
		}
	}
	return false, nil
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
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd ~/projects/so && go test ./internal/so/...`
Expected: PASS, or tmux integration tests SKIP if tmux not available.

- [ ] **Step 4: Commit**

```bash
cd ~/projects/so
git add internal/so/tmux.go internal/so/tmux_test.go
git commit -m "tmux: thin wrapper over tmux CLI via os/exec"
```

---

## Task 6: `so send` implementation

**Files:**
- Create: `internal/so/send.go`
- Create: `internal/so/send_test.go`
- Modify: `cmd/so/main.go` (wire up runSend)

- [ ] **Step 1: Write the failing test**

Create `internal/so/send_test.go`:

```go
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
	// Use a window running `cat` so its pane will hold whatever we paste.
	if err := tx.NewWindow("test-base", "claude@auth-bug", "cat"); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	// Give cat a moment to start
	time.Sleep(200 * time.Millisecond)

	if err := SendPrompt(tx, "test-base", "claude@auth-bug", "hello there"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
	// Give the keystrokes a moment to land in cat's output
	time.Sleep(200 * time.Millisecond)

	out, err := tx.run("capture-pane", "-p", "-t", "test-base:claude@auth-bug")
	if err != nil {
		t.Fatalf("capture-pane: %v", err)
	}
	if !strings.Contains(out, "hello there") {
		t.Errorf("expected 'hello there' in capture; got %q", out)
	}
}
```

Run: `cd ~/projects/so && go test ./internal/so/... -run Send`
Expected: FAIL — `SendPrompt` undefined.

- [ ] **Step 2: Implement send.go**

Create `internal/so/send.go`:

```go
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
// is still named "<agent>@new" it waits ~2s for TUI warmup before
// pasting. Empty prompts are rejected. Returns nil on successful paste.
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
```

- [ ] **Step 3: Wire runSend in cmd/so/main.go**

Modify `cmd/so/main.go`. Replace the stub `runSend` with:

```go
func runSend(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: so send <window> [<prompt>]")
		return 2
	}
	window := args[0]
	prompt, err := so.PromptFromArgsOrStdin(args[1:], os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "so send:", err)
		return 2
	}
	if err := so.SendPrompt(so.DefaultTmux(), sessionName, window, prompt); err != nil {
		fmt.Fprintln(os.Stderr, "so send:", err)
		return 1
	}
	return 0
}
```

Add the import at the top:
```go
import (
	"fmt"
	"os"

	"github.com/scottstav/so/internal/so"
)
```

- [ ] **Step 4: Run tests and verify build**

Run: `cd ~/projects/so && go test ./... && go build ./...`
Expected: tests PASS (or skip if no tmux); build succeeds.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/so
git add internal/so/send.go internal/so/send_test.go cmd/so/main.go
git commit -m "send: SendPrompt with @new readiness wait + paste-buffer delivery"
```

---

## Task 7: `so rename` implementation

**Files:**
- Create: `internal/so/rename.go`
- Create: `internal/so/rename_test.go`
- Modify: `cmd/so/main.go` (wire up runRename)

- [ ] **Step 1: Write the failing test**

Create `internal/so/rename_test.go`:

```go
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
	t.Setenv("TMUX_PANE", "%1") // pretend we're in a pane
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
	// Get the pane id of the window we just made.
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
			return // success
		}
	}
	t.Fatalf("expected claude@auth-bug in %v", wins)
}
```

Run: `cd ~/projects/so && go test ./internal/so/... -run Rename`
Expected: FAIL — `RenameCurrentWindow` and `ErrInvalidSuffix` undefined.

- [ ] **Step 2: Implement rename.go**

Create `internal/so/rename.go`:

```go
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
// window names if needed. Returns nil on success.
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
		// No @ in the existing name — unusual; fall back to bare suffix.
		// This shouldn't happen for so-launched windows.
		agent = "agent"
	}
	wantBase := agent + "@" + suffix
	wins, err := tx.ListWindows(session)
	if err != nil {
		return fmt.Errorf("rename: list windows: %w", err)
	}
	// Exclude the current window from the dedup list — we're renaming IT,
	// so its current name shouldn't block.
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
```

- [ ] **Step 3: Wire runRename in cmd/so/main.go**

Replace the stub `runRename`:

```go
func runRename(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: so rename <word>")
		return 2
	}
	if err := so.RenameCurrentWindow(so.DefaultTmux(), sessionName, args[0]); err != nil {
		fmt.Fprintln(os.Stderr, "so rename:", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Run tests and verify build**

Run: `cd ~/projects/so && go test ./... && go build ./...`
Expected: PASS / build succeeds.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/so
git add internal/so/rename.go internal/so/rename_test.go cmd/so/main.go
git commit -m "rename: RenameCurrentWindow preserves agent prefix, dedupes"
```

---

## Task 8: `so ls` implementation

**Files:**
- Create: `internal/so/ls.go`
- Create: `internal/so/ls_test.go`
- Modify: `cmd/so/main.go` (wire up runLs)

- [ ] **Step 1: Write the failing test**

Create `internal/so/ls_test.go`:

```go
package so

import (
	"strings"
	"testing"
)

func TestFormatLs(t *testing.T) {
	rows := []LsRow{
		{Window: "claude@auth-bug", Agent: "claude", Task: "auth-bug"},
		{Window: "cursor@review-pr-42", Agent: "cursor", Task: "review-pr-42"},
		{Window: "claude@new", Agent: "claude", Task: "(idle)"},
	}
	out := FormatLs(rows)
	for _, want := range []string{
		"WINDOW", "AGENT", "TASK",
		"claude@auth-bug", "auth-bug",
		"cursor@review-pr-42", "review-pr-42",
		"claude@new", "(idle)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatLs output missing %q; got:\n%s", want, out)
		}
	}
}

func TestBuildLsRows(t *testing.T) {
	wins := []string{"claude@new", "cursor@review-pr-42", "no-at-sign", "claude@auth-bug"}
	got := BuildLsRows(wins)
	want := map[string]LsRow{
		"claude@new":          {Window: "claude@new", Agent: "claude", Task: "(idle)"},
		"cursor@review-pr-42": {Window: "cursor@review-pr-42", Agent: "cursor", Task: "review-pr-42"},
		"no-at-sign":          {Window: "no-at-sign", Agent: "?", Task: "?"},
		"claude@auth-bug":     {Window: "claude@auth-bug", Agent: "claude", Task: "auth-bug"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows want %d", len(got), len(want))
	}
	for _, row := range got {
		if want[row.Window] != row {
			t.Errorf("for %q: got %+v want %+v", row.Window, row, want[row.Window])
		}
	}
}
```

Run: `cd ~/projects/so && go test ./internal/so/... -run Ls`
Expected: FAIL.

- [ ] **Step 2: Implement ls.go**

Create `internal/so/ls.go`:

```go
package so

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// LsRow is one row of `so ls` output.
type LsRow struct {
	Window string
	Agent  string
	Task   string
}

// BuildLsRows turns raw window names into rows.
func BuildLsRows(windows []string) []LsRow {
	rows := make([]LsRow, 0, len(windows))
	for _, w := range windows {
		agent, suffix, ok := ParseWindowName(w)
		if !ok {
			rows = append(rows, LsRow{Window: w, Agent: "?", Task: "?"})
			continue
		}
		task := suffix
		if suffix == "new" || strings.HasPrefix(suffix, "new-") {
			task = "(idle)"
		}
		rows = append(rows, LsRow{Window: w, Agent: agent, Task: task})
	}
	return rows
}

// FormatLs renders rows as a tab-aligned table to a string.
func FormatLs(rows []LsRow) string {
	var sb strings.Builder
	tw := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "WINDOW\tAGENT\tTASK")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", r.Window, r.Agent, r.Task)
	}
	tw.Flush()
	return sb.String()
}

// Ls runs the `so ls` command, writing output to w.
func Ls(tx *Tmux, session string, w io.Writer) error {
	exists, err := tx.SessionExists(session)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Fprintf(w, "no `%s` session running\n", session)
		return ErrNoSession
	}
	wins, err := tx.ListWindows(session)
	if err != nil {
		return err
	}
	fmt.Fprint(w, FormatLs(BuildLsRows(wins)))
	return nil
}

// ErrNoSession indicates the so session doesn't exist; callers use this
// to set a non-zero exit code while still printing a friendly message.
var ErrNoSession = fmt.Errorf("session not running")
```

- [ ] **Step 3: Wire runLs in cmd/so/main.go**

Replace the stub `runLs`:

```go
func runLs(_ []string) int {
	err := so.Ls(so.DefaultTmux(), sessionName, os.Stdout)
	if err == so.ErrNoSession {
		return 1
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "so ls:", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Run tests and verify build**

Run: `cd ~/projects/so && go test ./... && go build ./...`
Expected: PASS / build succeeds.

- [ ] **Step 5: Commit**

```bash
cd ~/projects/so
git add internal/so/ls.go internal/so/ls_test.go cmd/so/main.go
git commit -m "ls: BuildLsRows + FormatLs + Ls subcommand"
```

---

## Task 9: `so <agent>` launch implementation

**Files:**
- Create: `internal/so/briefing.go`
- Create: `internal/so/launch.go`
- Create: `internal/so/launch_test.go`
- Modify: `cmd/so/main.go` (wire up runLaunch)

- [ ] **Step 1: Implement briefing.go (no tests — trivial passthrough)**

Create `internal/so/briefing.go`:

```go
package so

import (
	"fmt"
	"os"
)

// LoadBriefing returns the contents of ~/.config/so/briefing.md.
// Callers should call EnsureDefaults() first if they want the file to
// be auto-created.
func LoadBriefing() (string, error) {
	b, err := os.ReadFile(BriefingPath())
	if err != nil {
		return "", fmt.Errorf("read briefing: %w", err)
	}
	return string(b), nil
}
```

- [ ] **Step 2: Write the failing test for launch**

Create `internal/so/launch_test.go`:

```go
package so

import (
	"os"
	"strings"
	"testing"
)

func TestLaunch_UnknownAgent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}
	opts := LaunchOpts{Agent: "nonexistent"}
	_, err := Launch(nil, "test-base", opts)
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention unknown agent name; got %v", err)
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("error should list known agents (e.g. claude); got %v", err)
	}
}

func TestLaunch_CreatesWindowAndDedupes(t *testing.T) {
	tx, teardown := withFreshSession(t)
	defer teardown()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := EnsureDefaults(); err != nil {
		t.Fatal(err)
	}

	// Use `cat` as a stand-in agent so we don't need claude installed.
	agentsPath := AgentsConfPath()
	if err := writeFile(agentsPath, "fake=cat\n"); err != nil {
		t.Fatal(err)
	}

	opts := LaunchOpts{
		Agent:           "fake",
		SkipBriefing:    true, // can't deliver briefing without a real TUI
		SkipFocus:       true, // no tmux client attached in tests
	}
	target1, err := Launch(tx, "test-base", opts)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if !strings.HasSuffix(target1, ":fake@new") {
		t.Errorf("got target %q, want suffix :fake@new", target1)
	}
	target2, err := Launch(tx, "test-base", opts)
	if err != nil {
		t.Fatalf("Launch (2nd): %v", err)
	}
	if !strings.HasSuffix(target2, ":fake@new-2") {
		t.Errorf("got target %q, want suffix :fake@new-2", target2)
	}
}

// writeFile is a tiny helper so the test stays clean.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
```

Run: `cd ~/projects/so && go test ./internal/so/... -run Launch`
Expected: FAIL.

- [ ] **Step 3: Implement launch.go**

Create `internal/so/launch.go`:

```go
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
	// Used in tests (no attached client) and may be set by callers that
	// explicitly want no focus change.
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

	// Ensure the session exists, computing the deduped window name based
	// on whatever windows are already present.
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
		// First launch: create session with our window in one shot so we
		// don't end up with tmux's default window hanging around.
		if err := tx.NewSessionWithWindow(session, winName, cmd); err != nil {
			return "", fmt.Errorf("launch: create session: %w", err)
		}
	} else {
		if err := tx.NewWindow(session, winName, cmd); err != nil {
			return "", fmt.Errorf("launch: new-window: %w", err)
		}
	}
	target := session + ":" + winName

	// Inject briefing.
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

	// Focus the new window unless suppressed.
	if !opts.SkipFocus {
		// switch-client only works if a client is attached to the session;
		// errors are non-fatal (we still launched successfully).
		_ = tx.SwitchClient(session)
		_ = tx.SelectWindow(target)
	}

	return target, nil
}
```

- [ ] **Step 4: Wire runLaunch in cmd/so/main.go**

Replace the stub `runLaunch`:

```go
func runLaunch(agent string, _ []string) int {
	// Detect: are we already inside tmux? If not, we need to attach
	// after creating the window.
	insideTmux := os.Getenv("TMUX") != ""

	tx := so.DefaultTmux()
	target, err := so.Launch(tx, sessionName, so.LaunchOpts{
		Agent:     agent,
		SkipFocus: !insideTmux, // outside tmux: we'll attach ourselves below
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "so:", err)
		return 1
	}
	fmt.Println(target)

	if !insideTmux {
		// Replace this process with `tmux attach -t so`, selecting the new
		// window first so it's the active one when we attach.
		_ = tx.SelectWindow(target)
		tmuxBin, err := exec.LookPath("tmux")
		if err != nil {
			fmt.Fprintln(os.Stderr, "so: tmux not found in PATH:", err)
			return 1
		}
		if err := syscall.Exec(tmuxBin, []string{"tmux", "attach-session", "-t", sessionName}, os.Environ()); err != nil {
			fmt.Fprintln(os.Stderr, "so: exec tmux attach:", err)
			return 1
		}
	}
	return 0
}
```

Add to the imports:
```go
import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/scottstav/so/internal/so"
)
```

- [ ] **Step 5: Run tests and verify build**

Run: `cd ~/projects/so && go test ./... && go build ./...`
Expected: PASS (or skip integration tests if no tmux) / build succeeds.

- [ ] **Step 6: Commit**

```bash
cd ~/projects/so
git add internal/so/briefing.go internal/so/launch.go internal/so/launch_test.go cmd/so/main.go
git commit -m "launch: so <agent> spawns window, injects briefing, switches focus"
```

---

## Task 10: Help text + final dispatch polish

**Files:**
- Modify: `cmd/so/main.go`

- [ ] **Step 1: Improve printUsage to reflect real behavior**

Replace `printUsage` in `cmd/so/main.go`:

```go
func printUsage() {
	fmt.Fprintln(os.Stderr, `so — Scott's orchestrator. Launches CLI agents in a shared tmux session.

Usage:
  so <agent> [name]       launch an agent (e.g. so claude, so cursor)
  so send <window> <msg>  feed a prompt to an agent's window
                          (msg may be passed via stdin if "-" or omitted)
  so rename <word>        rename the calling window's task suffix
  so ls                   list active agent windows
  so -h | --help          show this help

Configuration files (auto-created on first run):
  ~/.config/so/agents.conf   agent registry, format: name=command
  ~/.config/so/briefing.md   text injected as the first prompt at launch

The tmux session is named "so". Windows are named "<agent>@<task>".`)
}
```

- [ ] **Step 2: Add a smoke test for the binary's CLI behavior**

Create `cmd/so/main_test.go`:

```go
package main

import (
	"os/exec"
	"strings"
	"testing"
)

// build the binary into a temp dir and return its path.
func buildBin(t *testing.T) string {
	t.Helper()
	bin := t.TempDir() + "/so"
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func TestCLI_HelpFlag(t *testing.T) {
	bin := buildBin(t)
	out, err := exec.Command(bin, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("so --help: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Scott's orchestrator") {
		t.Errorf("expected help banner; got: %s", out)
	}
}

func TestCLI_NoArgs(t *testing.T) {
	bin := buildBin(t)
	out, err := exec.Command(bin).CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit; got success with: %s", out)
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Errorf("expected usage on no-args; got: %s", out)
	}
}
```

- [ ] **Step 3: Run tests and verify build**

Run: `cd ~/projects/so && go test ./... && go build ./...`
Expected: PASS / build succeeds.

- [ ] **Step 4: Commit**

```bash
cd ~/projects/so
git add cmd/so/main.go cmd/so/main_test.go
git commit -m "cli: real help text and basic CLI smoke tests"
```

---

## Task 11: Install + cc alias + v0 smoke

**Files:**
- Modify: `~/dotfiles/.profile` (alias `cc` to `so claude`)
- Optionally: install the binary somewhere on PATH.

- [ ] **Step 1: Build and install the binary**

Run:
```bash
cd ~/projects/so && go install ./cmd/so
which so
```

Expected: `~/go/bin/so` (or wherever `$GOBIN`/`$GOPATH/bin` points).

Confirm `~/go/bin` (or your `GOBIN`) is on `PATH`. If not, you'll need to add it. Sanity:
```bash
so --help
```

Expected: help banner prints, exit 0.

- [ ] **Step 2: Replace the cc function with an alias to so claude**

Read `~/dotfiles/.profile`. Find the `cc()` function block (around line 39 — verify with `grep -n '^cc()' ~/dotfiles/.profile`). Replace the entire `cc() { ... }` function (lines 39–76 inclusive — recount with grep) with:

```bash
# Claude Code in tmux — now delegates to `so claude` (Scott's orchestrator).
# Original cc() function archived in git history.
alias cc='so claude'
```

Leave the `cca()` function (lines 79–85ish) untouched for now — it still works because `cd "$dir" && cc "$@"` resolves the alias.

- [ ] **Step 3: Verify by opening a new terminal**

Source the profile (or open a fresh terminal):
```bash
source ~/dotfiles/.profile
type cc       # should show: cc is aliased to `so claude'
```

- [ ] **Step 4: Manual smoke test**

In a fresh terminal (outside tmux):
```bash
so claude
```

Expected: tmux session `so` opens with a `claude@new` window, Claude starts loading, and after a couple seconds the briefing is pasted into the prompt. Press Ctrl-B then `:kill-window` to clean up, or just exit Claude.

If you have cursor-agent installed and trusted in this dir:
```bash
# From inside the so session:
so cursor
```

Expected: new window `cursor@new` opens, cursor-agent launches, briefing arrives.

In a Claude pane that's been renamed via `so rename auth-bug`:
```bash
so ls
```

Expected: shows `claude@auth-bug` in the AGENT/TASK columns.

- [ ] **Step 5: Tag v0.0.1**

```bash
cd ~/projects/so
git tag v0.0.1
git log --oneline
```

The commit log should read like a clean walk-through of the implementation.

---

## Self-review against spec

After completing all tasks, verify each spec requirement has been addressed:

| Spec requirement | Implementing task |
| --- | --- |
| One Go binary named `so` | 1 |
| `so <agent>` subcommand | 9 |
| `so send` subcommand | 6 |
| `so rename` subcommand | 7 |
| `so ls` subcommand | 8 |
| Tmux session named `so` | 9 (`sessionName` const in main.go from Task 1) |
| Window names `<agent>@<suffix>`, dedup with `-N` | 4 + 9 |
| Initial suffix `new` | 9 |
| Rename preserves agent prefix | 7 |
| `~/.config/so/agents.conf` registry, embedded default | 2 + 3 |
| `~/.config/so/briefing.md`, embedded default | 2 + 9 |
| Briefing auto-injected at launch with sleep 2 | 9 |
| `so send` waits if target ends in `@new` | 6 |
| `so send` paste-buffer + Enter mechanism | 5 + 6 |
| Focus switches to new window on launch | 9 |
| `cc` aliased to `so claude` | 11 |
| Code-review acceptance scenario | Manual smoke in 11 |

If a row has no task, add a task before starting implementation.
