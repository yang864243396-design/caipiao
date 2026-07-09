package middleware

import "strings"

// IsAllowedOrigin 校验 CORS / WebSocket Origin。
// 除显式白名单外，允许与请求 Host 同源的 Origin（生产反代前后端同域时常见）。
func IsAllowedOrigin(origin, host string, allowedOrigins []string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	for _, o := range allowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	for _, scheme := range []string{"https://", "http://"} {
		if origin == scheme+host {
			return true
		}
	}
	return false
}
