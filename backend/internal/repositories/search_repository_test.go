package repositories

import "testing"

func TestLikePattern(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"abc", "%abc%"},
		{"100%", "%100\\%%"},
		{"under_score", "%under\\_score%"},
		{"back\\slash", "%back\\\\slash%"},
		{"", "%%"},
	}
	for _, c := range cases {
		if got := likePattern(c.input); got != c.want {
			t.Errorf("likePattern(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
