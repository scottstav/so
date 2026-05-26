package so

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// LsRow is one row of `so ls` output.
type LsRow struct {
	Pane   string // tmux pane id, e.g. "%42"
	Window string
	Agent  string
	Task   string
}

// BuildLsRows turns parallel slices of windows and pane ids into rows.
// panes must be the same length as windows; if a pane id is unknown,
// pass an empty string in that slot.
func BuildLsRows(windows, panes []string) []LsRow {
	rows := make([]LsRow, 0, len(windows))
	for i, w := range windows {
		var pane string
		if i < len(panes) {
			pane = panes[i]
		}
		agent, suffix, ok := ParseWindowName(w)
		if !ok {
			rows = append(rows, LsRow{Pane: pane, Window: w, Agent: "?", Task: "?"})
			continue
		}
		task := suffix
		if suffix == "new" || strings.HasPrefix(suffix, "new-") {
			task = "(idle)"
		}
		rows = append(rows, LsRow{Pane: pane, Window: w, Agent: agent, Task: task})
	}
	return rows
}

// FormatLs renders rows as a tab-aligned table.
func FormatLs(rows []LsRow) string {
	var sb strings.Builder
	tw := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PANE\tWINDOW\tAGENT\tTASK")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Pane, r.Window, r.Agent, r.Task)
	}
	tw.Flush()
	return sb.String()
}

// ErrNoSession indicates the so session doesn't exist.
var ErrNoSession = fmt.Errorf("session not running")

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
	// Use a combined list-windows format so we get windows and pane ids
	// in one tmux call, in matching order. The `=` prefix forces an exact
	// session match so a query for "so" doesn't pick up "so-personal".
	out, err := tx.run("list-windows", "-t", "="+session, "-F", "#W\t#{pane_id}")
	if err != nil {
		return err
	}
	var wins, panes []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		wins = append(wins, parts[0])
		if len(parts) > 1 {
			panes = append(panes, parts[1])
		} else {
			panes = append(panes, "")
		}
	}
	fmt.Fprint(w, FormatLs(BuildLsRows(wins, panes)))
	return nil
}
