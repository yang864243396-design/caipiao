// 补应用 00099：扩展 scheme_instances.status_reason 允许止盈/止损原因
// go run ./cmd/apply-limit-constraint/
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

const alterSQL = `
ALTER TABLE scheme_instances
    DROP CONSTRAINT IF EXISTS chk_scheme_instances_status_reason;

ALTER TABLE scheme_instances
    ADD CONSTRAINT chk_scheme_instances_status_reason CHECK (
        status_reason IN (
            '', 'manual', 'insufficient_funds', 'maintenance', 'end_time',
            'await_next_bet', 'cloud_active', 'bet_failed',
            'scheme_stop_loss', 'scheme_take_profit',
            'total_stop_loss', 'total_take_profit'
        )
    );
`

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx := context.Background()
	if _, err := pool.Exec(ctx, alterSQL); err != nil {
		fmt.Println("apply:", err)
		os.Exit(1)
	}
	fmt.Println("ok: chk_scheme_instances_status_reason updated (00099)")
}
