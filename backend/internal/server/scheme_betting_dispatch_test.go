package server

import (
	"strings"
	"testing"
	"time"

	"caipiao/backend/internal/config"
)

func TestSchemeBettingDispatchShadowDoesNotRequireInfrastructure(t *testing.T) {
	runtime, err := newSchemeBettingDispatchRuntime(config.Config{SchemeBettingMode: "shadow"}, nil, nil)
	if err != nil || runtime != nil {
		t.Fatalf("runtime=%v err=%v", runtime, err)
	}
}

func TestProductionSchemeBettingRequiresTLSAuthAndReplicas(t *testing.T) {
	base := config.Config{
		DBRequired: true, NATSURL: "tls://nats.internal:4222", NATSCredentialsFile: "service.creds",
		SchemeEventReplicas: 3, SchemeEventMaxAge: 72 * time.Hour,
	}
	if err := validateFormalSchemeInfrastructure(base, "production"); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		edit func(*config.Config)
		want string
	}{
		{name: "database optional", edit: func(c *config.Config) { c.DBRequired = false }, want: "DB_REQUIRED"},
		{name: "plaintext nats", edit: func(c *config.Config) { c.NATSURL = "nats://localhost:4222" }, want: "TLS"},
		{name: "anonymous nats", edit: func(c *config.Config) { c.NATSCredentialsFile = "" }, want: "authenticated"},
		{name: "single replica", edit: func(c *config.Config) { c.SchemeEventReplicas = 1 }, want: "three"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			tt.edit(&cfg)
			if err := validateFormalSchemeInfrastructure(cfg, "production"); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v want %q", err, tt.want)
			}
		})
	}
}

func TestSchemeBettingDispatchRejectsUnknownFormalMode(t *testing.T) {
	if _, err := newSchemeBettingDispatchRuntime(config.Config{SchemeBettingMode: "automatic"}, nil, nil); err == nil {
		t.Fatal("unknown mode accepted")
	}
}
