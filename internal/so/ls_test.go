package so

import (
	"strings"
	"testing"
)

func TestFormatLs(t *testing.T) {
	rows := []LsRow{
		{Pane: "%5", Window: "claude@auth-bug", Agent: "claude", Task: "auth-bug"},
		{Pane: "%7", Window: "cursor@review-pr-42", Agent: "cursor", Task: "review-pr-42"},
		{Pane: "%3", Window: "claude@new", Agent: "claude", Task: "(idle)"},
	}
	out := FormatLs(rows)
	for _, want := range []string{
		"PANE", "WINDOW", "AGENT", "TASK",
		"%5", "claude@auth-bug", "auth-bug",
		"%7", "cursor@review-pr-42", "review-pr-42",
		"%3", "claude@new", "(idle)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatLs output missing %q; got:\n%s", want, out)
		}
	}
}

func TestBuildLsRows(t *testing.T) {
	wins := []string{"claude@new", "cursor@review-pr-42", "no-at-sign", "claude@auth-bug"}
	panes := []string{"%1", "%2", "%3", "%4"}
	got := BuildLsRows(wins, panes)
	want := map[string]LsRow{
		"claude@new":          {Pane: "%1", Window: "claude@new", Agent: "claude", Task: "(idle)"},
		"cursor@review-pr-42": {Pane: "%2", Window: "cursor@review-pr-42", Agent: "cursor", Task: "review-pr-42"},
		"no-at-sign":          {Pane: "%3", Window: "no-at-sign", Agent: "?", Task: "?"},
		"claude@auth-bug":     {Pane: "%4", Window: "claude@auth-bug", Agent: "claude", Task: "auth-bug"},
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
