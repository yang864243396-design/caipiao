// 诊断 member_lookback 表结构；缺失时可用 -fix 补应用 00100 + 00101
// go run ./cmd/diag-lookback-schema/
// go run ./cmd/diag-lookback-schema/ -fix
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
ALTER TABLE member_lookback_settings
    DROP CONSTRAINT IF EXISTS chk_member_lookback_judgment;

ALTER TABLE member_lookback_settings
    ADD CONSTRAINT chk_member_lookback_judgment CHECK (
        judgment IN ('individual', 'overall', '')
    );

ALTER TABLE member_lookback_settings
    ADD COLUMN IF NOT EXISTS apply_formal BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE member_lookback_settings
    ADD COLUMN IF NOT EXISTS apply_sim BOOLEAN NOT NULL DEFAULT false;

UPDATE member_lookback_settings
SET apply_formal = (run_mode LIKE '%real%'),
    apply_sim    = (run_mode LIKE '%sim%')
WHERE apply_formal = false AND apply_sim = false AND run_mode <> '';

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
	fix := flag.Bool("fix", false, "补应用 00100 + 00101 迁移 SQL")
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
	cols, err := listColumns(ctx, pool, "member_lookback_settings")
	if err != nil {
		fmt.Println("schema:", err)
		os.Exit(1)
	}

	fmt.Println("member_lookback_settings columns:", strings.Join(cols, ", "))

	need := []string{"apply_formal", "apply_sim"}
	missing := missingCols(cols, need)
	if len(missing) == 0 {
		fmt.Println("ok: apply_formal / apply_sim 已存在，GET /client/cloud/lookback 不应因缺列而 500")
		return
	}

	fmt.Println("missing:", strings.Join(missing, ", "))
	fmt.Println("原因: 后端已升级至 simBet 回头，但库未执行 00100/00101 迁移")

	if !*fix {
		fmt.Println("\n修复: cd backend && go run ./cmd/diag-lookback-schema/ -fix")
		fmt.Println("或:   make migrate-up")
		os.Exit(2)
	}

	if _, err := pool.Exec(ctx, fixSQL); err != nil {
		fmt.Println("apply:", err)
		os.Exit(1)
	}
	fmt.Println("ok: 00100 + 00101 已应用，请重启后端并重试 GET /client/cloud/lookback")
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
