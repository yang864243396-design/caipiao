package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

func (s *Service) GetDefinition(ctx context.Context, account, definitionID string) (Definition, error) {
	if s == nil || s.q == nil {
		return Definition{}, ErrUnavailable
	}
	definitionID = strings.TrimSpace(definitionID)
	if definitionID == "" {
		return Definition{}, ErrDefinitionNotFound
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Definition{}, member.ErrNotFound
		}
		return Definition{}, err
	}

	def, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Definition{}, ErrDefinitionNotFound
		}
		return Definition{}, err
	}

	hasInstance := false
	if _, instErr := s.q.GetSchemeInstanceByDefinitionID(ctx, definitionID); instErr == nil {
		hasInstance = true
	} else if !errors.Is(instErr, pgx.ErrNoRows) {
		return Definition{}, instErr
	}

	return mapDefinitionFromGetRow(def, hasInstance), nil
}

func mapDefinitionFromGetRow(row sqlcdb.GetSchemeDefinitionByIDAndMemberRow, hasInstance bool) Definition {
	return mapDefinitionFields(
		row.ID, row.Kind, row.SchemeName, row.LotteryCode, row.LotteryLabel,
		row.ShareStatus, row.Config, row.CreatedAt, row.UpdatedAt, hasInstance,
	)
}

func (s *Service) PutBetMultiplier(ctx context.Context, account, definitionID string, payload json.RawMessage) (Definition, error) {
	if len(payload) == 0 || string(payload) == "null" {
		return Definition{}, ErrInvalidUpdatePatch
	}
	enriched, err := s.enrichBetMultiplierPayload(ctx, payload)
	if err != nil {
		return Definition{}, err
	}
	return s.UpdateDefinition(ctx, account, definitionID, UpdateDefinitionPatch{
		BetMultiplier:    enriched,
		HasBetMultiplier: true,
	})
}

func (s *Service) enrichBetMultiplierPayload(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return payload, err
	}
	kind, _ := m["kind"].(string)
	// 简单倍投：倍数串须解析出 ≥1 的正数序列（0 会被丢弃，不能只剩空表）
	if kind == "2" {
		if sm, ok := m["simple"].(map[string]interface{}); ok {
			ms, _ := sm["multiples"].(string)
			if len(parseMultSequence(ms)) == 0 {
				return nil, fmt.Errorf("%w: simple.multiples 须为不小于 1 的正数序列", ErrInvalidUpdatePatch)
			}
		}
	}
	if kind == "3" {
		if adv, ok := m["advanced"].(map[string]interface{}); ok {
			if raw, ok := adv["rounds"]; ok {
				b, _ := json.Marshal(raw)
				if len(b) > 0 && string(b) != "null" {
					if err := validateSchemeRoundsMult(b); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	if kind != "3" {
		return payload, nil
	}
	adv, ok := m["advanced"].(map[string]interface{})
	if !ok {
		return payload, nil
	}
	if raw, ok := adv["rounds"]; ok && len(parseSchemeRoundsFromRaw(raw)) > 0 {
		return payload, nil
	}
	selectedID, _ := adv["selectedId"].(string)
	selectedID = strings.TrimSpace(selectedID)
	if selectedID == "" {
		return payload, nil
	}
	rounds := s.TemplateRoundsByID(ctx, selectedID)
	if len(rounds) == 0 {
		return payload, nil
	}
	adv["rounds"] = rounds
	m["advanced"] = adv
	out, err := json.Marshal(m)
	if err != nil {
		return payload, err
	}
	return out, nil
}

func (s *Service) PutRounds(ctx context.Context, account, definitionID string, rounds json.RawMessage) (Definition, error) {
	if len(rounds) == 0 || string(rounds) == "null" {
		return Definition{}, ErrInvalidUpdatePatch
	}
	if err := validateSchemeRoundsMult(rounds); err != nil {
		return Definition{}, err
	}
	return s.UpdateDefinition(ctx, account, definitionID, UpdateDefinitionPatch{
		Rounds:    append(json.RawMessage(nil), rounds...),
		HasRounds: true,
	})
}
