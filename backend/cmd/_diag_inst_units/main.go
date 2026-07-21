package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

func main() {
	_ = godotenv.Load(".env")
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	ctx := context.Background()
	id := "inst-1-1784618332941"
	if len(os.Args) > 1 {
		id = os.Args[1]
	}
	var defID, lottery, status, name string
	err = pool.QueryRow(ctx, `
SELECT definition_id, lottery_code, status, scheme_name FROM scheme_instances WHERE id=$1`, id).
		Scan(&defID, &lottery, &status, &name)
	if err != nil {
		panic(err)
	}
	fmt.Printf("inst=%s name=%s def=%s lottery=%s status=%s\n", id, name, defID, lottery, status)

	var defCfg []byte
	_ = pool.QueryRow(ctx, `SELECT config FROM scheme_definitions WHERE id=$1`, defID).Scan(&defCfg)
	var pretty map[string]interface{}
	_ = json.Unmarshal(defCfg, &pretty)
	for _, k := range []string{"runTypeId", "playTypeId", "subPlayId", "betMode", "schemeGroups", "fixedPick", "playMethod", "builtinPlan"} {
		if v, ok := pretty[k]; ok {
			b, _ := json.Marshal(v)
			fmt.Printf("%s: %s\n", k, string(b))
		}
	}

	rows, err := pool.Query(ctx, `
SELECT period_no, amount::float8, status, multiplier, COALESCE(bet_content,''), placed_at::text
FROM cloud_bet_records WHERE scheme_id=$1 ORDER BY placed_at ASC LIMIT 5`, id)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var period, st, mult, content, at string
		var amount float64
		_ = rows.Scan(&period, &amount, &st, &mult, &content, &at)
		i++
		fmt.Printf("#%d period=%s amount=%.4f status=%s mult=%s at=%s\ncontent=%q\nlines:\n", i, period, amount, st, mult, at, content)
		for li, ln := range strings.Split(strings.ReplaceAll(content, "\r", ""), "\n") {
			fmt.Printf("  L%d: %q\n", li+1, ln)
		}
	}
}
