package so

import (
	"testing"
)

func TestParseWindowName(t *testing.T) {
	cases := []struct {
		in         string
		wantAgent  string
		wantSuffix string
		wantOK     bool
	}{
		{"claude@new", "claude", "new", true},
		{"cursor@auth-bug", "cursor", "auth-bug", true},
		{"claude@new-2", "claude", "new-2", true},
		{"no-at-sign", "", "", false},
		{"@onlyrhs", "", "", false},
		{"onlyat@", "", "", false},
	}
	for _, tc := range cases {
		agent, suffix, ok := ParseWindowName(tc.in)
		if ok != tc.wantOK || agent != tc.wantAgent || suffix != tc.wantSuffix {
			t.Errorf("ParseWindowName(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.in, agent, suffix, ok, tc.wantAgent, tc.wantSuffix, tc.wantOK)
		}
	}
}

func TestDedupName(t *testing.T) {
	cases := []struct {
		base     string
		existing []string
		want     string
	}{
		{"claude@new", nil, "claude@new"},
		{"claude@new", []string{"cursor@new"}, "claude@new"},
		{"claude@new", []string{"claude@new"}, "claude@new-2"},
		{"claude@new", []string{"claude@new", "claude@new-2"}, "claude@new-3"},
		{"claude@new", []string{"claude@new", "claude@new-3"}, "claude@new-2"},
	}
	for _, tc := range cases {
		got := DedupName(tc.base, tc.existing)
		if got != tc.want {
			t.Errorf("DedupName(%q, %v) = %q, want %q",
				tc.base, tc.existing, got, tc.want)
		}
	}
}

func TestIsValidSuffix(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"new", true},
		{"auth-bug", true},
		{"a_b_c", true},
		{"has space", false},
		{"has:colon", false},
		{"has.dot", false},
		{"", false},
		{"has@at", false},
	}
	for _, tc := range cases {
		if got := IsValidSuffix(tc.in); got != tc.want {
			t.Errorf("IsValidSuffix(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
