package schemes

import (
	"strings"
	"testing"
)

func TestResolveRandomDrawLHCRenyiDuipengCounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		counts string
		want   []int
	}{
		{name: "legacy total count", counts: "[5]", want: []int{2, 3}},
		{name: "two zones", counts: "[4,6]", want: []int{4, 6}},
		{name: "total cap", counts: "[9,9]", want: []int{9, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte(`{
				"runTypeId":"random_draw","playTemplate":"lhc_std","subPlayId":"284","catalogSubId":"284",
				"betMode":"renyi_dp","randomDraw":{"counts":` + tt.counts + `,"strategy":"every"}
			}`)
			cfg := parseSchemeConfig("custom", raw, 0, 0)
			if cfg.Random == nil {
				t.Fatal("random draw config is missing")
			}
			if got := cfg.Random.Counts; len(got) != 2 || got[0] != tt.want[0] || got[1] != tt.want[1] {
				t.Fatalf("counts=%v want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateRandomDrawContentLHCRenyiDuipengUsesSeparateDistinctZones(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"runTypeId":"random_draw","playTemplate":"lhc_std","subPlayId":"284","catalogSubId":"284",
		"betMode":"renyi_dp","randomDraw":{"counts":[4,6],"strategy":"every"}
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if !isLHCRenyiDuipengPlayRule(cfg.Play) {
		t.Fatal("expected renyi duipeng play")
	}

	for i := 0; i < 20; i++ {
		content := generateRandomDrawContent(cfg, 0)
		if err := validateGroupContent(cfg.Play, content); err != nil {
			t.Fatalf("iter %d: invalid content %q: %v", i, content, err)
		}
		parts := strings.Split(content, "|")
		if len(parts) != 2 {
			t.Fatalf("iter %d: want A|B got %q", i, content)
		}
		a, invalidA := parseLHCRenyiDuipengNumbers(parts[0])
		b, invalidB := parseLHCRenyiDuipengNumbers(parts[1])
		if len(invalidA) != 0 || len(invalidB) != 0 || len(a) != 4 || len(b) != 6 {
			t.Fatalf("iter %d: A=%v B=%v invalidA=%v invalidB=%v", i, a, b, invalidA, invalidB)
		}
		seen := make(map[int]struct{}, len(a)+len(b))
		for _, n := range append(a, b...) {
			seen[n] = struct{}{}
		}
		if len(seen) != 10 {
			t.Fatalf("iter %d: zones overlap in %q", i, content)
		}
		if got := countLHCRenyiDuipengBetUnits(content); got != 24 {
			t.Fatalf("iter %d: units=%d want 24 for %q", i, got, content)
		}
	}
}
