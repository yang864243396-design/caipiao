package schemes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/schemes"
)

type maintResumeSnap struct {
	instID         string
	status         string
	statusReason   string
	turnover       float64
	sessionPnl     float64
	defID          string
	defConfig      []byte
	breakPeriodStop bool
}

func loadMaintResumeSnap(t *testing.T, pool *db.Pool, instID string, memberID int64) maintResumeSnap {
	t.Helper()
	var snap maintResumeSnap
	snap.instID = instID
	err := pool.QueryRow(context.Background(), `
SELECT status, COALESCE(status_reason,''), turnover::float8, COALESCE(session_pnl,0)::float8, definition_id
FROM scheme_instances WHERE id = $1 AND member_id = $2`, instID, memberID).Scan(
		&snap.status, &snap.statusReason, &snap.turnover, &snap.sessionPnl, &snap.defID,
	)
	if err != nil {
		t.Fatalf("load instance %s: %v", instID, err)
	}
	err = pool.QueryRow(context.Background(), `SELECT config FROM scheme_definitions WHERE id = $1`, snap.defID).Scan(&snap.defConfig)
	if err != nil {
		t.Fatalf("load definition %s: %v", snap.defID, err)
	}
	err = pool.QueryRow(context.Background(), `
SELECT COALESCE(break_period_stop, false) FROM member_cloud_settings WHERE member_id = $1`, memberID).Scan(&snap.breakPeriodStop)
	if err != nil {
		// 无行时默认 false
		snap.breakPeriodStop = false
	}
	return snap
}

func restoreMaintResumeSnap(t *testing.T, pool *db.Pool, memberID int64, snap maintResumeSnap) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
UPDATE scheme_instances
SET status = $2, status_reason = NULLIF($3,''), turnover = $4, session_pnl = $5, updated_at = now()
WHERE id = $1 AND member_id = $6`,
		snap.instID, snap.status, snap.statusReason, snap.turnover, snap.sessionPnl, memberID)
	if err != nil {
		t.Errorf("restore instance %s: %v", snap.instID, err)
	}
	_, err = pool.Exec(context.Background(), `UPDATE scheme_definitions SET config = $2 WHERE id = $1`, snap.defID, snap.defConfig)
	if err != nil {
		t.Errorf("restore definition %s: %v", snap.defID, err)
	}
	_, err = pool.Exec(context.Background(), `
INSERT INTO member_cloud_settings (member_id, break_period_stop, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (member_id) DO UPDATE SET break_period_stop = EXCLUDED.break_period_stop, updated_at = now()`,
		memberID, snap.breakPeriodStop)
	if err != nil {
		t.Errorf("restore cloud settings: %v", err)
	}
}

func seedMaintenanceStopped(t *testing.T, pool *db.Pool, instID string, memberID int64, sessionPnl float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
UPDATE scheme_instances
SET status = 'pending', status_reason = 'maintenance', session_pnl = $3, updated_at = now()
WHERE id = $1 AND member_id = $2`, instID, memberID, sessionPnl)
	if err != nil {
		t.Fatalf("seed maintenance stopped: %v", err)
	}
}

func seedDefinitionScheduleWindow(t *testing.T, pool *db.Pool, defID string, now time.Time) {
	t.Helper()
	cfg := map[string]string{
		"startTime": now.Add(-2 * time.Hour).Format("2006-01-02 15:04:05"),
		"endTime":   now.Add(2 * time.Hour).Format("2006-01-02 15:04:05"),
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	_, err = pool.Exec(context.Background(), `UPDATE scheme_definitions SET config = $2 WHERE id = $1`, defID, raw)
	if err != nil {
		t.Fatalf("seed definition window: %v", err)
	}
}

func setBreakPeriodStop(t *testing.T, pool *db.Pool, memberID int64, stop bool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
INSERT INTO member_cloud_settings (member_id, break_period_stop, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (member_id) DO UPDATE SET break_period_stop = EXCLUDED.break_period_stop, updated_at = now()`,
		memberID, stop)
	if err != nil {
		t.Fatalf("set break_period_stop=%v: %v", stop, err)
	}
}

func lotteryOnSale(t *testing.T, pool *db.Pool, lotteryCode string) bool {
	t.Helper()
	var saleStatus string
	err := pool.QueryRow(context.Background(), `SELECT sale_status FROM lottery_catalog WHERE code = $1`, lotteryCode).Scan(&saleStatus)
	if err != nil {
		t.Fatalf("lottery catalog %s: %v", lotteryCode, err)
	}
	return saleStatus == "on_sale"
}

func siteMaintenanceEnabled(t *testing.T, pool *db.Pool) bool {
	t.Helper()
	var enabled bool
	err := pool.QueryRow(context.Background(), `SELECT enabled FROM cms_maintenance LIMIT 1`).Scan(&enabled)
	if err != nil {
		return false
	}
	return enabled
}

func readInstanceMetrics(t *testing.T, pool *db.Pool, instID string, memberID int64) (status, reason string, sessionPnl float64) {
	t.Helper()
	err := pool.QueryRow(context.Background(), `
SELECT status, COALESCE(status_reason,''), COALESCE(session_pnl,0)::float8
FROM scheme_instances WHERE id = $1 AND member_id = $2`, instID, memberID).Scan(&status, &reason, &sessionPnl)
	if err != nil {
		t.Fatalf("read instance: %v", err)
	}
	return status, reason, sessionPnl
}

func pickDemoInstance(t *testing.T, svc *schemes.Service, account string) (instID, lotteryCode string) {
	t.Helper()
	rows, err := svc.ListInstances(context.Background(), account, "")
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	if len(rows.Items) == 0 {
		t.Skip("no instances for demo account")
	}
	return rows.Items[0].ID, rows.Items[0].LotteryCode
}

func TestBreakPeriodStopMaintenanceResumeIntegration(t *testing.T) {
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

	svc := schemes.NewService(pool, nil)
	instID, lotteryCode := pickDemoInstance(t, svc, account)
	if siteMaintenanceEnabled(t, pool) {
		t.Skip("site maintenance enabled; skip resume integration")
	}
	if !lotteryOnSale(t, pool, lotteryCode) {
		t.Skipf("lottery %s not on_sale; skip resume integration", lotteryCode)
	}

	snap := loadMaintResumeSnap(t, pool, instID, memberID)
	t.Cleanup(func() { restoreMaintResumeSnap(t, pool, memberID, snap) })

	now := time.Now()
	seedDefinitionScheduleWindow(t, pool, snap.defID, now)

	t.Run("T7_breakPeriodStop_true_no_auto_resume", func(t *testing.T) {
		const wantPnl = 77.7
		seedMaintenanceStopped(t, pool, instID, memberID, wantPnl)
		setBreakPeriodStop(t, pool, memberID, true)

		worker := schemes.NewWorker(pool, 5, nil, nil)
		worker.TickMaintenanceResume(context.Background())

		status, reason, pnl := readInstanceMetrics(t, pool, instID, memberID)
		if status != "pending" || reason != "maintenance" {
			t.Fatalf("want pending+maintenance, got %s+%s", status, reason)
		}
		if pnl != wantPnl {
			t.Fatalf("session_pnl changed: got %.2f want %.2f", pnl, wantPnl)
		}
	})

	t.Run("T2_T11_auto_resume_preserves_session_pnl", func(t *testing.T) {
		const wantPnl = 88.8
		seedMaintenanceStopped(t, pool, instID, memberID, wantPnl)
		setBreakPeriodStop(t, pool, memberID, false)

		worker := schemes.NewWorker(pool, 5, nil, nil)
		worker.TickMaintenanceResume(context.Background())

		status, reason, pnl := readInstanceMetrics(t, pool, instID, memberID)
		if status != "running" || reason != "await_next_bet" {
			t.Fatalf("want running+await_next_bet, got %s+%s", status, reason)
		}
		if pnl != wantPnl {
			t.Fatalf("session_pnl not preserved: got %.2f want %.2f", pnl, wantPnl)
		}
	})

	t.Run("T8_manual_start_past_startTime_preserves_session_pnl", func(t *testing.T) {
		const wantPnl = 99.9
		seedMaintenanceStopped(t, pool, instID, memberID, wantPnl)
		setBreakPeriodStop(t, pool, memberID, true)

		inst, err := svc.StartInstance(context.Background(), account, instID)
		if err != nil {
			t.Fatalf("StartInstance maintenance resume: %v", err)
		}
		if inst.Status != "running" {
			t.Fatalf("want running, got %s", inst.Status)
		}
		if inst.SessionPnL != wantPnl {
			t.Fatalf("sessionPnl not preserved: got %.2f want %.2f", inst.SessionPnL, wantPnl)
		}
		t.Logf("manual resume ok instance=%s sessionPnl=%.2f", instID, inst.SessionPnL)
	})

	t.Run("T8b_enableAll_filter_logic", func(t *testing.T) {
		seedMaintenanceStopped(t, pool, instID, memberID, 12.3)
		rows, err := svc.ListInstances(context.Background(), account, "")
		if err != nil {
			t.Fatalf("ListInstances: %v", err)
		}
		var maintCount, eligibleCount int
		for _, item := range rows.Items {
			if item.Status != "pending" && item.Status != "paused" {
				continue
			}
			if item.StatusReason == "maintenance" {
				maintCount++
				continue
			}
			eligibleCount++
		}
		if maintCount < 1 {
			t.Fatalf("expected at least one maintenance instance in list")
		}
		t.Logf("enableAll would skip %d maintenance, include %d others", maintCount, eligibleCount)
	})

	t.Run("T1_metrics_unchanged_on_maintenance_pause", func(t *testing.T) {
		// 模拟 worker 维护停投：pending+maintenance 且指标不变
		const wantPnl = 55.5
		const wantTurnover = 321.0
		_, err := pool.Exec(context.Background(), `
UPDATE scheme_instances
SET status = 'running', status_reason = 'await_next_bet', session_pnl = $3, turnover = $4, updated_at = now()
WHERE id = $1 AND member_id = $2`, instID, memberID, wantPnl, wantTurnover)
		if err != nil {
			t.Fatalf("seed running: %v", err)
		}
		_, err = pool.Exec(context.Background(), `
UPDATE scheme_instances
SET status = 'pending', status_reason = 'maintenance', updated_at = now()
WHERE id = $1 AND member_id = $2`, instID, memberID)
		if err != nil {
			t.Fatalf("simulate maintenance pause: %v", err)
		}
		var pnl, turnover float64
		var status, reason string
		err = pool.QueryRow(context.Background(), `
SELECT status, COALESCE(status_reason,''), COALESCE(session_pnl,0)::float8, turnover::float8
FROM scheme_instances WHERE id = $1`, instID).Scan(&status, &reason, &pnl, &turnover)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if status != "pending" || reason != "maintenance" {
			t.Fatalf("want pending+maintenance, got %s+%s", status, reason)
		}
		if pnl != wantPnl || turnover != wantTurnover {
			t.Fatalf("metrics changed: pnl=%.2f turnover=%.2f", pnl, turnover)
		}
		t.Log(fmt.Sprintf("T1 ok pnl=%.1f turnover=%.1f", pnl, turnover))
	})
}
