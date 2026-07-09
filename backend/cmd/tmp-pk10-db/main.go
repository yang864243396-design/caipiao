package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sub_plays WHERE template_code='pk10_std'`).Scan(&n); err != nil {
		panic(err)
	}
	fmt.Println("pk10_std sub_plays count:", n)

	rows, err := pool.Query(ctx, `
SELECT sp.type_id, sp.sub_id, sp.label, COALESCE(sp.bet_mode,''), COALESCE(sp.outbound_play_code,''), COALESCE(sp.segment_rule::text,'')
FROM sub_plays sp
WHERE sp.template_code='pk10_std'
ORDER BY sp.type_id, sp.sub_id`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var t, s, l, b, o, seg string
		if err := rows.Scan(&t, &s, &l, &b, &o, &seg); err != nil {
			panic(err)
		}
		if s == "221" || s == "222" || s == "223" || s == "dxds_guanya" || s == "dxds_qian3" || s == "dxds_hou3" || t == "g010" {
			fmt.Printf("sub=%-12s type=%s label=%-16s outbound=%-6s mode=%s\n", s, t, l, o, b)
		}
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	_ = pgx.ErrNoRows
}
