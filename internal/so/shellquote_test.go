package so

import "testing"

func TestShellQuote(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "''"},
		{"hello", "hello"},
		{"--resume", "--resume"},
		{"sk-1234.abcd_5", "sk-1234.abcd_5"},
		{"hello world", "'hello world'"},
		{"with'quote", `'with'\''quote'`},
		{"$VAR", "'$VAR'"},
		{"semi;colon", "'semi;colon'"},
	}
	for _, tc := range cases {
		got := shellQuote(tc.in)
		if got != tc.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildAgentCommand(t *testing.T) {
	cases := []struct {
		cmd   string
		extra []string
		want  string
	}{
		{"claude", nil, "claude"},
		{"claude", []string{}, "claude"},
		{"claude", []string{"--resume"}, "claude --resume"},
		{"cursor-agent", []string{"--print", "review pr"}, "cursor-agent --print 'review pr'"},
		{"claude", []string{"--api-key", "sk-12345"}, "claude --api-key sk-12345"},
		{"claude", []string{"-p", "say 'hi'"}, `claude -p 'say '\''hi'\'''`},
	}
	for _, tc := range cases {
		got := buildAgentCommand(tc.cmd, tc.extra)
		if got != tc.want {
			t.Errorf("buildAgentCommand(%q, %v) = %q, want %q", tc.cmd, tc.extra, got, tc.want)
		}
	}
}
