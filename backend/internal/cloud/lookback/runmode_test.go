package lookback

import "testing"

func TestEncodeDecodeRunModes(t *testing.T) {
	cases := []struct {
		in  []RunMode
		out string
	}{
		{nil, ""},
		{[]RunMode{RunModeSim, RunModeReal}, "real,sim"},
		{[]RunMode{RunModeReal}, "real"},
	}
	for _, c := range cases {
		if got := EncodeRunModes(c.in); got != c.out {
			t.Fatalf("encode %v => %q want %q", c.in, got, c.out)
		}
		if dec := DecodeRunModes(c.out); EncodeRunModes(dec) != c.out {
			t.Fatalf("roundtrip %q => %v", c.out, dec)
		}
	}
}

func TestContainsRunMode(t *testing.T) {
	if ContainsRunMode(nil, "real") {
		t.Fatal("empty should not match")
	}
	modes := []RunMode{RunModeReal, RunModeSim}
	if !ContainsRunMode(modes, "sim") || !ContainsRunMode(modes, "real") {
		t.Fatal("both should match")
	}
	if ContainsRunMode(modes, "other") {
		t.Fatal("unknown mode")
	}
}
