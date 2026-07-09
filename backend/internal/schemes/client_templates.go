package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

type ClientSaveMemberTemplateInput struct {
	Name         string
	DefinitionID string
	Brief        string
	Rounds       json.RawMessage
}

type ClientUpdateMemberTemplateInput struct {
	Name   string
	Brief  string
	Rounds json.RawMessage
}

func (s *Service) ClientGetTemplate(ctx context.Context, account, id string) (TemplateRow, error) {
	if s == nil || s.q == nil {
		return TemplateRow{}, ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return TemplateRow{}, ErrTemplateNotFound
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRow{}, member.ErrNotFound
		}
		return TemplateRow{}, err
	}
	row, err := s.q.GetSchemeTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRow{}, ErrTemplateNotFound
		}
		return TemplateRow{}, err
	}
	if !templateVisibleToMember(ctx, s, row, m.ID) {
		return TemplateRow{}, ErrTemplateForbidden
	}
	return mapTemplateDetailRow(row), nil
}

func (s *Service) ClientCreateMemberTemplate(ctx context.Context, account string, in ClientSaveMemberTemplateInput) (TemplateRow, error) {
	if s == nil || s.q == nil {
		return TemplateRow{}, ErrUnavailable
	}
	name := strings.TrimSpace(in.Name)
	definitionID := strings.TrimSpace(in.DefinitionID)
	if name == "" || definitionID == "" {
		return TemplateRow{}, ErrInvalidTemplate
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRow{}, member.ErrNotFound
		}
		return TemplateRow{}, err
	}
	def, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRow{}, ErrDefinitionNotFound
		}
		return TemplateRow{}, err
	}
	cfg, err := templateConfigWithRounds(nil, in.Rounds)
	if err != nil {
		return TemplateRow{}, ErrInvalidTemplate
	}
	row, err := s.q.UpsertSchemeTemplate(ctx, sqlcdb.UpsertSchemeTemplateParams{
		ID:           newTemplateID(),
		Name:         name,
		LotteryCode:  def.LotteryCode,
		Brief:        pgtype.Text{String: strings.TrimSpace(in.Brief), Valid: strings.TrimSpace(in.Brief) != ""},
		SortOrder:    9000,
		Enabled:      true,
		Config:       cfg,
		MemberID:     pgtype.Int8{Int64: m.ID, Valid: true},
		DefinitionID: pgtype.Text{String: definitionID, Valid: true},
	})
	if err != nil {
		return TemplateRow{}, err
	}
	label, err := s.lotteryLabel(ctx, row.LotteryCode)
	if err != nil {
		return TemplateRow{}, err
	}
	return mapUpsertedTemplate(row, label), nil
}

func (s *Service) ClientUpdateMemberTemplate(ctx context.Context, account, id string, in ClientUpdateMemberTemplateInput) (TemplateRow, error) {
	if s == nil || s.q == nil {
		return TemplateRow{}, ErrUnavailable
	}
	id = strings.TrimSpace(id)
	name := strings.TrimSpace(in.Name)
	if id == "" || name == "" {
		return TemplateRow{}, ErrInvalidTemplate
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRow{}, member.ErrNotFound
		}
		return TemplateRow{}, err
	}
	existing, err := s.q.GetSchemeTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRow{}, ErrTemplateNotFound
		}
		return TemplateRow{}, err
	}
	if !existing.DefinitionID.Valid {
		return TemplateRow{}, ErrTemplateForbidden
	}
	definitionID := strings.TrimSpace(existing.DefinitionID.String)
	cfg, err := templateConfigWithRounds(existing.Config, in.Rounds)
	if err != nil {
		return TemplateRow{}, ErrInvalidTemplate
	}
	row, err := s.q.UpdateSchemeTemplateDefinitionOwned(ctx, sqlcdb.UpdateSchemeTemplateDefinitionOwnedParams{
		ID:           id,
		DefinitionID: definitionID,
		MemberID:     m.ID,
		Name:         name,
		Config:       cfg,
		Brief:        pgtype.Text{String: strings.TrimSpace(in.Brief), Valid: strings.TrimSpace(in.Brief) != ""},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRow{}, ErrTemplateNotFound
		}
		return TemplateRow{}, err
	}
	label, err := s.lotteryLabel(ctx, row.LotteryCode)
	if err != nil {
		return TemplateRow{}, err
	}
	return mapUpsertedTemplate(row, label), nil
}

func (s *Service) TemplateRoundsByID(ctx context.Context, templateID string) []schemeRound {
	if s == nil || s.q == nil {
		return nil
	}
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil
	}
	row, err := s.q.GetSchemeTemplateByID(ctx, templateID)
	if err != nil {
		return nil
	}
	cfg := parseTemplateConfig(row.Config)
	return parseSchemeRoundsFromRaw(cfg["rounds"])
}

func templateVisibleToMember(ctx context.Context, s *Service, row sqlcdb.GetSchemeTemplateByIDRow, viewerID int64) bool {
	if row.DefinitionID.Valid {
		definitionID := strings.TrimSpace(row.DefinitionID.String)
		if definitionID == "" {
			return false
		}
		_, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
			ID:       definitionID,
			MemberID: viewerID,
		})
		return err == nil
	}
	if row.MemberID.Valid {
		return row.MemberID.Int64 == viewerID
	}
	return row.Enabled
}

func templateConfigWithRounds(existing []byte, rounds json.RawMessage) ([]byte, error) {
	cfg := parseTemplateConfig(existing)
	if len(rounds) > 0 && string(rounds) != "null" {
		var v interface{}
		if err := json.Unmarshal(rounds, &v); err != nil {
			return nil, err
		}
		cfg["rounds"] = v
	}
	return encodeTemplateConfig(cfg)
}
