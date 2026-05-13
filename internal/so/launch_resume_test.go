package so

import "testing"

func TestLooksLikeResume(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{}, false},
		{[]string{"--print", "hello"}, false},
		{[]string{"--resume"}, true},
		{[]string{"--foo", "--resume"}, true},
		{[]string{"--resume=session-id"}, true},
		{[]string{"-r"}, true},
		{[]string{"--resume-something-else"}, false}, // prefix without =
	}
	for _, tc := range cases {
		if got := looksLikeResume(tc.args); got != tc.want {
			t.Errorf("looksLikeResume(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}
