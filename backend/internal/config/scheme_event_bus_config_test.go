package config

import (
	"testing"
	"time"
)

func TestLoadSchemeEventBusDefaultsDisabled(t *testing.T) {
	for _, key := range []string{"SCHEME_EVENT_BUS_ENABLED", "SCHEME_EVENT_STREAM", "SCHEME_EVENT_REPLICAS", "SCHEME_EVENT_MAX_AGE_MS"} {
		t.Setenv(key, "")
	}
	cfg := Load()
	if cfg.SchemeEventBusEnabled || cfg.SchemeEventStream != "SCHEME_EVENTS" || cfg.SchemeEventReplicas != 1 || cfg.SchemeEventMaxAge != 72*time.Hour {
		t.Fatalf("event bus defaults=%+v", cfg)
	}
}
