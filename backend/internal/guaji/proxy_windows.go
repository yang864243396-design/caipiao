//go:build windows

package guaji

import (
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func systemHTTPProxy(req *http.Request) (*url.URL, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return nil, nil
	}
	defer k.Close()

	enable, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil || enable == 0 {
		return nil, nil
	}
	server, _, err := k.GetStringValue("ProxyServer")
	server = strings.TrimSpace(server)
	if err != nil || server == "" {
		return nil, nil
	}
	// 可能是 "127.0.0.1:7897" 或 "http=...;https=..."
	proxyAddr := server
	if strings.Contains(server, "=") {
		parts := strings.Split(server, ";")
		var httpP, httpsP, fallback string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			kv := strings.SplitN(p, "=", 2)
			if len(kv) != 2 {
				fallback = p
				continue
			}
			switch strings.ToLower(strings.TrimSpace(kv[0])) {
			case "https":
				httpsP = strings.TrimSpace(kv[1])
			case "http":
				httpP = strings.TrimSpace(kv[1])
			}
		}
		if req != nil && req.URL != nil && req.URL.Scheme == "http" && httpP != "" {
			proxyAddr = httpP
		} else if httpsP != "" {
			proxyAddr = httpsP
		} else if httpP != "" {
			proxyAddr = httpP
		} else if fallback != "" {
			proxyAddr = fallback
		}
	}
	if proxyAddr == "" {
		return nil, nil
	}
	if !strings.Contains(proxyAddr, "://") {
		proxyAddr = "http://" + proxyAddr
	}
	u, err := url.Parse(proxyAddr)
	if err != nil || u.Host == "" {
		return nil, nil
	}
	return u, nil
}
