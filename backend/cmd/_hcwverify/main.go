// 验证指定冷热方案近 N 期实下注是否等于引擎按「近 totalPeriods 期」重算的热/冷整区。
//
//	go run ./cmd/_hcwverify -def def-1-1785294381718 -n 10
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

func main() {
	_ = godotenv.Load()
	defID := flag.String("def", "def-1-1785294381718", "scheme definition id")
	n := flag.Int("n", 10, "recent bet periods to check")
	flag.Parse()

	cfg := config.Load()
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL / DB_* 未配置")
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	var (
		name, lottery, kind string
		raw                 []byte
	)
	err = pool.QueryRow(ctx, `
SELECT scheme_name, lottery_code, COALESCE(kind,'custom'), config
FROM scheme_definitions WHERE id=$1`, *defID).Scan(&name, &lottery, &kind, &raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load definition:", err)
		os.Exit(1)
	}

	var pretty map[string]any
	_ = json.Unmarshal(raw, &pretty)
	hcw, _ := pretty["hotColdWarm"].(map[string]any)
	hcwJSON, _ := json.MarshalIndent(hcw, "", "  ")
	fmt.Printf("方案 %s  %s  lottery=%s  kind=%s\n", *defID, name, lottery, kind)
	fmt.Printf("runTypeId=%v\n", pretty["runTypeId"])
	fmt.Printf("hotColdWarm:\n%s\n\n", string(hcwJSON))

	// 探测期望期数（与引擎默认一致）
	_, periodsHint, ok := schemes.RebuildHotColdPickContent(kind, raw, nil)
	if !ok {
		fmt.Fprintln(os.Stderr, "非冷热方案或配置缺少 hotColdWarm")
		os.Exit(1)
	}

	rows, err := pool.Query(ctx, `
SELECT c.period_no, COALESCE(c.bet_content,''), COALESCE(c.status,''), c.placed_at
FROM cloud_bet_records c
JOIN scheme_instances si ON si.id = c.scheme_id
WHERE si.definition_id = $1
ORDER BY c.placed_at DESC
LIMIT $2`, *defID, *n)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list bets:", err)
		os.Exit(1)
	}
	defer rows.Close()

	type betRow struct {
		period  string
		content string
		status  string
	}
	var bets []betRow
	for rows.Next() {
		var b betRow
		var placed any
		if err := rows.Scan(&b.period, &b.content, &b.status, &placed); err != nil {
			panic(err)
		}
		bets = append(bets, b)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	if len(bets) == 0 {
		fmt.Println("无注单")
		os.Exit(1)
	}

	okCount := 0
	failCount := 0
	for i, b := range bets {
		draws := loadDrawsBefore(ctx, pool, lottery, b.period, periodsHint)
		expected, periods, ok := schemes.RebuildHotColdPickContent(kind, raw, draws)
		if !ok {
			fmt.Printf("[%d] period=%s  ERROR: rebuild failed\n", i+1, b.period)
			failCount++
			continue
		}
		got := normalizeContent(b.content)
		want := normalizeContent(expected)
		match := got == want
		mark := "OK"
		if !match {
			mark = "FAIL"
			failCount++
		} else {
			okCount++
		}
		fmt.Printf("── %s  period=%s  status=%s  statsN=%d drawsUsed=%d\n", mark, b.period, b.status, periods, len(draws))
		fmt.Printf("   实下:\n%s\n", indent(got))
		fmt.Printf("   应下:\n%s\n", indent(want))
		if !match {
			fmt.Printf("   diff:\n%s\n", diffLines(got, want))
		}
	}
	fmt.Printf("\n合计: OK=%d FAIL=%d / %d\n", okCount, failCount, len(bets))
	if failCount > 0 {
		os.Exit(2)
	}
}

// loadDrawsBefore 对齐 worker.recentDrawBalls：按 drawn_at 倒序取当期之前最近 N 期。
func loadDrawsBefore(ctx context.Context, pool *pgxpool.Pool, lottery, currentIssue string, periods int) [][]string {
	if periods <= 0 {
		periods = 20
	}
	rows, err := pool.Query(ctx, `
SELECT issue_no, balls
FROM lottery_draws
WHERE lottery_code = $1
ORDER BY drawn_at DESC, id DESC
LIMIT $2`, lottery, periods+8)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([][]string, 0, periods)
	for rows.Next() {
		var issue string
		var ballsRaw []byte
		if err := rows.Scan(&issue, &ballsRaw); err != nil {
			continue
		}
		// 与 worker.recentDrawBalls 相同过滤（同族期号字符串可比）
		if currentIssue != "" && (issue == currentIssue || issue >= currentIssue) {
			continue
		}
		balls := sqlcdb.ParseDrawBalls(ballsRaw)
		if len(balls) == 0 {
			continue
		}
		out = append(out, balls)
		if len(out) >= periods {
			break
		}
	}
	return out
}

func normalizeContent(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		toks := splitTokens(ln)
		sort.SliceStable(toks, func(a, b int) bool {
			na, ea := strconv.Atoi(toks[a])
			nb, eb := strconv.Atoi(toks[b])
			if ea == nil && eb == nil {
				return na < nb
			}
			return toks[a] < toks[b]
		})
		lines[i] = strings.Join(toks, ",")
	}
	return strings.Join(lines, "\n")
}

func splitTokens(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func indent(s string) string {
	if s == "" {
		return "   (empty)"
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = "   | " + lines[i]
	}
	return strings.Join(lines, "\n")
}

func diffLines(got, want string) string {
	ga := strings.Split(got, "\n")
	wa := strings.Split(want, "\n")
	n := len(ga)
	if len(wa) > n {
		n = len(wa)
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		g, w := "", ""
		if i < len(ga) {
			g = ga[i]
		}
		if i < len(wa) {
			w = wa[i]
		}
		if g == w {
			b.WriteString(fmt.Sprintf("   | pos%d OK  %s\n", i, g))
		} else {
			b.WriteString(fmt.Sprintf("   | pos%d MISMATCH  got=%q want=%q\n", i, g, w))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
