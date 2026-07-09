// 诊断 simBet 迁移（00102）：cloud_bet_records / scheme_instances 缺 sim_bet 列
// go run ./cmd/diag-simbet-schema/
// go run ./cmd/diag-simbet-schema/ -fix
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

const fixSQL = `
ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS sim_bet BOOLEAN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'cloud_bet_records' AND column_name = 'run_mode'
    ) THEN
        UPDATE cloud_bet_records SET sim_bet = (run_mode = 'sim') WHERE sim_bet IS NULL;
    END IF;
END $$;

UPDATE cloud_bet_records SET sim_bet = false WHERE sim_bet IS NULL;

ALTER TABLE cloud_bet_records
    ALTER COLUMN sim_bet SET NOT NULL;

DROP INDEX IF EXISTS idx_cloud_bet_records_member_mode_placed;
DROP INDEX IF EXISTS idx_cloud_bet_records_member_scheme_placed;

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_sim_bet_placed
    ON cloud_bet_records (member_id, sim_bet, placed_at DESC);

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_scheme_sim_bet_placed
    ON cloud_bet_records (member_id, scheme_id, sim_bet, placed_at DESC);

ALTER TABLE cloud_bet_records DROP CONSTRAINT IF EXISTS chk_cloud_bet_records_mode;
ALTER TABLE cloud_bet_records DROP COLUMN IF EXISTS run_mode;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'scheme_instances' AND column_name = 'run_mode'
    ) THEN
        UPDATE scheme_instances
        SET sim_bet = (run_mode = 'sim')
        WHERE run_mode IS NOT NULL
          AND sim_bet IS DISTINCT FROM (run_mode = 'sim');
    END IF;
END $$;

ALTER TABLE scheme_instances DROP CONSTRAINT IF EXISTS chk_scheme_instances_run_mode;
ALTER TABLE scheme_instances DROP COLUMN IF EXISTS run_mode;

ALTER TABLE member_lookback_runtime DROP COLUMN IF EXISTS run_mode;
`

func main() {
	fix := flag.Bool("fix", false, "补应用 00102 sim_bet 迁移")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx := context.Background()
	tables := []string{"cloud_bet_records", "scheme_instances"}
	var missingAll []string
	for _, t := range tables {
		cols, err := listColumns(ctx, pool, t)
		if err != nil {
			fmt.Println("schema:", err)
			os.Exit(1)
		}
		fmt.Printf("%s columns: %s\n", t, strings.Join(cols, ", "))
		if m := missingCols(cols, []string{"sim_bet"}); len(m) > 0 {
			missingAll = append(missingAll, t+".sim_bet")
		}
	}

	if len(missingAll) == 0 {
		fmt.Println("ok: sim_bet 列已存在，投注不应因缺列失败")
		return
	}

	fmt.Println("missing:", strings.Join(missingAll, ", "))
	fmt.Println("原因: 后端已切 simBet，但库未执行 00102 迁移")

	if !*fix {
		fmt.Println("\n修复: cd backend && go run ./cmd/diag-simbet-schema/ -fix")
		fmt.Println("或:   make migrate-up")
		os.Exit(2)
	}

	if _, err := pool.Exec(ctx, fixSQL); err != nil {
		fmt.Println("apply:", err)
		os.Exit(1)
	}
	fmt.Println("ok: 00102 已应用，请重试投注")
}

func listColumns(ctx context.Context, pool *db.Pool, table string) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func missingCols(have, need []string) []string {
	set := make(map[string]struct{}, len(have))
	for _, c := range have {
		set[c] = struct{}{}
	}
	var out []string
	for _, n := range need {
		if _, ok := set[n]; !ok {
			out = append(out, n)
		}
	}
	return out
}
