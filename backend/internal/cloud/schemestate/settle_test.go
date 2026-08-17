package schemestate

import (
	"context"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestFormalFinancialSettlementDoesNotAdvanceStrategyTwice(t *testing.T) {
	previous := FormalPickAdvancer
	t.Cleanup(func() { FormalPickAdvancer = previous })
	FormalPickAdvancer = func(string, []byte, sqlcdb.SchemeInstance, string, bool) (int32, string, string) {
		return 9, "next", "next"
	}
	inst := sqlcdb.SchemeInstance{PickIndex: 3, CurrentPick: "frozen", LastDirection: "up"}
	index, current, direction := formalPickStateAfterSettlement(context.Background(), false, nil, inst, "", nil, true)
	if index != 3 || current != "frozen" || direction != "up" {
		t.Fatalf("financial settlement changed strategy state: (%d,%q,%q)", index, current, direction)
	}
}
