package guaji

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds third-party Guaji platform connection settings (T0).
type Config struct {
	Enabled        bool
	HTTPBase       string
	AuthBase       string
	WSBase         string
	WSPath         string
	Origin         string
	Referer        string
	IsAI           bool
	HTTPTimeout    time.Duration
	CredentialsKey string
	TestUsername   string
	TestPassword   string
}

const defaultGuajiHTTPBase = "https://www.v6hs1.com"
const defaultGuajiWSBase = "wss://www.v6hs1.com"

func LoadConfigFromEnv() Config {
	timeoutSec := envInt("GUAJI_HTTP_TIMEOUT_SEC", 30)
	httpBase := trimSlash(env("GUAJI_HTTP_BASE", defaultGuajiHTTPBase))
	authBase := trimSlash(env("GUAJI_AUTH_BASE", ""))
	if authBase == "" {
		authBase = defaultAuthBase(httpBase)
	}
	wsBase := trimSlash(env("GUAJI_WS_BASE", defaultGuajiWSBase))
	origin := env("GUAJI_ORIGIN", httpBase)
	referer := env("GUAJI_REFERER", origin+"/")
	return Config{
		Enabled:        envBool("GUAJI_ENABLED", false),
		HTTPBase:       httpBase,
		AuthBase:       authBase,
		WSBase:         wsBase,
		WSPath:         env("GUAJI_WS_PATH", "/ws"),
		Origin:         origin,
		Referer:        referer,
		IsAI:           envBool("GUAJI_IS_AI", true),
		HTTPTimeout:    time.Duration(timeoutSec) * time.Second,
		CredentialsKey: strings.TrimSpace(os.Getenv("GUAJI_CREDENTIALS_KEY")),
		TestUsername:   env("GUAJI_TEST_USERNAME", ""),
		TestPassword:   env("GUAJI_TEST_PASSWORD", ""),
	}
}

func defaultAuthBase(httpBase string) string {
	switch {
	case strings.Contains(httpBase, "hash.iyes.dev"):
		return "https://hash-game-admin.iyes.dev"
	case strings.Contains(httpBase, "v6hs1.com"):
		return trimSlash(httpBase)
	case strings.Contains(httpBase, "5rf9q.com"):
		// 旧正式域名 auth 须运维显式配置 GUAJI_AUTH_BASE
		return ""
	default:
		return trimSlash(httpBase)
	}
}

func (c Config) Valid() error {
	if !c.Enabled {
		return nil
	}
	if c.HTTPBase == "" {
		return ErrMisconfigured("GUAJI_HTTP_BASE 未配置")
	}
	if c.AuthBase == "" {
		return ErrMisconfigured("GUAJI_AUTH_BASE 未配置（正式环境必填）")
	}
	if c.WSBase == "" {
		return ErrMisconfigured("GUAJI_WS_BASE 未配置")
	}
	return nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func trimSlash(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}
