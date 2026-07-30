package schemes_test

// 方案全链路 E2E：创建方案 → 加入云端 → 启动 → worker 出号下注 → 结算 → 查询。
//
// 此前每个环节都有组件单测，但没有一条测试把它们串起来跑过；
// 「能创建、能下注、能结算、查得到」这件事一直只靠人工点。
//
// 走真实 DB（无 DATABASE_URL 则跳过），全程 sim_bet=true，不碰第三方。
// 期号来自注入的 periods 缓存而不是第三方接口，所以不依赖真实时间流逝。
//
// 跑之前最好先停掉本地后端：它的 scheme worker 会并发推同一批实例。
// 断言写的是「终态如此」而不是「由我这一 tick 造成」，所以并发不会误判，只是会变慢。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/cloud/betrecords"
	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/schemes"
)

// e2eEnv 一次 E2E 用到的全部外部依赖。
type e2eEnv struct {
	pool    *db.Pool
	q       *sqlcdb.Queries
	svc      *schemes.Service
	worker   *schemes.Worker
	account  string
	memberID int64
	lottery  string
	// 该彩种模板下确实存在的一个玩法（定位胆个位）
	playTypeID, subPlayID string
}

func newE2EEnv(t *testing.T) *e2eEnv {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(pool.Close)

	account := cfg.ClientDemoAccount
	if account == "" {
		account = "vs8888"
	}
	var memberID int64
	if err := pool.QueryRow(ctx,
		`SELECT id FROM members WHERE account = $1`, account).Scan(&memberID); err != nil {
		t.Skipf("member %s 不存在：%v", account, err)
	}

	// 彩种与玩法都从库里取：sub_plays 的 id 会随第三方 rules 同步变动，硬编码迟早失效。
	// 选一星定位胆——单位单号，内容形态最简单，能把注意力留在链路本身而不是玩法解析上。
	var lotteryCode, playTypeID, subPlayID string
	err = pool.QueryRow(ctx, `
SELECT c.code, sp.type_id, sp.sub_id
FROM lottery_catalog c
JOIN sub_plays sp ON sp.template_code = c.play_template
WHERE c.sale_status = 'on_sale'
  AND c.play_template = 'ssc_std'
  AND sp.enabled
  AND sp.label = '一星定位胆'
ORDER BY c.code
LIMIT 1`).Scan(&lotteryCode, &playTypeID, &subPlayID)
	if err != nil {
		t.Skipf("找不到可用的 ssc_std 在售彩种 + 一星定位胆玩法：%v", err)
	}

	worker := schemes.NewWorker(pool, 1, nil, nil)
	if worker == nil {
		t.Fatal("NewWorker 返回 nil")
	}
	restoreSimQuota(t, pool, memberID)
	return &e2eEnv{
		pool: pool, q: sqlcdb.New(pool),
		svc:     schemes.NewService(pool, nil),
		worker:  worker,
		account: account, memberID: memberID, lottery: lotteryCode,
		playTypeID: playTypeID, subPlayID: subPlayID,
	}
}

// restoreSimQuota 快照并还原「今日模拟启动次数」。
//
// 每个用例都要启动一次实例，会真实消耗该会员的当日配额（上限 5 次），
// 不还原的话本测试跑两轮就把人的配额吃光了，第三个用例还会直接失败。
func restoreSimQuota(t *testing.T, pool *db.Pool, memberID int64) {
	t.Helper()
	ctx := context.Background()
	var date *time.Time
	var count int
	err := pool.QueryRow(ctx,
		`SELECT sim_scheme_starts_date, sim_scheme_starts_count FROM members WHERE id = $1`,
		memberID).Scan(&date, &count)
	if err != nil {
		t.Skipf("读模拟配额失败（可能未跑迁移 00130）：%v", err)
	}
	// 先清零，让本次运行不受历史次数影响
	if _, err := pool.Exec(ctx,
		`UPDATE members SET sim_scheme_starts_count = 0 WHERE id = $1`, memberID); err != nil {
		t.Fatalf("重置模拟配额：%v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE members SET sim_scheme_starts_date = $2, sim_scheme_starts_count = $3 WHERE id = $1`,
			memberID, date, count)
	})
}

// e2ePeriods 注入 periods 缓存，替代第三方接口。
//
// skipPeriod 已封盘（启动后要跳过的那一期），betPeriod 尚在开盘窗口内。
// 这样两次 tick 就能走完「跳过首期 → 激活 → 下注」，不用等真实时间。
func (e *e2eEnv) injectPeriods(skipPeriod, betPeriod string) {
	now := time.Now().UTC()
	lottery.UpdatePeriodsScheduleFullWithDuration(
		e.lottery,
		betPeriod, skipPeriod,
		now.Add(-time.Minute), // skipPeriod 已于 1 分钟前封盘
		now.Add(5*time.Minute), // betPeriod 5 分钟后封盘
		60, "", now.Add(-4*time.Minute),
	)
}

// createRunningInstance 建方案 → 写内容 → 加入云端 → 启动，返回实例 id。
func (e *e2eEnv) createRunningInstance(t *testing.T, name string, cfgPatch map[string]interface{}) string {
	t.Helper()
	ctx := context.Background()

	def, err := e.svc.CreateDefinition(ctx, e.account, schemes.CreateDefinitionInput{
		Kind:        "custom",
		SchemeName:  name,
		LotteryCode: e.lottery,
		RunTypeID:   cfgPatch["runTypeId"].(string),
		PlayTypeID:  e.playTypeID,
		SubPlayID:   e.subPlayID,
	})
	if err != nil {
		t.Fatalf("CreateDefinition: %v", err)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(),
			`DELETE FROM scheme_definitions WHERE id = $1`, def.ID)
	})

	patch := schemes.UpdateDefinitionPatch{HasSimBet: true, SimBet: true}
	applyE2EConfigPatch(t, &patch, cfgPatch)
	if _, err := e.svc.UpdateDefinition(ctx, e.account, def.ID, patch); err != nil {
		t.Fatalf("UpdateDefinition: %v", err)
	}

	added, err := e.svc.AddDefinitionToCloud(ctx, e.account, def.ID, "private",
		schemes.AddToCloudConfigPatch{SchemeFunds: "1000", BetUnit: "1"})
	if err != nil {
		t.Fatalf("AddDefinitionToCloud: %v", err)
	}
	instID := added.Instance.ID
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = e.pool.Exec(bg, `DELETE FROM cloud_bet_records WHERE scheme_id = $1`, instID)
		_, _ = e.pool.Exec(bg, `DELETE FROM scheme_instances WHERE id = $1`, instID)
	})

	// 每个用例都占一次当日模拟启动配额（上限 5），用例多于 5 个就会被挡住。
	// 环境初始化时已快照原值，这里清零只影响本次运行。
	if _, err := e.pool.Exec(ctx,
		`UPDATE members SET sim_scheme_starts_count = 0 WHERE id = $1`, e.memberID); err != nil {
		t.Fatalf("重置模拟配额：%v", err)
	}
	if _, err := e.svc.StartInstance(ctx, e.account, instID); err != nil {
		t.Fatalf("StartInstance: %v", err)
	}
	return instID
}

// applyE2EConfigPatch 把用例给的配置片段翻成 UpdateDefinitionPatch。
func applyE2EConfigPatch(t *testing.T, patch *schemes.UpdateDefinitionPatch, cfg map[string]interface{}) {
	t.Helper()
	for key, val := range cfg {
		switch key {
		case "runTypeId":
		case "schemeGroups":
			for _, g := range val.([]string) {
				patch.SchemeGroups = append(patch.SchemeGroups, g)
			}
		case "betMode":
			patch.BetMode, patch.HasBetMode = val.(string), true
		case "randomDraw":
			patch.RandomDraw, patch.HasRandomDraw = mustRawJSON(t, val), true
		case "hotColdWarm":
			patch.HotColdWarm, patch.HasHotColdWarm = mustRawJSON(t, val), true
		case "jushuList":
			patch.JushuList, patch.HasJushuList = mustRawJSON(t, val), true
		case "triggerBet":
			patch.TriggerBet, patch.HasTriggerBet = mustRawJSON(t, val), true
		case "fixedPick":
			patch.FixedPick, patch.HasFixedPick = mustRawJSON(t, val), true
		default:
			t.Fatalf("用例配置含未支持的键 %q，请在 applyE2EConfigPatch 里补上", key)
		}
	}
}

func mustRawJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

// driveUntilBet 推动实例直到落下一笔注单：第一轮跳过首期，第二轮激活并下注。
func (e *e2eEnv) driveUntilBet(t *testing.T, instID, betPeriod string) sqlcdb.SchemeInstance {
	t.Helper()
	ctx := context.Background()
	var inst sqlcdb.SchemeInstance
	for round := 1; round <= 3; round++ {
		fresh, err := e.q.GetSchemeInstanceFull(ctx, instID)
		if err != nil {
			t.Fatalf("第 %d 轮读实例：%v", round, err)
		}
		inst = fresh
		e.worker.TickInstanceForTest(ctx, inst)
		if e.betCount(t, instID, betPeriod) > 0 {
			return inst
		}
	}
	final, _ := e.q.GetSchemeInstanceFull(ctx, instID)
	t.Fatalf("推了 3 轮仍未下注：status=%s reason=%s period=%s",
		final.Status, final.StatusReason, betPeriod)
	return inst
}

func (e *e2eEnv) betCount(t *testing.T, instID, period string) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT count(*)::int FROM cloud_bet_records WHERE scheme_id = $1 AND period_no = $2`,
		instID, period).Scan(&n); err != nil {
		t.Fatalf("count bets: %v", err)
	}
	return n
}

// ensureDraw 保证该期有开奖号；本测试插入的行会在结束时删掉。
func (e *e2eEnv) ensureDraw(t *testing.T, period string, balls []string) {
	t.Helper()
	ctx := context.Background()
	var exists bool
	if err := e.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM lottery_draws WHERE lottery_code=$1 AND issue_no=$2)`,
		e.lottery, period).Scan(&exists); err != nil {
		t.Fatalf("check draw: %v", err)
	}
	if exists {
		return
	}
	ballsJSON, err := json.Marshal(balls)
	if err != nil {
		t.Fatalf("marshal balls: %v", err)
	}
	sum := 0
	for _, b := range balls {
		n := 0
		_, _ = fmt.Sscanf(b, "%d", &n)
		sum += n
	}
	if _, err := e.pool.Exec(ctx, `
INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at)
VALUES ($1, $2, $2, $3, $4, now())`, e.lottery, period, ballsJSON, sum); err != nil {
		t.Fatalf("insert draw: %v", err)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(),
			`DELETE FROM lottery_draws WHERE lottery_code=$1 AND issue_no=$2`, e.lottery, period)
	})
}

type e2eBet struct {
	RecordNo   string
	Status     string
	Amount     float64
	Pnl        float64
	BetContent string
	BetUnits   *int
}

func (e *e2eEnv) loadBet(t *testing.T, instID, period string) e2eBet {
	t.Helper()
	var b e2eBet
	err := e.pool.QueryRow(context.Background(), `
SELECT record_no, status, amount::float8, pnl::float8, bet_content, bet_units
FROM cloud_bet_records WHERE scheme_id = $1 AND period_no = $2`,
		instID, period).Scan(&b.RecordNo, &b.Status, &b.Amount, &b.Pnl, &b.BetContent, &b.BetUnits)
	if err != nil {
		t.Fatalf("load bet: %v", err)
	}
	return b
}

// e2ePeriodNo 造一个不会和真实期号撞车的期号。
func e2ePeriodNo(offset int) string {
	return fmt.Sprintf("99%d%03d", time.Now().Unix(), offset)
}

func TestSchemeLifecycleE2E(t *testing.T) {
	env := newE2EEnv(t)

	// 用的是一星定位胆五位面板：单行内容只作用于万位，所以开奖球的第一位决定中不中。
	cases := []struct {
		name string
		cfg  map[string]interface{}
		// priorDrawBalls 非空时，先给被跳过的首期造一条开奖，
		// 供依赖上期结果出号的运行类型（开某投某）取用。
		priorDrawBalls []string
		drawBalls      []string
		// wantStatus 出号确定的用例把判定也钉死；随机/冷热出号留空，只查通用不变量。
		wantStatus string
	}{
		{
			name: "fixed_number 固定号码",
			cfg: map[string]interface{}{
				"runTypeId": "fixed_number", "betMode": "dingwei",
				"schemeGroups": []string{"6,8"},
			},
			drawBalls: []string{"6", "2", "3", "4", "5"}, wantStatus: "hit",
		},
		{
			name: "fixed_rotate 定码轮换",
			cfg: map[string]interface{}{
				"runTypeId": "fixed_rotate", "betMode": "dingwei",
				"schemeGroups": []string{"1,2", "3,4"},
			},
			// 首期用第一组 1,2；万位开 9 → 必不中
			drawBalls: []string{"9", "2", "3", "4", "5"}, wantStatus: "miss",
		},
		{
			name: "random_draw 随机出号",
			cfg: map[string]interface{}{
				"runTypeId": "random_draw", "betMode": "dingwei",
				"randomDraw": map[string]interface{}{"counts": []int{2}, "strategy": "every"},
			},
			drawBalls: []string{"1", "2", "3", "4", "5"},
		},
		{
			name: "adv_fixed_rotate 高级定码轮换",
			cfg: map[string]interface{}{
				"runTypeId": "adv_fixed_rotate", "betMode": "dingwei",
				"jushuList": []map[string]interface{}{
					{"ju": 1, "content": "1", "afterHit": 2, "afterMiss": 2},
					{"ju": 2, "content": "3", "afterHit": 1, "afterMiss": 1},
				},
			},
			// 第 1 局投 1；万位开 1 → 必中
			drawBalls: []string{"1", "2", "3", "4", "5"}, wantStatus: "hit",
		},
		{
			name: "hot_cold_warm 冷热出号",
			cfg: map[string]interface{}{
				"runTypeId": "hot_cold_warm", "betMode": "dingwei",
				"hotColdWarm": map[string]interface{}{
					"totalPeriods": 20, "pickTypes": []string{"hot"},
					"faultCount": 0, "pickCount": 1, "strategy": "every",
				},
			},
			drawBalls: []string{"1", "2", "3", "4", "5"},
		},
		{
			name: "adv_trigger_bet 高级开某投某",
			// 触发投注要看上一期开奖：前置期被造成全 1，正投行 open=1 必定触发
			priorDrawBalls: []string{"1", "1", "1", "1", "1"},
			cfg: map[string]interface{}{
				"runTypeId": "adv_trigger_bet", "betMode": "dingwei",
				"triggerBet": map[string]interface{}{
					"mode": "always_pos",
					"rows": []map[string]interface{}{
						{"enabled": true, "open": "1", "pos": "1\n1\n1\n1\n1", "neg": "0\n0\n0\n0\n0"},
					},
				},
			},
			// 触发后五位全投 1；开奖全 1 → 必中
			drawBalls: []string{"1", "1", "1", "1", "1"}, wantStatus: "hit",
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			skipPeriod := e2ePeriodNo(i*2 + 1)
			betPeriod := e2ePeriodNo(i*2 + 2)
			env.injectPeriods(skipPeriod, betPeriod)
			if len(tc.priorDrawBalls) > 0 {
				env.ensureDraw(t, skipPeriod, tc.priorDrawBalls)
			}

			instID := env.createRunningInstance(t, fmt.Sprintf("E2E-%s-%d", tc.cfg["runTypeId"], time.Now().UnixNano()), tc.cfg)

			// 1) 出号 + 下注
			env.driveUntilBet(t, instID, betPeriod)
			bet := env.loadBet(t, instID, betPeriod)
			if bet.Status != "pending" {
				t.Fatalf("刚下的注应为 pending，实际 %s", bet.Status)
			}
			if strings.TrimSpace(bet.BetContent) == "" {
				t.Fatal("投注内容为空")
			}
			if bet.Amount <= 0 {
				t.Fatalf("投注金额应为正：%.2f", bet.Amount)
			}

			// 2) 投注内容必须落在该玩法的合法投注空间内
			def, err := env.q.GetSchemeDefinitionByID(ctx, instDefinitionID(t, env, instID))
			if err != nil {
				t.Fatalf("load definition: %v", err)
			}
			if vs := schemes.ValidateSchemeBetContent("custom", def.Config, bet.BetContent, 0); len(vs) > 0 {
				t.Fatalf("出号越出合法投注空间：%s（内容 %q）", vs[0].Detail, bet.BetContent)
			}

			// 3) 结算
			env.ensureDraw(t, betPeriod, tc.drawBalls)
			n, err := env.worker.SettleSimBetsForTest(ctx, instID)
			if err != nil {
				t.Fatalf("结算失败：%v", err)
			}
			if n != 1 {
				t.Fatalf("应结算 1 笔，实际 %d", n)
			}
			settled := env.loadBet(t, instID, betPeriod)
			if settled.Status != "hit" && settled.Status != "miss" {
				t.Fatalf("结算后状态应为 hit/miss，实际 %s", settled.Status)
			}
			if tc.wantStatus != "" && settled.Status != tc.wantStatus {
				t.Fatalf("开奖 %v 投注 %q，判定应为 %s，实际 %s",
					tc.drawBalls, settled.BetContent, tc.wantStatus, settled.Status)
			}
			// 资金守恒：未中亏满本金
			if settled.Status == "miss" && settled.Pnl != -settled.Amount {
				t.Fatalf("未中奖应亏满本金：pnl=%.2f amount=%.2f", settled.Pnl, settled.Amount)
			}
			if settled.Status == "hit" && settled.Pnl <= 0 {
				t.Fatalf("中奖盈亏应为正：%.2f", settled.Pnl)
			}

			// 4) 查询链路（真实 SQL，非内存 mock）
			item, err := betrecords.NewService(env.pool).ItemByRecordNo(ctx, env.memberID, settled.RecordNo)
			if err != nil {
				t.Fatalf("查询投注详情：%v", err)
			}
			if item.Period != betPeriod {
				t.Fatalf("详情期号 %s ≠ %s", item.Period, betPeriod)
			}
			if item.Status != settled.Status {
				t.Fatalf("详情状态 %s ≠ 落库 %s", item.Status, settled.Status)
			}
			if !item.SimBet {
				t.Fatal("详情应标记为模拟投注")
			}
			if item.BetContent != settled.BetContent {
				t.Fatalf("详情投注内容 %q ≠ 落库 %q", item.BetContent, settled.BetContent)
			}
		})
	}
}

// TestBuiltinPlanLifecycleE2E 内置计划的完整路径：
// 别人分享出方案 → 我收藏 → 建内置计划方案并物化该收藏 → 跑起来下注结算。
//
// 单独成一个用例是因为它比其它运行类型多两步前置（分享快照 + 收藏），
// 而这两步恰好是「收藏方案 → 物化 → 运行」里此前没有任何测试碰过的一段。
func TestBuiltinPlanLifecycleE2E(t *testing.T) {
	env := newE2EEnv(t)
	ctx := context.Background()

	// 1) 造一个公开分享的方案，拿到快照 id
	donor, err := env.svc.CreateDefinition(ctx, env.account, schemes.CreateDefinitionInput{
		Kind:        "custom",
		SchemeName:  fmt.Sprintf("E2E-donor-%d", time.Now().UnixNano()),
		LotteryCode: env.lottery,
		RunTypeID:   "fixed_number",
		PlayTypeID:  env.playTypeID,
		SubPlayID:   env.subPlayID,
	})
	if err != nil {
		t.Fatalf("建分享源方案：%v", err)
	}
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(),
			`DELETE FROM scheme_definitions WHERE id = $1`, donor.ID)
	})
	if _, err := env.svc.UpdateDefinition(ctx, env.account, donor.ID, schemes.UpdateDefinitionPatch{
		SchemeGroups: []string{"7"},
		BetMode:      "dingwei", HasBetMode: true,
	}); err != nil {
		t.Fatalf("写分享源内容：%v", err)
	}
	shared, err := env.svc.AddDefinitionToCloud(ctx, env.account, donor.ID, "public",
		schemes.AddToCloudConfigPatch{SchemeFunds: "1000", BetUnit: "1"})
	if err != nil {
		t.Fatalf("分享源加入云端：%v", err)
	}
	snapshotID := shared.ShareSnapshotID
	if snapshotID == "" {
		t.Fatal("公开分享未产出快照 id")
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = env.pool.Exec(bg, `DELETE FROM cloud_bet_records WHERE scheme_id = $1`, shared.Instance.ID)
		_, _ = env.pool.Exec(bg, `DELETE FROM scheme_instances WHERE id = $1`, shared.Instance.ID)
		_, _ = env.pool.Exec(bg, `DELETE FROM scheme_share_snapshots WHERE id = $1`, snapshotID)
	})

	// 2) 收藏该快照——不收藏就不允许物化
	if err := env.svc.AddFavorite(ctx, env.account, snapshotID); err != nil {
		t.Fatalf("收藏快照：%v", err)
	}
	t.Cleanup(func() {
		_ = env.svc.RemoveFavorite(context.Background(), env.account, snapshotID)
	})

	// 3) 建内置计划方案并物化
	plan, err := env.svc.CreateDefinition(ctx, env.account, schemes.CreateDefinitionInput{
		Kind:        "custom",
		SchemeName:  fmt.Sprintf("E2E-builtin-%d", time.Now().UnixNano()),
		LotteryCode: env.lottery,
		RunTypeID:   "builtin_plan",
		PlayTypeID:  env.playTypeID,
		SubPlayID:   env.subPlayID,
	})
	if err != nil {
		t.Fatalf("建内置计划方案：%v", err)
	}
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(),
			`DELETE FROM scheme_definitions WHERE id = $1`, plan.ID)
	})
	if _, err := env.svc.UpdateDefinition(ctx, env.account, plan.ID, schemes.UpdateDefinitionPatch{
		BuiltinPlanSnapshotID: snapshotID, HasBuiltinPlan: true,
		HasSimBet: true, SimBet: true,
	}); err != nil {
		t.Fatalf("物化内置计划：%v", err)
	}

	added, err := env.svc.AddDefinitionToCloud(ctx, env.account, plan.ID, "private",
		schemes.AddToCloudConfigPatch{SchemeFunds: "1000", BetUnit: "1"})
	if err != nil {
		t.Fatalf("内置计划加入云端：%v", err)
	}
	instID := added.Instance.ID
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = env.pool.Exec(bg, `DELETE FROM cloud_bet_records WHERE scheme_id = $1`, instID)
		_, _ = env.pool.Exec(bg, `DELETE FROM scheme_instances WHERE id = $1`, instID)
	})
	if _, err := env.pool.Exec(ctx,
		`UPDATE members SET sim_scheme_starts_count = 0 WHERE id = $1`, env.memberID); err != nil {
		t.Fatalf("重置模拟配额：%v", err)
	}
	if _, err := env.svc.StartInstance(ctx, env.account, instID); err != nil {
		t.Fatalf("启动内置计划实例：%v", err)
	}

	// 4) 跑起来：物化后应按快照里的 fixed_number 出号，投注内容就是分享方的 "7"
	skipPeriod, betPeriod := e2ePeriodNo(901), e2ePeriodNo(902)
	env.injectPeriods(skipPeriod, betPeriod)
	env.driveUntilBet(t, instID, betPeriod)
	bet := env.loadBet(t, instID, betPeriod)
	if strings.TrimSpace(bet.BetContent) != "7" {
		t.Fatalf("内置计划应沿用快照内容 7，实际 %q", bet.BetContent)
	}

	// 一星定位胆是五位面板，单行内容只作用于万位，所以要让万位开 7
	env.ensureDraw(t, betPeriod, []string{"7", "2", "3", "4", "5"})
	if _, err := env.worker.SettleSimBetsForTest(ctx, instID); err != nil {
		t.Fatalf("结算：%v", err)
	}
	settled := env.loadBet(t, instID, betPeriod)
	if settled.Status != "hit" {
		t.Fatalf("万位开 7 投 7 应中奖，实际 %s", settled.Status)
	}
	if settled.Pnl <= 0 {
		t.Fatalf("中奖盈亏应为正：%.2f", settled.Pnl)
	}
}

func instDefinitionID(t *testing.T, env *e2eEnv, instID string) string {
	t.Helper()
	var defID string
	if err := env.pool.QueryRow(context.Background(),
		`SELECT definition_id FROM scheme_instances WHERE id = $1`, instID).Scan(&defID); err != nil {
		t.Fatalf("definition_id: %v", err)
	}
	return defID
}
