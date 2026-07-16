package guaji

import (
	"net/http"
	"testing"
)

func TestHTTPProxyFunc_readsWindowsOrEnv(t *testing.T) {
	fn := httpProxyFunc()
	req, err := http.NewRequest(http.MethodGet, "https://www.v6hs1.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := fn(req)
	if err != nil {
		t.Fatalf("proxy err: %v", err)
	}
	// 本机若开了 IE 代理应非空；无代理则跳过。
	if u != nil {
		t.Logf("proxy=%s", u.String())
		if u.Host == "" {
			t.Fatal("empty proxy host")
		}
	}
}
