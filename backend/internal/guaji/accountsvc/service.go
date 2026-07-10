package accountsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/member"
)

const (
	defaultTokenTTL   = 23 * time.Hour
	maxReauthFailures = 3
)

type Service struct {
	pool    *db.Pool
	guaji   *guaji.Client
	credKey []byte
	jwtFallback string
}

func NewService(pool *db.Pool, client *guaji.Client, credentialsKey, jwtFallback string) *Service {
	if pool == nil {
		return nil
	}
	key, _ := guaji.CredentialsKey(credentialsKey, jwtFallback)
	return &Service{pool: pool, guaji: client, credKey: key, jwtFallback: jwtFallback}
}

func (s *Service) Bind(ctx context.Context, memberAccount string, in BindInput) (BindResult, error) {
	if s == nil {
		return BindResult{}, ErrUnavailable
	}
	if s.guaji == nil || !s.guaji.Enabled() {
		return BindResult{}, ErrGuajiDisabled
	}
	if len(s.credKey) == 0 {
		return BindResult{}, ErrCredentialsKey
	}
	username := strings.TrimSpace(in.Username)
	password := strings.TrimSpace(in.Password)
	if username == "" || password == "" {
		return BindResult{}, fmt.Errorf("%w: 用户名与密码不能为空", ErrInvalidCredentials)
	}

	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return BindResult{}, err
	}

	existing, err := s.getRowByUsername(ctx, username)
	if err == nil && existing.memberID != m {
		return BindResult{}, ErrUsernameTaken
	}
	if err != nil && !isNoRows(err) {
		return BindResult{}, err
	}

	loginRes, mfaErr, err := s.loginThirdParty(ctx, in)
	if mfaErr != nil {
		return BindResult{MFARequired: true, LoginKey: mfaErr.LoginKey}, nil
	}
	if err != nil {
		return BindResult{}, mapLoginErr(err)
	}

	passEnc, err := guaji.EncryptSecret(s.credKey, password)
	if err != nil {
		return BindResult{}, err
	}
	accessEnc, refreshEnc, expiresAt, err := s.encryptTokens(loginRes)
	if err != nil {
		return BindResult{}, err
	}

	count, err := s.countBindings(ctx, m)
	if err != nil {
		return BindResult{}, err
	}
	autoActive := count == 0

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BindResult{}, err
	}
	defer tx.Rollback(ctx)

	if autoActive {
		if err := s.deactivateAll(ctx, tx, m); err != nil {
			return BindResult{}, err
		}
		if _, err := s.pauseRunningPending(ctx, tx, m); err != nil {
			return BindResult{}, err
		}
	}

	row, err := s.insertRowTx(ctx, tx, m, username, passEnc, "", accessEnc, refreshEnc, expiresAt, autoActive)
	if err != nil {
		if strings.Contains(err.Error(), "member_guaji_accounts_username_unique") {
			return BindResult{}, ErrUsernameTaken
		}
		return BindResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return BindResult{}, err
	}
	acct := mapPublic(row)
	return BindResult{Account: &acct}, nil
}

func (s *Service) List(ctx context.Context, memberAccount string) ([]Account, error) {
	if s == nil {
		return nil, ErrUnavailable
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return nil, err
	}
	rows, err := s.listRows(ctx, m)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, r := range rows {
		out = append(out, mapPublic(r))
	}
	return out, nil
}

func (s *Service) Activate(ctx context.Context, memberAccount string, accountID int64) (Account, error) {
	if s == nil {
		return Account{}, ErrUnavailable
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return Account{}, err
	}
	if _, err := s.getRowByID(ctx, m, accountID); err != nil {
		if isNoRows(err) {
			return Account{}, ErrAccountNotFound
		}
		return Account{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := s.pauseRunningPending(ctx, tx, m); err != nil {
		return Account{}, err
	}
	if err := s.deactivateAll(ctx, tx, m); err != nil {
		return Account{}, err
	}
	if err := s.activateRow(ctx, tx, m, accountID); err != nil {
		return Account{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Account{}, err
	}
	row, err := s.getRowByID(ctx, m, accountID)
	if err != nil {
		return Account{}, err
	}
	return mapPublic(row), nil
}

func (s *Service) Unbind(ctx context.Context, memberAccount string, accountID int64) error {
	if s == nil {
		return ErrUnavailable
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return err
	}
	row, err := s.getRowByID(ctx, m, accountID)
	if err != nil {
		if isNoRows(err) {
			return ErrAccountNotFound
		}
		return err
	}

	bindingCount, err := s.countBindings(ctx, m)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 解绑启用中账号，或解绑最后一个授权时，暂停全部 running/pending 方案（§4.4）。
	if row.isActive || bindingCount <= 1 {
		if _, err := s.pauseRunningPending(ctx, tx, m); err != nil {
			return err
		}
	}
	if err := s.deleteRowTx(ctx, tx, m, accountID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) Reauth(ctx context.Context, memberAccount string, accountID int64) (Account, error) {
	if s == nil {
		return Account{}, ErrUnavailable
	}
	if s.guaji == nil || !s.guaji.Enabled() {
		return Account{}, ErrGuajiDisabled
	}
	if len(s.credKey) == 0 {
		return Account{}, ErrCredentialsKey
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return Account{}, err
	}
	row, err := s.getRowByID(ctx, m, accountID)
	if err != nil {
		if isNoRows(err) {
			return Account{}, ErrAccountNotFound
		}
		return Account{}, err
	}
	if row.reauthFailCount >= maxReauthFailures {
		return Account{}, ErrReauthNeedsBind
	}

	password, err := guaji.DecryptSecret(s.credKey, row.passwordEnc)
	if err != nil {
		return Account{}, err
	}

	var loginRes *guaji.LoginResult
	if row.refreshTokenEnc.Valid && row.refreshTokenEnc.String != "" {
		refresh, err := guaji.DecryptSecret(s.credKey, row.refreshTokenEnc.String)
		if err == nil {
			loginRes, err = s.guaji.RefreshToken(ctx, refresh)
			if err != nil {
				loginRes = nil
			}
		}
	}
	if loginRes == nil {
		in := BindInput{Username: row.guajiUsername, Password: password}
		if row.mfaMaterialEnc.Valid {
			var mat map[string]string
			_ = json.Unmarshal([]byte(row.mfaMaterialEnc.String), &mat)
			in.LoginKey = mat["loginKey"]
			in.GoogleCode = mat["googleCode"]
		}
		res, mfaErr, err := s.loginThirdParty(ctx, in)
		if mfaErr != nil || err != nil {
			msg := "重新授权失败"
			if mfaErr != nil {
				msg = "需要二次验证，请重新绑定授权"
			} else if err != nil {
				msg = guaji.ClassifyUpstreamError(err).UserMessage
			}
			_ = s.markTokenError(ctx, m, accountID, msg)
			if row.reauthFailCount+1 >= maxReauthFailures {
				return Account{}, ErrReauthNeedsBind
			}
			return Account{}, ErrTokenInvalid
		}
		loginRes = res
	}

	accessEnc, refreshEnc, expiresAt, err := s.encryptTokens(loginRes)
	if err != nil {
		return Account{}, err
	}
	if err := s.updateTokens(ctx, m, accountID, accessEnc, refreshEnc, expiresAt); err != nil {
		return Account{}, err
	}
	row, err = s.getRowByID(ctx, m, accountID)
	if err != nil {
		return Account{}, err
	}
	if !accountTokenValid(row) {
		return Account{}, ErrTokenInvalid
	}
	return mapPublic(row), nil
}

func (s *Service) Balance(ctx context.Context, memberAccount string) (BalanceResult, error) {
	if s == nil {
		return BalanceResult{}, ErrUnavailable
	}
	if s.guaji == nil || !s.guaji.Enabled() {
		return BalanceResult{}, ErrGuajiDisabled
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return BalanceResult{}, err
	}
	currency, err := s.primaryCurrency(ctx, m)
	if err != nil {
		return BalanceResult{}, err
	}
	row, err := s.getActiveRow(ctx, m)
	if err != nil {
		if isNoRows(err) {
			return BalanceResult{}, ErrNoActiveAccount
		}
		return BalanceResult{}, err
	}
	if !s.tokenHealthy(row) {
		return BalanceResult{}, ErrTokenInvalid
	}
	token, err := guaji.DecryptSecret(s.credKey, row.accessTokenEnc.String)
	if err != nil {
		return BalanceResult{}, err
	}
	info, err := s.guaji.UserInfo(ctx, token)
	if err != nil {
		fault := guaji.ClassifyUpstreamError(err)
		if fault.IsTokenInvalid {
			_ = s.markTokenError(ctx, m, row.id, fault.UserMessage)
			return BalanceResult{}, ErrTokenInvalid
		}
		return BalanceResult{}, ErrGuajiUpstream
	}
	s.persistGuajiBalances(ctx, row.id, multiCurrencyFromInfo(info))
	return BalanceResult{
		Currency: currency,
		Amount:   info.BalanceByCurrency(currency),
		Username: info.Username,
	}, nil
}

// PrimaryCurrency 返回会员主币种（默认 CNY）。
func (s *Service) PrimaryCurrency(ctx context.Context, memberAccount string) (string, error) {
	if s == nil {
		return "", ErrUnavailable
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return "", err
	}
	return s.primaryCurrency(ctx, m)
}

// SetPrimaryCurrency 切换主币种；同切换授权逻辑：全部 running+pending → paused（§4.4）。
func (s *Service) SetPrimaryCurrency(ctx context.Context, memberAccount, currency string) (string, error) {
	if s == nil {
		return "", ErrUnavailable
	}
	next := guaji.NormalizeCurrency(currency)
	if next != currency {
		return "", ErrInvalidCurrency
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return "", err
	}
	cur, err := s.primaryCurrency(ctx, m)
	if err != nil {
		return "", err
	}
	if cur == next {
		return next, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	if _, err := s.pauseRunningPending(ctx, tx, m); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE members SET primary_currency = $2, updated_at = now() WHERE id = $1`,
		m, next); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return next, nil
}

func (s *Service) primaryCurrency(ctx context.Context, memberID int64) (string, error) {
	var c string
	err := s.pool.QueryRow(ctx, `SELECT primary_currency FROM members WHERE id = $1`, memberID).Scan(&c)
	if err != nil {
		if isNoRows(err) {
			return guaji.CurrencyCNY, nil
		}
		return "", err
	}
	return guaji.NormalizeCurrency(c), nil
}

// HasHealthyAuthForMember 报告会员是否有可用（未过期）的启用授权。
func (s *Service) HasHealthyAuthForMember(ctx context.Context, memberAccount string) (bool, error) {
	st, err := s.AuthStatus(ctx, memberAccount)
	if err != nil {
		return false, err
	}
	return st.HasActiveGuajiAuth, nil
}

func (s *Service) AuthStatus(ctx context.Context, memberAccount string) (AuthStatus, error) {
	if s == nil {
		return AuthStatus{}, ErrUnavailable
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return AuthStatus{}, err
	}
	count, err := s.countBindings(ctx, m)
	if err != nil {
		return AuthStatus{}, err
	}
	st := AuthStatus{BindingCount: count}
	row, err := s.getActiveRow(ctx, m)
	if err != nil {
		if isNoRows(err) {
			return st, nil
		}
		return AuthStatus{}, err
	}
	st.ActiveUsername = row.guajiUsername
	st.HasActiveGuajiAuth = s.tokenHealthy(row)
	st.ActiveAuthExpired = !st.HasActiveGuajiAuth
	return st, nil
}

func (s *Service) AdminList(ctx context.Context, memberID int64) ([]AdminAccountRow, error) {
	if s == nil {
		return nil, ErrUnavailable
	}
	var exists int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM members WHERE id = $1`, memberID).Scan(&exists)
	if err != nil {
		if isNoRows(err) {
			return nil, member.ErrNotFound
		}
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, guaji_username, is_active, bound_at, last_sync_at, last_token_error, last_bet_at
FROM member_guaji_accounts
WHERE member_id = $1
ORDER BY is_active DESC, bound_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminAccountRow
	for rows.Next() {
		var r AdminAccountRow
		var lastSync, lastBet pgtype.Timestamptz
		var lastErr pgtype.Text
		if err := rows.Scan(&r.ID, &r.GuajiUsername, &r.IsActive, &r.BoundAt, &lastSync, &lastErr, &lastBet); err != nil {
			return nil, err
		}
		r.LastSyncAt = tsPtr(lastSync)
		r.LastTokenError = textPtr(lastErr)
		r.LastBetAt = tsPtr(lastBet)
		out = append(out, r)
	}
	return out, rows.Err()
}

// AdminClearAllAuth 停止该会员全部运行中/待开启方案，并清空全部第三方授权。
func (s *Service) AdminClearAllAuth(ctx context.Context, memberID int64) (pausedCount int, clearedCount int64, err error) {
	if s == nil || s.pool == nil {
		return 0, 0, ErrUnavailable
	}
	if memberID <= 0 {
		return 0, 0, member.ErrNotFound
	}
	var exists int64
	err = s.pool.QueryRow(ctx, `SELECT id FROM members WHERE id = $1`, memberID).Scan(&exists)
	if err != nil {
		if isNoRows(err) {
			return 0, 0, member.ErrNotFound
		}
		return 0, 0, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx)

	pausedCount, err = s.pauseRunningPending(ctx, tx, memberID)
	if err != nil {
		return 0, 0, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM member_guaji_accounts WHERE member_id = $1`, memberID)
	if err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return pausedCount, tag.RowsAffected(), nil
}

func (s *Service) memberID(ctx context.Context, account string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM members WHERE account = $1`, account).Scan(&id)
	if err != nil {
		if isNoRows(err) {
			return 0, member.ErrNotFound
		}
		return 0, err
	}
	return id, nil
}

func (s *Service) tokenHealthy(r row) bool {
	return r.isActive && accountTokenValid(r)
}

func (s *Service) loginThirdParty(ctx context.Context, in BindInput) (*guaji.LoginResult, *guaji.MFARequiredError, error) {
	if in.LoginKey != "" || in.GoogleCode != "" || in.EmailCode != "" || in.PhoneCode != "" {
		req := guaji.LoginRequest{
			Username:   in.Username,
			Password:   in.Password,
			LoginKey:   in.LoginKey,
			GoogleCode: in.GoogleCode,
			EmailCode:  in.EmailCode,
			PhoneCode:  in.PhoneCode,
		}
		res, err := s.guaji.LoginWithMFA(ctx, req)
		if err != nil {
			var mfa *guaji.MFARequiredError
			if errors.As(err, &mfa) {
				return nil, mfa, nil
			}
			return nil, nil, err
		}
		return res, nil, nil
	}
	res, err := s.guaji.Login(ctx, in.Username, in.Password)
	if err != nil {
		var mfa *guaji.MFARequiredError
		if errors.As(err, &mfa) {
			return nil, mfa, nil
		}
		return nil, nil, err
	}
	return res, nil, nil
}

func (s *Service) encryptTokens(res *guaji.LoginResult) (accessEnc, refreshEnc string, expiresAt time.Time, err error) {
	expiresAt = time.Now().Add(defaultTokenTTL)
	if len(s.credKey) == 0 {
		return "", "", time.Time{}, ErrCredentialsKey
	}
	if res == nil || strings.TrimSpace(res.Token) == "" {
		return "", "", time.Time{}, fmt.Errorf("第三方未返回 access token")
	}
	accessEnc, err = guaji.EncryptSecret(s.credKey, res.Token)
	if err != nil || accessEnc == "" {
		return "", "", time.Time{}, fmt.Errorf("加密 access token 失败: %w", err)
	}
	if res.RefreshToken != "" {
		refreshEnc, err = guaji.EncryptSecret(s.credKey, res.RefreshToken)
		if err != nil {
			return "", "", time.Time{}, fmt.Errorf("加密 refresh token 失败: %w", err)
		}
	}
	return accessEnc, refreshEnc, expiresAt, nil
}

func mapLoginErr(err error) error {
	var api *guaji.APIError
	if errors.As(err, &api) {
		return ErrInvalidCredentials
	}
	return err
}

func (s *Service) insertRowTx(ctx context.Context, tx pgx.Tx, memberID int64, username, passEnc, mfaEnc, accessEnc, refreshEnc string, expiresAt time.Time, active bool) (row, error) {
	return scanRow(tx.QueryRow(ctx, `
INSERT INTO member_guaji_accounts (
    member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active, last_sync_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now())
RETURNING `+accountCols,
		memberID, username, passEnc, nullText(mfaEnc), nullText(accessEnc), nullText(refreshEnc), expiresAt, active))
}

func (s *Service) deleteRowTx(ctx context.Context, tx pgx.Tx, memberID, id int64) error {
	tag, err := tx.Exec(ctx, `DELETE FROM member_guaji_accounts WHERE id = $1 AND member_id = $2`, id, memberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}
