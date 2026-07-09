// 手动触发单笔/全部 pending 第三方注单派奖同步
// go run ./cmd/sync-pending-payout/
// go run ./cmd/sync-pending-payout/ BO11782110976816
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/orders/bets"
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

	svc := accountsvc.NewService(pool, guaji.NewClient(cfg.Guaji), cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	w := svc.NewPayoutSyncWorker(nil, bets.LocalGuajiDrawFallback(pool))
	if w == nil {
		fmt.Println("payout sync worker unavailable (GUAJI_ENABLED? credentials?)")
		os.Exit(1)
	}

	ctx := context.Background()
	orderFilter := ""
	if len(os.Args) > 1 {
		orderFilter = os.Args[1]
	}

	var targets []sqlcdb.ListPendingGuajiBetOrdersRow
	if orderFilter != "" {
		row, err := svc.LoadPendingGuajiBetOrder(ctx, orderFilter)
		if err != nil {
			if err == pgx.ErrNoRows {
				fmt.Printf("pending guaji order not found: %s\n", orderFilter)
			} else {
				fmt.Println("load:", err)
			}
			os.Exit(1)
		}
		targets = []sqlcdb.ListPendingGuajiBetOrdersRow{row}
	} else {
		q := sqlcdb.New(pool)
		targets, err = q.ListPendingGuajiBetOrders(ctx, 50)
		if err != nil {
			fmt.Println("list:", err)
			os.Exit(1)
		}
	}

	fmt.Printf("pending guaji orders: %d\n", len(targets))
	synced := 0
	for _, row := range targets {
		fmt.Printf("syncing %s thirdPartyBetId=%s ...\n", row.OrderNo, row.ThirdPartyBetID.String)
		if err := w.SyncOne(ctx, row); err != nil {
			fmt.Printf("FAIL %s: %v\n", row.OrderNo, err)
			continue
		}
		fmt.Printf("OK %s\n", row.OrderNo)
		synced++
	}
	fmt.Printf("done synced=%d\n", synced)
}
