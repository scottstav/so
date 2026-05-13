package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/scottstav/so/internal/so"
)

const defaultSession = "so"

// sessionName resolves the tmux session name. SO_SESSION env var
// overrides the default — used by sibling launchers like claude-personal
// to keep their work isolated.
func sessionName() string {
	if s := os.Getenv("SO_SESSION"); s != "" {
		return s
	}
	return defaultSession
}

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
	case "brief":
		os.Exit(runBrief(args))
	case "-h", "--help", "help":
		printUsage()
		os.Exit(0)
	default:
		os.Exit(runLaunch(cmd, args))
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `so — Scott's orchestrator. Launches CLI agents in a shared tmux session.

Usage:
  so <agent> [-- args...] launch an agent (e.g. so claude, so cursor --resume)
                          args after the optional '--' are passed verbatim
                          to the agent's command (e.g. claude --resume).
                          prints the new pane id to stdout.
  so send <target> <msg>  feed a prompt to an agent's pane
                          <target> = pane id (%42), window (cursor@task), or so:window
                          <msg> may be passed via stdin if "-" or omitted
                          waits for the target pane to be idle before delivering
  so rename <word>        rename the calling window's task suffix
  so ls                   list active agent panes (PANE / WINDOW / AGENT / TASK)
  so brief                print the so briefing (useful for resumed sessions)
  so -h | --help          show this help

Configuration files (auto-created on first run):
  ~/.config/so/agents.conf   agent registry, format: name=command
  ~/.config/so/briefing.md   text injected as the first prompt at launch

The tmux session is named "so". Windows are named "<agent>@<task>".
Pane ids are the stable routing addresses (they survive window renames).`)
}

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
	if err := so.SendPrompt(so.DefaultTmux(), sessionName(), window, prompt); err != nil {
		fmt.Fprintln(os.Stderr, "so send:", err)
		return 1
	}
	return 0
}

func runRename(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: so rename <word>")
		return 2
	}
	if err := so.RenameCurrentWindow(so.DefaultTmux(), sessionName(), args[0]); err != nil {
		fmt.Fprintln(os.Stderr, "so rename:", err)
		return 1
	}
	return 0
}

func runBrief(_ []string) int {
	if err := so.Brief(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "so brief:", err)
		return 1
	}
	return 0
}

func runLs(_ []string) int {
	err := so.Ls(so.DefaultTmux(), sessionName(), os.Stdout)
	if err == so.ErrNoSession {
		return 1
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "so ls:", err)
		return 1
	}
	return 0
}

func runLaunch(agent string, args []string) int {
	// Optional `--` separator: `so claude -- --resume`. Strip if present.
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	insideTmux := os.Getenv("TMUX") != ""

	tx := so.DefaultTmux()
	result, err := so.Launch(tx, sessionName(), so.LaunchOpts{
		Agent:     agent,
		ExtraArgs: args,
		SkipFocus: !insideTmux,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "so:", err)
		return 1
	}
	// Print the pane id — the stable routing target. Window name is
	// visible in `so ls`.
	fmt.Println(result.PaneID)

	if !insideTmux {
		_ = tx.SelectWindow(result.Target)
		tmuxBin, err := exec.LookPath("tmux")
		if err != nil {
			fmt.Fprintln(os.Stderr, "so: tmux not found in PATH:", err)
			return 1
		}
		if err := syscall.Exec(tmuxBin, []string{"tmux", "attach-session", "-t", sessionName()}, os.Environ()); err != nil {
			fmt.Fprintln(os.Stderr, "so: exec tmux attach:", err)
			return 1
		}
	}
	return 0
}
