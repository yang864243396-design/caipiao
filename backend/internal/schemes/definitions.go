package schemes

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

var ErrInvalidKind = errors.New("invalid scheme kind")

type DefinitionListResult struct {
	Items []Definition `json:"items"`
}

func (s *Service) ListDefinitions(ctx context.Context, account, kind string) (DefinitionListResult, error) {
	if s == nil || s.q == nil {
		return DefinitionListResult{}, ErrUnavailable
	}
	if kind != "" && kind != "custom" && kind != "contrary" && kind != "follow" {
		return DefinitionListResult{}, ErrInvalidKind
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DefinitionListResult{}, member.ErrNotFound
		}
		return DefinitionListResult{}, err
	}

	kindParam := pgtype.Text{}
	if kind != "" {
		kindParam = pgtype.Text{String: kind, Valid: true}
	}

	rows, err := s.q.ListSchemeDefinitionsByMember(ctx, sqlcdb.ListSchemeDefinitionsByMemberParams{
		MemberID: m.ID,
		Kind:     kindParam,
	})
	if err != nil {
		return DefinitionListResult{}, err
	}

	items := make([]Definition, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapDefinitionListRow(row))
	}
	return DefinitionListResult{Items: items}, nil
}

func mapDefinitionListRow(row sqlcdb.ListSchemeDefinitionsByMemberRow) Definition {
	return mapDefinitionFields(
		row.ID, row.Kind, row.SchemeName, row.LotteryCode, row.LotteryLabel,
		row.ShareStatus, row.Config, row.CreatedAt, row.UpdatedAt, row.HasInstance,
	)
}
