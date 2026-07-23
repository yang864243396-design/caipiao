package accountsvc

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/guaji"
)

type row struct {
	id              int64
	memberID        int64
	guajiUsername   string
	passwordEnc     string
	mfaMaterialEnc  pgtype.Text
	accessTokenEnc  pgtype.Text
	refreshTokenEnc pgtype.Text
	tokenExpiresAt  pgtype.Timestamptz
	isActive        bool
	boundAt         time.Time
	lastSyncAt      pgtype.Timestamptz
	lastTokenError  pgtype.Text
	lastBetAt       pgtype.Timestamptz
	reauthFailCount int32
}

const accountCols = `id, member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error, last_bet_at, reauth_fail_count`

func scanRow(r pgx.Row) (row, error) {
	var x row
	err := r.Scan(
		&x.id, &x.memberID, &x.guajiUsername, &x.passwordEnc, &x.mfaMaterialEnc,
		&x.accessTokenEnc, &x.refreshTokenEnc, &x.tokenExpiresAt, &x.isActive,
		&x.boundAt, &x.lastSyncAt, &x.lastTokenError, &x.lastBetAt, &x.reauthFailCount,
	)
	return x, err
}

func (s *Service) listRows(ctx context.Context, memberID int64) ([]row, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+accountCols+`
FROM member_guaji_accounts
WHERE member_id = $1
ORDER BY is_active DESC, bound_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []row
	for rows.Next() {
		x, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Service) getRowByID(ctx context.Context, memberID, id int64) (row, error) {
	return scanRow(s.pool.QueryRow(ctx, `
SELECT `+accountCols+`
FROM member_guaji_accounts
WHERE id = $1 AND member_id = $2`, id, memberID))
}

func (s *Service) getRowByUsername(ctx context.Context, username string) (row, error) {
	return scanRow(s.pool.QueryRow(ctx, `
SELECT `+accountCols+`
FROM member_guaji_accounts
WHERE guaji_username = $1`, username))
}

func (s *Service) getRowByIDAny(ctx context.Context, id int64) (row, error) {
	return scanRow(s.pool.QueryRow(ctx, `
SELECT `+accountCols+`
FROM member_guaji_accounts
WHERE id = $1`, id))
}

func (s *Service) getActiveRow(ctx context.Context, memberID int64) (row, error) {
	return scanRow(s.pool.QueryRow(ctx, `
SELECT `+accountCols+`
FROM member_guaji_accounts
WHERE member_id = $1 AND is_active = true
LIMIT 1`, memberID))
}

func (s *Service) countBindings(ctx context.Context, memberID int64) (int, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `
SELECT COUNT(*) FROM member_guaji_accounts WHERE member_id = $1`, memberID).Scan(&n)
	return int(n), err
}

func (s *Service) insertRow(ctx context.Context, memberID int64, username, passEnc, mfaEnc, accessEnc, refreshEnc string, expiresAt time.Time, active bool) (row, error) {
	return scanRow(s.pool.QueryRow(ctx, `
INSERT INTO member_guaji_accounts (
    member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active, last_sync_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now())
RETURNING `+accountCols,
		memberID, username, passEnc, nullText(mfaEnc), nullText(accessEnc), nullText(refreshEnc), expiresAt, active))
}

func (s *Service) deactivateAll(ctx context.Context, tx pgx.Tx, memberID int64) error {
	_, err := tx.Exec(ctx, `
UPDATE member_guaji_accounts SET is_active = false, updated_at = now()
WHERE member_id = $1 AND is_active = true`, memberID)
	return err
}

func (s *Service) activateRow(ctx context.Context, tx pgx.Tx, memberID, id int64) error {
	tag, err := tx.Exec(ctx, `
UPDATE member_guaji_accounts
SET is_active = true, last_token_error = NULL, updated_at = now()
WHERE id = $1 AND member_id = $2`, id, memberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (s *Service) updateTokens(ctx context.Context, memberID, id int64, accessEnc, refreshEnc string, expiresAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE member_guaji_accounts
SET access_token_enc = $3, refresh_token_enc = $4, token_expires_at = $5,
    last_sync_at = now(), last_token_error = NULL, reauth_fail_count = 0, updated_at = now()
WHERE id = $1 AND member_id = $2`, id, memberID, nullText(accessEnc), nullText(refreshEnc), expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// markTokenError 仅当库内 access_token_enc 仍与失败请求所用密文一致时写入。
// 避免 reauth 已换新 token 后，旧 in-flight 请求的 401 把账号重新标为失效。
func (s *Service) markTokenError(ctx context.Context, memberID, id int64, accessTokenEnc, msg string) error {
	_, err := s.pool.Exec(ctx, `
UPDATE member_guaji_accounts
SET last_token_error = $4, reauth_fail_count = reauth_fail_count + 1, updated_at = now()
WHERE id = $1 AND member_id = $2
  AND access_token_enc IS NOT DISTINCT FROM $3`,
		id, memberID, nullText(accessTokenEnc), msg)
	return err
}

func (s *Service) clearTokenError(ctx context.Context, memberID, id int64) error {
	_, err := s.pool.Exec(ctx, `
UPDATE member_guaji_accounts
SET last_token_error = NULL, updated_at = now()
WHERE id = $1 AND member_id = $2`, id, memberID)
	return err
}

func (s *Service) deleteRow(ctx context.Context, memberID, id int64) error {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM member_guaji_accounts WHERE id = $1 AND member_id = $2`, id, memberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (s *Service) pauseRunningPending(ctx context.Context, tx pgx.Tx, memberID int64) (int, error) {
	tag, err := tx.Exec(ctx, `
UPDATE scheme_instances
SET status = 'paused', status_reason = 'manual', updated_at = now()
WHERE member_id = $1 AND status IN ('running', 'pending')`, memberID)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func nullText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func accountTokenValid(r row) bool {
	if !r.accessTokenEnc.Valid || r.accessTokenEnc.String == "" {
		return false
	}
	if r.lastTokenError.Valid && r.lastTokenError.String != "" {
		if guaji.ClassifyUpstreamError(errors.New(r.lastTokenError.String)).IsTokenInvalid {
			return false
		}
	}
	if r.tokenExpiresAt.Valid && r.tokenExpiresAt.Time.Before(time.Now()) {
		return false
	}
	return true
}

func mapPublic(r row) Account {
	return Account{
		ID:              r.id,
		GuajiUsername:   r.guajiUsername,
		IsActive:        r.isActive,
		BoundAt:         r.boundAt,
		LastSyncAt:      tsPtr(r.lastSyncAt),
		LastTokenError:  sanitizeLastTokenError(r.lastTokenError),
		LastBetAt:       tsPtr(r.lastBetAt),
		ReauthFailCount: int(r.reauthFailCount),
		AuthExpired:     r.isActive && !accountTokenValid(r),
	}
}

func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func sanitizeLastTokenError(t pgtype.Text) *string {
	if !t.Valid || t.String == "" {
		return nil
	}
	fault := guaji.ClassifyUpstreamError(errors.New(t.String))
	if !fault.IsTokenInvalid {
		return nil
	}
	msg := fault.UserMessage
	return &msg
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}
