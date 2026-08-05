package schemes

import (
	"strings"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestPickRandomDraw_ren4Zu12DualCounts(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"random_draw",
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"144","catalogSubId":"144",
		"betMode":"zu12","playMethodLabel":"任四组选12","renPositionCount":4,"segmentLen":1,
		"randomDraw":{"counts":[2,4],"strategy":"every","positionIdxs":[0,1,2,3]}
	}`)
	if !isZu12PlayRule(cfg.Play) {
		t.Fatal("expected zu12 play")
	}
	for i := 0; i < 40; i++ {
		dec := pickRandomDraw(cfg, sqlcdb.SchemeInstance{Kind: "custom"})
		body := dec.Content
		// 任选带选位前缀时取末行
		if idx := strings.LastIndex(body, "\n"); idx >= 0 {
			body = strings.TrimSpace(body[idx+1:])
		}
		parts := strings.Split(body, ",")
		if len(parts) != 2 {
			t.Fatalf("want doubles,singles got %q", dec.Content)
		}
		doubles := uniqueDigitRunSchemes(parts[0])
		singles := uniqueDigitRunSchemes(parts[1])
		if len(doubles) != 2 {
			t.Fatalf("doubles want 2 got %q from %q", doubles, dec.Content)
		}
		if len(singles) != 4 {
			t.Fatalf("singles want 4 got %q from %q", singles, dec.Content)
		}
		if n := countZu12DualZoneBetUnits(body); n <= 0 {
			t.Fatalf("units=0 content=%q", dec.Content)
		}
	}
}

func TestResolveRandomDraw_zu12PadsSinglesCount(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"random_draw",
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"144","catalogSubId":"144",
		"betMode":"zu12","playMethodLabel":"任四组选12","renPositionCount":4,"segmentLen":1,
		"randomDraw":{"counts":[3],"strategy":"every"}
	}`)
	if cfg.Random == nil || len(cfg.Random.Counts) != 2 {
		t.Fatalf("counts=%v want [3,2]", cfg.Random)
	}
	if cfg.Random.Counts[0] != 3 || cfg.Random.Counts[1] != 2 {
		t.Fatalf("counts=%v want [3,2]", cfg.Random.Counts)
	}
}
