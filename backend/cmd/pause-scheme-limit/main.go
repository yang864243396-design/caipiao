// 手动触发方案止盈/止损停投（Worker 同等逻辑）
// go run ./cmd/pause-scheme-limit/ inst-1-1782018133383
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/cloudlimits"
	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
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
	q := sqlcdb.New(pool)

	inst, err := q.GetSchemeInstanceByID(ctx, schemeID)
	if err != nil {
		fmt.Println("instance:", err)
		os.Exit(1)
	}
	def, err := q.GetSchemeDefinitionByID(ctx, inst.DefinitionID)
	if err != nil {
		fmt.Println("definition:", err)
		os.Exit(1)
	}

	sessionPnl := 0.0
	if f, ferr := inst.SessionPnl.Float64Value(); ferr == nil && f.Valid {
		sessionPnl = f.Float64
	}
	reason, hit := schemelimits.Evaluate(sessionPnl, def.Config)
	if hit {
		n, perr := q.PauseSchemeInstanceByWorker(ctx, sqlcdb.PauseSchemeInstanceByWorkerParams{
			ID:           inst.ID,
			StatusReason: reason,
		})
		if perr != nil {
			fmt.Println("scheme pause:", perr)
			os.Exit(1)
		}
		fmt.Printf("scheme paused: id=%s reason=%s rows=%d sessionPnl=%.2f\n", inst.ID, reason, n, sessionPnl)
	} else {
		fmt.Printf("scheme limit not hit: sessionPnl=%.2f\n", sessionPnl)
	}

	settings, serr := q.GetMemberCloudSettings(ctx, inst.MemberID)
	if serr == nil {
		sum, _ := q.SumMemberFormalSessionPnl(ctx, inst.MemberID)
		total := 0.0
		if f, ferr := sum.Float64Value(); ferr == nil && f.Valid {
			total = f.Float64
		}
		limits := cloudlimits.LimitsFromSettings(settings.TotalStopLoss, settings.TotalTakeProfit)
		cloudReason, cloudHit := cloudlimits.Evaluate(total, limits)
		if cloudHit {
			rows, perr := q.PauseAllRunningInstancesByMember(ctx, sqlcdb.PauseAllRunningInstancesByMemberParams{
				MemberID:     inst.MemberID,
				StatusReason: cloudReason,
			})
			if perr != nil {
				fmt.Println("cloud pause:", perr)
				os.Exit(1)
			}
			fmt.Printf("cloud limit pause: reason=%s totalPnl=%.2f count=%d\n", cloudReason, total, len(rows))
		} else {
			fmt.Printf("cloud limit not hit: totalPnl=%.2f\n", total)
		}
	}
}
