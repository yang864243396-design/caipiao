package accountsvc_test

import (
	"context"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
)

func TestAuthStatusEmptyBindings(t *testing.T) {
	cfg := config.Load()
	_ = godotenv.Load("../../.env")
	cfg = config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	defer pool.Close()

	svc := accountsvc.NewService(pool, guaji.NewClient(cfg.Guaji), cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	st, err := svc.AuthStatus(context.Background(), cfg.ClientDemoAccount)
	if err != nil {
		t.Fatalf("AuthStatus: %v", err)
	}
	if st.HasActiveGuajiAuth {
		t.Fatalf("expected no active auth for demo account")
	}
}
