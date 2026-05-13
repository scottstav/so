# so

Scott's [agent] orchestrator

An opinianated platform for managing coding agents via tmux.

## Installation

Requires Go 1.26+ and `tmux` on `$PATH`.

```sh
go install github.com/scottstav/so/cmd/so@latest
```

This drops the `so` binary in `$(go env GOBIN)` (or `$(go env GOPATH)/bin` if `GOBIN` is unset) — make sure that directory is on your `$PATH`. To pin to a specific release, swap `@latest` for `@v0.0.7`.

Or build from source:

```sh
git clone https://github.com/scottstav/so
cd so
go install ./cmd/so
```

## Commands

Most commands are run by a "briefed" `so` agent.

| Command | Description |
| --- | --- |
| `so <agent> [-- args...]` | Launch an agent in a new tmux window (e.g. `so claude`, `so cursor -- --resume`). Args after `--` pass through verbatim. Prints the new pane id to stdout. This is probably the only command youll ever need to run manually.|
| `so brief` | Print the so briefing. You might also run this one manually to give an agent access to the system.|
| `so send <target> <msg>` | Feed a prompt into another pane. `<target>` is a pane id (`%42`), window name (`cursor@task`), or `so:window`. `<msg>` can come from stdin if omitted or `-`. Waits for the target to be idle. |
| `so rename <word>` | Rename the calling window's task suffix to `<word>`. |
| `so ls` | List active agent panes — `PANE`, `WINDOW`, `AGENT`, `TASK`. |
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

## Releases

Releases are plain git tags following [semver](https://semver.org/) — `vMAJOR.MINOR.PATCH`. To cut a new release:

```sh
git tag -a v0.0.8 -m "v0.0.8: <one-line summary>"
git push origin v0.0.8
```

`go install github.com/scottstav/so/cmd/so@v0.0.8` will then resolve to that tag via the Go module proxy. See the [tags page](https://github.com/scottstav/so/tags) for the full release list.
