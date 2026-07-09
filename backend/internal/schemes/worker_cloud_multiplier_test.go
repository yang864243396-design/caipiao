package schemes

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestMemberPlanMultiplierDefault(t *testing.T) {
	w := &Worker{}
	if got := w.memberPlanMultiplier(nil, 0); got != 1 {
		t.Fatalf("got %v want 1", got)
	}
}

func TestPlanBaseCoef(t *testing.T) {
	if planBaseCoef(2.5) != 2.5 {
		t.Fatal(planBaseCoef(2.5))
	}
	if planBaseCoef(-1) != 1 {
		t.Fatal("negative should default to 1")
	}
}

func TestCombinedBaseCoefUsesBothMultipliers(t *testing.T) {
	var n pgtype.Numeric
	_ = n.Scan("4")
	if got := combinedBaseCoef(n, 2); got != 8 {
		t.Fatalf("got %v want 8", got)
	}
}
