// 补应用 00101：回头 apply_formal/apply_sim + runtime sim_bet/total_hit_count
// go run ./cmd/apply-lookback-sim-bet/
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
    ADD COLUMN IF NOT EXISTS apply_formal BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE member_lookback_settings
    ADD COLUMN IF NOT EXISTS apply_sim BOOLEAN NOT NULL DEFAULT false;

UPDATE member_lookback_settings
SET apply_formal = (run_mode LIKE '%real%'),
    apply_sim    = (run_mode LIKE '%sim%');

ALTER TABLE member_lookback_runtime
    ADD COLUMN IF NOT EXISTS sim_bet BOOLEAN;

UPDATE member_lookback_runtime
SET sim_bet = (run_mode = 'sim')
WHERE sim_bet IS NULL;

UPDATE member_lookback_runtime
SET sim_bet = false
WHERE sim_bet IS NULL;

ALTER TABLE member_lookback_runtime
    ALTER COLUMN sim_bet SET NOT NULL;

ALTER TABLE member_lookback_runtime
    ADD COLUMN IF NOT EXISTS total_hit_count INT NOT NULL DEFAULT 0;

ALTER TABLE member_lookback_runtime
    DROP CONSTRAINT IF EXISTS member_lookback_runtime_pkey;

ALTER TABLE member_lookback_runtime
    ADD PRIMARY KEY (member_id, sim_bet);
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
	fmt.Println("ok: lookback sim_bet migration applied (00101)")
}
