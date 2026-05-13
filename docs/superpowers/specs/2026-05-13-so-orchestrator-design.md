# `so` — v0 design

Date: 2026-05-13
Status: design (pre-implementation)

## What this is

`so` is a small CLI that orchestrates command-line AI agents (Claude Code, cursor-agent, anything else that runs in a TTY) inside a single shared tmux session. It provides a primitive for spawning agents in panes and a primitive for feeding prompts to a running agent. Those two primitives compose to express multi-agent flows — the first concrete one being a peer-review loop where one agent dispatches another to review its work.

`so` replaces the user's existing `cc` shell function (which only launches Claude). The name is short for "Scott's orchestrator".

## Motivation

The user has been working with Claude Code (`cc` launcher) and wants to:
1. Launch cursor-agent sessions the same way `cc` launches Claude sessions.
2. Feed an existing, already-running interactive agent a prompt as if the user typed it.
3. Use (2) from anywhere — including from inside *another* agent's session — so agents can hand work to each other.

The motivating use case: Claude finishes a PR, the user asks "have cursor review this and address its feedback," cursor spawns in a new pane, reviews the diff, then delivers its review back into Claude's pane (where Claude reads it as a user message and addresses the feedback). The design must support that flow as a *composition* of small primitives — not as a special-cased feature.

The user explicitly rejected:
- A "wrapper process" model where each agent runs inside an `so`-aware shim that mediates I/O.
- A "mailbox / message bus" model with structured envelopes between agents.
- A "master agent" (Gastown-style mayor) that coordinates other agents.

Those were rejected for being too much machinery for a v0 that fits in a few hundred lines.

## Building blocks (the mental model)

The design is structured around four building blocks. v0 implements blocks 2, 3, and 4 (block 1 already exists).

1. **`cc`** — already exists. Launches a Claude session in a known tmux topology. **Will be replaced by `so claude`.**
2. **Launch any agent** — `so <agent>` spawns a new tmux window running that agent. Same shape as today's `cc`, but agent-agnostic. Does not give the agent a task; just stands it up.
3. **Feed a prompt into a running agent** — `so send <window> <prompt>` delivers a prompt into the live, interactive agent in that window, as if a human typed it. Critically: not via the agent's `-p`/print mode. The agent is already running; we paste input at it. Works identically regardless of which agent is in the window.
4. **Block 3 is callable from anywhere** — including from inside another agent's session. When cursor finishes a review, cursor itself runs `so send claude@xyz "<review>"` to deliver back. There is no special "send result back" pathway; that flow is just *a use* of block 3.

The code-review feature is then: block 2 + block 3 (with self-routed return instructions embedded in the prompt) + block 3 again (from cursor, sending back). No new code paths needed for it.

## Goals

- One Go binary, `so`, installed to `$PATH`.
- Four subcommands in v0: launch (`so <agent>`), `so send`, `so rename`, `so ls`.
- All agent sessions live in one tmux session named `so`.
- Window names encode `<agent>@<task>` so the same window name reads as a routing address.
- Agents are briefed about the `so` ecosystem at launch via the same prompt-feeding mechanism users will use.
- Adding a new agent type = one line in a config file. No code change.
- Replaces `cc` for the user's daily workflow (with a `cc` alias kept for muscle memory and to avoid breaking Emacs/Hyprland callers).

## Non-goals (deliberately out of v0)

- No long-running daemon, broker, message bus, or watcher service.
- No structured envelope/protocol between agents. Messages are plain text pastes.
- No agent-supervision features (timeouts, retries, "is the agent stuck"). If they're needed later, they become new subcommands.
- No "master" / orchestrator agent reasoning about other agents.
- No multi-turn back-and-forth choreography beyond what `so send` already enables (repeated calls).
- No persistence/logging of conversations across sessions. Tmux scrollback is the log.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ tmux session: `so`                                              │
│                                                                 │
│  ┌──────────────────────┐    ┌──────────────────────┐           │
│  │ window: claude@new   │    │ window: cursor@new   │           │
│  │  $ claude            │    │  $ cursor-agent      │           │
│  │  > [agent briefing]  │    │  > [agent briefing]  │           │
│  │  > [user task]       │    │  > [task from claude]│           │
│  │  ...                 │    │  ...                 │           │
│  └──────────────────────┘    └──────────────────────┘           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

External state:
  ~/.config/so/agents.conf   # agent registry (name → command)
  ~/.config/so/briefing.md   # text injected as first prompt at launch
```

All `so` does is shell out to `tmux` (and to the agent binary at launch time). It is stateless across invocations except for whatever tmux + config files persist.

## Subcommand surface (v0)

### `so <agent> [name]`

Launches an agent (block 2). Aliases like `so claude`, `so cursor`. Behavior:

- Resolves `<agent>` against `~/.config/so/agents.conf`. Errors if unknown.
- Ensures the tmux session `so` exists; creates if not.
- Creates a new window in `so` with name `<agent>@<name>`. `<name>` defaults to `new`. Deduped with `-2`, `-3` if a window of that name already exists.
- Starts the agent's command inside the new window.
- Sleeps 2 seconds to let the agent's TUI warm up.
- Injects the briefing (contents of `~/.config/so/briefing.md`) into the new window using the same mechanism as `so send`.
- Behavior re: focus (mirrors `cc`'s current pattern — always bring the user to the new window):
  - If user is outside any tmux session: `exec tmux attach -t so` selecting the new window.
  - If user is inside a tmux session that isn't `so`: `tmux switch-client -t so` and select the new window.
  - If user is inside the `so` session: `tmux select-window` to the new window.
  - Rationale: even when an agent (block 4 case) spawns another agent, switching focus is desirable — the human gets to watch the new agent's work. Process activity is unaffected by focus; the sender pane keeps running.
- Prints the new window's tmux target (e.g. `so:claude@new`) to stdout. Scripts and agents capture this even though their pane is no longer focused.
- Exit code 0 on success, non-zero on resolution or tmux errors.

### `so send <window> <prompt>`

Feeds a prompt to the running agent at `<window>` (block 3 / block 4).

- `<window>` is a bare window name resolved within the `so` session (e.g. `cursor@auth-bug`). Not a full tmux target — the session prefix is implicit.
- `<prompt>` may be passed as a positional argument or read from stdin (if positional is omitted or set to `-`).
- An empty prompt (no positional arg and empty stdin) is an error — `so send` will not paste an empty message.
- If the target window's name ends in `@new`, the agent hasn't done its first task yet — `so send` waits briefly (`sleep 2`) before delivering, to absorb TUI warmup. Once the agent renames its window, future sends are immediate.
- Delivery mechanism:
  1. `tmux load-buffer -b so-<pid> -` (load prompt from stdin into a tmux buffer).
  2. `tmux paste-buffer -b so-<pid> -t so:<window>` (paste into target pane; relies on bracketed-paste mode so multi-line lands as one input).
  3. `tmux send-keys -t so:<window> Enter` (submit).
  4. Delete the tmux buffer.
- Returns 0 on successful delivery (paste-buffer succeeded). Returns non-zero if the target window doesn't exist.

### `so rename <word>`

Renames the *calling* window (block 2's "agent self-rename after first task").

- Reads `$TMUX_PANE` to identify the calling pane. Errors out if unset (must be called from inside tmux).
- Reads the calling window's current name. Splits on the first `@`.
- Keeps the part before `@` (the agent prefix). Replaces the suffix with `<word>`.
- E.g. from `claude@new` → `so rename auth-bug` → `claude@auth-bug`.
- Dedups against existing window names in the `so` session with `-2`, `-3` suffix (rare but possible).
- If the calling window has no `@` in its name, prepend the calling pane's command type as best-guess — but this case shouldn't happen for `so`-launched windows.

### `so ls`

Lists windows in the `so` session.

- Output format (tab-separated columns):
  ```
  WINDOW                 AGENT    TASK
  claude@auth-bug        claude   auth-bug
  cursor@review-pr-42    cursor   review-pr-42
  claude@new             claude   (idle)
  ```
- Where `AGENT` is the prefix before `@` and `TASK` is the suffix after `@` (or `(idle)` if still `@new`).
- Exit code 0 even with zero windows. Exit 1 if the `so` session doesn't exist (and emit a friendly "no `so` session is running" message).

## Configuration

### `~/.config/so/agents.conf`

Simple `key=value` lines, `#` for comments. Default ships with:

```
# Agent registry. Format: <name>=<command>
# Adding an agent: add a line. No code change required.
claude=claude
cursor=cursor-agent
```

The value is the binary name (or full path) that gets executed in the new tmux window. On first run, if the file doesn't exist, `so` writes the default. v0 does **not** support per-agent flags, env, or readiness regex — the command is just `exec` after `tmux new-window`.

### `~/.config/so/briefing.md`

The text injected as the first prompt into every freshly-spawned agent. Default content:

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

On first run, if the file doesn't exist, `so` writes this default. Users can edit it freely; their edits persist.

## Window naming and dedup

- Format: `<agent>@<suffix>`.
- Initial suffix on launch: `new`.
- Dedup: if `claude@new` exists, next is `claude@new-2`, then `claude@new-3`, etc.
- Renaming via `so rename`: replaces only the suffix, preserves the agent prefix.
- Names should not contain spaces or characters that confuse tmux's target syntax (`:`, `.`, whitespace). v0 rejects invalid characters in `so rename` with a clear error.

## End-to-end: the code-review story

This is the v0 acceptance scenario. No new code is written for it — it's pure composition of `so <agent>`, `so send`, and `so rename`.

User context: Claude has opened a PR. User wants cursor's review.

1. User in Claude pane: *"Have cursor-agent review this PR and address its feedback."*
2. Claude (per its briefing) runs (its own window is, say, `claude@feature-x`):
   ```
   target=$(so cursor)                                 # → so:cursor@new (block 2)
   so send cursor@new "Review the diff for PR #N in this repo. \
     When done, run: so send claude@feature-x '<your review>'."
                                                       # block 3, with self-routed return
   ```
3. cursor-agent's pane (named `cursor@new`):
   - Receives briefing on launch.
   - Receives the review task as its first "user" prompt.
   - Renames its window: `so rename review-pr-N` → `cursor@review-pr-N`.
   - Performs the review.
   - Runs `so send claude@feature-x "[review of PR #N] ..."`.
4. Claude's pane receives the review as a user message. Claude addresses the feedback.

The platform did three things: spawned a pane, pasted a prompt, pasted another prompt. The "review" framing lives in the prompt text, not in any code.

## Migration from `cc`

- `cc` is a shell function in `~/dotfiles/.profile`. It launches Claude in a tmux session named `claude`.
- After `so` ships: `cc` becomes a thin alias to `so claude`. Same behavior from the user's perspective.
- `cc-picker` (the Hyprland-bound fzf dir picker) keeps working unchanged — it ends with `bash -lic 'cc $CC_ARGS'`. We keep `cc` working so this is invisible.
- The tmux session name flips from `claude` to `so`. Pre-existing `claude` session can be killed; no migration of running panes.
- Emacs integrations that send to `claude` session windows will need to point at `so` session windows when the user gets around to updating them. Not blocking for v0.

## Implementation notes

- Language: Go (1.24+).
- Module: `github.com/scottstav/so`.
- Layout:
  ```
  cmd/so/main.go         # CLI entry, subcommand dispatch
  internal/agents/       # agents.conf loader, default writer
  internal/briefing/     # briefing.md loader, default writer
  internal/tmux/         # thin wrapper over `os/exec tmux ...`
  internal/window/       # window-name parsing, dedup
  ```
- Subcommand dispatch: stdlib `flag` + a small switch on `os.Args[1]`. Cobra is overkill for 4 commands.
- All tmux interaction shells out via `os/exec`. No tmux library; tmux's CLI is the API.
- Embedded defaults: `//go:embed` the default `agents.conf` and `briefing.md` so the binary can self-bootstrap config on first run.
- Tests: unit-test the pure helpers (config parsing, window-name parsing, dedup). The tmux-shelling side is integration-tested by exercising the binary against a real tmux server in CI (or skipped if no tmux on the runner).

## Failure modes & v0 handling

- **Unknown agent**: `so foo` → error `unknown agent "foo"; defined agents: claude, cursor` (read from agents.conf).
- **No tmux installed**: `so` errors out early with "tmux is required".
- **Target window doesn't exist for `so send`**: error with the list of currently active windows (`so ls`-style hint).
- **Agent fails to start**: tmux window will show the error; `so` itself can't tell. (Acceptable for v0.)
- **Briefing injection lands during a "trust this folder" prompt**: dismiss manually that one time; folder is now trusted. Not worth automating.
- **`so send` called from a pane outside the `so` session**: still works — `so send` doesn't care where the caller is, only where the target is. Useful for scripts and Emacs.
- **`so rename` called outside tmux**: error with "must be called from inside a tmux pane".

## What this enables for v1+ (without forcing now)

- Adding a new agent (codex, aider, a local "run-tests-and-summarize" wrapper): one line in `agents.conf`.
- Multi-turn back-and-forth between same two agents: just repeat `so send` from each side. Topology stays flat.
- Per-agent readiness regex / startup-prompt dismissal: add fields to `agents.conf` (`command=`, `ready_regex=`, etc.) and a richer parser. Only if the simple `sleep 2` proves insufficient.
- A coordinator/master agent: ship it as just another agent type, invoked via `so send` like anything else. The platform doesn't need special support for it.
- Routing by intent ("send this to whichever agent is reviewing PR #42"): a `so route` subcommand that resolves windows by name patterns. Add when there's an actual need.

## Acceptance criteria for v0

- `so claude` from a terminal outside tmux: tmux session `so` exists, window `claude@new` is running Claude, briefing has been delivered, user is attached and focused on the new window.
- `so cursor` while inside the `so` session: new window `cursor@new` exists, briefing delivered, focus switches to the new window.
- From inside Claude's pane: `so send cursor@new "hello"` causes "hello" to appear as a user input in the cursor pane and be submitted.
- From inside cursor's pane after `so rename review-pr-1`: `so ls` shows `cursor@review-pr-1` with `TASK` column = `review-pr-1`.
- `cc` continues to work for users still typing `cc` out of habit (aliased to `so claude`).
