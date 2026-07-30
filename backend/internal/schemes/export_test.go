package schemes

import (
	"context"

	"caipiao/backend/internal/db/sqlcdb"
)

// 仅测试可见的 worker 驱动入口，供 e2e_lifecycle_test.go（外部测试包）使用。
//
// 不直接用 w.tick：它会捞出全库所有 running 实例挨个推一遍，
// 在开发库上等于替别人的方案下注。E2E 必须只推自己造出来的那一个。

// TickInstanceForTest 推动单个实例走完一轮：门禁 → 出号 → 下注。
func (w *Worker) TickInstanceForTest(ctx context.Context, inst sqlcdb.SchemeInstance) {
	w.tickInstance(ctx, inst, 1, nil)
}

// SettleSimBetsForTest 只结算指定实例的待结算模拟注单，返回结算笔数。
func (w *Worker) SettleSimBetsForTest(ctx context.Context, instanceID string) (int, error) {
	rows, err := w.q.ListPendingSimCloudBetsReady(ctx, simSettlementBatchSize)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, row := range rows {
		if row.SchemeID != instanceID {
			continue
		}
		if err := w.settleSimCloudBet(ctx, row); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
