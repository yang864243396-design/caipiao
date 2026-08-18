package schemes

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

type CloudCenterChannelStats struct {
	TotalTurnover float64 `json:"totalTurnover"`
	// TotalSessionPnl 顶部「总盈亏」：该通道全部实例 session_pnl（本次盈亏）之和
	TotalSessionPnl float64 `json:"totalSessionPnl"`
	// RunningSessionPnl 运行中盈亏：running 实例 session_pnl 之和
	RunningSessionPnl float64 `json:"runningSessionPnl"`
}

type CloudCenterStats struct {
	GeneratedAt string                  `json:"generatedAt,omitempty"`
	Formal      CloudCenterChannelStats `json:"formal"`
	Sim         CloudCenterChannelStats `json:"sim"`
	// 模拟方案配额（北京时间自然日 / 同时运行）
	SimQuota SimSchemeQuota `json:"simQuota"`
}

func (s *Service) GetCloudCenterStats(ctx context.Context, account string) (CloudCenterStats, error) {
	if s == nil || s.q == nil {
		return CloudCenterStats{}, ErrUnavailable
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CloudCenterStats{}, member.ErrNotFound
		}
		return CloudCenterStats{}, err
	}
	statsByMember, err := s.LoadRealtimeStats(ctx, []int64{m.ID})
	if err != nil {
		return CloudCenterStats{}, err
	}
	stats := statsByMember[m.ID]
	stats.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return stats, nil
}

func mapCloudCenterChannelStats(row sqlcdb.MemberCloudCenterStatsRow) CloudCenterChannelStats {
	return CloudCenterChannelStats{
		// 投注流水必须按第三方实际扣款金额返回，不能先按一位小数四舍五入。
		TotalTurnover:     TruncateBetAmount(statsNumericToFloatRaw(row.TotalTurnover)),
		TotalSessionPnl:   statsNumericToFloat(row.TotalSessionPnl),
		RunningSessionPnl: statsNumericToFloat(row.RunningSessionPnl),
	}
}

func statsNumericToFloat(n pgtype.Numeric) float64 {
	return math.Round(statsNumericToFloatRaw(n)*10) / 10
}

func statsNumericToFloatRaw(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}
