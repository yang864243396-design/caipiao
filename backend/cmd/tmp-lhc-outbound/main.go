package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()
	pool, _ := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	defer pool.Close()
	rows, _ := pool.Query(ctx, `
SELECT type_id, sub_id, label, outbound_play_code, segment_rule::text
FROM sub_plays
WHERE template_code = 'lhc_std'
  AND sub_id IN ('tema_a', 'zheng1_te', 'zongxiao', 'tematouwei', 'qima')
ORDER BY type_id, sub_id`)
	defer rows.Close()
	for rows.Next() {
		var a, b, c, d, e string
		_ = rows.Scan(&a, &b, &c, &d, &e)
		fmt.Printf("%s/%s label=%s outbound=%s seg=%s\n", a, b, c, d, e)
	}
}
