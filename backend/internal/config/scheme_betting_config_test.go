package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadSchemeBettingDefaultsStayShadowAndNonDispatching(t *testing.T) {
	for _, key := range []string{
		"SCHEME_BETTING_SHARD_COUNT",
		"SCHEME_BETTING_MODE", "SCHEME_BETTING_LOTTERIES", "SCHEME_BETTING_SHARDS",
		"SCHEME_BETTING_DISPATCHER_OWNER", "SCHEME_BETTING_BATCH", "SCHEME_BETTING_CONCURRENCY",
		"SCHEME_BETTING_LEASE_MS", "SCHEME_BETTING_POLL_MS",
	} {
		t.Setenv(key, "")
	}
	cfg := Load()
	if cfg.SchemeBettingMode != "shadow" || len(cfg.SchemeBettingLotteries) != 0 || len(cfg.SchemeBettingShards) != 0 {
		t.Fatalf("unsafe defaults: mode=%q lotteries=%v shards=%v", cfg.SchemeBettingMode, cfg.SchemeBettingLotteries, cfg.SchemeBettingShards)
		if cfg.SchemeBettingShardCount != 64 {
			t.Fatalf("shard count default = %d", cfg.SchemeBettingShardCount)
		}
	}
}

func TestLoadSchemeBettingFormalOverrides(t *testing.T) {
	t.Setenv("SCHEME_BETTING_MODE", "gray")
	t.Setenv("SCHEME_BETTING_LOTTERIES", "six-a, six-b")
	t.Setenv("SCHEME_BETTING_SHARDS", "0,3,63,3,bad")
	t.Setenv("SCHEME_BETTING_DISPATCHER_OWNER", "node-a")
	t.Setenv("SCHEME_BETTING_SHARD_COUNT", "64")
	t.Setenv("SCHEME_BETTING_BATCH", "24")
	t.Setenv("SCHEME_BETTING_CONCURRENCY", "6")
	t.Setenv("SCHEME_BETTING_LEASE_MS", "4500")
	t.Setenv("SCHEME_BETTING_POLL_MS", "80")
	cfg := Load()
	if cfg.SchemeBettingMode != "gray" || !reflect.DeepEqual(cfg.SchemeBettingLotteries, []string{"six-a", "six-b"}) || !reflect.DeepEqual(cfg.SchemeBettingShards, []int32{0, 3, 63}) {
		t.Fatalf("mode=%q lotteries=%v shards=%v", cfg.SchemeBettingMode, cfg.SchemeBettingLotteries, cfg.SchemeBettingShards)
	}
	if cfg.SchemeBettingDispatcherOwner != "node-a" || cfg.SchemeBettingBatch != 24 || cfg.SchemeBettingConcurrency != 6 || cfg.SchemeBettingLease != 4500*time.Millisecond || cfg.SchemeBettingPoll != 80*time.Millisecond {
		t.Fatalf("dispatcher config=%+v", cfg)
	}
}
