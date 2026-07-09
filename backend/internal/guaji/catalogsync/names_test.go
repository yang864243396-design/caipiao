package catalogsync

import "testing"

func TestNormalizeLotteryName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"波场一分彩", "波场1分彩"},
		{"波场1分彩", "波场1分彩"},
		{"哈希三分彩", "哈希3分彩"},
		{"新以太坊分分彩", "新以太坊分分彩"},
		{"以太5分赛车", "以太5分赛车"},
	}
	for _, c := range cases {
		if got := NormalizeLotteryName(c.in); got != c.want {
			t.Errorf("NormalizeLotteryName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
