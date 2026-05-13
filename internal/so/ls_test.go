package so

import (
	"strings"
	"testing"
)

func TestFormatLs(t *testing.T) {
	rows := []LsRow{
		{Window: "claude@auth-bug", Agent: "claude", Task: "auth-bug"},
		{Window: "cursor@review-pr-42", Agent: "cursor", Task: "review-pr-42"},
		{Window: "claude@new", Agent: "claude", Task: "(idle)"},
	}
	out := FormatLs(rows)
	for _, want := range []string{
		"WINDOW", "AGENT", "TASK",
		"claude@auth-bug", "auth-bug",
		"cursor@review-pr-42", "review-pr-42",
		"claude@new", "(idle)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatLs output missing %q; got:\n%s", want, out)
		}
	}
}

func TestBuildLsRows(t *testing.T) {
	wins := []string{"claude@new", "cursor@review-pr-42", "no-at-sign", "claude@auth-bug"}
	got := BuildLsRows(wins)
	want := map[string]LsRow{
		"claude@new":          {Window: "claude@new", Agent: "claude", Task: "(idle)"},
		"cursor@review-pr-42": {Window: "cursor@review-pr-42", Agent: "cursor", Task: "review-pr-42"},
		"no-at-sign":          {Window: "no-at-sign", Agent: "?", Task: "?"},
		"claude@auth-bug":     {Window: "claude@auth-bug", Agent: "claude", Task: "auth-bug"},
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
