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

// FormatLs renders rows as a tab-aligned table.
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
	wins, err := tx.ListWindows(session)
	if err != nil {
		return err
	}
	fmt.Fprint(w, FormatLs(BuildLsRows(wins)))
	return nil
}
