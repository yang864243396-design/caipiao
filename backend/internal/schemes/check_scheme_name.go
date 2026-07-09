package schemes

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

type CheckSchemeNameResult struct {
	Available             bool   `json:"available"`
	ExistingDefinitionID  string `json:"existingDefinitionId,omitempty"`
	ExistingHasInstance   bool   `json:"existingHasInstance,omitempty"`
}

func (s *Service) CheckSchemeName(ctx context.Context, account, schemeName string) (CheckSchemeNameResult, error) {
	if s == nil || s.q == nil {
		return CheckSchemeNameResult{}, ErrUnavailable
	}
	schemeName = strings.TrimSpace(schemeName)
	if schemeName == "" {
		return CheckSchemeNameResult{}, ErrInvalidCreateRequest
	}
	if len([]rune(schemeName)) > 128 {
		return CheckSchemeNameResult{}, ErrInvalidCreateRequest
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CheckSchemeNameResult{}, member.ErrNotFound
		}
		return CheckSchemeNameResult{}, err
	}

	row, err := s.q.GetSchemeDefinitionNameStatusByMember(ctx, sqlcdb.GetSchemeDefinitionNameStatusByMemberParams{
		MemberID:   m.ID,
		SchemeName: schemeName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CheckSchemeNameResult{Available: true}, nil
		}
		return CheckSchemeNameResult{}, err
	}
	return CheckSchemeNameResult{
		Available:            false,
		ExistingDefinitionID: row.ID,
		ExistingHasInstance:  row.HasInstance,
	}, nil
}

// CheckSchemeNameAvailable 兼容旧调用方。
func (s *Service) CheckSchemeNameAvailable(ctx context.Context, account, schemeName string) (bool, error) {
	res, err := s.CheckSchemeName(ctx, account, schemeName)
	if err != nil {
		return false, err
	}
	return res.Available, nil
}
