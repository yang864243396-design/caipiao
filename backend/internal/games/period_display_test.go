package games

import "testing"

func TestClampDrawDisplayIssue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		issue      string
		openPeriod string
		want       string
	}{
		{name: "no open keeps issue", issue: "100", openPeriod: "", want: "100"},
		{name: "empty issue uses prev open", issue: "", openPeriod: "1014046800043", want: "1014046800042"},
		{name: "clamps open period itself", issue: "1014046800043", openPeriod: "1014046800043", want: "1014046800042"},
		{name: "clamps future issue", issue: "1014046800044", openPeriod: "1014046800043", want: "1014046800042"},
		{name: "keeps previous drawn", issue: "1014046800041", openPeriod: "1014046800043", want: "1014046800041"},
		{name: "keeps exact prev", issue: "1014046800042", openPeriod: "1014046800043", want: "1014046800042"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := clampDrawDisplayIssue(tc.issue, tc.openPeriod)
			if got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
		})
	}
}
