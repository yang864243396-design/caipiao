package server

import (
	"errors"
	"strings"
	"time"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/schemebettingdispatch"
)

func newSchemeBettingDispatchRuntime(cfg config.Config, pool *db.Pool, accounts *accountsvc.Service) (*schemebettingdispatch.Runtime, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.SchemeBettingMode))
	if mode == "" || mode == "shadow" {
		return nil, nil
	}
	if mode != "gray" && mode != "production" {
		return nil, errors.New("SCHEME_BETTING_MODE must be shadow, gray, or production")
	}
	if err := validateFormalSchemeInfrastructure(cfg, mode); err != nil {
		return nil, err
	}
	if !cfg.SchemeEventBusEnabled {
		if cfg.SchemeBettingShardCount <= 0 {
			return nil, errors.New("formal scheme betting requires SCHEME_BETTING_SHARD_COUNT > 0")
		}
		for _, shard := range cfg.SchemeBettingShards {
			if shard < 0 || int(shard) >= cfg.SchemeBettingShardCount {
				return nil, errors.New("SCHEME_BETTING_SHARDS contains a shard outside SCHEME_BETTING_SHARD_COUNT")
			}
		}
		return nil, errors.New("formal scheme betting requires SCHEME_EVENT_BUS_ENABLED=true")
	}
	if pool == nil {
		return nil, errors.New("formal scheme betting requires PostgreSQL")
	}
	if accounts == nil || !accounts.Enabled() {
		return nil, errors.New("formal scheme betting requires enabled guaji single-attempt placement")
	}
	runtime, err := schemebettingdispatch.New(sqlcdb.New(pool), accounts, schemebettingdispatch.Config{
		Mode: mode, Owner: cfg.SchemeBettingDispatcherOwner, LotteryCodes: cfg.SchemeBettingLotteries,
		Shards: cfg.SchemeBettingShards, Batch: int32(cfg.SchemeBettingBatch), Concurrency: cfg.SchemeBettingConcurrency,
		LeaseDuration: cfg.SchemeBettingLease, PollInterval: cfg.SchemeBettingPoll,
	})
	if err != nil {
		return nil, err
	}
	runtime.SetAcceptanceFinalizer(schemebettingdispatch.NewAcceptanceFinalizer(pool, accounts))
	runtime.SetDispatchLimiter(schemebettingdispatch.NewDispatchRateLimiter(pool))
	return runtime, nil
}

func validateFormalSchemeInfrastructure(cfg config.Config, mode string) error {
	if mode != "production" {
		return nil
	}
	if !cfg.DBRequired {
		return errors.New("production scheme betting requires DB_REQUIRED=true")
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(cfg.NATSURL)), "tls://") {
		return errors.New("production scheme betting requires a TLS NATS URL")
	}
	if strings.TrimSpace(cfg.NATSCredentialsFile) == "" && strings.TrimSpace(cfg.NATSToken) == "" &&
		(strings.TrimSpace(cfg.NATSUser) == "" || strings.TrimSpace(cfg.NATSPassword) == "") {
		return errors.New("production scheme betting requires authenticated NATS credentials")
	}
	if cfg.SchemeEventReplicas < 3 {
		return errors.New("production scheme betting requires at least three JetStream replicas")
	}
	if cfg.SchemeEventMaxAge < 72*time.Hour {
		return errors.New("production scheme betting requires at least 72 hours of event retention")
	}
	return nil
}
