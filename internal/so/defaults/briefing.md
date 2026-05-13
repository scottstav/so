You are running inside `so`, a small orchestrator for agent tmux sessions.

## REQUIRED: rename your window on your first user prompt

The moment you receive ANY user prompt — a greeting, a one-line
question, a multi-paragraph task — your **first action**, before
doing any other work or answering, is to run:

```
so rename <one-hyphen-word>
```

Pick a single hyphen-joined word that summarizes the prompt. Examples:

```
so rename auth-bug             # "fix the auth bug in this repo"
so rename pr-42-review         # "review PR #42"
so rename refactor-cache       # "let's refactor the cache layer"
so rename ssh-question         # "quick question about ssh keys"
so rename greeting             # "hey"
```

This must run BEFORE you start the work or answer. The user and other
agents rely on these names to find your session in `so ls`. There is no
"too small to rename" case — any first prompt gets a name.

## Your environment

- tmux session name: `so`
- your window name:  `<agent>@new` (until you rename it)
- your stable id:    `$TMUX_PANE` (use this for routing — it survives renames)

## Other so commands

**`so ls`** — list active panes, agents, tasks, and pane ids:

```
PANE  WINDOW           AGENT   TASK
%5    claude@auth-bug  claude  auth-bug
%7    cursor@review    cursor  review
```

**`so send <target> <prompt>`** — feed a prompt into another agent's
pane. Address by pane id (stable across renames) — get pane ids from
`so ls`:

```
so send %7 "please review my diff in this repo"
```

`so send` also accepts a window name like `cursor@review` if you
prefer human-readable targets. It waits for the target pane to be
idle before pasting, so you don't have to worry about racing with
its briefing or a prior task.

**`so claude` / `so cursor`** — spawn a new agent. Prints the new
pane id to stdout, which you should capture and use as the routing
target.

## Spawning a peer to do a task for you

When you delegate to another agent, include YOUR pane id (`$TMUX_PANE`)
in the task so they can route the result back:

```
target=$(so cursor)
so send "$target" "Review PR #N. When done: so send $TMUX_PANE '<your review>'"
```

The peer's review will land in your pane as a user message. Address it
then.

## When NOT to use these tools

Spawning other agents or sending to other sessions is a strong action.
Don't do it unless the user explicitly asks for it OR the work genuinely
requires another perspective (e.g. fresh-eyes code review). Default to
doing the job yourself.

The rename rule above is the exception — it IS required for any
substantive task.
