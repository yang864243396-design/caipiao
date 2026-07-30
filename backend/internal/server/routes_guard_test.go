package server_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// 这一层守的是越权。registerRoutes 里两百多条路由，每条都靠人手挂
// clientAuth 或 adminAuth，漏挂一条照样编译通过、照样能跑，只是那个接口
// 对所有人敞开——不会报错，不会写脏数据，只有被人用了才知道。
//
// 所以这里不测「接口能不能用」，测的是「注册表本身是否自洽」：
// 从源码里把每条路由连同它的守卫一起抽出来，按路径前缀比对该挂哪个。
// 新增路由时忘了挂守卫，这里会直接红。

// route 一条注册的路由及其守卫。guard 为空表示没有任何鉴权包装。
type route struct {
	method string
	path   string
	guard  string // "", "clientAuth", "adminAuth"
	line   int
}

func (r route) String() string {
	g := r.guard
	if g == "" {
		g = "(无守卫)"
	}
	return fmt.Sprintf("%s %s [%s] server.go:%d", r.method, r.path, g, r.line)
}

// 有意公开的路由。除这些之外，/client/ 一律要 clientAuth，/admin/ 一律要 adminAuth。
var (
	publicExact = map[string]bool{
		"/health":            true,
		"/client/auth/login": true, // 登录接口本身不能要求已登录
		"/admin/auth/login":  true,
	}
	publicPrefixes = []string{
		"/public/", // 未登录也要能看的落地页内容
		"/ws/",     // WS 在 ws.Server 内部按 token 自行校验
	}
)

func isIntentionallyPublic(path string) bool {
	if publicExact[path] {
		return true
	}
	for _, p := range publicPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// wantGuard 按路径前缀给出该挂的守卫。
func wantGuard(path string) string {
	switch {
	case isIntentionallyPublic(path):
		return ""
	case strings.HasPrefix(path, "/client/"):
		return "clientAuth"
	case strings.HasPrefix(path, "/admin/"):
		return "adminAuth"
	}
	return ""
}

// parseRoutes 从 server.go 抽出 registerRoutes 中注册的全部路由。
func parseRoutes(t *testing.T) []route {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "server.go", nil, 0)
	if err != nil {
		t.Fatalf("解析 server.go: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if ok && d.Name.Name == "registerRoutes" {
			fn = d
			break
		}
	}
	if fn == nil {
		t.Fatal("server.go 里找不到 registerRoutes——本测试的抽取逻辑已与实现脱节")
	}

	var routes []route
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok || recv.Name != "api" {
			return true
		}
		if sel.Sel.Name != "Handle" && sel.Sel.Name != "HandleFunc" {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		pattern, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		method, path := splitPattern(pattern)

		r := route{method: method, path: path, line: fset.Position(call.Pos()).Line}
		// api.Handle 的第二个参数若是 clientAuth(...) / adminAuth(...)，即为守卫
		if guardCall, ok := call.Args[1].(*ast.CallExpr); ok {
			if id, ok := guardCall.Fun.(*ast.Ident); ok {
				switch id.Name {
				case "clientAuth", "adminAuth":
					r.guard = id.Name
				}
			}
		}
		routes = append(routes, r)
		return true
	})
	return routes
}

func splitPattern(pattern string) (method, path string) {
	if i := strings.IndexByte(pattern, ' '); i > 0 {
		return pattern[:i], strings.TrimSpace(pattern[i+1:])
	}
	return "", pattern
}

// TestRouteGuardsMatchPathPrefix 每条路由的守卫必须与其路径前缀相符。
func TestRouteGuardsMatchPathPrefix(t *testing.T) {
	routes := parseRoutes(t)

	// 抽取逻辑一旦失效就会「零条路由全部通过」，这里兜住这种空转。
	// 当前实现是 148 条，取 140 作下界：容得下少量删减，抽取真断了也拦得住。
	if len(routes) < 140 {
		t.Fatalf("只抽到 %d 条路由，实现里远不止这些——抽取逻辑可能已失效", len(routes))
	}
	t.Logf("共检查 %d 条路由", len(routes))

	for _, r := range routes {
		want := wantGuard(r.path)
		if r.guard == want {
			continue
		}
		switch {
		case want != "" && r.guard == "":
			t.Errorf("%s 未挂任何鉴权守卫，应为 %s——该接口当前对所有人敞开", r, want)
		case want == "" && r.guard != "":
			t.Errorf("%s 挂了 %s，但按路径它应是公开接口", r, r.guard)
		default:
			t.Errorf("%s 挂错守卫，应为 %s", r, want)
		}
	}
}

// TestNoDuplicateRoutes 同一 方法+路径 只应注册一次。
// 重复注册在 Go 1.22 的 ServeMux 上会 panic，但那要等到进程启动才暴露。
func TestNoDuplicateRoutes(t *testing.T) {
	seen := map[string]route{}
	for _, r := range parseRoutes(t) {
		key := r.method + " " + r.path
		if prev, dup := seen[key]; dup {
			t.Errorf("路由重复注册: %s（另一处在 server.go:%d）", r, prev.line)
			continue
		}
		seen[key] = r
	}
}

// TestPublicRoutesAreDeliberate 公开路由数量不多，逐条列出来。
// 新增一个免鉴权接口时，必须同步改这份名单，避免「悄悄多一个公开接口」。
func TestPublicRoutesAreDeliberate(t *testing.T) {
	want := map[string]bool{
		"GET /health":                            true,
		"GET /public/maintenance":                true,
		"GET /public/lobby-slots":                true,
		"GET /public/banners":                    true,
		"GET /public/site-brand":                 true,
		"GET /public/cms-uploads/{filename}":     true,
		"GET /public/lotteries":                  true,
		"GET /public/lotteries/{code}/status":    true,
		"GET /public/lotteries/{code}/play-tree": true,
		"POST /client/auth/login":                true,
		"POST /admin/auth/login":                 true,
		"GET /ws/public":                         true,
		"GET /ws/client":                         true,
		"GET /ws/admin":                          true,
	}
	got := map[string]bool{}
	for _, r := range parseRoutes(t) {
		if r.guard == "" {
			got[r.method+" "+r.path] = true
		}
	}
	for k := range got {
		if !want[k] {
			t.Errorf("新增了未在名单内的公开接口: %s——确认这是有意为之后再加进名单", k)
		}
	}
	for k := range want {
		if !got[k] {
			t.Errorf("名单里的公开接口 %s 已不存在或已改为鉴权，请更新名单", k)
		}
	}
}
