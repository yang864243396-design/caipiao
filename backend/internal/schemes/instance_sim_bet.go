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

	row, err := s.q.UpdateSchemeInstanceSimBet(ctx, sqlcdb.UpdateSchemeInstanceSimBetParams{
		ID:       instanceID,
		MemberID: m.ID,
		SimBet:   simBet,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 区分不存在 vs running 禁止修改
			if _, getErr := s.q.GetSchemeInstanceByIDAndMember(ctx, sqlcdb.GetSchemeInstanceByIDAndMemberParams{
				ID: instanceID, MemberID: m.ID,
			}); getErr == nil {
				return Instance{}, ErrInstanceRunningSimBet
			}
			return Instance{}, ErrDefinitionNotFound
		}
		return Instance{}, err
	}
	if err := s.syncDefinitionSimBet(ctx, m.ID, row.DefinitionID, simBet); err != nil {
		return Instance{}, err
	}
	return s.enrichInstanceForDisplay(ctx, sqlcdb.SchemeInstanceFromSimBetRow(row), time.Now()), nil
}

func (s *Service) syncDefinitionSimBet(ctx context.Context, memberID int64, definitionID string, simBet bool) error {
	def, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
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
	_, err = s.q.UpdateSchemeDefinitionConfig(ctx, sqlcdb.UpdateSchemeDefinitionConfigParams{
		ID:       definitionID,
		MemberID: memberID,
		Config:   cfgBytes,
	})
	return err
}
