package betrecords

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

// service_test.go 里那套跑的是 NewService(nil) 的内存 mock（service_memory.go），
// 真正跑 SQL 的 service_db.go 此前零测试——分组、汇总、分页、筛选、详情
// 全都发生在 SQL 和 Go 的交界处，mock 一行也覆盖不到。
//
// 这里往库里播几条注单，跑真实查询再删干净。

type dbEnv struct {
	svc      *Service
	pool     *db.Pool
	memberID int64
	schemeID string
	lottery  string
}

// seedBet 一条待播的注单。
type seedBet struct {
	period  string
	status  string
	amount  float64
	pnl     float64
	simBet  bool
	placed  time.Time
	units   int
	payout  float64
	content string
}

func newDBEnv(t *testing.T) *dbEnv {
	t.Helper()
	_ = godotenv.Load("../../../.env")
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
		t.Skipf("会员 %s 不存在：%v", account, err)
	}
	var lottery string
	if err := pool.QueryRow(ctx,
		`SELECT code FROM lottery_catalog WHERE sale_status = 'on_sale' ORDER BY code LIMIT 1`).
		Scan(&lottery); err != nil {
		t.Skipf("没有在售彩种：%v", err)
	}

	return &dbEnv{
		svc: NewService(pool), pool: pool, memberID: memberID,
		// record_no 是 varchar(32)，实例 id 还要给 "-Rnn" 后缀留位置，所以只取纳秒的低位
		schemeID: fmt.Sprintf("bt%d", time.Now().UnixNano()%1e12),
		lottery:  lottery,
	}
}

// seed 播入实例与注单并登记清理。record_no 用实例 id 做前缀，确保只删自己的。
//
// 实例行是必须的：分组查询 join 了 scheme_instances 取彩种，
// 明细查询也要靠它认出方案，只播注单会一条都查不出来。
func (e *dbEnv) seed(t *testing.T, bets ...seedBet) []string {
	t.Helper()
	ctx := context.Background()
	defID := "d" + e.schemeID
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = e.pool.Exec(bg, `DELETE FROM cloud_bet_records WHERE scheme_id = $1`, e.schemeID)
		_, _ = e.pool.Exec(bg, `DELETE FROM scheme_instances WHERE id = $1`, e.schemeID)
		_, _ = e.pool.Exec(bg, `DELETE FROM scheme_definitions WHERE id = $1`, defID)
	})
	if _, err := e.pool.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', $3, $4, '测试彩种', 'private', '{}'::jsonb)`,
		defID, e.memberID, "betrecords 测试方案", e.lottery); err != nil {
		t.Fatalf("播方案定义失败：%v", err)
	}
	// 状态用 paused：worker 只捞 running，不会误推这条测试实例
	if _, err := e.pool.Exec(ctx, `
INSERT INTO scheme_instances (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status)
VALUES ($1, $2, $3, 'custom', $4, $5, '测试彩种', 'paused')`,
		e.schemeID, defID, e.memberID, "betrecords 测试方案", e.lottery); err != nil {
		t.Fatalf("播实例失败：%v", err)
	}

	recordNos := make([]string, 0, len(bets))
	for i, b := range bets {
		recordNo := fmt.Sprintf("%s-R%02d", e.schemeID, i)
		placed := b.placed
		if placed.IsZero() {
			placed = time.Now().Add(-time.Duration(i) * time.Minute)
		}
		content := b.content
		if content == "" {
			content = "1,2"
		}
		var units interface{}
		if b.units > 0 {
			units = b.units
		}
		var payout interface{}
		if b.payout > 0 {
			payout = b.payout
		}
		_, err := e.pool.Exec(ctx, `
INSERT INTO cloud_bet_records (
    record_no, member_id, scheme_id, scheme_name, period_no, play_type,
    multiplier, round_label, amount, pnl, status, placed_at, bet_content,
    sim_bet, currency, lottery_code, lottery_label, definition_id, bet_units, payout_amount
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
			recordNo, e.memberID, e.schemeID, "betrecords 测试方案", b.period, "一星定位胆",
			"1", "第1局", b.amount, b.pnl, b.status, placed, content,
			b.simBet, "CNY", e.lottery, "测试彩种", "def-betrec-test", units, payout)
		if err != nil {
			t.Fatalf("播注单 %s 失败：%v", recordNo, err)
		}
		recordNos = append(recordNos, recordNo)
	}
	return recordNos
}

// findGroup 在分组结果里找出本测试造的那一组。
// 库里本来就有别人的数据，不能按下标取。
func findGroup(t *testing.T, res GroupsResult, schemeID string) Group {
	t.Helper()
	for _, g := range res.Groups.Items {
		if g.SchemeID == schemeID {
			return g
		}
	}
	t.Fatalf("分组结果里找不到 %s（共 %d 组）", schemeID, len(res.Groups.Items))
	return Group{}
}

// TestGroupsAggregatesFromSQL 分组的投注额与盈亏必须由 SQL 里的注单如实汇总出来。
func TestGroupsAggregatesFromSQL(t *testing.T) {
	env := newDBEnv(t)
	env.seed(t,
		seedBet{period: "20260101001", status: "hit", amount: 10, pnl: 8, units: 2, payout: 18, simBet: true},
		seedBet{period: "20260101002", status: "miss", amount: 10, pnl: -10, units: 2, simBet: true},
		seedBet{period: "20260101003", status: "pending", amount: 10, units: 2, simBet: true},
	)

	res, err := env.svc.GroupsWithFilter(context.Background(), env.memberID, GroupsFilter{
		Mode: string(ModeSim), Days: 3, Limit: -1,
	})
	if err != nil {
		t.Fatalf("查分组：%v", err)
	}
	g := findGroup(t, res, env.schemeID)
	if g.TotalBet != 30 {
		t.Errorf("投注额 = %.2f，期望 30", g.TotalBet)
	}
	// 8 + (-10) + 0 = -2
	if g.DayPnL != -2 {
		t.Errorf("盈亏 = %.2f，期望 -2", g.DayPnL)
	}
	// 中奖派彩 = 本金 + 盈利 = 10 + 8
	if g.TotalPrize != 18 {
		t.Errorf("派彩 = %.2f，期望 18", g.TotalPrize)
	}
	// 中奖率 = 中奖笔数 / 总笔数 = 1/3。
	// 注意分母把待开奖的那笔也算进去了（summarize 用的是 len(rows)），
	// 所以方案刚跑起来、大部分注单还没开奖时，这个数会被稀释得偏低。
	// 这里按现状钉住：真要改口径，得是一次有意识的产品决定，而不是顺手改查询改掉的。
	if g.WinRate != 33.3 {
		t.Errorf("中奖率 = %.2f，期望 33.3", g.WinRate)
	}
}

// TestGroupsSeparatesSimAndFormal 模拟盘与正式盘各查各的，互不串台。
//
// 顺带钉住一条不太显眼的规则：正式盘分组还要求注单绑定了挂机账号、
// 且已回填第三方注单号；只有本地记录、没落到第三方的正式注单不会出现在分组里。
// 这条藏在 ListCloudBetRecordsFiltered 的 guaji_account_id 条件里，
// 改查询时很容易顺手改坏。
func TestGroupsSeparatesSimAndFormal(t *testing.T) {
	env := newDBEnv(t)
	env.seed(t,
		seedBet{period: "20260102001", status: "hit", amount: 10, pnl: 5},
		seedBet{period: "20260102002", status: "hit", amount: 20, pnl: 7, simBet: true},
	)
	ctx := context.Background()

	sim, err := env.svc.GroupsWithFilter(ctx, env.memberID,
		GroupsFilter{Mode: string(ModeSim), Days: 3, Limit: -1})
	if err != nil {
		t.Fatalf("查模拟盘：%v", err)
	}
	g := findGroup(t, sim, env.schemeID)
	if g.TotalBet != 20 {
		t.Errorf("模拟盘投注额应为 20（不能把正式盘那 10 元算进来），实际 %.2f", g.TotalBet)
	}

	formal, err := env.svc.GroupsWithFilter(ctx, env.memberID,
		GroupsFilter{Mode: string(ModeReal), Days: 3, Limit: -1})
	if err != nil {
		t.Fatalf("查正式盘：%v", err)
	}
	for _, item := range formal.Groups.Items {
		if item.SchemeID == env.schemeID {
			t.Fatalf("未绑定挂机账号、无第三方注单号的正式注单不应出现在分组里：%+v", item)
		}
	}
}

// TestGroupsFiltersByLottery 彩种筛选要真的落到 SQL 上。
func TestGroupsFiltersByLottery(t *testing.T) {
	env := newDBEnv(t)
	env.seed(t, seedBet{period: "20260103001", status: "hit", amount: 10, pnl: 3, simBet: true})
	ctx := context.Background()

	hit, err := env.svc.GroupsWithFilter(ctx, env.memberID, GroupsFilter{
		Mode: string(ModeSim), Days: 3, Limit: -1, LotteryCode: env.lottery,
	})
	if err != nil {
		t.Fatalf("按彩种查：%v", err)
	}
	findGroup(t, hit, env.schemeID)

	miss, err := env.svc.GroupsWithFilter(ctx, env.memberID, GroupsFilter{
		Mode: string(ModeSim), Days: 3, Limit: -1, LotteryCode: "no_such_lottery",
	})
	if err != nil {
		t.Fatalf("按不存在的彩种查：%v", err)
	}
	for _, g := range miss.Groups.Items {
		if g.SchemeID == env.schemeID {
			t.Fatal("筛了别的彩种却仍返回了本方案")
		}
	}
}

// TestGroupsRejectsBadCursor 游标非法要报错，而不是当成第一页悄悄返回。
func TestGroupsRejectsBadCursor(t *testing.T) {
	env := newDBEnv(t)
	_, err := env.svc.GroupsWithFilter(context.Background(), env.memberID, GroupsFilter{
		Mode: string(ModeReal), Days: 3, Limit: 10, Cursor: "not-a-number",
	})
	if err == nil {
		t.Fatal("非法游标应当报错")
	}
}

// TestDetailListsBetsFromSQL 方案明细要按真实 SQL 返回该方案下的注单。
func TestDetailListsBetsFromSQL(t *testing.T) {
	env := newDBEnv(t)
	now := time.Now()
	env.seed(t,
		seedBet{period: "20260104001", status: "hit", amount: 10, pnl: 8, simBet: true, placed: now.Add(-3 * time.Minute)},
		seedBet{period: "20260104002", status: "miss", amount: 10, pnl: -10, simBet: true, placed: now.Add(-2 * time.Minute)},
		seedBet{period: "20260104003", status: "pending", amount: 10, simBet: true, placed: now.Add(-time.Minute)},
	)

	res, ok, err := env.svc.Detail(context.Background(), env.memberID, env.schemeID,
		ModeSim, 3, 50, "")
	if err != nil {
		t.Fatalf("查明细：%v", err)
	}
	if !ok {
		t.Fatal("方案存在却报找不到")
	}
	if len(res.Records.Items) != 3 {
		t.Fatalf("明细应有 3 笔，实际 %d", len(res.Records.Items))
	}
	// 最近下注的排最前
	if res.Records.Items[0].Period != "20260104003" {
		t.Errorf("首条期号 = %s，期望最近的 20260104003", res.Records.Items[0].Period)
	}
	for _, it := range res.Records.Items {
		if it.RecordNo == "" {
			t.Errorf("期号 %s 的注单编号为空", it.Period)
		}
	}
}

// TestDetailPaginatesFromSQL 明细分页要能翻到第二页，且两页不重叠。
func TestDetailPaginatesFromSQL(t *testing.T) {
	env := newDBEnv(t)
	now := time.Now()
	var bets []seedBet
	for i := 0; i < 5; i++ {
		bets = append(bets, seedBet{
			period: fmt.Sprintf("2026010500%d", i), status: "miss", amount: 10, pnl: -10,
			simBet: true, placed: now.Add(-time.Duration(i) * time.Minute),
		})
	}
	env.seed(t, bets...)
	ctx := context.Background()

	first, ok, err := env.svc.Detail(ctx, env.memberID, env.schemeID, ModeSim, 3, 2, "")
	if err != nil || !ok {
		t.Fatalf("查第一页：ok=%v err=%v", ok, err)
	}
	if len(first.Records.Items) != 2 {
		t.Fatalf("第一页应有 2 笔，实际 %d", len(first.Records.Items))
	}
	if !first.Records.Page.HasMore || first.Records.Page.NextCursor == nil {
		t.Fatalf("第一页应标记还有更多：%+v", first.Records.Page)
	}

	second, _, err := env.svc.Detail(ctx, env.memberID, env.schemeID,
		ModeSim, 3, 2, *first.Records.Page.NextCursor)
	if err != nil {
		t.Fatalf("查第二页：%v", err)
	}
	seen := map[string]bool{}
	for _, it := range first.Records.Items {
		seen[it.RecordNo] = true
	}
	for _, it := range second.Records.Items {
		if seen[it.RecordNo] {
			t.Errorf("注单 %s 在两页里都出现了", it.RecordNo)
		}
	}
}

// TestDetailRejectsOtherMembersScheme 别人的方案查不到，防止越权。
func TestDetailRejectsOtherMembersScheme(t *testing.T) {
	env := newDBEnv(t)
	env.seed(t, seedBet{period: "20260106001", status: "hit", amount: 10, pnl: 5})

	_, ok, err := env.svc.Detail(context.Background(), env.memberID+9999, env.schemeID,
		ModeReal, 3, 50, "")
	if err != nil {
		t.Fatalf("查别人的方案不该报错，应当是查不到：%v", err)
	}
	if ok {
		t.Fatal("查到了不属于该会员的方案")
	}
}

// TestItemByRecordNoFromSQL 单笔详情要能按注单编号取到，字段与落库一致。
func TestItemByRecordNoFromSQL(t *testing.T) {
	env := newDBEnv(t)
	nos := env.seed(t, seedBet{
		period: "20260107001", status: "hit", amount: 12, pnl: 8,
		units: 2, payout: 20, content: "3,7",
	})
	ctx := context.Background()

	item, err := env.svc.ItemByRecordNo(ctx, env.memberID, nos[0])
	if err != nil {
		t.Fatalf("查单笔详情：%v", err)
	}
	if item.RecordNo != nos[0] {
		t.Errorf("注单编号 = %q，期望 %q", item.RecordNo, nos[0])
	}
	if item.Period != "20260107001" {
		t.Errorf("期号 = %q", item.Period)
	}
	if item.Status != "hit" {
		t.Errorf("状态 = %q", item.Status)
	}
	if item.Amount != 12 {
		t.Errorf("金额 = %.2f，期望 12", item.Amount)
	}
	if item.BetContent != "3,7" {
		t.Errorf("投注内容 = %q，期望 3,7", item.BetContent)
	}
	if item.BetUnits == nil || *item.BetUnits != 2 {
		t.Errorf("投注注数 = %v，期望 2", item.BetUnits)
	}
	if item.PayoutAmount == nil || *item.PayoutAmount != 20 {
		t.Errorf("返奖金额 = %v，期望 20", item.PayoutAmount)
	}
}

// TestItemByRecordNoRejectsOtherMember 单笔详情同样要挡住越权。
func TestItemByRecordNoRejectsOtherMember(t *testing.T) {
	env := newDBEnv(t)
	nos := env.seed(t, seedBet{period: "20260108001", status: "hit", amount: 10, pnl: 5})
	ctx := context.Background()

	if _, err := env.svc.ItemByRecordNo(ctx, env.memberID+9999, nos[0]); err == nil {
		t.Fatal("查别人的注单应当报找不到")
	}
	if _, err := env.svc.ItemByRecordNo(ctx, env.memberID, "no-such-record-no"); err == nil {
		t.Fatal("查不存在的注单应当报找不到")
	}
	if _, err := env.svc.ItemByRecordNo(ctx, env.memberID, ""); err == nil {
		t.Fatal("空注单编号应当报找不到")
	}
}
