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
	Port                         string
	Env                          string
	JWTSecret                    string
	CORSOrigins                  []string
	ClientDemoAccount            string
	ClientDemoPass               string
	AdminDemoAccount             string
	AdminDemoPass                string
	TokenTTL                     time.Duration
	DatabaseURL                  string
	DBRequired                   bool
	DBMaxConns                   int
	DBMinConns                   int
	SchemeWorkerEnabled          bool
	SchemeWorkerTickSec          int
	SchemeWorkerConcurrency      int
	SchemeWorkerPlaceConcurrency int
	SchemeBettingMode            string
	SchemeBettingLotteries       []string
	SchemeBettingShards          []int32
	SchemeBettingShardCount      int
	SchemeBettingDispatcherOwner string
	SchemeBettingBatch           int
	SchemeBettingConcurrency     int
	SchemeBettingLease           time.Duration
	SchemeBettingPoll            time.Duration
	SchemeEventBusEnabled        bool
	SchemeEventStream            string
	SchemeEventReplicas          int
	SchemeEventMaxAge            time.Duration
	WSEnabled                    bool
	WSAuthViaQuery               bool
	CloudRealtimeEnabled         bool
	CloudRealtimeBus             string
	NATSURL                      string
	NATSUser                     string
	NATSPassword                 string
	NATSToken                    string
	NATSCredentialsFile          string
	NATSSubjectPrefix            string
	CloudRealtimeCoalesce        time.Duration
	CloudStatsCoalesce           time.Duration
	CloudReconcileInterval       time.Duration
	CloudReconcileBatch          int
	Guaji                        guaji.Config
	CMSUploadDir                 string
}

func Load() Config {
	ttlHours := envInt("TOKEN_TTL_HOURS", 8)
	dbRequired := envBool("DB_REQUIRED", true)
	return Config{
		Port:                         env("PORT", "8080"),
		Env:                          env("ENV", "development"),
		JWTSecret:                    env("JWT_SECRET", "dev-change-me-in-production"),
		CORSOrigins:                  splitCSV(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:5174,http://127.0.0.1:5173,http://127.0.0.1:5174")),
		ClientDemoAccount:            env("CLIENT_DEMO_ACCOUNT", "vs8888"),
		ClientDemoPass:               env("CLIENT_DEMO_PASSWORD", "vs8888"),
		AdminDemoAccount:             env("ADMIN_DEMO_ACCOUNT", "admin"),
		AdminDemoPass:                env("ADMIN_DEMO_PASSWORD", "admin123"),
		TokenTTL:                     time.Duration(ttlHours) * time.Hour,
		DatabaseURL:                  buildDatabaseURL(),
		DBRequired:                   dbRequired,
		DBMaxConns:                   envInt("DB_MAX_CONNS", 25),
		DBMinConns:                   envInt("DB_MIN_CONNS", 2),
		SchemeWorkerEnabled:          envBool("SCHEME_WORKER_ENABLED", true),
		SchemeWorkerTickSec:          envInt("SCHEME_WORKER_TICK_SEC", 1),
		SchemeWorkerConcurrency:      envInt("SCHEME_WORKER_CONCURRENCY", 32),
		SchemeWorkerPlaceConcurrency: envInt("SCHEME_WORKER_PLACE_CONCURRENCY", 16),
		SchemeBettingMode:            strings.ToLower(env("SCHEME_BETTING_MODE", "shadow")),
		SchemeBettingLotteries:       splitCSV(env("SCHEME_BETTING_LOTTERIES", "")),
		SchemeBettingShards:          splitInt32CSV(env("SCHEME_BETTING_SHARDS", "")),
		SchemeBettingShardCount:      envInt("SCHEME_BETTING_SHARD_COUNT", 64),
		SchemeBettingDispatcherOwner: env("SCHEME_BETTING_DISPATCHER_OWNER", ""),
		SchemeBettingBatch:           envInt("SCHEME_BETTING_BATCH", 32),
		SchemeBettingConcurrency:     envInt("SCHEME_BETTING_CONCURRENCY", 8),
		SchemeBettingLease:           envDurationMS("SCHEME_BETTING_LEASE_MS", 5*time.Second),
		SchemeBettingPoll:            envDurationMS("SCHEME_BETTING_POLL_MS", 100*time.Millisecond),
		WSEnabled:                    envBool("WS_ENABLED", true),
		SchemeEventBusEnabled:        envBool("SCHEME_EVENT_BUS_ENABLED", false),
		SchemeEventStream:            env("SCHEME_EVENT_STREAM", "SCHEME_EVENTS"),
		SchemeEventReplicas:          envInt("SCHEME_EVENT_REPLICAS", 1),
		SchemeEventMaxAge:            envDurationMS("SCHEME_EVENT_MAX_AGE_MS", 72*time.Hour),
		WSAuthViaQuery:               envBool("WS_AUTH_VIA_QUERY", true),
		CloudRealtimeEnabled:         envBool("CLOUD_REALTIME_ENABLED", true),
		CloudRealtimeBus:             env("CLOUD_REALTIME_BUS", "nats"),
		NATSURL:                      env("NATS_URL", "nats://127.0.0.1:4222"),
		NATSUser:                     env("NATS_USER", ""),
		NATSPassword:                 env("NATS_PASSWORD", ""),
		NATSToken:                    env("NATS_TOKEN", ""),
		NATSCredentialsFile:          env("NATS_CREDENTIALS_FILE", ""),
		NATSSubjectPrefix:            env("NATS_SUBJECT_PREFIX", "caipiao"),
		CloudRealtimeCoalesce:        envDurationMS("CLOUD_REALTIME_COALESCE_MS", 200*time.Millisecond),
		CloudStatsCoalesce:           envDurationMS("CLOUD_STATS_COALESCE_MS", time.Second),
		CloudReconcileInterval:       envDurationMS("CLOUD_RECONCILE_INTERVAL_MS", 5*time.Second),
		CloudReconcileBatch:          envInt("CLOUD_RECONCILE_BATCH", 500),
		Guaji:                        guaji.LoadConfigFromEnv(),
		CMSUploadDir:                 env("CMS_UPLOAD_DIR", "./data/uploads/cms"),
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

func envDurationMS(key string, fallback time.Duration) time.Duration {
	n := envInt(key, int(fallback/time.Millisecond))
	if n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Millisecond
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
