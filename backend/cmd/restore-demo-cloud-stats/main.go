// 一次性脚本：还原 vs8888 演示账号 scheme_instances 测试脏数据。
// 用法：go run ./cmd/restore-demo-cloud-stats/
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	account := cfg.ClientDemoAccount
	if account == "" {
		account = "vs8888"
	}
	tag, err := pool.Exec(context.Background(), `
UPDATE scheme_instances si
SET
    sim_bet = false,
    status = CASE WHEN si.scheme_name = '1111' THEN 'running' ELSE 'pending' END,
    turnover = 0,
    session_pnl = 0,
    updated_at = now()
FROM members m
WHERE si.member_id = m.id AND m.account = $1`, account)
	if err != nil {
		fmt.Println("restore:", err)
		os.Exit(1)
	}
	fmt.Printf("restored %d instances for %s\n", tag.RowsAffected(), account)
}
