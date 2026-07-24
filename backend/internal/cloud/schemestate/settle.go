package schemestate

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemerounds"
)

// FormalPickAdvancer 正式盘派奖后推进出号游标的注入点。
//
// 出号推进逻辑依赖 schemes 包的运行类型解析，而 schemes -> periodsync -> accountsvc
// -> schemestate 已构成依赖链，schemestate 无法反向 import schemes。故由 schemes 包在
// init() 时注入本函数；未注入（如未链接 schemes 的工具二进制）时退回冻结旧行为。
//
// 入参：方案 kind、方案定义 config、实例快照、该期实际下注内容、是否命中。
// 返回：推进后的 pick_index / current_pick / last_direction。
var FormalPickAdvancer func(
	kind string,
	definitionConfig []byte,
	inst sqlcdb.SchemeInstance,
	betContent string,
	hit bool,
) (pickIndex int32, currentPick string, lastDirection string)

// ProcessFormalAfterSettlement 正式盘派奖后：回头盈亏 + 按实际中/未中推进倍投轮次
// 与出号游标（定码轮换/高级定码轮换等运行类型的下注内容切换）。
// 轮次与出号游标均不在下单时推进（第三方待开奖），派奖后在此统一更新。
func ProcessFormalAfterSettlement(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	periodNo string,
	pnl float64,
	hit bool,
	definitionConfig []byte,
	numericFromFloat func(float64) pgtype.Numeric,
) error {
	return ProcessAfterSettlement(ctx, q, inst, periodNo, pnl, hit, definitionConfig, numericFromFloat)
}

// ProcessAfterSettlement 正式盘/模拟盘派奖后共用：回头盈亏 + 倍投轮次 + 出号游标。
// 模拟盘与正式盘一致，下单时冻结游标，待真实开奖入库后再按中/未中推进。
func ProcessAfterSettlement(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	periodNo string,
	pnl float64,
	hit bool,
	definitionConfig []byte,
	numericFromFloat func(float64) pgtype.Numeric,
) error {
	if q == nil || inst.Status != "running" {
		return nil
	}

	simBet := inst.SimBet
	engine := lookback.NewEngine(q)
	settings := engine.LoadSettings(ctx, inst.MemberID)
	currentLookback := numericToFloat(inst.LookbackPnl)
	var overallRT lookback.Runtime
	if lookback.AppliesTo(settings, simBet) && settings.Judgment == lookback.JudgmentOverall {
		overallRT = engine.LoadRuntime(ctx, inst.MemberID, simBet)
	}
	lbEval := lookback.Evaluate(settings, simBet, currentLookback, overallRT, periodNo, pnl, hit)
	lookbackDelta := lbEval.LookbackAfter - currentLookback

	applyRoundIndex := inst.RoundIndex
	if lbEval.ResetIndividual || lbEval.ResetOverall {
		applyRoundIndex = 0
	} else {
		rounds := schemerounds.ParseFromDefinitionConfig(definitionConfig)
		applyRoundIndex = int32(schemerounds.NextIndex(rounds, int(inst.RoundIndex), hit))
	}

	// 出号体系：下单时（待开奖）冻结的出号游标在此按实际中/未中补推进，
	// 使定码轮换/高级定码轮换等运行类型逐期切换下注内容（与倍投轮次推进独立）。
	applyPickIndex, applyCurrentPick, applyLastDirection := inst.PickIndex, inst.CurrentPick, inst.LastDirection
	if FormalPickAdvancer != nil {
		betContent := ""
		if snap, serr := q.GetCloudBetPeriodSnapshot(ctx, inst.ID, periodNo); serr == nil {
			betContent = snap.BetContent
		}
		applyPickIndex, applyCurrentPick, applyLastDirection = FormalPickAdvancer(
			inst.Kind, definitionConfig, inst, betContent, hit,
		)
	}

	if _, err := q.ApplySchemeInstanceBet(ctx, sqlcdb.ApplySchemeInstanceBetParams{
		ID:               inst.ID,
		CountdownSec:     inst.CountdownSec,
		Turnover:         numericFromFloat(0),
		Pnl:              numericFromFloat(0),
		Multiplier:       inst.Multiplier,
		RoundIndex:       applyRoundIndex,
		LastSettledIssue: inst.LastSettledIssue,
		LookbackPnl:      numericFromFloat(lookbackDelta),
		PickIndex:        applyPickIndex,
		CurrentPick:      applyCurrentPick,
		LastDirection:    applyLastDirection,
	}); err != nil {
		return err
	}

	if lbEval.TrackOverall {
		if err := engine.SaveRuntime(ctx, inst.MemberID, simBet, lbEval.OverallRT, lbEval.ResetOverall); err != nil {
			return err
		}
	}
	if lbEval.ResetIndividual || lbEval.ResetOverall {
		if _, err := engine.ApplyInstanceResets(ctx, inst, lbEval.ResetIndividual, lbEval.ResetOverall); err != nil {
			return err
		}
		if lbEval.ResetIndividual && !lbEval.ResetOverall {
			mode := "formal"
			if simBet {
				mode = "sim"
			}
			slog.Info("lookback reset individual ("+mode+")", "instanceId", inst.ID, "memberId", inst.MemberID)
		}
	}
	return nil
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}
