package guaji

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// httpProxyFunc 解析顺序：GUAJI_HTTP_PROXY / HTTPS_PROXY 环境变量 → 系统代理（Windows IE）。
// 本机 V6 常需走 Clash 等本地代理（如 127.0.0.1:7897）；仅 ProxyFromEnvironment 会直连超时。
func httpProxyFunc() func(*http.Request) (*url.URL, error) {
	explicit := firstEnvNonEmpty(
		os.Getenv("GUAJI_HTTP_PROXY"),
		os.Getenv("HTTPS_PROXY"),
		os.Getenv("https_proxy"),
		os.Getenv("HTTP_PROXY"),
		os.Getenv("http_proxy"),
	)
	var fixed *url.URL
	if explicit != "" {
		if u, err := url.Parse(strings.TrimSpace(explicit)); err == nil && u.Host != "" {
			fixed = u
		}
	}
	return func(req *http.Request) (*url.URL, error) {
		if fixed != nil {
			return fixed, nil
		}
		if u, err := http.ProxyFromEnvironment(req); u != nil || err != nil {
			return u, err
		}
		return systemHTTPProxy(req)
	}
}

func firstEnvNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
