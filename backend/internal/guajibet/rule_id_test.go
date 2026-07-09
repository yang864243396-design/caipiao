package guajibet

import "testing"

func TestExtractGuajiRuleID(t *testing.T) {
	cases := []struct {
		outbound, subID, want string
		segment               []byte
	}{
		{"13", "", "13", nil},
		{"ssc_std:dingwei:dingwei_wan", "dingwei_wan", "", nil},
		{"ssc_std:g006:13", "13", "13", nil},
		{"", "42", "42", nil},
		{"composite", "99", "99", []byte(`{"guajiRuleId":"99"}`)},
	}
	for _, c := range cases {
		got := ExtractGuajiRuleID(c.outbound, c.segment, c.subID)
		if got != c.want {
			t.Fatalf("outbound=%q sub=%q: got %q want %q", c.outbound, c.subID, got, c.want)
		}
	}
}
