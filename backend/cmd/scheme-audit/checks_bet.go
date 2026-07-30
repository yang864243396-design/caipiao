package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

// 金额比对容差：库内均为 2 位小数。
const moneyEpsilon = 0.005

// 开奖后多久仍查不到开奖号 / 仍是 pending 才算异常，避开正常的同步与结算延迟。
const (
	drawGrace   = 30 * time.Minute
	settleGrace = 15 * time.Minute
)

type betRow struct {
	RecordNo     string
	MemberID     int64
	SchemeID     string
	SchemeName   string
	SimBet       bool
	LotteryCode  string
	LotteryLabel string
	PeriodNo     string
	TPPeriod     string
	TPBetID      string
	PlayType     string
	BetContent   string
	UnitsNull    bool
	Units        int32
	Amount       float64
	Pnl          float64
	PayoutNull   bool
	Payout       float64
	Status       string
	PlacedAt     time.Time
	Kind         string
	RoundIndex   int32
	Config       string
	BallsRaw     string
}

const betQuery = `
SELECT c.record_no,
       c.member_id,
       c.scheme_id,
       c.scheme_name,
       c.sim_bet,
       COALESCE(c.lottery_code, ''),
       COALESCE(c.lottery_label, ''),
       c.period_no,
       COALESCE(c.third_party_period, ''),
       COALESCE(c.third_party_bet_id, ''),
       COALESCE(c.play_type, ''),
       COALESCE(c.bet_content, ''),
       c.bet_units IS NULL,
       COALESCE(c.bet_units, 0),
       c.amount::float8,
       c.pnl::float8,
       c.payout_amount IS NULL,
       COALESCE(c.payout_amount, 0)::float8,
       c.status,
       c.placed_at,
       COALESCE(si.kind, ''),
       COALESCE(si.round_index, 0),
       COALESCE(sd.config::text, ''),
       COALESCE(d.balls::text, '')
FROM cloud_bet_records c
LEFT JOIN scheme_instances si ON si.id = c.scheme_id
LEFT JOIN scheme_definitions sd ON sd.id = si.definition_id
LEFT JOIN LATERAL (
    SELECT ld.balls
    FROM lottery_draws ld
    WHERE ld.lottery_code = c.lottery_code
      AND ld.issue_no = ANY (ARRAY[c.period_no, COALESCE(c.third_party_period, '')])
    LIMIT 1
) d ON TRUE
WHERE c.placed_at >= now() - make_interval(days => $1)
ORDER BY c.placed_at DESC
LIMIT $2`

func loadBets(ctx context.Context, pool *pgxpool.Pool, days, limit int) ([]betRow, error) {
	rows, err := pool.Query(ctx, betQuery, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []betRow
	for rows.Next() {
		var b betRow
		if err := rows.Scan(
			&b.RecordNo, &b.MemberID, &b.SchemeID, &b.SchemeName, &b.SimBet,
			&b.LotteryCode, &b.LotteryLabel, &b.PeriodNo, &b.TPPeriod, &b.TPBetID,
			&b.PlayType, &b.BetContent, &b.UnitsNull, &b.Units,
			&b.Amount, &b.Pnl, &b.PayoutNull, &b.Payout,
			&b.Status, &b.PlacedAt, &b.Kind, &b.RoundIndex, &b.Config, &b.BallsRaw,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// loadColumnEpochs 探测后加的列各自从哪一刻开始有值，用首条非空记录的投注时间作准。
func loadColumnEpochs(ctx context.Context, pool *pgxpool.Pool) (columnEpochs, error) {
	var e columnEpochs
	err := pool.QueryRow(ctx, `
SELECT COALESCE((SELECT MIN(placed_at) FROM cloud_bet_records WHERE bet_units IS NOT NULL),
                'infinity'::timestamptz),
       COALESCE((SELECT MIN(placed_at) FROM cloud_bet_records WHERE payout_amount IS NOT NULL),
                'infinity'::timestamptz)`).Scan(&e.BetUnits, &e.PayoutAmount)
	if err != nil {
		return e, err
	}

	rows, err := pool.Query(ctx, `SELECT code, updated_at FROM lottery_catalog WHERE updated_at IS NOT NULL`)
	if err != nil {
		return e, err
	}
	defer rows.Close()
	e.LotteryChanged = map[string]time.Time{}
	for rows.Next() {
		var code string
		var at time.Time
		if err := rows.Scan(&code, &at); err != nil {
			return e, err
		}
		e.LotteryChanged[code] = at
	}
	return e, rows.Err()
}

// lotteryTally 按彩种统计期号命中率，用于识别下注链路与开奖链路期号族不一致。
//
// 必须同时留一份近窗口计数：整窗命中率会被历史坏数据长期压低，
// 让「昨天刚修好」和「现在还坏着」看起来一模一样。断言只看近窗口。
type lotteryTally struct {
	Total, DrawFound             int
	RecentTotal, RecentDrawFound int
}

// columnEpochs 记录后加的列各自从什么时候开始有值。
// 迁移之前的历史行天然为空，把它们算成缺陷只会把报告淹掉。
//
// LotteryChanged 是各彩种 lottery_catalog 最后一次改动的时刻。改 game_id
// 会整体换掉下注链路的期号族（00136 对调极速彩即是），改动之前的注单记的是
// 旧族期号、永远查不到开奖，把它们计入命中率会让「昨天已修好」一直标红：
// 00136 生效后 tron_jisu 实测 96/96 全中，而跨越生效点的 24h 窗口只有 47%。
type columnEpochs struct {
	BetUnits, PayoutAmount time.Time
	LotteryChanged         map[string]time.Time
}

// periodFamilyEpoch 返回该彩种期号族的起算时刻（彩种配置最后一次改动）。
func (e columnEpochs) periodFamilyEpoch(code string) time.Time {
	if e.LotteryChanged == nil {
		return time.Time{}
	}
	return e.LotteryChanged[strings.TrimSpace(code)]
}

func (e columnEpochs) covers(t time.Time, epoch time.Time) bool {
	return !epoch.IsZero() && !t.Before(epoch)
}

func auditBets(
	c *collector, bets []betRow, now time.Time, recent time.Duration, maxUnits int, epochs columnEpochs,
) map[string]*lotteryTally {
	tally := map[string]*lotteryTally{}

	for _, b := range bets {
		c.scanned["bet"]++
		balls := sqlcdb.ParseDrawBalls([]byte(b.BallsRaw))

		if code := strings.TrimSpace(b.LotteryCode); code != "" {
			t := tally[code]
			if t == nil {
				t = &lotteryTally{}
				tally[code] = t
			}
			t.Total++
			if len(balls) > 0 {
				t.DrawFound++
			}
			// 近窗口：已过开奖宽限、落在近窗口内、且晚于该彩种最后一次配置改动，
			// 反映的才是当下状态（改 game_id 会换掉整个期号族）。
			age := now.Sub(b.PlacedAt)
			if age > drawGrace && age <= recent && !b.PlacedAt.Before(epochs.periodFamilyEpoch(code)) {
				t.RecentTotal++
				if len(balls) > 0 {
					t.RecentDrawFound++
				}
			}
		}

		checkFundConservation(c, b)
		checkPayoutSource(c, b, epochs)
		checkDisplayFields(c, b, epochs)
		checkDrawAndStatus(c, b, balls, now)
		checkBetContent(c, b, maxUnits, epochs)
		checkAdjudication(c, b, balls)
	}
	return tally
}

// 资金守恒：pnl 与 status 必须自洽，返奖金额必须等于本金 + 盈亏。
func checkFundConservation(c *collector, b betRow) {
	switch b.Status {
	case "pending":
		if math.Abs(b.Pnl) > moneyEpsilon {
			c.add(betFinding(P0, "fund_conservation", b,
				fmt.Sprintf("未结算却已有盈亏 %.2f", b.Pnl)))
		}
	case "miss":
		if math.Abs(b.Pnl+b.Amount) > moneyEpsilon {
			c.add(betFinding(P0, "fund_conservation", b,
				fmt.Sprintf("未中奖应亏满本金：pnl=%.2f amount=%.2f", b.Pnl, b.Amount)))
		}
	case "hit":
		if b.Pnl <= -b.Amount-moneyEpsilon {
			c.add(betFinding(P0, "fund_conservation", b,
				fmt.Sprintf("中奖却亏损超过本金：pnl=%.2f amount=%.2f", b.Pnl, b.Amount)))
		}
		if !b.PayoutNull && math.Abs(b.Payout-(b.Amount+b.Pnl)) > moneyEpsilon {
			c.add(betFinding(P0, "fund_conservation", b,
				fmt.Sprintf("返奖 %.2f ≠ 本金 %.2f + 盈亏 %.2f", b.Payout, b.Amount, b.Pnl)))
		}
	}
}

// 返奖来源：模拟盘必须落库返奖；正式盘缺失说明第三方未回传、走了本地兜底结算。
func checkPayoutSource(c *collector, b betRow, epochs columnEpochs) {
	if b.Status != "hit" {
		if !b.PayoutNull && b.Status == "pending" && b.Payout != 0 {
			c.add(betFinding(P1, "payout_source", b,
				fmt.Sprintf("未开奖却已写入返奖 %.2f", b.Payout)))
		}
		return
	}
	if !epochs.covers(b.PlacedAt, epochs.PayoutAmount) {
		c.skip("payout_source（列启用前）")
		return
	}
	if b.PayoutNull {
		sev, why := P2, "第三方未回传返奖，或走了本地兜底结算"
		if b.SimBet {
			sev, why = P1, "模拟盘返奖由本端计算，不应缺失"
		}
		c.add(betFinding(sev, "payout_source", b, "中奖但返奖金额为空："+why))
	}
}

func checkDisplayFields(c *collector, b betRow, epochs columnEpochs) {
	var missing []string
	if strings.TrimSpace(b.PlayType) == "" {
		missing = append(missing, "play_type")
	}
	if strings.TrimSpace(b.BetContent) == "" {
		missing = append(missing, "bet_content")
	}
	if strings.TrimSpace(b.LotteryLabel) == "" {
		missing = append(missing, "lottery_label")
	}
	if b.UnitsNull {
		if epochs.covers(b.PlacedAt, epochs.BetUnits) {
			missing = append(missing, "bet_units")
		} else {
			c.skip("display_fields.bet_units（列启用前）")
		}
	}
	if len(missing) > 0 {
		c.add(betFinding(P3, "display_fields", b, "详情字段缺失："+strings.Join(missing, "、")))
	}
}

// 期号归属 + 状态时效：开奖号查不到说明下注期号与开奖期号不同族（极速彩那类 bug）。
func checkDrawAndStatus(c *collector, b betRow, balls []string, now time.Time) {
	aged := now.Sub(b.PlacedAt)
	if len(balls) == 0 {
		if aged > drawGrace {
			c.add(betFinding(P1, "draw_missing", b,
				fmt.Sprintf("投注 %s 后仍查不到期号 %s 的开奖号（third_party_period=%q）",
					humanDur(aged), b.PeriodNo, b.TPPeriod)))
		} else {
			c.skip("draw_missing")
		}
		return
	}
	if b.Status == "pending" && aged > settleGrace {
		c.add(betFinding(P2, "status_stale", b,
			fmt.Sprintf("开奖号已入库，注单仍 pending 达 %s", humanDur(aged))))
	}
}

// 号池闭包 + 注数上下限：投注内容必须落在该玩法的合法投注空间内。
func checkBetContent(c *collector, b betRow, maxUnits int, epochs columnEpochs) {
	if strings.TrimSpace(b.Config) == "" || strings.TrimSpace(b.BetContent) == "" {
		c.skip("bet_content（缺方案定义或投注内容）")
		return
	}
	cfg := []byte(b.Config)
	if schemes.UniverseKindForScheme(b.Kind, cfg) == "" {
		c.skip("bet_content（玩法内容形态未知）")
		return
	}
	for _, v := range schemes.ValidateSchemeBetContent(b.Kind, cfg, b.BetContent, maxUnits) {
		sev := P1
		if v.Code == schemes.ViolationEmptyContent {
			sev = P3
		}
		c.add(betFinding(sev, "content_"+v.Code, b, v.Detail))
	}

	if b.UnitsNull || !epochs.covers(b.PlacedAt, epochs.BetUnits) {
		return
	}
	got, ok := schemes.CountBetUnitsForScheme(b.Kind, cfg, b.BetContent)
	if !ok {
		c.skip("bet_units_mismatch（口径不可比）")
		return
	}
	if got != int(b.Units) {
		c.add(betFinding(P2, "bet_units_mismatch", b,
			fmt.Sprintf("落库注数 %d，按投注内容重算为 %d（形态 %s、内容 %q）",
				b.Units, got, schemes.UniverseKindForScheme(b.Kind, cfg), truncate(b.BetContent, 40))))
	}
}

// 判定一致性：用开奖号本地重算，与落库 status 比对。
func checkAdjudication(c *collector, b betRow, balls []string) {
	if len(balls) == 0 || strings.TrimSpace(b.Config) == "" {
		c.skip("adjudication_mismatch")
		return
	}
	if b.Status != "hit" && b.Status != "miss" {
		return
	}
	hit, _, ok := schemes.AdjudicateSchemeBet(
		b.Kind, []byte(b.Config), b.LotteryCode, b.BetContent, int(b.RoundIndex), balls,
	)
	if !ok {
		c.skip("adjudication_mismatch")
		return
	}
	want := "miss"
	if hit {
		want = "hit"
	}
	if want == b.Status {
		return
	}
	// 正式盘以第三方判定为准，本地重算不一致说明两端验奖逻辑已分叉
	source := "第三方判定"
	if b.SimBet {
		source = "本端结算"
	}
	c.add(betFinding(P0, "adjudication_mismatch", b,
		fmt.Sprintf("%s为 %s，按开奖号 [%s] 本地重算为 %s",
			source, b.Status, strings.Join(balls, " "), want)))
}

func betFinding(sev, check string, b betRow, detail string) Finding {
	return Finding{
		Severity: sev, Check: check, Scope: "bet", Key: b.RecordNo,
		Lottery: b.LotteryCode, Play: b.PlayType, SimBet: b.SimBet,
		Detail: detail, PlacedAt: b.PlacedAt.Format(time.RFC3339),
	}
}

func humanDur(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%.1f 天", d.Hours()/24)
	case d >= time.Hour:
		return fmt.Sprintf("%.1f 小时", d.Hours())
	default:
		return fmt.Sprintf("%.0f 分钟", d.Minutes())
	}
}
