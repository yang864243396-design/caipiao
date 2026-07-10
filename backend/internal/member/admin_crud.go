package member

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"caipiao/backend/internal/db/sqlcdb"
)

var (
	ErrInvalidInput      = errors.New("invalid member input")
	ErrDuplicateAccount  = errors.New("member account duplicate")
	ErrPasswordTooShort  = errors.New("password too short")
)

type AdminMemberCreateInput struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

type AdminMemberUpdateInput struct {
	Password string `json:"password,omitempty"`
	Status   string `json:"status"`
}

func (s *Service) AdminCreateMember(ctx context.Context, in AdminMemberCreateInput) (AdminMemberRow, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return AdminMemberRow{}, ErrUnavailable
	}
	account := strings.TrimSpace(in.Account)
	password := strings.TrimSpace(in.Password)
	status, err := normalizeMemberStatus(in.Status)
	if err != nil {
		return AdminMemberRow{}, err
	}
	if account == "" {
		return AdminMemberRow{}, fmt.Errorf("%w: 会员账号不能为空", ErrInvalidInput)
	}
	if utf8.RuneCountInString(account) > 32 {
		return AdminMemberRow{}, fmt.Errorf("%w: 会员账号过长", ErrInvalidInput)
	}
	if len(password) < 6 {
		return AdminMemberRow{}, ErrPasswordTooShort
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AdminMemberRow{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AdminMemberRow{}, err
	}
	defer tx.Rollback(ctx)

	qtx := sqlcdb.New(tx)
	row, err := qtx.AdminInsertMember(ctx, sqlcdb.AdminInsertMemberParams{
		Account:      account,
		PasswordHash: string(hash),
		DisplayName:  account,
		Status:       status,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return AdminMemberRow{}, ErrDuplicateAccount
		}
		return AdminMemberRow{}, err
	}
	if err := qtx.AdminInsertMemberWallet(ctx, row.ID); err != nil {
		return AdminMemberRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminMemberRow{}, err
	}
	return s.AdminGetMember(ctx, row.ID)
}

func (s *Service) AdminUpdateMember(ctx context.Context, memberID int64, in AdminMemberUpdateInput) (AdminMemberRow, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return AdminMemberRow{}, ErrUnavailable
	}
	if memberID <= 0 {
		return AdminMemberRow{}, ErrNotFound
	}
	status, err := normalizeMemberStatus(in.Status)
	if err != nil {
		return AdminMemberRow{}, err
	}
	password := strings.TrimSpace(in.Password)
	if password != "" && len(password) < 6 {
		return AdminMemberRow{}, ErrPasswordTooShort
	}

	cur, err := s.q.GetMemberByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminMemberRow{}, ErrNotFound
		}
		return AdminMemberRow{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AdminMemberRow{}, err
	}
	defer tx.Rollback(ctx)
	qtx := sqlcdb.New(tx)

	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return AdminMemberRow{}, err
		}
		rows, err := qtx.AdminUpdateMemberPasswordByID(ctx, sqlcdb.AdminUpdateMemberPasswordByIDParams{
			ID: memberID, PasswordHash: string(hash),
		})
		if err != nil {
			return AdminMemberRow{}, err
		}
		if rows == 0 {
			return AdminMemberRow{}, ErrNotFound
		}
	}

	var paused []sqlcdb.PauseRunningPendingInstancesByMemberRow
	if status != cur.Status {
		rows, err := qtx.AdminUpdateMemberStatus(ctx, sqlcdb.AdminUpdateMemberStatusParams{
			ID: memberID, Status: status,
		})
		if err != nil {
			return AdminMemberRow{}, err
		}
		if rows == 0 {
			return AdminMemberRow{}, ErrNotFound
		}
		if status == "frozen" {
			paused, err = qtx.PauseRunningPendingInstancesByMember(ctx, memberID)
			if err != nil {
				return AdminMemberRow{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return AdminMemberRow{}, err
	}

	if n := len(paused); n > 0 {
		s.notifyPausedSchemeInstances(cur.Account, paused)
		slog.Info("member disabled, schemes paused", "memberId", memberID, "account", cur.Account, "count", n)
	}

	return s.AdminGetMember(ctx, memberID)
}

func normalizeMemberStatus(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case "", "active", "正常":
		return "active", nil
	case "frozen", "禁用", "禁止":
		return "frozen", nil
	default:
		return "", fmt.Errorf("%w: 状态须为正常或禁用", ErrInvalidInput)
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
