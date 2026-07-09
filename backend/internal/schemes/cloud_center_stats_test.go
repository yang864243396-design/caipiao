package schemes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
)

type instSnapshot struct {
	id         string
	simBet     bool
	status     string
	turnover   float64
	sessionPnl float64
}

func loadDemoAccountInstances(t *testing.T, pool *db.Pool, memberID int64) []instSnapshot {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
SELECT id, sim_bet, status, turnover::float8, COALESCE(session_pnl, 0)::float8
FROM scheme_instances WHERE member_id = $1 ORDER BY id`, memberID)
	if err != nil {
		t.Fatalf("load instances: %v", err)
	}
	defer rows.Close()
	var out []instSnapshot
	for rows.Next() {
		var s instSnapshot
		if err := rows.Scan(&s.id, &s.simBet, &s.status, &s.turnover, &s.sessionPnl); err != nil {
			t.Fatalf("scan instance: %v", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("instances rows: %v", err)
	}
	return out
}

func restoreDemoAccountInstances(t *testing.T, pool *db.Pool, snaps []instSnapshot) {
	t.Helper()
	for _, s := range snaps {
		_, err := pool.Exec(context.Background(), `
UPDATE scheme_instances
SET sim_bet = $2, status = $3, turnover = $4, session_pnl = $5, updated_at = now()
WHERE id = $1`,
			s.id, s.simBet, s.status, s.turnover, s.sessionPnl)
		if err != nil {
			t.Errorf("restore %s: %v", s.id, err)
		}
	}
}

func TestGetCloudCenterStatsSeededE2E(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { pool.Close() })

	account := cfg.ClientDemoAccount
	if account == "" {
		account = "vs8888"
	}
	var memberID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM members WHERE account = $1`, account).Scan(&memberID); err != nil {
		t.Fatalf("member: %v", err)
	}
	snaps := loadDemoAccountInstances(t, pool, memberID)
	if len(snaps) < 2 {
		t.Skip("need at least 2 scheme instances for seeded e2e")
	}
	t.Cleanup(func() { restoreDemoAccountInstances(t, pool, snaps) })

	// 正式 running + 模拟 pending，验证分路与 running 过滤
	_, err = pool.Exec(context.Background(), `
UPDATE scheme_instances SET sim_bet=false, status='running', turnover=120, session_pnl=15.3, updated_at=now() WHERE id=$1`,
		snaps[0].id)
	if err != nil {
		t.Fatalf("seed formal: %v", err)
	}
	_, err = pool.Exec(context.Background(), `
UPDATE scheme_instances SET sim_bet=true, status='pending', turnover=50, session_pnl=8.7, updated_at=now() WHERE id=$1`,
		snaps[1].id)
	if err != nil {
		t.Fatalf("seed sim: %v", err)
	}

	svc := schemes.NewService(pool, nil)
	got, err := svc.GetCloudCenterStats(context.Background(), account)
	if err != nil {
		t.Fatalf("GetCloudCenterStats: %v", err)
	}

	wantFormal := schemes.CloudCenterChannelStats{TotalTurnover: 120, TotalSessionPnl: 15.3, RunningSessionPnl: 15.3}
	wantSim := schemes.CloudCenterChannelStats{TotalTurnover: 50, TotalSessionPnl: 8.7, RunningSessionPnl: 0}
	assertChannelStats(t, "formal", got.Formal, wantFormal)
	assertChannelStats(t, "sim", got.Sim, wantSim)

	base := strings.TrimRight(os.Getenv("E2E_API_BASE"), "/")
	if base == "" {
		base = "http://127.0.0.1:8080/api/v1"
	}
	loginBody := fmt.Sprintf(`{"account":%q,"password":%q}`, account, cfg.ClientDemoPass)
	if cfg.ClientDemoPass == "" {
		loginBody = fmt.Sprintf(`{"account":%q,"password":%q}`, account, account)
	}
	resp, err := http.Post(base+"/client/auth/login", "application/json", strings.NewReader(loginBody))
	if err != nil {
		t.Skipf("http login unavailable: %v", err)
	}
	defer resp.Body.Close()
	loginRaw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login http %d: %s", resp.StatusCode, loginRaw)
	}
	var loginEnv struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRaw, &loginEnv); err != nil || loginEnv.Code != 0 || loginEnv.Data.AccessToken == "" {
		t.Fatalf("login parse: %v body=%s", err, loginRaw)
	}

	req, _ := http.NewRequest(http.MethodGet, base+"/client/cloud/schemes/stats", nil)
	req.Header.Set("Authorization", "Bearer "+loginEnv.Data.AccessToken)
	statsResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stats http: %v", err)
	}
	defer statsResp.Body.Close()
	statsRaw, _ := io.ReadAll(statsResp.Body)
	if statsResp.StatusCode != http.StatusOK {
		t.Fatalf("stats http %d: %s", statsResp.StatusCode, statsRaw)
	}
	var statsEnv struct {
		Code int                    `json:"code"`
		Data schemes.CloudCenterStats `json:"data"`
	}
	if err := json.Unmarshal(statsRaw, &statsEnv); err != nil || statsEnv.Code != 0 {
		t.Fatalf("stats parse: %v body=%s", err, statsRaw)
	}
	assertChannelStats(t, "http formal", statsEnv.Data.Formal, wantFormal)
	assertChannelStats(t, "http sim", statsEnv.Data.Sim, wantSim)
	t.Logf("seeded e2e ok formal=%+v sim=%+v", statsEnv.Data.Formal, statsEnv.Data.Sim)
}

func assertChannelStats(t *testing.T, name string, got, want schemes.CloudCenterChannelStats) {
	t.Helper()
	if !approxStat(got.TotalTurnover, want.TotalTurnover) {
		t.Errorf("%s totalTurnover: got %.1f want %.1f", name, got.TotalTurnover, want.TotalTurnover)
	}
	if !approxStat(got.TotalSessionPnl, want.TotalSessionPnl) {
		t.Errorf("%s totalSessionPnl: got %.1f want %.1f", name, got.TotalSessionPnl, want.TotalSessionPnl)
	}
	if !approxStat(got.RunningSessionPnl, want.RunningSessionPnl) {
		t.Errorf("%s runningSessionPnl: got %.1f want %.1f", name, got.RunningSessionPnl, want.RunningSessionPnl)
	}
}

func TestGetCloudCenterStatsIntegration(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	defer pool.Close()

	account := cfg.ClientDemoAccount
	if account == "" {
		account = "vs8888"
	}

	svc := schemes.NewService(pool, nil)
	got, err := svc.GetCloudCenterStats(context.Background(), account)
	if err != nil {
		t.Fatalf("GetCloudCenterStats: %v", err)
	}

	var memberID int64
	err = pool.QueryRow(context.Background(), `SELECT id FROM members WHERE account = $1`, account).Scan(&memberID)
	if err != nil {
		t.Fatalf("member lookup: %v", err)
	}

	type channelExpect struct {
		totalTurnover     float64
		totalSessionPnl   float64
		runningSessionPnl float64
	}
	expect := map[bool]channelExpect{false: {}, true: {}}

	rows, err := pool.Query(context.Background(), `
SELECT
    sim_bet,
    COALESCE(SUM(turnover), 0)::float8 AS total_turnover,
    COALESCE(SUM(session_pnl), 0)::float8 AS total_session_pnl,
    COALESCE(SUM(session_pnl) FILTER (WHERE status = 'running'), 0)::float8 AS running_session_pnl
FROM scheme_instances
WHERE member_id = $1
GROUP BY sim_bet`, memberID)
	if err != nil {
		t.Fatalf("direct sql: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var simBet bool
		var ch channelExpect
		if err := rows.Scan(&simBet, &ch.totalTurnover, &ch.totalSessionPnl, &ch.runningSessionPnl); err != nil {
			t.Fatalf("scan: %v", err)
		}
		expect[simBet] = ch
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	assertChannel := func(name string, got schemes.CloudCenterChannelStats, exp channelExpect) {
		t.Helper()
		if !approxStat(got.TotalTurnover, roundStat(exp.totalTurnover)) {
			t.Errorf("%s totalTurnover: got %.1f want %.1f", name, got.TotalTurnover, roundStat(exp.totalTurnover))
		}
		if !approxStat(got.TotalSessionPnl, roundStat(exp.totalSessionPnl)) {
			t.Errorf("%s totalSessionPnl: got %.1f want %.1f", name, got.TotalSessionPnl, roundStat(exp.totalSessionPnl))
		}
		if !approxStat(got.RunningSessionPnl, roundStat(exp.runningSessionPnl)) {
			t.Errorf("%s runningSessionPnl: got %.1f want %.1f", name, got.RunningSessionPnl, roundStat(exp.runningSessionPnl))
		}
	}

	assertChannel("formal", got.Formal, expect[false])
	assertChannel("sim", got.Sim, expect[true])

	if math.Abs(got.Formal.RunningSessionPnl) > math.Abs(got.Formal.TotalSessionPnl)+0.05 {
		t.Errorf("formal |runningSessionPnl| %.1f exceeds |totalSessionPnl| %.1f",
			got.Formal.RunningSessionPnl, got.Formal.TotalSessionPnl)
	}
	if math.Abs(got.Sim.RunningSessionPnl) > math.Abs(got.Sim.TotalSessionPnl)+0.05 {
		t.Errorf("sim |runningSessionPnl| %.1f exceeds |totalSessionPnl| %.1f",
			got.Sim.RunningSessionPnl, got.Sim.TotalSessionPnl)
	}

	t.Logf("account=%s formal=%+v sim=%+v", account, got.Formal, got.Sim)
}

func TestGetCloudCenterStatsMemberNotFound(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	defer pool.Close()

	svc := schemes.NewService(pool, nil)
	_, err = svc.GetCloudCenterStats(context.Background(), "__no_such_member__")
	if !errors.Is(err, member.ErrNotFound) {
		t.Fatalf("expected member.ErrNotFound, got %v", err)
	}
}

func roundStat(v float64) float64 {
	return math.Round(v*10) / 10
}

func approxStat(got, want float64) bool {
	return math.Abs(got-want) < 0.05
}
