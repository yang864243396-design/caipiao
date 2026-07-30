package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// 走库的登录分支。这里最要紧的一条是「状态不对的账号，即使密码正确也必须登不进」——
// 风控冻结一个账号后如果他还能拿到 Token，冻结就是摆设。

const dbTestPassword = "S3cret-For-Test"

func newDBService(t *testing.T) (*Service, *db.Pool) {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL 未配置")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(pool.Close)

	// 固定测试密钥，避免依赖环境里的 JWT_SECRET
	cfg.JWTSecret = testSecret
	cfg.TokenTTL = time.Hour
	return NewService(cfg, pool), pool
}

func hashFor(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(h)
}

// seedMember 建一个临时会员，测试结束即删。
func seedMember(t *testing.T, pool *db.Pool, status string) string {
	t.Helper()
	account := fmt.Sprintf("authtest%d", time.Now().UnixNano()%1e10)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO members (account, password_hash, display_name, status)
		VALUES ($1, $2, $3, $4)`,
		account, hashFor(t, dbTestPassword), "鉴权测试会员", status)
	if err != nil {
		t.Fatalf("造会员: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM members WHERE account = $1`, account)
	})
	return account
}

func TestLoginClientDB(t *testing.T) {
	svc, pool := newDBService(t)

	active := seedMember(t, pool, "active")
	frozen := seedMember(t, pool, "frozen")

	t.Run("正常账号密码正确", func(t *testing.T) {
		res, err := svc.LoginClient(active, dbTestPassword)
		if err != nil {
			t.Fatalf("应登录成功: %v", err)
		}
		if res.Account != active {
			t.Errorf("Account = %q，期望 %q", res.Account, active)
		}
		c, err := svc.ParseBearer(res.AccessToken)
		if err != nil {
			t.Fatalf("签发的 Token 无法解析: %v", err)
		}
		if c.Role != RoleClient {
			t.Errorf("Role = %q，期望 client", c.Role)
		}
	})

	t.Run("正常账号密码错", func(t *testing.T) {
		if _, err := svc.LoginClient(active, "wrong-password"); err != ErrInvalidCredentials {
			t.Errorf("err = %v，期望 ErrInvalidCredentials", err)
		}
	})

	t.Run("冻结账号密码正确也不得登入", func(t *testing.T) {
		res, err := svc.LoginClient(frozen, dbTestPassword)
		if err != ErrAccountFrozen {
			t.Fatalf("err = %v，期望 ErrAccountFrozen", err)
		}
		if res.AccessToken != "" {
			t.Error("冻结账号拿到了 Token——冻结形同虚设")
		}
	})

	t.Run("账号不存在", func(t *testing.T) {
		if _, err := svc.LoginClient("no_such_account_zzz", dbTestPassword); err != ErrInvalidCredentials {
			t.Errorf("err = %v，期望 ErrInvalidCredentials", err)
		}
	})

	t.Run("不存在与密码错返回同一个错误", func(t *testing.T) {
		// 两者必须不可区分，否则可以用错误信息枚举出哪些账号真实存在
		_, errNoUser := svc.LoginClient("no_such_account_zzz", dbTestPassword)
		_, errBadPw := svc.LoginClient(active, "wrong-password")
		if errNoUser != errBadPw {
			t.Errorf("账号不存在返回 %v，密码错返回 %v，两者可区分即可枚举账号", errNoUser, errBadPw)
		}
	})
}

// TestLoginClientDBTouchesLastLogin 登录成功要更新最近登录时间。
func TestLoginClientDBTouchesLastLogin(t *testing.T) {
	svc, pool := newDBService(t)
	account := seedMember(t, pool, "active")

	var before *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT last_login_at FROM members WHERE account = $1`, account).Scan(&before); err != nil {
		t.Fatalf("读取 last_login_at: %v", err)
	}
	if before != nil {
		t.Fatalf("新建会员的 last_login_at 应为 NULL，实际 %v", before)
	}

	if _, err := svc.LoginClient(account, dbTestPassword); err != nil {
		t.Fatalf("登录: %v", err)
	}

	var after *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT last_login_at FROM members WHERE account = $1`, account).Scan(&after); err != nil {
		t.Fatalf("重读 last_login_at: %v", err)
	}
	if after == nil {
		t.Error("登录成功后 last_login_at 仍为 NULL")
	}
}

// seedAdmin 建一个临时管理员账号，复用库里已有的角色以满足外键。
func seedAdmin(t *testing.T, pool *db.Pool, status string) string {
	t.Helper()
	var roleID string
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM admin_roles ORDER BY id LIMIT 1`).Scan(&roleID); err != nil {
		t.Skipf("库中没有可用的 admin_roles: %v", err)
	}
	account := fmt.Sprintf("authadm%d", time.Now().UnixNano()%1e10)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO admin_users (account, password_hash, display_name, role_id, status)
		VALUES ($1, $2, $3, $4, $5)`,
		account, hashFor(t, dbTestPassword), "鉴权测试管理员", roleID, status)
	if err != nil {
		t.Fatalf("造管理员: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM admin_users WHERE account = $1`, account)
	})
	return account
}

func TestLoginAdminDB(t *testing.T) {
	svc, pool := newDBService(t)

	active := seedAdmin(t, pool, "active")
	disabled := seedAdmin(t, pool, "disabled")

	t.Run("正常管理员", func(t *testing.T) {
		res, err := svc.LoginAdmin(active, dbTestPassword)
		if err != nil {
			t.Fatalf("应登录成功: %v", err)
		}
		c, err := svc.ParseBearer(res.AccessToken)
		if err != nil {
			t.Fatalf("解析 Token: %v", err)
		}
		if c.Role != RoleAdmin {
			t.Errorf("Role = %q，期望 admin", c.Role)
		}
		if c.AdminRoleID == "" {
			t.Error("管理员 Token 未带 adminRoleId，权限判定将无据可依")
		}
	})

	t.Run("已停用管理员密码正确也不得登入", func(t *testing.T) {
		res, err := svc.LoginAdmin(disabled, dbTestPassword)
		if err != ErrInvalidCredentials {
			t.Fatalf("err = %v，期望 ErrInvalidCredentials", err)
		}
		if res.AccessToken != "" {
			t.Error("停用管理员拿到了 Token")
		}
	})

	t.Run("密码错", func(t *testing.T) {
		if _, err := svc.LoginAdmin(active, "wrong-password"); err != ErrInvalidCredentials {
			t.Errorf("err = %v，期望 ErrInvalidCredentials", err)
		}
	})

	t.Run("会员账号不能从管理端登入", func(t *testing.T) {
		memberAccount := seedMember(t, pool, "active")
		if _, err := svc.LoginAdmin(memberAccount, dbTestPassword); err != ErrInvalidCredentials {
			t.Errorf("会员账号登管理端 err = %v，应被拒", err)
		}
	})
}
