// scheme-audit：只读批量对账，把方案/注单的业务断言跑一遍并输出异常清单。
//
// 不写库、不调第三方。抓的是「能正常下注、能正常结算，但选出来的号或算出来的钱在业务上是错的」
// 这一类——这类问题不会报错，只能靠断言体检翻出来。
//
// 用法：
//
//	go run ./cmd/scheme-audit                    # 近 30 天，报告写到 reports/
//	go run ./cmd/scheme-audit -days 7 -limit 5000
//	go run ./cmd/scheme-audit -ci                # 有 P0/P1 则 exit 1
//
// 环境：DATABASE_URL。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
)

func main() {
	days := flag.Int("days", 30, "回溯天数")
	limit := flag.Int("limit", 50000, "注单扫描上限")
	maxUnits := flag.Int("max-units", 0, "注数上限；0=按玩法自身上限")
	minSample := flag.Int("min-sample", 10, "彩种期号命中率断言所需的最小样本量")
	recent := flag.Duration("recent", 24*time.Hour,
		"近窗口：报告的「近窗口」列与彩种期号断言都只看这段时间。修完映射想立刻验证就调小")
	outDir := flag.String("out", "reports", "报告输出目录")
	ciMode := flag.Bool("ci", false, "CI 模式：存在 P0/P1 则 exit 1")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal("连接数据库失败: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		fatal("数据库不可达: %v", err)
	}

	now := time.Now()
	c := newCollector()

	epochs, err := loadColumnEpochs(ctx, pool)
	if err != nil {
		fatal("探测列启用时间失败: %v", err)
	}
	bets, err := loadBets(ctx, pool, *days, *limit)
	if err != nil {
		fatal("读取注单失败: %v", err)
	}
	tally := auditBets(c, bets, now, *recent, *maxUnits, epochs)
	auditLotteryPeriods(c, tally, *minSample, *recent)

	schemeRows, err := loadSchemes(ctx, pool)
	if err != nil {
		fatal("读取方案失败: %v", err)
	}
	auditSchemes(c, schemeRows, *maxUnits)

	stamp := now.Format("20060102-150405")
	jsonlPath := filepath.Join(*outDir, "scheme-audit-"+stamp+".jsonl")
	mdPath := filepath.Join(*outDir, "scheme-audit-"+stamp+".md")
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal("创建输出目录失败: %v", err)
	}
	if err := writeJSONL(jsonlPath, c.items); err != nil {
		fatal("写 JSONL 失败: %v", err)
	}
	report := buildReport(c, *days, *recent, len(bets), len(schemeRows), now, epochs)
	if err := os.WriteFile(mdPath, []byte(report), 0o644); err != nil {
		fatal("写报告失败: %v", err)
	}

	fmt.Print(report)
	fmt.Printf("\n明细：%s\n报告：%s\n", jsonlPath, mdPath)

	if *ciMode && c.hasBlocking() {
		os.Exit(1)
	}
}

func writeJSONL(path string, items []Finding) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, it := range items {
		if err := enc.Encode(it); err != nil {
			return err
		}
	}
	return nil
}

func buildReport(
	c *collector, days int, recent time.Duration, betCount, schemeCount int,
	now time.Time, epochs columnEpochs,
) string {
	var b strings.Builder
	sev := c.countBySeverity()
	recentSince := now.Add(-recent)

	fmt.Fprintf(&b, "# 方案 / 注单对账报告\n\n")
	fmt.Fprintf(&b, "- 生成时间：%s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- 扫描范围：近 %d 天，注单 %d 笔、方案 %d 个、彩种 %d 种\n",
		days, betCount, schemeCount, c.scanned["lottery"])
	fmt.Fprintf(&b, "- 异常合计：%d（P0 %d / P1 %d / P2 %d / P3 %d）\n",
		len(c.items), sev[P0], sev[P1], sev[P2], sev[P3])
	fmt.Fprintf(&b, "- 后加列启用时间：bet_units %s、payout_amount %s（更早的行不参与这两项检查）\n\n",
		fmtEpoch(epochs.BetUnits), fmtEpoch(epochs.PayoutAmount))

	if len(c.items) == 0 {
		b.WriteString("未发现异常。\n")
	} else {
		fmt.Fprintf(&b, "「近窗口」= 最近 %s，区分历史遗留与当下仍在发生；"+
			"方案 / 彩种级检查无时间维度，计入当前状态。\n\n", recent)
		b.WriteString("## 按检查项汇总\n\n")
		b.WriteString("| 级别 | 检查项 | 命中数 | 近窗口 | 样例 |\n|---|---|---:|---:|---|\n")
		for _, s := range c.summarize(recentSince) {
			fmt.Fprintf(&b, "| %s | %s | %d | %d | %s |\n",
				s.Severity, s.Check, s.Count, s.Recent, mdEscape(truncate(s.Sample, 90)))
		}
		b.WriteString("\n")
	}

	if len(c.skipped) > 0 {
		b.WriteString("## 未覆盖（缺输入而跳过，不代表无问题）\n\n")
		keys := make([]string, 0, len(c.skipped))
		for k := range c.skipped {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "- %s：%d 次\n", k, c.skipped[k])
		}
		b.WriteString("\n")
	}

	b.WriteString("## 级别含义\n\n")
	b.WriteString("- P0 资金或输赢已经算错\n")
	b.WriteString("- P1 投注内容非法，会被第三方拒单或静默错投\n")
	b.WriteString("- P2 结果暂时还对，但选号 / 状态推导已经偏了\n")
	b.WriteString("- P3 展示瑕疵\n")
	return b.String()
}

func fmtEpoch(t time.Time) string {
	if t.IsZero() || t.Year() > 9000 {
		return "尚无数据"
	}
	return t.Local().Format("2006-01-02 15:04")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func mdEscape(s string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(s)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
