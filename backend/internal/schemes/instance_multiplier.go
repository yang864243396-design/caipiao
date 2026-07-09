package schemes

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

func (s *Service) UpdateInstanceMultiplier(ctx context.Context, account, instanceID string, multiplier float64) (Instance, error) {
	if s == nil || s.q == nil {
		return Instance{}, ErrUnavailable
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return Instance{}, ErrDefinitionNotFound
	}
	if multiplier < 1 || math.Mod(multiplier, 1) != 0 {
		return Instance{}, fmt.Errorf("%w: multiplier 须为正整数且不小于 1", ErrInvalidCreateRequest)
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, member.ErrNotFound
		}
		return Instance{}, err
	}

	row, err := s.q.UpdateSchemeInstanceMultiplier(ctx, sqlcdb.UpdateSchemeInstanceMultiplierParams{
		ID:         instanceID,
		MemberID:   m.ID,
		Multiplier: floatToNumeric(multiplier),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrDefinitionNotFound
		}
		return Instance{}, err
	}
	return s.enrichInstanceForDisplay(ctx, sqlcdb.SchemeInstanceFromMultiplierRow(row), time.Now()), nil
}
