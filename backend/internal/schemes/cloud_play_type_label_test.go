package schemes

import "testing"

func TestCloudPlayTypeLabel(t *testing.T) {
	cases := []struct {
		play, sub, want string
	}{
		{"定位胆", "定位胆 · 万位", "定位胆 · 万位"},
		{"大小单双", "五星和值大小", "大小单双 · 五星和值大小"},
		{"定位胆", "", "定位胆"},
		{"", "定位胆 · 万位", "定位胆 · 万位"},
		{"定位胆 · 万位", "万位", "定位胆 · 万位"},
	}
	for _, tc := range cases {
		got := cloudPlayTypeLabel(tc.play, tc.sub)
		if got != tc.want {
			t.Fatalf("cloudPlayTypeLabel(%q,%q)=%q want %q", tc.play, tc.sub, got, tc.want)
		}
	}
}
