// bnb_pk10_jisu 存量脏数据盘点：错线开奖行数、注单期号族、那笔 lose 是怎么结算的。
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
		fmt.Println("db:", err)
		return
	}
	defer pool.Close()

	fmt.Println("=== bnb_pk10_jisu 开奖行按期号族 ===")
	rows, err := pool.Query(ctx, `
SELECT left(issue_no, 5) AS fam, length(issue_no), count(*),
       min(to_char(drawn_at AT TIME ZONE 'Asia/Shanghai','MM-DD HH24:MI')),
       max(to_char(drawn_at AT TIME ZONE 'Asia/Shanghai','MM-DD HH24:MI'))
FROM lottery_draws WHERE lottery_code='bnb_pk10_jisu'
GROUP BY 1,2 ORDER BY 3 DESC`)
	if err != nil {
		fmt.Println("q1:", err)
		return
	}
	for rows.Next() {
		var fam, lo, hi string
		var n, ln int
		if err := rows.Scan(&fam, &ln, &n, &lo, &hi); err != nil {
			fmt.Println("scan:", err)
			break
		}
		fmt.Printf("  前缀=%-6s %2d位 %7d 行  %s ~ %s\n", fam, ln, n, lo, hi)
	}
	rows.Close()

	fmt.Println("\n=== bnb_pk10_jisu 注单期号族 × 状态 ===")
	rows2, err := pool.Query(ctx, `
SELECT left(issue_no,5), length(issue_no), status, count(*)
FROM bet_orders WHERE lottery_code='bnb_pk10_jisu'
GROUP BY 1,2,3 ORDER BY 4 DESC`)
	if err != nil {
		fmt.Println("q2:", err)
		return
	}
	for rows2.Next() {
		var fam, st string
		var ln, n int
		if err := rows2.Scan(&fam, &ln, &st, &n); err != nil {
			fmt.Println("scan:", err)
			break
		}
		fmt.Printf("  前缀=%-6s %2d位 %-10s %5d 笔\n", fam, ln, st, n)
	}
	rows2.Close()

	fmt.Println("\n=== 那笔 lose 注单 ===")
	rows3, err := pool.Query(ctx, `
SELECT o.id, o.issue_no, o.status, o.bet_amount, o.payout_amount,
       to_char(o.created_at AT TIME ZONE 'Asia/Shanghai','MM-DD HH24:MI:SS'),
       COALESCE(d.issue_no,'(无匹配开奖)'), COALESCE(d.balls::text,'')
FROM bet_orders o
LEFT JOIN lottery_draws d ON d.lottery_code=o.lottery_code AND d.issue_no=o.issue_no
WHERE o.lottery_code='bnb_pk10_jisu' AND o.status <> 'cancel'`)
	if err != nil {
		fmt.Println("q3:", err)
		return
	}
	for rows3.Next() {
		var id int64
		var issue, st, amt, payout, at, dIssue, balls string
		if err := rows3.Scan(&id, &issue, &st, &amt, &payout, &at, &dIssue, &balls); err != nil {
			fmt.Println("scan:", err)
			break
		}
		fmt.Printf("  #%d 期号=%s 状态=%s 投注=%s 返奖=%s 时间=%s\n    匹配开奖=%s %s\n",
			id, issue, st, amt, payout, at, dIssue, balls)
	}
	rows3.Close()

	fmt.Println("\n=== 云端注单（cloud_bet_records）===")
	var cn int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM cloud_bet_records WHERE lottery_code='bnb_pk10_jisu'`).Scan(&cn); err != nil {
		fmt.Println("q4:", err)
	} else {
		fmt.Printf("  %d 条\n", cn)
	}
}
