package schemes

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

func (s *Service) DeleteDefinition(ctx context.Context, account, definitionID string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	definitionID = strings.TrimSpace(definitionID)
	if definitionID == "" {
		return ErrDefinitionNotFound
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return member.ErrNotFound
		}
		return err
	}

	if _, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: m.ID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDefinitionNotFound
		}
		return err
	}

	if inst, err := s.q.GetSchemeInstanceByDefinitionID(ctx, definitionID); err == nil {
		if inst.Status == "running" {
			return ErrDeleteWhileRunning
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	rows, err := s.q.DeleteSchemeDefinitionByIDAndMember(ctx, sqlcdb.DeleteSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: m.ID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDefinitionNotFound
	}
	return nil
}
