package auth

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"caipiao/backend/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// 这一层错了不是脏数据，是越权：签发的 Token 谁都能造、过期的还认、
// 别人签的也认，任何一条都是直接的安全事故。所以这里覆盖的重点不是
// 「正常登录能拿到 Token」，而是「不该通过的一定通不过」。

const (
	testSecret = "test-secret-please-do-not-reuse"
	otherKey   = "another-secret-entirely"
)

func testCfg() config.Config {
	return config.Config{
		JWTSecret:         testSecret,
		TokenTTL:          time.Hour,
		ClientDemoAccount: "demo_client",
		ClientDemoPass:    "demo_client_pw",
		AdminDemoAccount:  "demo_admin",
		AdminDemoPass:     "demo_admin_pw",
	}
}

// newEnvService 不接数据库的服务，走 env 演示账号分支。
func newEnvService(t *testing.T) *Service {
	t.Helper()
	return NewService(testCfg(), nil)
}

func TestLoginClientEnv(t *testing.T) {
	svc := newEnvService(t)
	for _, tc := range []struct {
		name        string
		account     string
		password    string
		wantErr     error
		wantAccount string
	}{
		{name: "账号密码正确", account: "demo_client", password: "demo_client_pw", wantAccount: "demo_client"},
		{name: "首尾空格应被裁剪", account: "  demo_client  ", password: "demo_client_pw", wantAccount: "demo_client"},
		{name: "密码错", account: "demo_client", password: "wrong", wantErr: ErrInvalidCredentials},
		{name: "账号错", account: "nobody", password: "demo_client_pw", wantErr: ErrInvalidCredentials},
		{name: "空账号", account: "", password: "demo_client_pw", wantErr: ErrInvalidCredentials},
		{name: "空密码", account: "demo_client", password: "", wantErr: ErrInvalidCredentials},
		{name: "纯空格账号", account: "   ", password: "demo_client_pw", wantErr: ErrInvalidCredentials},
		{name: "密码不做裁剪", account: "demo_client", password: " demo_client_pw ", wantErr: ErrInvalidCredentials},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.LoginClient(tc.account, tc.password)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("err = %v，期望 %v", err, tc.wantErr)
				}
				if got.AccessToken != "" {
					t.Error("失败时不应返回 Token")
				}
				return
			}
			if err != nil {
				t.Fatalf("登录失败: %v", err)
			}
			if got.Account != tc.wantAccount {
				t.Errorf("Account = %q，期望 %q", got.Account, tc.wantAccount)
			}
			if got.AccessToken == "" {
				t.Fatal("成功时应返回 Token")
			}
		})
	}
}

// TestLoginRoleSeparation 会员口令登不进管理端，反之亦然。
func TestLoginRoleSeparation(t *testing.T) {
	svc := newEnvService(t)
	if _, err := svc.LoginAdmin("demo_client", "demo_client_pw"); err != ErrInvalidCredentials {
		t.Errorf("会员凭据登管理端 err = %v，应为 ErrInvalidCredentials", err)
	}
	if _, err := svc.LoginClient("demo_admin", "demo_admin_pw"); err != ErrInvalidCredentials {
		t.Errorf("管理员凭据登会员端 err = %v，应为 ErrInvalidCredentials", err)
	}
}

// TestIssuedTokenCarriesRole 签发的 Token 必须带上正确的角色，
// 角色是 middleware.RequireRole 唯一的判据。
func TestIssuedTokenCarriesRole(t *testing.T) {
	svc := newEnvService(t)

	client, err := svc.LoginClient("demo_client", "demo_client_pw")
	if err != nil {
		t.Fatalf("client 登录: %v", err)
	}
	cc, err := svc.ParseBearer(client.AccessToken)
	if err != nil {
		t.Fatalf("解析 client Token: %v", err)
	}
	if cc.Role != RoleClient {
		t.Errorf("client Token 的 Role = %q，期望 %q", cc.Role, RoleClient)
	}
	if cc.Subject != "demo_client" {
		t.Errorf("Subject = %q，期望 demo_client", cc.Subject)
	}
	if cc.AdminRoleID != "" {
		t.Errorf("会员 Token 不应带 adminRoleId，实际 %q", cc.AdminRoleID)
	}

	admin, err := svc.LoginAdmin("demo_admin", "demo_admin_pw")
	if err != nil {
		t.Fatalf("admin 登录: %v", err)
	}
	ac, err := svc.ParseBearer(admin.AccessToken)
	if err != nil {
		t.Fatalf("解析 admin Token: %v", err)
	}
	if ac.Role != RoleAdmin {
		t.Errorf("admin Token 的 Role = %q，期望 %q", ac.Role, RoleAdmin)
	}
	if ac.AdminRoleID != "r_super" {
		t.Errorf("AdminRoleID = %q，期望 r_super", ac.AdminRoleID)
	}
	if admin.RoleID != "r_super" {
		t.Errorf("TokenResult.RoleID = %q，期望 r_super", admin.RoleID)
	}
}

func TestIssuedTokenExpiry(t *testing.T) {
	svc := newEnvService(t)
	before := time.Now().UTC()
	res, err := svc.LoginClient("demo_client", "demo_client_pw")
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	after := time.Now().UTC()

	// 过期时间应落在 [before+TTL, after+TTL] 内
	if res.ExpiresAt.Before(before.Add(time.Hour)) || res.ExpiresAt.After(after.Add(time.Hour)) {
		t.Errorf("ExpiresAt = %v，不在 TTL 窗口 [%v, %v] 内",
			res.ExpiresAt, before.Add(time.Hour), after.Add(time.Hour))
	}
	c, err := svc.ParseBearer(res.AccessToken)
	if err != nil {
		t.Fatalf("解析: %v", err)
	}
	if c.ExpiresAt == nil {
		t.Fatal("Token 未带 exp——不过期的 Token 等于永久凭据")
	}
	if c.Issuer != "caipiao-backend" {
		t.Errorf("Issuer = %q", c.Issuer)
	}
}

// TestParseBearerRejectsBadTokens 所有不该被接受的 Token 形态。
func TestParseBearerRejectsBadTokens(t *testing.T) {
	svc := newEnvService(t)
	good, err := svc.LoginClient("demo_client", "demo_client_pw")
	if err != nil {
		t.Fatalf("登录: %v", err)
	}

	// 用别的密钥签一个内容完全合法的 Token
	forged := signWith(t, otherKey, claims{
		Role: RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "demo_admin",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "caipiao-backend",
		},
	})
	// 用正确密钥签一个已过期的
	expired := signWith(t, testSecret, claims{
		Role: RoleClient,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "demo_client",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			Issuer:    "caipiao-backend",
		},
	})
	// 篡改签名段
	tampered := good.AccessToken[:len(good.AccessToken)-4] + "AAAA"
	// 改载荷（把 role 改成 admin）后保留原签名
	payloadSwapped := swapPayloadRole(t, good.AccessToken)

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "空串", token: ""},
		{name: "纯空格", token: "   "},
		{name: "非 JWT 结构", token: "not-a-token"},
		{name: "段数不足", token: "aaa.bbb"},
		{name: "别的密钥签的", token: forged},
		{name: "已过期", token: expired},
		{name: "签名被改", token: tampered},
		{name: "载荷被改角色", token: payloadSwapped},
		{name: "alg=none", token: noneAlgToken(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := svc.ParseBearer(tc.token)
			if err == nil {
				t.Fatalf("被接受了，解析出 role=%q subject=%q", c.Role, c.Subject)
			}
			if err != ErrInvalidCredentials {
				t.Errorf("err = %v，期望 ErrInvalidCredentials（不应泄漏内部细节）", err)
			}
		})
	}
}

// TestParseBearerTrimsSpace 带首尾空格的 Token 仍应可用，
// 前端拼 "Bearer " 时多一个空格不该导致登出。
func TestParseBearerTrimsSpace(t *testing.T) {
	svc := newEnvService(t)
	res, err := svc.LoginClient("demo_client", "demo_client_pw")
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	if _, err := svc.ParseBearer("  " + res.AccessToken + "  "); err != nil {
		t.Errorf("带空格的 Token 应被接受: %v", err)
	}
}

// TestTokenNotAcceptedAcrossSecrets 换了 JWT 密钥后，旧 Token 必须全部失效。
func TestTokenNotAcceptedAcrossSecrets(t *testing.T) {
	old := newEnvService(t)
	res, err := old.LoginClient("demo_client", "demo_client_pw")
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	cfg := testCfg()
	cfg.JWTSecret = otherKey
	rotated := NewService(cfg, nil)
	if _, err := rotated.ParseBearer(res.AccessToken); err != ErrInvalidCredentials {
		t.Errorf("轮换密钥后旧 Token 仍被接受，err = %v", err)
	}
}

func signWith(t *testing.T, secret string, c claims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("签名: %v", err)
	}
	return s
}

// noneAlgToken 构造 alg=none 的 Token，验证服务端不接受无签名算法。
func noneAlgToken(t *testing.T) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims{
		Role: RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "demo_admin",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("none 签名: %v", err)
	}
	return s
}

// swapPayloadRole 把载荷里的 client 换成 admin，签名段原样保留。
func swapPayloadRole(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("Token 结构异常: %q", token)
	}
	payload := decodeSegment(t, parts[1])
	swapped := strings.Replace(string(payload), `"role":"client"`, `"role":"admin"`, 1)
	if swapped == string(payload) {
		t.Fatalf("载荷中未找到 role=client：%s", payload)
	}
	return parts[0] + "." + encodeSegment(swapped) + "." + parts[2]
}

func decodeSegment(t *testing.T, seg string) []byte {
	t.Helper()
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		t.Fatalf("解码载荷: %v", err)
	}
	return b
}

func encodeSegment(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
