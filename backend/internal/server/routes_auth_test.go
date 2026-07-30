package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/config"
	"caipiao/backend/internal/server"
)

// 静态检查只能证明源码里「挂了」守卫，证明不了守卫真的会拒。
// 这一层把整个 Server 起起来，用上一层抽出的同一份路由表逐条打真实请求：
// 不带 Token 必须 401，拿错角色的 Token 必须 403。
//
// 这里刻意不连数据库：鉴权在中间件里完成，被拒的请求根本到不了 handler，
// 所以 pool 为 nil 不影响结论，反而让测试跑得快且不碰生产库。

const (
	testJWTSecret = "route-guard-test-secret"
	demoClient    = "route_test_client"
	demoClientPw  = "route_test_client_pw"
	demoAdmin     = "route_test_admin"
	demoAdminPw   = "route_test_admin_pw"
)

func testServerConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		Port:              "0",
		Env:               "test",
		JWTSecret:         testJWTSecret,
		TokenTTL:          time.Hour,
		ClientDemoAccount: demoClient,
		ClientDemoPass:    demoClientPw,
		AdminDemoAccount:  demoAdmin,
		AdminDemoPass:     demoAdminPw,
		// 不连库、不起 worker、不开 WS：本测试只关心鉴权中间件
		DatabaseURL:         "",
		DBRequired:          false,
		SchemeWorkerEnabled: false,
		WSEnabled:           false,
		CMSUploadDir:        t.TempDir(),
	}
}

type authEnv struct {
	ts          *httptest.Server
	clientToken string
	adminToken  string
}

func newAuthEnv(t *testing.T) *authEnv {
	t.Helper()
	cfg := testServerConfig(t)

	srv, err := server.New(cfg)
	if err != nil {
		t.Fatalf("起 Server: %v", err)
	}
	t.Cleanup(srv.Close)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	// 无库时走 env 演示账号分支，签出的 Token 与线上同一套签发逻辑
	authSvc := auth.NewService(cfg, nil)
	c, err := authSvc.LoginClient(demoClient, demoClientPw)
	if err != nil {
		t.Fatalf("签会员 Token: %v", err)
	}
	a, err := authSvc.LoginAdmin(demoAdmin, demoAdminPw)
	if err != nil {
		t.Fatalf("签管理员 Token: %v", err)
	}
	return &authEnv{ts: ts, clientToken: c.AccessToken, adminToken: a.AccessToken}
}

// do 发一个请求；token 为空表示不带 Authorization 头。
func (e *authEnv) do(t *testing.T, method, path, token string) int {
	t.Helper()
	req, err := http.NewRequest(method, e.ts.URL+"/api/v1"+path, nil)
	if err != nil {
		t.Fatalf("构造请求 %s %s: %v", method, path, err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := e.ts.Client().Do(req)
	if err != nil {
		t.Fatalf("请求 %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// fillPathParams 把 {memberId} 之类的占位段替换成具体值，否则路由匹配不上。
func fillPathParams(path string) string {
	if !strings.ContainsRune(path, '{') {
		return path
	}
	segs := strings.Split(path, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			segs[i] = "1"
		}
	}
	return strings.Join(segs, "/")
}

// guardedRoutes 按「路径前缀要求」筛出应当鉴权的路由。
//
// 这里刻意用 wantGuard(r.path)（策略）而不是 r.guard（实现）来判定：
// 若照着实现走，源码把 /admin/xxx 挂成 clientAuth 时，本测试会顺从地
// 用会员 Token 去试并得到「符合预期」的结果，挂错反而测不出来；
// 守卫被整个删掉时该路由更会直接从待测集合里消失。
func guardedRoutes(t *testing.T) []route {
	t.Helper()
	var out []route
	for _, r := range parseRoutes(t) {
		if r.method == "" || wantGuard(r.path) == "" {
			continue
		}
		out = append(out, r)
	}
	// 当前 148 条路由中 134 条需鉴权、14 条有意公开；取 125 作下界兜住抽取失效。
	if len(out) < 125 {
		t.Fatalf("只有 %d 条待测路由，抽取逻辑可能已失效", len(out))
	}
	return out
}

// TestGuardedRoutesRejectAnonymous 不带 Token 访问任何受保护接口都必须 401。
func TestGuardedRoutesRejectAnonymous(t *testing.T) {
	env := newAuthEnv(t)
	routes := guardedRoutes(t)
	t.Logf("逐条验证 %d 条受保护路由", len(routes))

	for _, r := range routes {
		got := env.do(t, r.method, fillPathParams(r.path), "")
		if got != http.StatusUnauthorized {
			t.Errorf("%s 匿名访问返回 %d，应为 401——该接口未被真正保护", r, got)
		}
	}
}

// TestGuardedRoutesRejectWrongRole 拿会员 Token 访问管理端接口必须 403，反之亦然。
// 这条抓的是「挂了守卫但挂成了另一种角色」。
func TestGuardedRoutesRejectWrongRole(t *testing.T) {
	env := newAuthEnv(t)

	for _, r := range guardedRoutes(t) {
		// 用与该路由「按路径应有的角色」相反的角色去访问
		var wrongToken, desc string
		switch wantGuard(r.path) {
		case "clientAuth":
			wrongToken, desc = env.adminToken, "管理员 Token 访问会员接口"
		case "adminAuth":
			wrongToken, desc = env.clientToken, "会员 Token 访问管理端接口"
		default:
			continue
		}
		got := env.do(t, r.method, fillPathParams(r.path), wrongToken)
		if got != http.StatusForbidden {
			t.Errorf("%s %s 返回 %d，应为 403", r, desc, got)
		}
	}
}

// TestGuardedRoutesAcceptRightRole 角色正确时必须能过鉴权这一关。
//
// 只断言「不是 401/403」：本测试没连库，请求穿过中间件后 handler 多半会
// 因为 pool 为 nil 而报错，那是 500，不影响这里要证明的事——守卫没有误杀。
func TestGuardedRoutesAcceptRightRole(t *testing.T) {
	env := newAuthEnv(t)

	for _, r := range guardedRoutes(t) {
		var token string
		switch wantGuard(r.path) {
		case "clientAuth":
			token = env.clientToken
		case "adminAuth":
			token = env.adminToken
		default:
			continue
		}
		got := env.do(t, r.method, fillPathParams(r.path), token)
		if got == http.StatusUnauthorized || got == http.StatusForbidden {
			t.Errorf("%s 用正确角色访问却返回 %d——守卫误杀", r, got)
		}
	}
}

// TestGuardedRoutesRejectBadToken 伪造或过期的 Token 一律 401。
func TestGuardedRoutesRejectBadToken(t *testing.T) {
	env := newAuthEnv(t)

	// 用另一把密钥签出的、内容完全合法的管理员 Token
	forgedCfg := testServerConfig(t)
	forgedCfg.JWTSecret = "a-completely-different-secret"
	forged, err := auth.NewService(forgedCfg, nil).LoginAdmin(demoAdmin, demoAdminPw)
	if err != nil {
		t.Fatalf("签伪造 Token: %v", err)
	}

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "别的密钥签的", token: forged.AccessToken},
		{name: "不是 JWT", token: "garbage-token"},
		{name: "签名被改", token: env.adminToken[:len(env.adminToken)-4] + "AAAA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// 挑一条管理端接口即可，中间件对所有路由是同一个
			if got := env.do(t, "GET", "/admin/members", tc.token); got != http.StatusUnauthorized {
				t.Errorf("返回 %d，应为 401", got)
			}
		})
	}
}

// TestAuthHeaderFormats Authorization 头的各种写法。
func TestAuthHeaderFormats(t *testing.T) {
	env := newAuthEnv(t)
	const path = "/client/member/profile"

	for _, tc := range []struct {
		name   string
		header string
		want   int
	}{
		{name: "缺 Bearer 前缀", header: env.clientToken, want: http.StatusUnauthorized},
		{name: "小写 bearer", header: "bearer " + env.clientToken, want: http.StatusUnauthorized},
		{name: "只有 Bearer", header: "Bearer ", want: http.StatusUnauthorized},
		{name: "Basic 认证", header: "Basic dXNlcjpwYXNz", want: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", env.ts.URL+"/api/v1"+path, nil)
			if err != nil {
				t.Fatalf("构造请求: %v", err)
			}
			req.Header.Set("Authorization", tc.header)
			resp, err := env.ts.Client().Do(req)
			if err != nil {
				t.Fatalf("请求: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("返回 %d，期望 %d", resp.StatusCode, tc.want)
			}
		})
	}
}
