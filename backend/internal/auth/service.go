package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleClient Role = "client"
	RoleAdmin  Role = "admin"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountFrozen      = errors.New("account frozen")
)

type TokenResult struct {
	AccessToken string
	ExpiresAt   time.Time
	Account     string
	DisplayName string
	RoleID      string
}

type Service struct {
	cfg config.Config
	q   *sqlcdb.Queries
}

func NewService(cfg config.Config, pool *db.Pool) *Service {
	s := &Service{cfg: cfg}
	if pool != nil {
		s.q = sqlcdb.New(pool)
	}
	return s
}

type claims struct {
	Role        Role   `json:"role"`
	DisplayName string `json:"displayName"`
	AdminRoleID string `json:"adminRoleId,omitempty"`
	jwt.RegisteredClaims
}

// Claims is exported for middleware context.
type Claims = claims

func (s *Service) LoginClient(account, password string) (TokenResult, error) {
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		return TokenResult{}, ErrInvalidCredentials
	}
	if s.q != nil {
		return s.loginClientDB(context.Background(), account, password)
	}
	return s.loginClientEnv(account, password)
}

func (s *Service) loginClientDB(ctx context.Context, account, password string) (TokenResult, error) {
	row, err := s.q.GetMemberForLogin(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenResult{}, ErrInvalidCredentials
		}
		return TokenResult{}, fmt.Errorf("login lookup: %w", err)
	}
	if row.Status == "frozen" {
		return TokenResult{}, ErrAccountFrozen
	}
	if row.Status != "active" {
		return TokenResult{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)); err != nil {
		return TokenResult{}, ErrInvalidCredentials
	}
	_ = s.q.TouchMemberLastLogin(ctx, row.ID)
	return s.issue(RoleClient, row.Account, row.DisplayName, "")
}

func (s *Service) loginClientEnv(account, password string) (TokenResult, error) {
	if account != s.cfg.ClientDemoAccount || password != s.cfg.ClientDemoPass {
		return TokenResult{}, ErrInvalidCredentials
	}
	return s.issue(RoleClient, account, "演示会员", "")
}

func (s *Service) LoginAdmin(account, password string) (TokenResult, error) {
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		return TokenResult{}, ErrInvalidCredentials
	}
	if s.q != nil {
		return s.loginAdminDB(context.Background(), account, password)
	}
	return s.loginAdminEnv(account, password)
}

func (s *Service) loginAdminDB(ctx context.Context, account, password string) (TokenResult, error) {
	row, err := s.q.GetAdminUserForLogin(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenResult{}, ErrInvalidCredentials
		}
		return TokenResult{}, fmt.Errorf("admin login lookup: %w", err)
	}
	if row.Status != "active" {
		return TokenResult{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)); err != nil {
		return TokenResult{}, ErrInvalidCredentials
	}
	_ = s.q.TouchAdminLastLogin(ctx, row.ID)
	return s.issue(RoleAdmin, row.Account, row.DisplayName, row.RoleID)
}

func (s *Service) loginAdminEnv(account, password string) (TokenResult, error) {
	if account != s.cfg.AdminDemoAccount || password != s.cfg.AdminDemoPass {
		return TokenResult{}, ErrInvalidCredentials
	}
	return s.issue(RoleAdmin, account, "管理员", "r_super")
}

func (s *Service) issue(role Role, account, displayName, adminRoleID string) (TokenResult, error) {
	expiresAt := time.Now().UTC().Add(s.cfg.TokenTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role:        role,
		DisplayName: displayName,
		AdminRoleID: strings.TrimSpace(adminRoleID),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   account,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Issuer:    "caipiao-backend",
		},
	})
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return TokenResult{}, fmt.Errorf("sign token: %w", err)
	}
	return TokenResult{
		AccessToken: signed,
		ExpiresAt:   expiresAt,
		Account:     account,
		DisplayName: displayName,
		RoleID:      adminRoleID,
	}, nil
}

func (s *Service) ParseBearer(tokenString string) (Claims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return Claims{}, ErrInvalidCredentials
	}
	parsed, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return Claims{}, ErrInvalidCredentials
	}
	c, ok := parsed.Claims.(*claims)
	if !ok {
		return Claims{}, ErrInvalidCredentials
	}
	return *c, nil
}
