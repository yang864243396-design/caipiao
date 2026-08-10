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

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Instance{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	row, err := qtx.UpdateSchemeInstanceMultiplier(ctx, sqlcdb.UpdateSchemeInstanceMultiplierParams{
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
	def, err := qtx.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       row.DefinitionID,
		MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrDefinitionNotFound
		}
		return Instance{}, err
	}
	cfgBytes, err := setSchemeConfigMultiplier(def.Config, multiplier)
	if err != nil {
		return Instance{}, err
	}
	if _, err := qtx.UpdateSchemeDefinitionConfig(ctx, sqlcdb.UpdateSchemeDefinitionConfigParams{
		ID:       row.DefinitionID,
		MemberID: m.ID,
		Config:   cfgBytes,
	}); err != nil {
		return Instance{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Instance{}, err
	}
	return s.enrichInstanceForDisplay(ctx, sqlcdb.SchemeInstanceFromMultiplierRow(row), time.Now()), nil
}
