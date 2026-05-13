# so

Scott's [agent] orchestrator

An opinianated platform for managing coding agents via tmux.

## Commands

| Command | Description |
| --- | --- |
| `so <agent> [-- args...]` | Launch an agent in a new tmux window (e.g. `so claude`, `so cursor -- --resume`). Args after `--` pass through verbatim. Prints the new pane id to stdout. |
| `so send <target> <msg>` | Feed a prompt into another pane. `<target>` is a pane id (`%42`), window name (`cursor@task`), or `so:window`. `<msg>` can come from stdin if omitted or `-`. Waits for the target to be idle. |
| `so rename <word>` | Rename the calling window's task suffix to `<word>`. |
| `so ls` | List active agent panes — `PANE`, `WINDOW`, `AGENT`, `TASK`. |
| `so brief` | Print the so briefing (useful when resuming a session). |
| `so -h`, `so --help` | Show help. |

## Briefing

On launch (`so <agent>`), the contents of `briefing.md` are auto-injected as the agent's first prompt. This way agents can use the system.

Resumed launches skip the auto-briefing. Run `so brief` from inside the resumed pane to pull the briefing on demand.

## Configuration

Config lives under `$XDG_CONFIG_HOME/so` (defaults to `~/.config/so`). Both files are auto-created with defaults on first run.

| File | Purpose |
| --- | --- |
| `agents.conf` | Agent registry. One `name=command` per line (e.g. `claude=claude`, `cursor=cursor-agent`). Determines what `so <agent>` will exec. |
| `briefing.md` | Markdown text injected as the agent's first prompt at launch. Edit this to change what every agent sees on spawn. |

The agents-config path can be overridden with the `SO_AGENTS_CONF` env var. The tmux session name (default `so`) can be overridden with `SO_SESSION` — useful when running sibling launchers in parallel.
