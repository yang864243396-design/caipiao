package config

import (
	"testing"
	"time"
)

func TestLoadCloudRealtimeConfiguration(t *testing.T) {
	// This catches incorrect defaults, environment keys, or duration units in
	// the realtime configuration consumed by the runtime wiring.
	tests := []struct {
		name string
		env  map[string]string
		want struct {
			enabled  bool
			bus      string
			url      string
			prefix   string
			coalesce time.Duration
			stats    time.Duration
			interval time.Duration
			batch    int
		}
	}{
		{
			name: "uses cloud realtime defaults",
			want: struct {
				enabled  bool
				bus      string
				url      string
				prefix   string
				coalesce time.Duration
				stats    time.Duration
				interval time.Duration
				batch    int
			}{
				enabled: true, bus: "nats", url: "nats://127.0.0.1:4222", prefix: "caipiao",
				coalesce: 200 * time.Millisecond, stats: time.Second, interval: 5 * time.Second, batch: 500,
			},
		},
		{
			name: "uses cloud realtime overrides",
			env: map[string]string{
				"CLOUD_REALTIME_ENABLED":      "false",
				"CLOUD_REALTIME_BUS":          "memory",
				"NATS_URL":                    "nats://bus.internal:4222",
				"NATS_SUBJECT_PREFIX":         "tenant",
				"CLOUD_REALTIME_COALESCE_MS":  "250",
				"CLOUD_STATS_COALESCE_MS":     "1250",
				"CLOUD_RECONCILE_INTERVAL_MS": "6000",
				"CLOUD_RECONCILE_BATCH":       "250",
			},
			want: struct {
				enabled  bool
				bus      string
				url      string
				prefix   string
				coalesce time.Duration
				stats    time.Duration
				interval time.Duration
				batch    int
			}{
				enabled: false, bus: "memory", url: "nats://bus.internal:4222", prefix: "tenant",
				coalesce: 250 * time.Millisecond, stats: 1250 * time.Millisecond, interval: 6 * time.Second, batch: 250,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{
				"CLOUD_REALTIME_ENABLED", "CLOUD_REALTIME_BUS", "NATS_URL", "NATS_USER", "NATS_PASSWORD", "NATS_TOKEN", "NATS_CREDENTIALS_FILE", "NATS_SUBJECT_PREFIX",
				"CLOUD_REALTIME_COALESCE_MS", "CLOUD_STATS_COALESCE_MS", "CLOUD_RECONCILE_INTERVAL_MS", "CLOUD_RECONCILE_BATCH",
			} {
				t.Setenv(key, "")
			}
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			got := Load()
			if got.CloudRealtimeEnabled != tt.want.enabled || got.CloudRealtimeBus != tt.want.bus || got.NATSURL != tt.want.url || got.NATSSubjectPrefix != tt.want.prefix || got.CloudRealtimeCoalesce != tt.want.coalesce || got.CloudStatsCoalesce != tt.want.stats || got.CloudReconcileInterval != tt.want.interval || got.CloudReconcileBatch != tt.want.batch {
				t.Fatalf("got enabled=%v bus=%q url=%q prefix=%q coalesce=%s stats=%s interval=%s batch=%d", got.CloudRealtimeEnabled, got.CloudRealtimeBus, got.NATSURL, got.NATSSubjectPrefix, got.CloudRealtimeCoalesce, got.CloudStatsCoalesce, got.CloudReconcileInterval, got.CloudReconcileBatch)
			}
			if got.NATSUser != "" || got.NATSPassword != "" || got.NATSToken != "" || got.NATSCredentialsFile != "" {
				t.Fatal("unexpected NATS credentials")
			}
		})
	}
}

func TestBuildDatabaseURLFromParts(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "192.168.100.239")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "caipiao")
	t.Setenv("DB_USER", "caipiaoapp")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_SSLMODE", "disable")

	got := buildDatabaseURL()
	want := "postgres://caipiaoapp:secret@192.168.100.239:5432/caipiao?sslmode=disable"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildDatabaseURLPrefersDATABASE_URL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@host:5432/db?sslmode=require")
	t.Setenv("DB_HOST", "ignored")

	got := buildDatabaseURL()
	if got != "postgres://u:p@host:5432/db?sslmode=require" {
		t.Fatalf("got %q", got)
	}
}
