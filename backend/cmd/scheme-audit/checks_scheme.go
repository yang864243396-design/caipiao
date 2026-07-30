package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"caipiao/backend/internal/schemes"
)

type schemeRow struct {
	ID          string
	MemberID    int64
	Kind        string
	LotteryCode string
	Status      string
	SimBet      bool
	Config      string
}

const schemeQuery = `
SELECT si.id,
       si.member_id,
       COALESCE(si.kind, ''),
       COALESCE(si.lottery_code, ''),
       COALESCE(si.status, ''),
       si.sim_bet,
       COALESCE(sd.config::text, '')
FROM scheme_instances si
LEFT JOIN scheme_definitions sd ON sd.id = si.definition_id
ORDER BY si.id`

func loadSchemes(ctx context.Context, pool *pgxpool.Pool) ([]schemeRow, error) {
	rows, err := pool.Query(ctx, schemeQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []schemeRow
	for rows.Next() {
		var s schemeRow
		if err := rows.Scan(&s.ID, &s.MemberID, &s.Kind, &s.LotteryCode,
			&s.Status, &s.SimBet, &s.Config); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func auditSchemes(c *collector, rows []schemeRow, maxUnits int) {
	for _, s := range rows {
		if strings.TrimSpace(s.Config) == "" {
			c.skip("scheme_config_missing")
			continue
		}
		c.scanned["scheme"]++
		cfg := []byte(s.Config)

		digest, ok := schemes.DigestScheme(s.Kind, cfg)
		if !ok {
			c.skip("scheme_digest")
			continue
		}
		play := digest.PlayLabel
		if play == "" {
			play = digest.PlayTypeID
		}
		if schemes.UniverseKindForScheme(s.Kind, cfg) == "" {
			c.skip("scheme（玩法内容形态未知）")
			continue
		}

		// 理论可达性：候选宇宙里频次恒为 0 的选项，取冷号会稳定押中
		if bad := schemes.UnreachableHotColdOptions(s.Kind, cfg); len(bad) > 0 {
			c.add(schemeFinding(P2, "hotcold_unreachable", s, play,
				fmt.Sprintf("冷热候选含 %d 个永不开出的选项：%s",
					len(bad), strings.Join(clip(bad, 12), ","))))
		}

		// 冷热分流：属性玩法与按位玩法用的是两套计频对象，走错就是统计错东西
		if actual, expected := schemes.HotColdRouting(s.Kind, cfg); actual != "" && actual != expected {
			why := "统计的是原始球号频次，而非该玩法的属性取值频次"
			if actual == "attribute" {
				why = "候选宇宙是一组固定文字选项，不是该玩法能下的内容"
			}
			c.add(schemeFinding(P2, "hotcold_routing", s, play,
				fmt.Sprintf("冷热计频走 %s 分支、应为 %s（betMode=%s）：%s",
					actual, expected, digest.BetMode, why)))
		}

		// 号池：默认 0-9 兜底会让 PK10/11 选 5/快三/六合彩静默降级
		if want, ok := expectedPool(digest.PlayTemplate); ok &&
			(digest.PoolMin != want[0] || digest.PoolMax != want[1]) {
			c.add(schemeFinding(P1, "pool_range", s, play,
				fmt.Sprintf("%s 号池应为 %d-%d，实际解析为 %d-%d",
					digest.PlayTemplate, want[0], want[1], digest.PoolMin, digest.PoolMax)))
		}

		// 方案保存下来的投注内容也要过一遍合法投注空间。
		// 随机 / 冷热出号会在下注时另行生成内容覆盖它，但保存的内容本身非法说明
		// 配置校验有缺口——正是"测试脚本随手填个数字"这类问题的来源。
		if content := strings.TrimSpace(digest.GroupContent); content != "" {
			for _, v := range schemes.ValidateSchemeBetContent(s.Kind, cfg, content, maxUnits) {
				c.add(schemeFinding(P1, "config_"+v.Code, s, play,
					fmt.Sprintf("%s（runType=%s、内容 %q）", v.Detail, digest.RunTypeID, clipStr(content, 40))))
			}
		}
	}
}

// expectedPool 各彩种模板的号池；未列出的模板不做断言。
func expectedPool(template string) ([2]int, bool) {
	switch strings.TrimSpace(template) {
	case "pk10_std":
		return [2]int{1, 10}, true
	case "syxw_std":
		return [2]int{1, 11}, true
	case "k3_std":
		return [2]int{1, 6}, true
	case "lhc_std":
		return [2]int{1, 49}, true
	case "ssc_std", "fast_ssc_std", "pc28_std":
		return [2]int{0, 9}, true
	}
	return [2]int{}, false
}

// auditLotteryPeriods 某彩种下注期号与开奖期号完全对不上，说明两条链路取的是不同期号族。
// 极速彩 game_id 对调那个 bug 正是这个形状：能下注、能结算，但期号永远查不到。
func auditLotteryPeriods(
	c *collector, tally map[string]*lotteryTally, minSample int, recent time.Duration,
) {
	codes := make([]string, 0, len(tally))
	for code := range tally {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	for _, code := range codes {
		t := tally[code]
		// 只对近一天的注单下结论：整窗命中率会被迁移前的历史坏数据长期压低，
		// 把"已修好"误报成"还坏着"。整窗数字只作为背景写进 detail。
		if t.RecentTotal < minSample {
			// 带上彩种码：这条断言被挂起意味着一个 P0 检查当前没在跑，
			// 报告里必须看得出是哪个彩种（改完配置后样本重新积累时会自动恢复）。
			c.skip("period_family（近窗口样本不足）：" + code)
			continue
		}
		c.scanned["lottery"]++
		rate := float64(t.RecentDrawFound) / float64(t.RecentTotal)
		if rate >= 0.5 {
			continue
		}
		sev, check := P1, "period_family_partial"
		if t.RecentDrawFound == 0 {
			sev, check = P0, "period_family_mismatch"
		}
		c.add(Finding{
			Severity: sev, Check: check, Scope: "lottery", Key: code, Lottery: code,
			Detail: fmt.Sprintf(
				"近 %s 内 %d/%d 笔注单查得到开奖号（%.0f%%），下注链路与开奖链路期号族不一致；整窗 %d/%d",
				recent, t.RecentDrawFound, t.RecentTotal, rate*100, t.DrawFound, t.Total),
		})
	}
}

func schemeFinding(sev, check string, s schemeRow, play, detail string) Finding {
	return Finding{
		Severity: sev, Check: check, Scope: "scheme", Key: s.ID,
		Lottery: s.LotteryCode, Play: play, SimBet: s.SimBet, Detail: detail,
	}
}

func clip(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return append(append([]string{}, in[:n]...), "…")
}

func clipStr(s string, n int) string {
	r := []rune(strings.ReplaceAll(s, "\n", "|"))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
