package periodissue

import "testing"

func TestAdvances(t *testing.T) {
	tests := []struct {
		name     string
		previous string
		current  string
		want     bool
	}{
		{
			name:     "arbitrarily long decimal rollover",
			previous: "9999999999999999999999999999999999999999",
			current:  "10000000000000000000000000000000000000000",
			want:     true,
		},
		{
			name:     "production decimal issue",
			previous: "10114255902821",
			current:  "10114255902822",
			want:     true,
		},
		{
			name:     "same prefix decimal suffix rollover",
			previous: "P99",
			current:  "P100",
			want:     true,
		},
		{
			name:     "production style prefix",
			previous: "lottery_log101-10113906900723",
			current:  "lottery_log101-10113906900724",
			want:     true,
		},
		{
			name:     "older prefixed replay",
			previous: "P100",
			current:  "P99",
		},
		{
			name:     "older arbitrary length decimal",
			previous: "10000000000000000000000000000000000000000",
			current:  "9999999999999999999999999999999999999999",
		},
		{
			name:     "identical",
			previous: "P100",
			current:  "P100",
		},
		{
			name:     "equal value with different zero padding",
			previous: "P099",
			current:  "P99",
		},
		{
			name:     "different prefixes are ambiguous",
			previous: "P99",
			current:  "Q100",
		},
		{
			name:     "non decimal suffixes are ambiguous",
			previous: "P-ws-current",
			current:  "P-ws-next",
		},
		{
			name:     "empty previous",
			previous: "",
			current:  "P100",
		},
		{
			name:     "empty current",
			previous: "P99",
			current:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Advances(test.previous, test.current); got != test.want {
				t.Fatalf("Advances(%q, %q) = %v, want %v", test.previous, test.current, got, test.want)
			}
		})
	}
}
