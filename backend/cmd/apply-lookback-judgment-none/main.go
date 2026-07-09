// 补应用 00100：member_lookback_settings.judgment 允许空（未选择）
// go run ./cmd/apply-lookback-judgment-none/
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
ALTER TABLE member_lookback_settings
    DROP CONSTRAINT IF EXISTS chk_member_lookback_judgment;

ALTER TABLE member_lookback_settings
    ADD CONSTRAINT chk_member_lookback_judgment CHECK (
        judgment IN ('individual', 'overall', '')
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

	if _, err := pool.Exec(context.Background(), alterSQL); err != nil {
		fmt.Println("apply:", err)
		os.Exit(1)
	}
	fmt.Println("ok: chk_member_lookback_judgment updated (00100)")
}
