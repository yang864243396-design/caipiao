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
	// 断言的是「零绑定 ⇒ 无可用授权」这个不变量，不是 demo 账号当前的状态。
	// 该账号一旦授权过就长期绑定着（真实投注要用），直接断言「无授权」只在全新库上成立。
	if st.BindingCount > 0 {
		t.Skipf("demo 账号已有 %d 个绑定，无法验证空绑定分支", st.BindingCount)
	}
	if st.HasActiveGuajiAuth {
		t.Fatalf("零绑定却报告有可用授权：%+v", st)
	}
}
