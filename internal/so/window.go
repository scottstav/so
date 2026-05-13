package so

import (
	"fmt"
	"strings"
)

// ParseWindowName splits a window name "<agent>@<suffix>" into its parts.
// Returns (agent, suffix, true) on success; ("", "", false) otherwise.
func ParseWindowName(name string) (agent, suffix string, ok bool) {
	at := strings.IndexByte(name, '@')
	if at <= 0 || at == len(name)-1 {
		return "", "", false
	}
	return name[:at], name[at+1:], true
}

// DedupName returns base if it's not in existing; otherwise appends -2, -3, ...
// until it finds an unused name. existing may be in any order.
func DedupName(base string, existing []string) string {
	set := map[string]struct{}{}
	for _, e := range existing {
		set[e] = struct{}{}
	}
	if _, taken := set[base]; !taken {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if _, taken := set[candidate]; !taken {
			return candidate
		}
	}
}

// IsValidSuffix returns true if s is a legal window-name suffix:
// non-empty, no whitespace, no `:`, `.`, or `@`.
func IsValidSuffix(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == ':' || r == '.' || r == '@' {
			return false
		}
	}
	return true
}
