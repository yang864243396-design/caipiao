package schemes

import (
	"context"
	"errors"
	"math"

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
	Formal CloudCenterChannelStats `json:"formal"`
	Sim    CloudCenterChannelStats `json:"sim"`
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
	rows, err := s.q.ListMemberCloudCenterStatsBySimBet(ctx, m.ID)
	if err != nil {
		return CloudCenterStats{}, err
	}
	out := CloudCenterStats{
		SimQuota: SimSchemeQuota{
			TodayStartsLimit: maxSimSchemeDailyStarts,
			RunningLimit:     maxSimSchemeConcurrent,
		},
	}
	for _, row := range rows {
		ch := mapCloudCenterChannelStats(row)
		if row.SimBet {
			out.Sim = ch
		} else {
			out.Formal = ch
		}
	}
	quota, qerr := s.simSchemeQuotaForMember(ctx, m.ID)
	if qerr == nil {
		out.SimQuota = quota
	}
	return out, nil
}

func mapCloudCenterChannelStats(row sqlcdb.MemberCloudCenterStatsRow) CloudCenterChannelStats {
	return CloudCenterChannelStats{
		TotalTurnover:     statsNumericToFloat(row.TotalTurnover),
		TotalSessionPnl:   statsNumericToFloat(row.TotalSessionPnl),
		RunningSessionPnl: statsNumericToFloat(row.RunningSessionPnl),
	}
}

func statsNumericToFloat(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return math.Round(f.Float64*10) / 10
}
