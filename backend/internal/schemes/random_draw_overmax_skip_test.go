package schemes

import (
	"strings"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestRandomOverMaxSkipStreakMarker(t *testing.T) {
	t.Parallel()
	if isRandomOverMaxSkipMarker("1,2,3") {
		t.Fatal("normal pick should not be marker")
	}
	marker := formatRandomOverMaxSkipMarker(3)
	if !isRandomOverMaxSkipMarker(marker) {
		t.Fatal("formatted marker should match")
	}
	if randomOverMaxSkipStreak("") != 0 {
		t.Fatal("empty streak want 0")
	}
	if randomOverMaxSkipStreak(formatRandomOverMaxSkipMarker(7)) != 7 {
		t.Fatal("streak want 7")
	}
	if randomOverMaxSkipStreak("__rd_overmax_skip:x") != 0 {
		t.Fatal("bad marker streak want 0")
	}
}

func TestRandomOverMaxUnsolvableDetail(t *testing.T) {
	t.Parallel()
	detail := randomOverMaxUnsolvableDetail()
	if detail != "随机出号连续10期超过注数上限" {
		t.Fatalf("detail=%q", detail)
	}
	if strings.Contains(detail, errBetUnitsExceeded.Error()) {
		t.Fatal("unsolvable detail must not match max-units refuse filter")
	}
}

func TestPickRandomDrawIgnoresOverMaxMarker(t *testing.T) {
	t.Parallel()
	cfg := pickTestConfig(t, `{
		"runTypeId":"random_draw","betMode":"hezhi","playTypeId":"g004","subPlayId":"40",
		"playTemplate":"ssc_std","randomDraw":{"counts":[3],"strategy":"keep"}
	}`)
	inst := sqlcdb.SchemeInstance{Kind: "custom", CurrentPick: formatRandomOverMaxSkipMarker(5)}
	dec := pickRandomDraw(cfg, inst)
	if isRandomOverMaxSkipMarker(dec.Content) {
		t.Fatalf("must not bet marker, content=%q", dec.Content)
	}
	if strings.TrimSpace(dec.Content) == "" {
		t.Fatal("expected redraw content")
	}
}
