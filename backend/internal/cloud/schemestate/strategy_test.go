package schemestate

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestNextStrategyAfterDrawAdvancesRoundAndPickWithoutFinancialState(t *testing.T) {
	previous := FormalPickAdvancer
	t.Cleanup(func() { FormalPickAdvancer = previous })
	FormalPickAdvancer = func(_ string, _ []byte, _ sqlcdb.SchemeInstance, betContent string, hit bool) (int32, string, string) {
		if betContent != "1\n2\n3" || hit {
			t.Fatalf("pick input content=%q hit=%v", betContent, hit)
		}
		return 4, "next-pick", "neg"
	}

	inst := sqlcdb.SchemeInstance{RoundIndex: 0, PickIndex: 1, CurrentPick: "old", LastDirection: "pos"}
	got := nextStrategyAfterDraw(inst, "custom", []byte(`{"rounds":[{"mult":1,"afterHit":0,"afterMiss":1},{"mult":2,"afterHit":0,"afterMiss":0}]}`), "1\n2\n3", false)
	if got.RoundIndex != 1 || got.PickIndex != 4 || got.CurrentPick != "next-pick" || got.LastDirection != "neg" {
		t.Fatalf("state = %+v, want advanced miss state", got)
	}
}
