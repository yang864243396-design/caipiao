package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

const protectedAdminRoleID = "r_super"

var (
	ErrRoleNotFound  = errors.New("admin role not found")
	ErrProtectedRole = errors.New("protected admin role")
)

type AdminRole struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	MenuPaths []string `json:"menuPaths"`
	CreatedAt string   `json:"createdAt,omitempty"`
	UpdatedAt string   `json:"updatedAt,omitempty"`
}

func (s *Service) AdminListRoles(ctx context.Context) ([]AdminRole, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListAdminRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminRole, 0, len(rows))
	for _, row := range rows {
		item, err := mapAdminRoleRow(row.ID, row.Name, row.MenuPaths, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) AdminSaveRole(ctx context.Context, in AdminRole) (AdminRole, error) {
	if s == nil || s.q == nil {
		return AdminRole{}, ErrUnavailable
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = fmt.Sprintf("r_%d", time.Now().UnixNano())
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return AdminRole{}, ErrInvalid
	}
	paths := normalizeMenuPaths(in.MenuPaths)
	raw, err := json.Marshal(paths)
	if err != nil {
		return AdminRole{}, ErrInvalid
	}
	row, err := s.q.UpsertAdminRole(ctx, sqlcdb.UpsertAdminRoleParams{
		ID: id, Name: name, Column3: raw,
	})
	if err != nil {
		return AdminRole{}, err
	}
	return mapAdminRoleRow(row.ID, row.Name, row.MenuPaths, row.CreatedAt, row.UpdatedAt)
}

func (s *Service) AdminDeleteRole(ctx context.Context, id string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	if id == protectedAdminRoleID {
		return ErrProtectedRole
	}
	n, err := s.q.DeleteAdminRole(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRoleNotFound
	}
	return nil
}

func normalizeMenuPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"/dashboard"}
	}
	return out
}

func mapAdminRoleRow(id, name string, raw []byte, createdAt, updatedAt pgtype.Timestamptz) (AdminRole, error) {
	paths := []string{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &paths); err != nil {
			return AdminRole{}, err
		}
	}
	item := AdminRole{ID: id, Name: name, MenuPaths: paths}
	if createdAt.Valid {
		item.CreatedAt = timeutil.FormatISO(createdAt.Time)
	}
	if updatedAt.Valid {
		item.UpdatedAt = timeutil.FormatISO(updatedAt.Time)
	}
	return item, nil
}
