package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji/rulessync"
)

func main() {
	apply := flag.Bool("apply", false, "写入数据库")
	tpl := flag.String("template", "", "仅同步指定 play_template（如 ssc_std）")
	lottery := flag.String("lottery", "", "预览指定彩种 play-tree 映射（如 taiwan_ssc_5m）")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()
	httpBase := cfg.Guaji.HTTPBase
	if httpBase == "" {
		httpBase = "https://www.v6hs1.com"
	}

	ctx := context.Background()
	rules, err := rulessync.FetchRulesV2(ctx, httpBase)
	if err != nil {
		fmt.Println("fetch rules/v2:", err)
		os.Exit(1)
	}
	fmt.Printf("rules/v2 玩法类型模板: %d 个 (base=%s)\n\n", len(rules), httpBase)

	bindings := rulessync.DefaultBindings
	if t := strings.TrimSpace(*tpl); t != "" {
		var filtered []rulessync.TemplateBinding
		for _, b := range bindings {
			if b.TemplateCode == t {
				filtered = append(filtered, b)
			}
		}
		if len(filtered) == 0 {
			fmt.Println("未知 template:", t)
			os.Exit(1)
		}
		bindings = filtered
	}

	var plans []rulessync.SyncPlan
	for _, b := range bindings {
		rt, ok := rules[b.GuajiRulesTypeID]
		if !ok {
			fmt.Printf("⚠ %s: rules/v2 无 type_id=%s\n", b.TemplateCode, b.GuajiRulesTypeID)
			continue
		}
		plan, err := rulessync.BuildPlan(b.TemplateCode, b.GuajiRulesTypeID, rt)
		if err != nil {
			fmt.Printf("⚠ %s: %v\n", b.TemplateCode, err)
			continue
		}
		plans = append(plans, plan)
		fmt.Printf("%s ↔ rules[%s] %s → play_types=%d sub_plays=%d\n",
			plan.TemplateCode, plan.GuajiRulesTypeID, plan.RulesTypeName, len(plan.PlayTypes), len(plan.SubPlays))
	}

	if code := strings.TrimSpace(*lottery); code != "" {
		printLotteryExample(ctx, cfg, code, plans)
	}

	if !*apply {
		fmt.Println("\n未写入（加 -apply 执行）")
		return
	}
	if len(plans) == 0 {
		os.Exit(1)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	for _, plan := range plans {
		if err := rulessync.ApplyPlan(ctx, pool, plan); err != nil {
			fmt.Println("apply", plan.TemplateCode, err)
			os.Exit(1)
		}
		fmt.Printf("已同步 %s: %d 玩法类型 / %d 子玩法\n", plan.TemplateCode, len(plan.PlayTypes), len(plan.SubPlays))
	}
}

func printLotteryExample(ctx context.Context, cfg config.Config, lotteryCode string, plans []rulessync.SyncPlan) {
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		return
	}
	defer pool.Close()

	var playTemplate, displayName string
	err = pool.QueryRow(ctx, `
		SELECT COALESCE(play_template,''), display_name
		FROM lottery_catalog WHERE code = $1`, lotteryCode).Scan(&playTemplate, &displayName)
	if err != nil {
		fmt.Printf("\n彩种 %s 未找到\n", lotteryCode)
		return
	}

	var plan *rulessync.SyncPlan
	for i := range plans {
		if plans[i].TemplateCode == playTemplate {
			plan = &plans[i]
			break
		}
	}
	if plan == nil {
		fmt.Printf("\n彩种 %s 使用 template=%s，当前未在同步列表\n", lotteryCode, playTemplate)
		return
	}

	fmt.Printf("\n=== 示例：%s (%s) ===\n", displayName, lotteryCode)
	fmt.Printf("彩种类型 (rules/v2 name): %s\n", plan.RulesTypeName)
	fmt.Printf("玩法类型数: %d | 子玩法数: %d\n\n", len(plan.PlayTypes), len(plan.SubPlays))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "玩法类型(group)\t子玩法(rule.name)\trule_id\tfull_name")
	for _, pt := range plan.PlayTypes {
		first := true
		for _, sp := range plan.SubPlays {
			if sp.TypeID != pt.TypeID {
				continue
			}
			full := ""
			var seg map[string]string
			if json.Unmarshal(sp.SegmentRule, &seg) == nil {
				full = seg["guajiFullName"]
			}
			groupLabel := pt.Label
			if !first {
				groupLabel = ""
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", groupLabel, sp.Label, sp.SubID, full)
			first = false
		}
	}
	w.Flush()

	// 前三码首组示例
	if len(plan.PlayTypes) > 0 {
		g := plan.PlayTypes[0]
		fmt.Printf("\n首组对照: groups[0].name=%q\n", g.Label)
		for _, sp := range plan.SubPlays {
			if sp.TypeID == g.TypeID {
				fmt.Printf("  team.rule → subId=%s label=%q outbound=%s\n", sp.SubID, sp.Label, sp.OutboundPlayCode)
			}
		}
	}
}
