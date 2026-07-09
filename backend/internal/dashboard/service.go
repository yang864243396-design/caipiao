package dashboard

import (
	"context"
	"errors"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

var ErrUnavailable = errors.New("dashboard service unavailable")

type Kpi struct {
	TodayRecharge          float64 `json:"todayRecharge"`
	TodayBetVolume         float64 `json:"todayBetVolume"`
	MemberTotalPnl         float64 `json:"memberTotalPnl"`
	RunningSchemesReal     int64   `json:"runningSchemesReal"`
	RunningSchemesSim      int64   `json:"runningSchemesSim"`
	RegistrationsLast7Days int64   `json:"registrationsLast7Days"`
}

type Service struct {
	q *sqlcdb.Queries
}

func NewService(pool *db.Pool) *Service {
	if pool == nil {
		return nil
	}
	return &Service{q: sqlcdb.New(pool)}
}

func (s *Service) AdminKpi(ctx context.Context) (Kpi, error) {
	if s == nil || s.q == nil {
		return Kpi{}, ErrUnavailable
	}
	row, err := s.q.AdminDashboardKpi(ctx)
	if err != nil {
		return Kpi{}, err
	}
	return Kpi{
		TodayRecharge:          row.TodayRecharge,
		TodayBetVolume:         row.TodayBetVolume,
		MemberTotalPnl:         row.MemberTotalPnl,
		RunningSchemesReal:     row.RunningSchemesReal,
		RunningSchemesSim:      row.RunningSchemesSim,
		RegistrationsLast7Days: row.RegistrationsLast7Days,
	}, nil
}
