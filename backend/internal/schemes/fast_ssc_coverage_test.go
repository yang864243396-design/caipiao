package schemes

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
)

// fast_ssc_std 此前零覆盖：它服务波场 3/6/15 秒彩三个在售彩种，
// 但子玩法只存在于 DB 迁移里，docs/seeds/sub_plays.csv 一条都没有。
//
// 更要紧的是编号体系对不上：种子 CSV 用语义 id（dingwei/dingwei_ge），
// 库里和生产配置用第三方数字 id（g006/13）。ssc_coverage_test.go 那 175 条遍历
// 走的全是语义 id，等于没覆盖到线上真正在用的那套编号。
//
// 所以这里全部从库和生产配置取数：
//   - 子玩法清单来自 sub_plays（该模板的唯一事实来源）
//   - betMode 来自 scheme_definitions 里真实存在过的配置
//     （sub_plays.bet_mode 是 NULL，生产的 betMode 由前端选玩法时写进方案配置，
//      resolveSSCPlayRule 只是把它原样收下——见 ssc_play_resolver.go 第 21 行）

// hashPlayBetModes g017 哈希玩法的内容形态。
// 这 4 个是 fast_ssc_std 独有的，生产里未必有人配过，兜底显式写死。
var hashPlayBetModes = map[string]string{
	"387": "danshuang",  // 尾数单双
	"388": "zhuangxian", // 幸运庄闲（庄/和/闲，单选）
	"389": "danshuang",  // 和值单双
	"390": "daxiao",     // 尾数大小
}

// 组选4的覆盖样例必须是「三重号,单号」双区内容；扁平号码池会被算成 0 注。
func TestFastSSCCoverageSampleForZu4HasBetUnits(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, typeID, subID string
	}{
		{name: "四星组选4", typeID: "g013", subID: "133"},
		{name: "前后四组选4", typeID: "g014", subID: "140"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := playRule{
				PlayTemplate: "fast_ssc_std",
				PlayTypeID:   tc.typeID,
				SubPlayID:    tc.subID,
				BetMode:      "zu4",
				SegmentLen:   4,
			}
			ev := evaluatePlayHit(rule, []string{"1", "3", "5", "7", "9"}, sampleSSCContent(rule), false, "", rule.PositionIdx)
			if ev.BetUnits <= 0 {
				t.Fatalf("sample=%q betUnits=%d, want a valid 组选4 dual-zone sample", sampleSSCContent(rule), ev.BetUnits)
			}
		})
	}
}

// TestFastSSCSubPlayCoverage 遍历 fast_ssc_std 全部启用子玩法，
// 逐个断言号池解析得出、验奖算得出正注数。
func TestFastSSCSubPlayCoverage(t *testing.T) {
	env := newSubPlayEnv(t)
	rows := env.subPlays("fast_ssc_std")
	// 132 条是当前迁移的规模；掉到 100 以下说明同步把玩法整批冲掉了
	if len(rows) < 100 {
		t.Fatalf("fast_ssc_std 子玩法只剩 %d 条，疑似被同步冲掉", len(rows))
	}

	balls := []string{"1", "3", "5", "7", "9"}
	var unresolved, zeroUnits, noOracle []string
	modes := map[string]int{}
	for _, sp := range rows {
		betMode, ok := env.betModeFor("fast_ssc_std", sp)
		if !ok {
			noOracle = append(noOracle, sp.String())
			continue
		}
		modes[betMode]++
		rule, ok := resolveCatalogPlayRule(fastSSCConfig(sp, betMode))
		if !ok {
			unresolved = append(unresolved, sp.String())
			continue
		}
		ev := evaluatePlayHit(rule, balls, sampleSSCContent(rule), false, "", rule.PositionIdx)
		if ev.BetUnits <= 0 {
			zeroUnits = append(zeroUnits, fmt.Sprintf("%s betMode=%s segLen=%d",
				sp.String(), rule.BetMode, rule.SegmentLen))
		}
	}
	if len(unresolved) > 0 {
		t.Errorf("号池解析不出的子玩法 %d 个：\n  %s",
			len(unresolved), strings.Join(unresolved, "\n  "))
	}
	if len(zeroUnits) > 0 {
		t.Errorf("验奖算出零注的子玩法 %d 个：\n  %s",
			len(zeroUnits), strings.Join(zeroUnits, "\n  "))
	}
	// 没有 oracle 的是从没被配置过的玩法，只报不判：
	// 数量突然变大说明玩法同步引入了一批线上没人用过的新 id，值得看一眼。
	if len(noOracle) > 0 {
		t.Logf("无历史配置可参照、本轮跳过的子玩法 %d 个：\n  %s",
			len(noOracle), strings.Join(noOracle, "\n  "))
	}
	if covered := len(rows) - len(noOracle); covered < len(rows)*2/3 {
		t.Errorf("只覆盖到 %d/%d 个子玩法，可参照的历史配置太少", covered, len(rows))
	}
	logSortedCounts(t, "betMode", modes)
}

// TestFastSSCMatchesSSCForSharedSubPlays 共享子玩法在两个模板下必须解析出同一套规则。
//
// 今天这条必然成立——resolveSSCPlayRule 把 PlayTemplate 硬编码成了 "ssc_std"，
// 两个模板走的是同一段代码。正因为这个前提没写在任何地方，
// 才需要一条测试把它钉住：一旦有人给某个模板加特判，这里会立刻炸出来。
func TestFastSSCMatchesSSCForSharedSubPlays(t *testing.T) {
	env := newSubPlayEnv(t)
	fast := env.subPlays("fast_ssc_std")
	sscKeys := map[string]bool{}
	for _, sp := range env.subPlays("ssc_std") {
		sscKeys[sp.key()] = true
	}

	var diverged []string
	shared := 0
	for _, sp := range fast {
		if !sscKeys[sp.key()] {
			continue // g017 哈希玩法：ssc_std 没有，无从比对
		}
		betMode, ok := env.betModeFor("fast_ssc_std", sp)
		if !ok {
			continue
		}
		shared++
		fastRule, okFast := resolveCatalogPlayRule(fastSSCConfig(sp, betMode))
		sscCfg := fastSSCConfig(sp, betMode)
		sscCfg["playTemplate"] = "ssc_std"
		sscRule, okSSC := resolveCatalogPlayRule(sscCfg)
		if okFast != okSSC {
			diverged = append(diverged, fmt.Sprintf("%s 解析成功与否不一致：fast=%v ssc=%v",
				sp.String(), okFast, okSSC))
			continue
		}
		if !reflect.DeepEqual(fastRule, sscRule) {
			diverged = append(diverged, fmt.Sprintf("%s 规则不一致：\n    fast=%+v\n    ssc =%+v",
				sp.String(), fastRule, sscRule))
		}
	}
	if shared < 80 {
		t.Fatalf("只比对到 %d 个共享子玩法，预期 120 上下；玩法集合可能变了", shared)
	}
	if len(diverged) > 0 {
		t.Errorf("两模板解析结果不一致的子玩法 %d 个：\n  %s",
			len(diverged), strings.Join(diverged, "\n  "))
	}
}

// TestFastSSCHashPlaysAreAttribute g017 哈希玩法是 fast_ssc_std 独有的，
// 它们都是属性盘（单双 / 大小 / 庄闲），不能被当成按位选号。
func TestFastSSCHashPlaysAreAttribute(t *testing.T) {
	env := newSubPlayEnv(t)
	seen := 0
	for _, sp := range env.subPlays("fast_ssc_std") {
		if sp.TypeID != "g017" {
			continue
		}
		seen++
		betMode, ok := hashPlayBetModes[sp.SubID]
		if !ok {
			t.Errorf("%s 未登记预期形态", sp.String())
			continue
		}
		u, ok := UniverseForScheme("custom", mustCfgJSON(t, fastSSCConfig(sp, betMode)))
		if !ok {
			t.Errorf("%s 推不出合法投注空间", sp.String())
			continue
		}
		if u.Kind != "attribute" {
			t.Errorf("%s 内容形态为 %s，应为 attribute（选项 %v）", sp.String(), u.Kind, u.Tokens)
		}
		if len(u.Tokens) < 2 {
			t.Errorf("%s 属性选项只有 %v，至少应有两个对立选项", sp.String(), u.Tokens)
		}
	}
	if seen != len(hashPlayBetModes) {
		t.Errorf("库里 g017 哈希玩法 %d 个，登记了 %d 个", seen, len(hashPlayBetModes))
	}
}

// TestFastSSCUniverseWellFormed 每个子玩法的合法投注空间都要能推出来，且号池非空。
// 号池 fallback 静默降级正是这类 bug 的形状：能下注、能结算，只是选出来的号是错的。
func TestFastSSCUniverseWellFormed(t *testing.T) {
	env := newSubPlayEnv(t)
	var bad []string
	kinds := map[string]int{}
	for _, sp := range env.subPlays("fast_ssc_std") {
		betMode, ok := env.betModeFor("fast_ssc_std", sp)
		if !ok {
			continue
		}
		u, ok := UniverseForScheme("custom", mustCfgJSON(t, fastSSCConfig(sp, betMode)))
		if !ok {
			// 推不出来只记形态：teshu 这类本来就存在已知信息缺口，见 scheme-audit.md
			kinds["<unknown>"]++
			continue
		}
		kinds[u.Kind]++
		if len(u.Tokens) == 0 {
			bad = append(bad, fmt.Sprintf("%s kind=%s 号池为空", sp.String(), u.Kind))
		}
	}
	if len(bad) > 0 {
		t.Errorf("投注空间号池为空的子玩法 %d 个：\n  %s",
			len(bad), strings.Join(bad, "\n  "))
	}
	logSortedCounts(t, "内容形态", kinds)
}

func fastSSCConfig(sp templateSubPlay, betMode string) map[string]interface{} {
	return map[string]interface{}{
		"playTemplate": "fast_ssc_std",
		"typeId":       sp.TypeID,
		"subId":        sp.SubID,
		"betMode":      betMode,
		"playMethod":   sp.Team,
	}
}

func mustCfgJSON(t *testing.T, cfg map[string]interface{}) []byte {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

type templateSubPlay struct {
	TypeID, SubID, Label, Team string
}

func (s templateSubPlay) key() string { return s.TypeID + "/" + s.SubID }

func (s templateSubPlay) String() string {
	return fmt.Sprintf("%s/%s %s", s.TypeID, s.SubID, s.Label)
}

// subPlayEnv 子玩法清单 + betMode oracle 的取数入口。
type subPlayEnv struct {
	pool   *pgxpool.Pool
	oracle map[string]string
}

func newSubPlayEnv(t *testing.T) *subPlayEnv {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(pool.Close)
	env := &subPlayEnv{pool: pool}
	env.loadBetModeOracle(t)
	return env
}

// loadBetModeOracle 从历史方案配置里取 (playTemplate/typeId/subId) → betMode。
//
// 同一个玩法对应多个 betMode 说明有人存进了自相矛盾的配置，
// 那么后续所有以它为前提的断言都不可信，直接判失败。
func (e *subPlayEnv) loadBetModeOracle(t *testing.T) {
	t.Helper()
	rows, err := e.pool.Query(context.Background(), `
SELECT config->>'playTemplate', config->>'typeId', config->>'subId', config->>'betMode', count(*)::int
FROM scheme_definitions
WHERE config->>'playTemplate' IN ('ssc_std', 'fast_ssc_std')
  AND coalesce(config->>'typeId', '') <> ''
  AND coalesce(config->>'subId', '') <> ''
  AND coalesce(config->>'betMode', '') <> ''
GROUP BY 1, 2, 3`)
	if err != nil {
		t.Skipf("读历史方案配置失败：%v", err)
	}
	defer rows.Close()

	e.oracle = map[string]string{}
	conflicts := map[string][]string{}
	for rows.Next() {
		var template, typeID, subID, betMode string
		var n int
		if err := rows.Scan(&template, &typeID, &subID, &betMode, &n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		key := template + "/" + typeID + "/" + subID
		if prev, ok := e.oracle[key]; ok && prev != betMode {
			conflicts[key] = append(conflicts[key], prev, betMode)
			continue
		}
		e.oracle[key] = betMode
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(conflicts) > 0 {
		var lines []string
		for k, v := range conflicts {
			lines = append(lines, fmt.Sprintf("%s → %s", k, strings.Join(v, " / ")))
		}
		sort.Strings(lines)
		t.Fatalf("同一玩法存在互相矛盾的 betMode 配置 %d 处：\n  %s",
			len(conflicts), strings.Join(lines, "\n  "))
	}
}

// betModeFor 该子玩法的 betMode：先查历史配置，g017 哈希玩法回退到登记表。
func (e *subPlayEnv) betModeFor(template string, sp templateSubPlay) (string, bool) {
	if mode, ok := e.oracle[template+"/"+sp.key()]; ok {
		return mode, true
	}
	mode, ok := hashPlayBetModes[sp.SubID]
	return mode, ok
}

func (e *subPlayEnv) subPlays(template string) []templateSubPlay {
	rows, err := e.pool.Query(context.Background(), `
SELECT type_id, sub_id, label, coalesce(segment_rule->>'guajiTeam', '')
FROM sub_plays
WHERE template_code = $1 AND enabled
ORDER BY type_id, sort_order`, template)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var out []templateSubPlay
	for rows.Next() {
		var sp templateSubPlay
		if err := rows.Scan(&sp.TypeID, &sp.SubID, &sp.Label, &sp.Team); err != nil {
			panic(err)
		}
		out = append(out, sp)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return out
}

func logSortedCounts(t *testing.T, label string, counts map[string]int) {
	t.Helper()
	names := make([]string, 0, len(counts))
	for k := range counts {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		t.Logf("%s %-14s %d 个", label, k, counts[k])
	}
}
