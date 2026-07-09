// 诊断方案止盈/止损与停投
// go run ./cmd/diag-scheme-limits/ inst-1-1782018133383
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/cloudlimits"
	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/schemelimits"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	schemeID := "inst-1-1782018133383"
	if len(os.Args) > 1 {
		schemeID = os.Args[1]
	}
	ctx := context.Background()

	var memberID int64
	var status, statusReason, runMode string
	var sessionPnl, pnl, turnover float64
	err = pool.QueryRow(ctx, `
SELECT member_id, status, status_reason, run_mode,
       COALESCE(session_pnl,0)::float8, COALESCE(pnl,0)::float8, COALESCE(turnover,0)::float8
FROM scheme_instances WHERE id = $1`, schemeID).Scan(
		&memberID, &status, &statusReason, &runMode, &sessionPnl, &pnl, &turnover)
	if err != nil {
		fmt.Println("instance:", err)
		os.Exit(1)
	}

	var defConfig []byte
	var defID string
	err = pool.QueryRow(ctx, `
SELECT sd.id, sd.config
FROM scheme_instances si
JOIN scheme_definitions sd ON sd.id = si.definition_id
WHERE si.id = $1`, schemeID).Scan(&defID, &defConfig)
	if err != nil {
		fmt.Println("definition:", err)
		os.Exit(1)
	}

	var totalStopLoss, totalTakeProfit float64
	err = pool.QueryRow(ctx, `
SELECT COALESCE(total_stop_loss,0)::float8, COALESCE(total_take_profit,0)::float8
FROM member_cloud_settings WHERE member_id = $1`, memberID).Scan(&totalStopLoss, &totalTakeProfit)
	if err != nil {
		fmt.Println("cloud settings:", err)
	}

	var sumFormalSessionPnl float64
	_ = pool.QueryRow(ctx, `
SELECT COALESCE(SUM(session_pnl),0)::float8
FROM scheme_instances
WHERE member_id = $1 AND run_mode = 'real' AND status IN ('running','pending')`, memberID).Scan(&sumFormalSessionPnl)

	schemeReason, schemeHit := schemelimits.Evaluate(sessionPnl, defConfig)
	schemeLimits := schemelimits.Parse(defConfig)
	cloudLimits := cloudlimits.Limits{StopLossYuan: totalStopLoss, TakeProfitYuan: totalTakeProfit}
	cloudReason, cloudHit := cloudlimits.Evaluate(sumFormalSessionPnl, cloudLimits)

	// check constraint values
	var constraintDef string
	_ = pool.QueryRow(ctx, `
SELECT pg_get_constraintdef(oid) FROM pg_constraint
WHERE conname = 'chk_scheme_instances_status_reason'`).Scan(&constraintDef)

	var pendingUnsettled int
	_ = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM cloud_bet_records
WHERE scheme_id = $1 AND status = 'pending'`, schemeID).Scan(&pendingUnsettled)

	out := map[string]any{
		"schemeId": schemeID, "definitionId": defID, "memberId": memberID,
		"status": status, "statusReason": statusReason, "runMode": runMode,
		"sessionPnl": sessionPnl, "pnl": pnl, "turnover": turnover,
		"definitionConfig": json.RawMessage(defConfig),
		"schemeLimits": map[string]any{
			"stopLoss": schemeLimits.StopLossYuan, "takeProfit": schemeLimits.TakeProfitYuan,
			"shouldStop": schemeHit, "reason": schemeReason,
			"detail": schemelimits.Detail(schemeReason, sessionPnl, schemeLimits),
		},
		"cloudSettings": map[string]any{
			"totalStopLoss": totalStopLoss, "totalTakeProfit": totalTakeProfit,
		},
		"memberFormalSessionPnlSum": sumFormalSessionPnl,
		"cloudLimitCheck": map[string]any{
			"shouldStop": cloudHit, "reason": cloudReason,
			"detail": cloudlimits.Detail(cloudReason, sumFormalSessionPnl, cloudLimits),
		},
		"pendingCloudBets": pendingUnsettled,
		"statusReasonConstraint": constraintDef,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
