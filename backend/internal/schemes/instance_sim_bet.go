package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

func (s *Service) UpdateInstanceSimBet(ctx context.Context, account, instanceID string, simBet bool) (Instance, error) {
	if s == nil || s.q == nil {
		return Instance{}, ErrUnavailable
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return Instance{}, ErrDefinitionNotFound
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, member.ErrNotFound
		}
		return Instance{}, err
	}

	tx, err := s.beginTransaction(ctx)
	if err != nil {
		return Instance{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	row, err := qtx.UpdateSchemeInstanceSimBet(ctx, sqlcdb.UpdateSchemeInstanceSimBetParams{
		ID:       instanceID,
		MemberID: m.ID,
		SimBet:   simBet,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 区分不存在 vs running 禁止修改
			if _, getErr := qtx.GetSchemeInstanceByIDAndMember(ctx, sqlcdb.GetSchemeInstanceByIDAndMemberParams{
				ID: instanceID, MemberID: m.ID,
			}); getErr == nil {
				return Instance{}, ErrInstanceRunningSimBet
			}
			return Instance{}, ErrDefinitionNotFound
		}
		return Instance{}, err
	}
	if err := s.syncDefinitionSimBet(ctx, qtx, m.ID, row.DefinitionID, simBet); err != nil {
		return Instance{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Instance{}, err
	}
	return s.enrichInstanceForDisplay(ctx, sqlcdb.SchemeInstanceFromSimBetRow(row), time.Now()), nil
}

func (s *Service) syncDefinitionSimBet(ctx context.Context, q *sqlcdb.Queries, memberID int64, definitionID string, simBet bool) error {
	def, err := q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: memberID,
	})
	if err != nil {
		return err
	}
	if configSimBet(def.Config) == simBet {
		return nil
	}
	cfg := map[string]interface{}{}
	_ = json.Unmarshal(def.Config, &cfg)
	setConfigSimBet(cfg, simBet)
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = q.UpdateSchemeDefinitionConfig(ctx, sqlcdb.UpdateSchemeDefinitionConfigParams{
		ID:       definitionID,
		MemberID: memberID,
		Config:   cfgBytes,
	})
	return err
}
