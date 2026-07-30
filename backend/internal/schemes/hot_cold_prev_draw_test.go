package schemes

import "testing"

func TestHotColdAdjacentPrevMissing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		expected string
		latest   string
		want     bool
	}{
		{"ready exact", "1014147100158", "1014147100158", false},
		{"missing adjacent", "1014147100158", "1014147100157", true},
		{"no draws", "1014147100158", "", true},
		{"empty expected", "", "1014147100157", false},
		{"family gap not blocking", "1014146800039", "1014146700039", false},
		{"latest already ahead", "1014147100158", "1014147100159", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := hotColdAdjacentPrevMissing(tc.expected, tc.latest); got != tc.want {
				t.Fatalf("hotColdAdjacentPrevMissing(%q,%q)=%v want %v", tc.expected, tc.latest, got, tc.want)
			}
		})
	}
}
