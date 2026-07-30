package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	defID := "def-1-1785384039382"
	rows, err := pool.Query(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_name='scheme_definitions' ORDER BY ordinal_position`)
	if err != nil {
		panic(err)
	}
	var cols []string
	for rows.Next() {
		var c string
		_ = rows.Scan(&c)
		cols = append(cols, c)
	}
	rows.Close()
	fmt.Println("DEF_COLS", strings.Join(cols, ","))

	var id, lottery, cfg string
	err = pool.QueryRow(ctx, `
		SELECT id, lottery_code, config::text
		FROM scheme_definitions WHERE id=$1`, defID).Scan(&id, &lottery, &cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println("DEF", id, lottery)
	if len(cfg) > 3000 {
		fmt.Println(cfg[:3000])
	} else {
		fmt.Println(cfg)
	}

	irows, err := pool.Query(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_name='scheme_instances' ORDER BY ordinal_position`)
	if err != nil {
		panic(err)
	}
	var icols []string
	for irows.Next() {
		var c string
		_ = irows.Scan(&c)
		icols = append(icols, c)
	}
	irows.Close()
	fmt.Println("INST_COLS", strings.Join(icols, ","))

	rows2, err := pool.Query(ctx, `
		SELECT id, status, status_reason, lottery_code
		FROM scheme_instances WHERE definition_id=$1
		ORDER BY updated_at DESC LIMIT 8`, defID)
	if err != nil {
		panic(err)
	}
	for rows2.Next() {
		var iid, st, sr, lc string
		_ = rows2.Scan(&iid, &st, &sr, &lc)
		fmt.Println("INST", iid, st, sr, lc)
	}
	rows2.Close()
}

func trim(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
