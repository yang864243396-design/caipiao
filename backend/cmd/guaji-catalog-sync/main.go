package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji/catalogsync"
)

func main() {
	apply := flag.Bool("apply", false, "写入数据库（默认仅预览）")
	limit := flag.Int("limit", 299, "new_lott limit 参数")
	page := flag.Int("page", 1, "new_lott page 参数")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()
	httpBase := cfg.Guaji.HTTPBase
	if httpBase == "" {
		httpBase = "https://www.v6hs1.com"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	remote, err := catalogsync.FetchNewLott(ctx, httpBase, *limit, *page)
	if err != nil {
		fmt.Println("fetch new_lott:", err)
		os.Exit(1)
	}
	fmt.Printf("第三方彩种: %d 条 (base=%s)\n", len(remote), httpBase)

	rows, err := pool.Query(ctx, `
		SELECT code, display_name, COALESCE(outbound_lottery_code, '')
		FROM lottery_catalog
		ORDER BY sort_order, code`)
	if err != nil {
		fmt.Println("query local:", err)
		os.Exit(1)
	}
	defer rows.Close()

	var local []catalogsync.LocalLottery
	for rows.Next() {
		var row catalogsync.LocalLottery
		if err := rows.Scan(&row.Code, &row.DisplayName, &row.OutboundLotteryCode); err != nil {
			fmt.Println("scan:", err)
			os.Exit(1)
		}
		local = append(local, row)
	}
	fmt.Printf("本地彩种: %d 条\n\n", len(local))

	report := catalogsync.BuildMatchReport(local, remote)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CODE\t旧名称\t新名称\t旧outbound\t新outbound\tremote_id\t变更")
	for _, m := range report.Matched {
		mark := "-"
		if m.Changed {
			mark = "✓"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			m.Code, m.OldName, m.NewName, orDash(m.OldOutbound), m.NewOutbound, m.RemoteID, mark)
	}
	w.Flush()

	if len(report.Unmatched) > 0 {
		fmt.Printf("\n⚠ 本地未匹配 (%d):\n", len(report.Unmatched))
		for _, u := range report.Unmatched {
			fmt.Printf("  %s  %s  outbound=%s\n", u.Code, u.OldName, orDash(u.OldOutbound))
		}
	}
	if len(report.RemoteOnly) > 0 {
		fmt.Printf("\nℹ 第三方未纳入本地目录 (%d):\n", len(report.RemoteOnly))
		for _, r := range report.RemoteOnly {
			fmt.Printf("  id=%d  %s\n", r.ID, r.Name)
		}
	}

	changed := 0
	for _, m := range report.Matched {
		if m.Changed {
			changed++
		}
	}
	fmt.Printf("\n预览: 可更新 %d / 匹配 %d\n", changed, len(report.Matched))

	if !*apply {
		fmt.Println("\n未写入（加 -apply 执行更新）")
		return
	}
	if changed == 0 {
		fmt.Println("无需更新")
		return
	}

	n, err := catalogsync.Apply(ctx, pool, report)
	if err != nil {
		fmt.Println("apply:", err)
		os.Exit(1)
	}
	fmt.Printf("已更新 %d 条\n", n)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
