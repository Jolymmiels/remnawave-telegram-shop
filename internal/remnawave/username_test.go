package remnawave

import (
	"math"
	"regexp"
	"testing"
)

func TestGenerateUsernameSpecCompliance(t *testing.T) {
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	cases := []struct{ c, tg int64 }{
		{1, 1},
		{123456, 987654321},
		{math.MaxInt64, math.MaxInt64},
		{math.MaxInt64, math.MinInt64},
	}
	for _, tc := range cases {
		u := generateUsername(tc.c, tc.tg)
		if l := len(u); l < 3 || l > 36 {
			t.Fatalf("length out of range for c=%d tg=%d: %q (%d)", tc.c, tc.tg, u, l)
		}
		if !re.MatchString(u) {
			t.Fatalf("pattern mismatch: %q", u)
		}
	}
}
