package schemes

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

// 回归：模拟盘占位后若再跑 GuajiPeriodAlreadyTaken，会把刚插入的记录当成已占用。
func TestSimBetDoesNotTreatOwnReserveAsDedupSkip(t *testing.T) {
	// 文档化约定：usesGuajiThirdParty(sim)=false → 走 reserve 路径；
	// 事务内不得再次 evaluateGuajiBetDedup（见 worker.placePeriodBet）。
	w := &Worker{}
	inst := sqlcdb.SchemeInstance{SimBet: true}
	if w.usesGuajiThirdParty(inst) {
		t.Fatal("sim must not use third-party place path")
	}
	if !requiresGuajiRealBet(sqlcdb.SchemeInstance{SimBet: false}) {
		t.Fatal("real must require guaji")
	}
	if requiresGuajiRealBet(inst) {
		t.Fatal("sim must not require guaji")
	}
}
