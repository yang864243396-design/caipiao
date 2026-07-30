package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://caipiaoapp:123456@192.168.100.239:5432/caipiao?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	defID := "def-1-1785393275074"
	var id, lottery, name, cfg string
	err = pool.QueryRow(ctx, `SELECT id, lottery_code, scheme_name, config::text FROM scheme_definitions WHERE id=$1`, defID).Scan(&id, &lottery, &name, &cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println("DEF", id, lottery, name)
	fmt.Println(cfg)
	rows, err := pool.Query(ctx, `
		SELECT id, status, COALESCE(status_reason,''), COALESCE(bet_failed_detail,''), updated_at
		FROM scheme_instances WHERE definition_id=$1 ORDER BY updated_at DESC LIMIT 3`, defID)
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var iid, st, sr, detail string
		var upd time.Time
		_ = rows.Scan(&iid, &st, &sr, &detail, &upd)
		fmt.Printf("---INST %s status=%s reason=%s detail=%q upd=%s\n", iid, st, sr, detail, upd.Format(time.RFC3339))
	}
}
