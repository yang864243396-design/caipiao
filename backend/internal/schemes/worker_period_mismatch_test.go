package schemes

import "testing"

func TestIsAcceptedPeriodMismatch(t *testing.T) {
	cases := []struct {
		target, accepted string
		want             bool
	}{
		{"", "101", false},
		{"101", "", false},
		{"101", "101", false},
		{"101", "102", true},
	}
	for _, tc := range cases {
		if got := isAcceptedPeriodMismatch(tc.target, tc.accepted); got != tc.want {
			t.Fatalf("target=%q accepted=%q got=%t want=%t", tc.target, tc.accepted, got, tc.want)
		}
	}
}
