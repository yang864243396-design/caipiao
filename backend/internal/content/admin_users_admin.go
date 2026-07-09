package content

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

const protectedAdminAccount = "admin"

var (
	ErrAdminUserNotFound   = errors.New("admin user not found")
	ErrProtectedAdminUser  = errors.New("protected admin user")
	ErrDuplicateAdminAcct  = errors.New("admin account duplicate")
	ErrAdminUserSelfDelete = errors.New("cannot delete current admin account")
)

type AdminUser struct {
	ID          int64  `json:"id"`
	Account     string `json:"account"`
	DisplayName string `json:"displayName"`
	RoleID      string `json:"roleId"`
	RoleName    string `json:"roleName,omitempty"`
	Status      string `json:"status"`
	LastLoginAt string `json:"lastLoginAt,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type AdminUserSaveInput struct {
	Account     string `json:"account"`
	DisplayName string `json:"displayName"`
	RoleID      string `json:"roleId"`
	Status      string `json:"status"`
	Password    string `json:"password,omitempty"`
}

func (s *Service) AdminListUsers(ctx context.Context) ([]AdminUser, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListAdminUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapListAdminUserRow(row))
	}
	return out, nil
}

func (s *Service) AdminCreateUser(ctx context.Context, in AdminUserSaveInput) (AdminUser, error) {
	if s == nil || s.q == nil {
		return AdminUser{}, ErrUnavailable
	}
	account := strings.TrimSpace(in.Account)
	displayName := strings.TrimSpace(in.DisplayName)
	roleID := strings.TrimSpace(in.RoleID)
	status := normalizeAdminUserStatus(in.Status)
	password := strings.TrimSpace(in.Password)
	if account == "" || displayName == "" || roleID == "" || password == "" {
		return AdminUser{}, ErrInvalid
	}
	if len(password) < 6 {
		return AdminUser{}, ErrInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AdminUser{}, fmt.Errorf("hash password: %w", err)
	}
	row, err := s.q.CreateAdminUser(ctx, sqlcdb.CreateAdminUserParams{
		Account: account, PasswordHash: string(hash), DisplayName: displayName, RoleID: roleID, Status: status,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return AdminUser{}, ErrDuplicateAdminAcct
		}
		return AdminUser{}, err
	}
	return s.mapAdminUserByID(ctx, row.ID)
}

func (s *Service) AdminUpdateUser(ctx context.Context, id int64, in AdminUserSaveInput) (AdminUser, error) {
	if s == nil || s.q == nil {
		return AdminUser{}, ErrUnavailable
	}
	if id <= 0 {
		return AdminUser{}, ErrInvalid
	}
	displayName := strings.TrimSpace(in.DisplayName)
	roleID := strings.TrimSpace(in.RoleID)
	status := normalizeAdminUserStatus(in.Status)
	if displayName == "" || roleID == "" {
		return AdminUser{}, ErrInvalid
	}
	row, err := s.q.UpdateAdminUser(ctx, sqlcdb.UpdateAdminUserParams{
		ID: id, DisplayName: displayName, RoleID: roleID, Status: status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminUser{}, ErrAdminUserNotFound
		}
		return AdminUser{}, err
	}
	password := strings.TrimSpace(in.Password)
	if password != "" {
		if len(password) < 6 {
			return AdminUser{}, ErrInvalid
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return AdminUser{}, fmt.Errorf("hash password: %w", err)
		}
		if err := s.q.UpdateAdminUserPassword(ctx, sqlcdb.UpdateAdminUserPasswordParams{
			ID: id, PasswordHash: string(hash),
		}); err != nil {
			return AdminUser{}, err
		}
	}
	return s.mapAdminUserByID(ctx, row.ID)
}

func (s *Service) AdminDeleteUser(ctx context.Context, id int64, currentAccount string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	if id <= 0 {
		return ErrInvalid
	}
	row, err := s.q.GetAdminUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAdminUserNotFound
		}
		return err
	}
	if row.Account == protectedAdminAccount {
		return ErrProtectedAdminUser
	}
	if strings.TrimSpace(currentAccount) != "" && row.Account == strings.TrimSpace(currentAccount) {
		return ErrAdminUserSelfDelete
	}
	n, err := s.q.DeleteAdminUser(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrAdminUserNotFound
	}
	return nil
}

func normalizeAdminUserStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "disabled" {
		return "disabled"
	}
	return "active"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func mapListAdminUserRow(row sqlcdb.ListAdminUsersRow) AdminUser {
	item := AdminUser{
		ID: row.ID, Account: row.Account, DisplayName: row.DisplayName,
		RoleID: row.RoleID, RoleName: row.RoleName, Status: row.Status,
	}
	if row.LastLoginAt.Valid {
		item.LastLoginAt = timeutil.FormatISO(row.LastLoginAt.Time)
	}
	if row.CreatedAt.Valid {
		item.CreatedAt = timeutil.FormatISO(row.CreatedAt.Time)
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = timeutil.FormatISO(row.UpdatedAt.Time)
	}
	return item
}

func (s *Service) mapAdminUserByID(ctx context.Context, id int64) (AdminUser, error) {
	full, err := s.q.GetAdminUserByID(ctx, id)
	if err != nil {
		return AdminUser{}, err
	}
	return mapGetAdminUserRow(full), nil
}

func mapGetAdminUserRow(row sqlcdb.GetAdminUserByIDRow) AdminUser {
	item := AdminUser{
		ID: row.ID, Account: row.Account, DisplayName: row.DisplayName,
		RoleID: row.RoleID, RoleName: row.RoleName, Status: row.Status,
	}
	if row.LastLoginAt.Valid {
		item.LastLoginAt = timeutil.FormatISO(row.LastLoginAt.Time)
	}
	if row.CreatedAt.Valid {
		item.CreatedAt = timeutil.FormatISO(row.CreatedAt.Time)
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = timeutil.FormatISO(row.UpdatedAt.Time)
	}
	return item
}
