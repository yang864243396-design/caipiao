package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"caipiao/backend/internal/guaji"
)

type Config struct {
	Port              string
	Env               string
	JWTSecret         string
	CORSOrigins       []string
	ClientDemoAccount string
	ClientDemoPass    string
	AdminDemoAccount  string
	AdminDemoPass     string
	TokenTTL          time.Duration
	DatabaseURL       string
	DBRequired        bool
	DBMaxConns        int
	DBMinConns        int
	SchemeWorkerEnabled bool
	SchemeWorkerTickSec int
	WSEnabled           bool
	WSAuthViaQuery      bool
	Guaji               guaji.Config
	CMSUploadDir        string
}

func Load() Config {
	ttlHours := envInt("TOKEN_TTL_HOURS", 8)
	dbRequired := envBool("DB_REQUIRED", true)
	return Config{
		Port:              env("PORT", "8080"),
		Env:               env("ENV", "development"),
		JWTSecret:         env("JWT_SECRET", "dev-change-me-in-production"),
		CORSOrigins:       splitCSV(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:5174,http://127.0.0.1:5173,http://127.0.0.1:5174")),
		ClientDemoAccount: env("CLIENT_DEMO_ACCOUNT", "vs8888"),
		ClientDemoPass:    env("CLIENT_DEMO_PASSWORD", "vs8888"),
		AdminDemoAccount:  env("ADMIN_DEMO_ACCOUNT", "admin"),
		AdminDemoPass:     env("ADMIN_DEMO_PASSWORD", "admin123"),
		TokenTTL:            time.Duration(ttlHours) * time.Hour,
		DatabaseURL:         buildDatabaseURL(),
		DBRequired:          dbRequired,
		DBMaxConns:          envInt("DB_MAX_CONNS", 25),
		DBMinConns:          envInt("DB_MIN_CONNS", 2),
		SchemeWorkerEnabled: envBool("SCHEME_WORKER_ENABLED", true),
		SchemeWorkerTickSec: envInt("SCHEME_WORKER_TICK_SEC", 1),
		WSEnabled:           envBool("WS_ENABLED", true),
		WSAuthViaQuery:      envBool("WS_AUTH_VIA_QUERY", true),
		Guaji:               guaji.LoadConfigFromEnv(),
		CMSUploadDir:        env("CMS_UPLOAD_DIR", "./data/uploads/cms"),
	}
}

func buildDatabaseURL() string {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw
	}

	host := env("DB_HOST", "")
	if host == "" {
		return ""
	}

	user := env("DB_USER", "caipiaoapp")
	pass := os.Getenv("DB_PASSWORD")
	name := env("DB_NAME", "caipiao")
	port := env("DB_PORT", "5432")
	sslmode := env("DB_SSLMODE", "disable")

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, pass),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String()
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

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
