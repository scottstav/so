package main

import (
	"fmt"
	"os"

	"github.com/scottstav/so/internal/so"
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

// Stubs — real impls land in later tasks.
func runLs(args []string) int     { fmt.Fprintln(os.Stderr, "ls: not implemented"); return 1 }
func runLaunch(agent string, args []string) int {
	fmt.Fprintln(os.Stderr, "launch: not implemented")
	return 1
}
