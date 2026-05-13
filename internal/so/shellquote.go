package so

import "strings"

// shellQuote returns s in a form safe for inclusion in a POSIX shell
// command line. "Safe" characters are passed through verbatim; anything
// else is single-quoted with embedded single quotes escaped.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		if !isShellSafe(r) {
			return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
		}
	}
	return s
}

func isShellSafe(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z',
		r >= 'A' && r <= 'Z',
		r >= '0' && r <= '9':
		return true
	}
	switch r {
	case '-', '_', '.', '/', '+', ',', '=', ':', '@', '%':
		return true
	}
	return false
}

// buildAgentCommand joins cmd with extra args, shell-quoting each arg.
// Returns a single command string suitable for `tmux new-window <cmd>`.
func buildAgentCommand(cmd string, extraArgs []string) string {
	if len(extraArgs) == 0 {
		return cmd
	}
	parts := make([]string, 0, 1+len(extraArgs))
	parts = append(parts, cmd)
	for _, a := range extraArgs {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}
