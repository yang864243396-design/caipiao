package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebettingdispatch"
)

func (p *StrategyProcessor) buildFormalFrozenRequest(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	targetPeriod string,
	requestID string,
) ([]byte, error) {
	if p == nil || p.pool == nil {
		return nil, errors.New("formal command planner has no database")
	}
	def, err := q.GetSchemeDefinitionByID(ctx, inst.DefinitionID)
	if err != nil {
		return nil, err
	}
	groupIndex := 0
	if inst.RoundIndex > 0 {
		groupIndex = int(inst.RoundIndex)
	}
	cfg := parseSchemeConfig(inst.Kind, def.Config, int(inst.RoundIndex), groupIndex)
	cfg.Play = attachOddsBase(cfg.Play, inst.LotteryCode)
	roundIndex := int(inst.RoundIndex)
	if roundIndex < 0 || roundIndex >= len(cfg.Rounds) {
		roundIndex = 0
	}
	round := cfg.Rounds[roundIndex]
	betMultiple := effectiveBetMultiple(combinedBaseCoef(inst.Multiplier, 1), round)
	planner := &Worker{pool: p.pool, q: q, ruleRegistry: p.ruleRegistry}
	draw := sqlcdb.LotteryDraw{LotteryCode: inst.LotteryCode, IssueNo: strings.TrimSpace(targetPeriod)}
	decision := planner.resolvePick(ctx, cfg, inst, draw)
	if decision.Skip {
		return nil, errors.New("formal command pick strategy skipped target")
	}
	content := normalizeResolvedBetContent(cfg, &decision)
	evaluation := evaluatePlayHit(cfg.Play, nil, content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)
	syncEvalBetUnitsWithWire(cfg.Play, content, &evaluation)
	if evaluation.BetUnits <= 0 {
		return nil, guajibet.ErrZeroBets
	}
	if max := maxBetUnitsForPlay(cfg.Play); max > 0 && evaluation.BetUnits > max {
		return nil, errMaxBetUnitsExceeded(max)
	}

	cat, err := q.GetLotteryCatalogByCode(ctx, inst.LotteryCode)
	if err != nil {
		return nil, err
	}
	gameID := strings.TrimSpace(textVal(cat.OutboundLotteryCode))
	if gameID == "" {
		gameID = inst.LotteryCode
	}
	ruleID, subPlay, err := resolveOutboundPlayCode(ctx, q, cfg, textVal(cat.PlayTemplate))
	if err != nil {
		return nil, err
	}
	amountUnit := cfg.BetUnitYuan
	if amountUnit <= 0 {
		amountUnit = baseBetUnitYuan
	}
	multiple := betMultipleAsInt(betMultiple)
	template := strings.TrimSpace(cfg.Play.PlayTemplate)
	if template == "" {
		template = strings.TrimSpace(textVal(cat.PlayTemplate))
	}
	ruleMeta := guajibet.ParseRuleMeta(template, subPlay.TypeID, subPlay.SubID, strings.TrimSpace(subPlay.Label), cfg.PlayTypeLabel, subPlay.SegmentRule, ruleID)
	if mode := strings.TrimSpace(cfg.Play.BetMode); mode != "" {
		ruleMeta.ForcedBetMode = mode
	}
	normalized := normalizeZhixuanDanshiContent(cfg.Play, content)
	wireContent := guajibet.FormatBetContentForRule(ruleMeta, normalized)
	bets := guajibet.ResolveBetsNums(ruleMeta, wireContent, 0, amountUnit, multiple)
	if bets <= 0 {
		return nil, guajibet.ErrZeroBets
	}
	amount := calcBetAmount(bets, float64(multiple), amountUnit)
	if betAmountExceedsMax(amount) {
		return nil, errMaxBetAmountExceeded(cfg.Currency)
	}
	var memberAccount string
	if err := p.pool.QueryRow(ctx, `SELECT account FROM members WHERE id = $1`, inst.MemberID).Scan(&memberAccount); err != nil {
		return nil, err
	}
	memberAccount = strings.TrimSpace(memberAccount)
	if memberAccount == "" {
		return nil, fmt.Errorf("member %d has no account", inst.MemberID)
	}
	ruleSnapshot, ok := planner.resolvePublishedRule(inst.LotteryCode, cfg.Play)
	if !ok {
		return nil, errors.New("published rule snapshot unavailable for formal command")
	}
	ruleSnapshotJSON, err := json.Marshal(ruleSnapshot)
	if err != nil {
		return nil, err
	}
	frozen := schemebettingdispatch.FrozenGuajiRequest{
		RequestID:     requestID,
		MemberAccount: memberAccount,
		SchemeName:    inst.SchemeName, LotteryLabel: inst.LotteryLabel,
		LotteryCategory: lotteryCategoryForCode(inst.LotteryCode), DefinitionID: inst.DefinitionID,
		PlayType: cloudPlayTypeLabel(cfg.PlayTypeLabel, subPlay.Label), RoundLabel: betRoundLabel(cfg, roundIndex, int(inst.PickIndex)),
		BetContent: normalized, BetUnits: bets,
		RuleSnapshot: ruleSnapshotJSON, RuleVersion: ruleSnapshot.RuleVersion,
		RuleSnapshotHash: ruleSnapshot.ContentHash,
		Request: guajibet.Request{
			LotteryCode: inst.LotteryCode, GameID: gameID, RuleID: ruleID, IssueNo: targetPeriod,
			Content: wireContent, PlayMethod: playMethodForPayload(cfg.PlayTypeLabel, subPlay.Label),
			Amount: amount, Multiplier: multiple, BetsNums: bets, AmountUnit: amountUnit,
			Currency: cfg.Currency, RuleMeta: ruleMeta,
		},
	}
	return json.Marshal(frozen)
}
