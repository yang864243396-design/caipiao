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
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	rows, err := pool.Query(ctx, `
SELECT sp.sub_id, sp.label, sp.outbound_play_code, COALESCE(pt.label,''), sp.segment_rule::text
FROM sub_plays sp
LEFT JOIN play_types pt ON pt.template_code=sp.template_code AND pt.type_id=sp.type_id
WHERE sp.template_code='lhc_std' AND sp.sub_id IN ('278','279','280','281','282','319','321','335','290','296')
ORDER BY sp.type_id, sp.sub_id`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var sub, label, mode, tl, seg string
		_ = rows.Scan(&sub, &label, &mode, &tl, &seg)
		fmt.Printf("sub=%s label=%s outbound=%s type=%s seg=%s\n", sub, label, mode, tl, seg)
	}
}
