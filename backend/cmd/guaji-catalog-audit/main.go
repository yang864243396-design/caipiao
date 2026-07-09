package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji/catalogsync"
)

// docIndex：generate_p0_seeds.py CATALOG 序号 → game_id（00087 文档假设，iyes.dev 见 catalogsync.IyesDevOutboundByCode）
var docIndex = catalogsync.DocOutboundByCode

type row struct {
	code, name, outbound, wsKey, saleStatus string
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	rows, err := pool.Query(context.Background(), `
		SELECT code, display_name,
		       COALESCE(outbound_lottery_code,''),
		       COALESCE(guaji_ws_key,''),
		       sale_status
		FROM lottery_catalog
		ORDER BY sort_order, code`)
	if err != nil {
		fmt.Println("query:", err)
		os.Exit(1)
	}
	defer rows.Close()

	var items []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.code, &r.name, &r.outbound, &r.wsKey, &r.saleStatus); err != nil {
			fmt.Println("scan:", err)
			os.Exit(1)
		}
		items = append(items, r)
	}

	fmt.Printf("平台彩种总数: %d (P0 种子 50 + 秒彩 3)\n\n", len(items))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CODE\t名称\tDB_outbound\tiyes.dev\t文档§8\tiyes一致?\tWS_KEY\t状态")
	var missingOutbound, iyesMismatch, docMismatch, noWsKey int
	for _, r := range items {
		doc := docIndex[r.code]
		iyes := catalogsync.IyesDevOutboundByCode[r.code]
		docStr := "-"
		iyesStr := "-"
		matchIyes := "-"
		if iyes > 0 {
			iyesStr = fmt.Sprintf("%d", iyes)
			if r.outbound == fmt.Sprintf("%d", iyes) {
				matchIyes = "✓"
			} else if r.outbound != "" {
				matchIyes = "✗"
				iyesMismatch++
			}
		}
		if doc > 0 {
			docStr = fmt.Sprintf("%d", doc)
			if r.outbound != "" && r.outbound != fmt.Sprintf("%d", doc) {
				docMismatch++
			}
		}
		if r.outbound == "" {
			matchIyes = "缺"
			missingOutbound++
		}
		if r.wsKey == "" {
			noWsKey++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.code, r.name, orDash(r.outbound), iyesStr, docStr, matchIyes, orDash(r.wsKey), r.saleStatus)
	}
	w.Flush()

	fmt.Printf("\n汇总:\n")
	fmt.Printf("  outbound 未配置: %d\n", missingOutbound)
	fmt.Printf("  outbound ≠ iyes.dev 实测: %d\n", iyesMismatch)
	fmt.Printf("  outbound ≠ 文档§8序号: %d\n", docMismatch)
	fmt.Printf("  guaji_ws_key 为空: %d\n", noWsKey)

	// 重复 outbound 检测
	type pair struct{ code, name string }
	byOutbound := map[string][]pair{}
	for _, r := range items {
		if r.outbound == "" {
			continue
		}
		byOutbound[r.outbound] = append(byOutbound[r.outbound], pair{r.code, r.name})
	}
	var dupes []string
	for gid, ps := range byOutbound {
		if len(ps) > 1 {
			var parts []string
			for _, p := range ps {
				parts = append(parts, p.code)
			}
			sort.Strings(parts)
			dupes = append(dupes, fmt.Sprintf("game_id=%s → %s", gid, strings.Join(parts, ", ")))
		}
	}
	sort.Strings(dupes)
	if len(dupes) > 0 {
		fmt.Printf("\n⚠ outbound 重复（多彩种共用同一 game_id）:\n")
		for _, d := range dupes {
			fmt.Println(" ", d)
		}
	}

	var total, numeric, empty int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM sub_plays WHERE enabled=true`).Scan(&total)
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM sub_plays WHERE enabled=true AND outbound_play_code ~ '^[0-9]+$'`).Scan(&numeric)
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM sub_plays WHERE enabled=true AND (outbound_play_code IS NULL OR outbound_play_code='')`).Scan(&empty)
	fmt.Printf("\n玩法 outbound：enabled=%d  numeric_rule_id=%d  composite=%d  empty=%d\n",
		total, numeric, total-numeric-empty, empty)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
