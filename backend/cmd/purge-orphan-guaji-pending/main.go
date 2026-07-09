// 清理孤儿第三方 pending 注单（挂机账号已删除、无法派奖同步的矩阵测试残留）。
//
//	go run ./cmd/purge-orphan-guaji-pending              # 预览
//	go run ./cmd/purge-orphan-guaji-pending -apply       # 执行
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

const previewBetSQL = `
SELECT b.order_no, b.guaji_account_id, b.lottery_code, b.issue_no, b.placed_at
FROM bet_orders b
WHERE b.status = 'pending'
  AND b.guaji_account_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM member_guaji_accounts ga WHERE ga.id = b.guaji_account_id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM cloud_bet_records c
    JOIN scheme_instances si ON si.id = c.scheme_id AND si.status = 'running'
    WHERE c.bet_order_no = b.order_no
      AND c.status = 'pending'
  )
ORDER BY b.placed_at ASC
LIMIT 20`

const previewCloudSQL = `
SELECT c.record_no, c.scheme_id, c.guaji_account_id, c.bet_order_no, c.placed_at
FROM cloud_bet_records c
WHERE c.status = 'pending'
  AND NOT EXISTS (
    SELECT 1 FROM scheme_instances si WHERE si.id = c.scheme_id AND si.status = 'running'
  )
  AND (
    (
      c.guaji_account_id IS NOT NULL
      AND NOT EXISTS (SELECT 1 FROM member_guaji_accounts ga WHERE ga.id = c.guaji_account_id)
    )
    OR c.third_party_bet_id IS NULL
    OR TRIM(c.third_party_bet_id) = ''
  )
ORDER BY c.placed_at ASC
LIMIT 20`

const countBetSQL = `
SELECT COUNT(*)
FROM bet_orders b
WHERE b.status = 'pending'
  AND b.guaji_account_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM member_guaji_accounts ga WHERE ga.id = b.guaji_account_id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM cloud_bet_records c
    JOIN scheme_instances si ON si.id = c.scheme_id AND si.status = 'running'
    WHERE c.bet_order_no = b.order_no
      AND c.status = 'pending'
  )`

const countCloudSQL = `
SELECT COUNT(*)
FROM cloud_bet_records c
WHERE c.status = 'pending'
  AND NOT EXISTS (
    SELECT 1 FROM scheme_instances si WHERE si.id = c.scheme_id AND si.status = 'running'
  )
  AND (
    (
      c.guaji_account_id IS NOT NULL
      AND NOT EXISTS (SELECT 1 FROM member_guaji_accounts ga WHERE ga.id = c.guaji_account_id)
    )
    OR c.third_party_bet_id IS NULL
    OR TRIM(c.third_party_bet_id) = ''
  )`

const cancelBetSQL = `
UPDATE bet_orders b
SET status = 'cancel',
    settled_at = now(),
    updated_at = now()
WHERE b.status = 'pending'
  AND b.guaji_account_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM member_guaji_accounts ga WHERE ga.id = b.guaji_account_id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM cloud_bet_records c
    JOIN scheme_instances si ON si.id = c.scheme_id AND si.status = 'running'
    WHERE c.bet_order_no = b.order_no
      AND c.status = 'pending'
  )`

const deleteCloudSQL = `
DELETE FROM cloud_bet_records c
WHERE c.status = 'pending'
  AND NOT EXISTS (
    SELECT 1 FROM scheme_instances si WHERE si.id = c.scheme_id AND si.status = 'running'
  )
  AND (
    (
      c.guaji_account_id IS NOT NULL
      AND NOT EXISTS (SELECT 1 FROM member_guaji_accounts ga WHERE ga.id = c.guaji_account_id)
    )
    OR c.third_party_bet_id IS NULL
    OR TRIM(c.third_party_bet_id) = ''
  )`

func main() {
	_ = godotenv.Load()
	apply := flag.Bool("apply", false, "执行清理（默认仅预览）")
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	var betCount, cloudCount int
	if err := pool.QueryRow(ctx, countBetSQL).Scan(&betCount); err != nil {
		fmt.Println("count bets:", err)
		os.Exit(1)
	}
	if err := pool.QueryRow(ctx, countCloudSQL).Scan(&cloudCount); err != nil {
		fmt.Println("count cloud:", err)
		os.Exit(1)
	}

	var pendingBefore int
	_ = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM bet_orders
WHERE status = 'pending' AND guaji_account_id IS NOT NULL`).Scan(&pendingBefore)

	fmt.Printf("当前 pending 第三方注单: %d\n", pendingBefore)
	fmt.Printf("将撤单 bet_orders (cancel): %d\n", betCount)
	fmt.Printf("将删除 cloud_bet_records (pending 孤儿): %d\n", cloudCount)
	fmt.Println("保护: 跳过 running 方案仍 pending 的注单")

	fmt.Println("\n--- bet_orders 样例 (最多 20) ---")
	rows, err := pool.Query(ctx, previewBetSQL)
	if err != nil {
		fmt.Println("preview bets:", err)
		os.Exit(1)
	}
	for rows.Next() {
		var orderNo, lottery, issue string
		var guajiID int64
		var placedAt any
		if err := rows.Scan(&orderNo, &guajiID, &lottery, &issue, &placedAt); err != nil {
			rows.Close()
			fmt.Println("scan bets:", err)
			os.Exit(1)
		}
		fmt.Printf("  %s guaji=%d %s %s placed=%v\n", orderNo, guajiID, lottery, issue, placedAt)
	}
	rows.Close()

	fmt.Println("\n--- cloud_bet_records 样例 (最多 20) ---")
	rows2, err := pool.Query(ctx, previewCloudSQL)
	if err != nil {
		fmt.Println("preview cloud:", err)
		os.Exit(1)
	}
	for rows2.Next() {
		var recordNo, schemeID string
		var betOrderNo pgtype.Text
		var guajiID pgtype.Int8
		var placedAt any
		if err := rows2.Scan(&recordNo, &schemeID, &guajiID, &betOrderNo, &placedAt); err != nil {
			rows2.Close()
			fmt.Println("scan cloud:", err)
			os.Exit(1)
		}
		fmt.Printf("  %s scheme=%s guaji=%d order=%s placed=%v\n",
			recordNo, schemeID, guajiID.Int64, betOrderNo.String, placedAt)
	}
	rows2.Close()

	if !*apply {
		if betCount > 0 || cloudCount > 0 {
			fmt.Println("\n预览模式，未修改。执行: go run ./cmd/purge-orphan-guaji-pending -apply")
		}
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Println("begin:", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	cloudTag, err := tx.Exec(ctx, deleteCloudSQL)
	if err != nil {
		fmt.Println("delete cloud:", err)
		os.Exit(1)
	}
	betTag, err := tx.Exec(ctx, cancelBetSQL)
	if err != nil {
		fmt.Println("cancel bets:", err)
		os.Exit(1)
	}
	if err := tx.Commit(ctx); err != nil {
		fmt.Println("commit:", err)
		os.Exit(1)
	}

	var pendingAfter int
	_ = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM bet_orders
WHERE status = 'pending' AND guaji_account_id IS NOT NULL`).Scan(&pendingAfter)

	fmt.Printf("\n已删除 cloud pending: %d\n", cloudTag.RowsAffected())
	fmt.Printf("已撤单 bet_orders: %d\n", betTag.RowsAffected())
	fmt.Printf("pending 第三方注单: %d -> %d\n", pendingBefore, pendingAfter)
}
