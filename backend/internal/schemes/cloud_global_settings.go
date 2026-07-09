package schemes

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

type CloudGlobalSettings struct {
	TotalStopLoss   float64 `json:"totalStopLoss"`
	TotalTakeProfit float64 `json:"totalTakeProfit"`
	PlanMultiplier  float64 `json:"planMultiplier"`
	BreakPeriodStop bool    `json:"breakPeriodStop"`
}

func defaultCloudGlobalSettings() CloudGlobalSettings {
	return CloudGlobalSettings{
		TotalStopLoss:   0,
		TotalTakeProfit: 0,
		PlanMultiplier:  1,
		BreakPeriodStop: false,
	}
}

func (s *Service) GetCloudGlobalSettings(ctx context.Context, account string) (CloudGlobalSettings, error) {
	if s == nil || s.q == nil {
		return CloudGlobalSettings{}, ErrUnavailable
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CloudGlobalSettings{}, member.ErrNotFound
		}
		return CloudGlobalSettings{}, err
	}

	row, err := s.q.GetMemberCloudSettings(ctx, m.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultCloudGlobalSettings(), nil
		}
		return CloudGlobalSettings{}, err
	}
	return mapCloudGlobalSettings(row), nil
}

func (s *Service) PutCloudGlobalSettings(ctx context.Context, account string, input CloudGlobalSettings) (CloudGlobalSettings, error) {
	if s == nil || s.q == nil {
		return CloudGlobalSettings{}, ErrUnavailable
	}
	if input.PlanMultiplier <= 0 {
		return CloudGlobalSettings{}, fmt.Errorf("%w: planMultiplier 须大于 0", ErrInvalidCreateRequest)
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CloudGlobalSettings{}, member.ErrNotFound
		}
		return CloudGlobalSettings{}, err
	}

	row, err := s.q.UpsertMemberCloudSettings(ctx, sqlcdb.UpsertMemberCloudSettingsParams{
		MemberID:        m.ID,
		TotalStopLoss:   floatToNumeric(input.TotalStopLoss),
		TotalTakeProfit: floatToNumeric(input.TotalTakeProfit),
		PlanMultiplier:  floatToNumeric(input.PlanMultiplier),
		BreakPeriodStop: input.BreakPeriodStop,
	})
	if err != nil {
		return CloudGlobalSettings{}, err
	}
	return mapCloudGlobalSettings(row), nil
}

func mapCloudGlobalSettings(row sqlcdb.MemberCloudSetting) CloudGlobalSettings {
	return CloudGlobalSettings{
		TotalStopLoss:   numericToFloat(row.TotalStopLoss),
		TotalTakeProfit: numericToFloat(row.TotalTakeProfit),
		PlanMultiplier:  numericToFloat(row.PlanMultiplier),
		BreakPeriodStop: row.BreakPeriodStop,
	}
}

func floatToNumeric(v float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.4f", v))
	return n
}
