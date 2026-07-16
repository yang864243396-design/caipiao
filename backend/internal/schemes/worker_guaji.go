package schemes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/member"
)

// guajiBetPlacer 与 guajibet.Placer 对齐；accountsvc.Service 实现该接口。
type guajiBetPlacer = guajibet.Placer

type schemeGuajiBetMeta struct {
	OrderNo         string
	ThirdPartyBetID string
	Periods         string
	Amount          float64 // 与第三方 bets_nums×单位×倍数对齐后的实扣金额
}

func (w *Worker) SetGuajiBetPlacer(p guajiBetPlacer) {
	if w == nil {
		return
	}
	w.guajiBets = p
}

func (w *Worker) guajiRealEnabled() bool {
	return w != nil && w.guajiBets != nil && w.guajiBets.Enabled()
}

func (w *Worker) placeGuajiSchemeBet(
	ctx context.Context,
	qtx *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	cfg parsedSchemeConfig,
	draw sqlcdb.LotteryDraw,
	betContent string,
	amount float64,
	betsNums int,
	mult float64,
) (schemeGuajiBetMeta, error) {
	if !w.guajiRealEnabled() {
		return schemeGuajiBetMeta{}, errors.New("guaji disabled")
	}
	var account string
	if err := w.pool.QueryRow(ctx, `SELECT account FROM members WHERE id = $1`, inst.MemberID).Scan(&account); err != nil {
		return schemeGuajiBetMeta{}, err
	}
	account = strings.TrimSpace(account)
	if account == "" {
		return schemeGuajiBetMeta{}, member.ErrNotFound
	}

	cat, err := qtx.GetLotteryCatalogByCode(ctx, inst.LotteryCode)
	if err != nil {
		return schemeGuajiBetMeta{}, err
	}
	gameID := strings.TrimSpace(textVal(cat.OutboundLotteryCode))
	if gameID == "" {
		gameID = inst.LotteryCode
	}
	ruleID, subPlay, err := resolveOutboundPlayCode(ctx, qtx, cfg, textVal(cat.PlayTemplate))
	if err != nil {
		return schemeGuajiBetMeta{}, err
	}
	multInt := betMultipleAsInt(mult)
	amountUnit := cfg.BetUnitYuan
	if amountUnit <= 0 {
		amountUnit = baseBetUnitYuan
	}
	tpl := strings.TrimSpace(cfg.Play.PlayTemplate)
	if tpl == "" {
		tpl = strings.TrimSpace(textVal(cat.PlayTemplate))
	}
	ruleMeta := guajibet.ParseRuleMeta(
		tpl,
		subPlay.TypeID,
		subPlay.SubID,
		strings.TrimSpace(subPlay.Label),
		cfg.PlayTypeLabel,
		subPlay.SegmentRule,
		ruleID,
	)
	guajiContent := guajibet.FormatBetContentForRule(ruleMeta, betContent)
	betsNums = guajibet.ResolveBetsNums(ruleMeta, guajiContent, amount, amountUnit, multInt)
	if betsNums <= 0 {
		betsNums = 1
	}
	// 本端 evaluate 注数偶发偏少（如单式未按逗号切分）；以 wire 注数为准同步金额，避免账本少扣、对账差一倍。
	amount = calcBetAmount(betsNums, float64(multInt), amountUnit)

	periodNo := strings.TrimSpace(draw.IssueNo)
	if !guajiBetPeriodMatches(inst.LotteryCode, periodNo) {
		return schemeGuajiBetMeta{}, fmt.Errorf("%w: period %s not current guaji open issue", guajibet.ErrPeriodClosed, periodNo)
	}

	betRes, err := w.guajiBets.PlaceRealBet(ctx, account, guajibet.Request{
		LotteryCode: inst.LotteryCode,
		GameID:     gameID,
		RuleID:     ruleID,
		IssueNo:    draw.IssueNo,
		Content:    guajiContent,
		PlayMethod: cfg.PlayTypeLabel,
		Amount:     amount,
		Multiplier: multInt,
		BetsNums:   betsNums,
		AmountUnit: amountUnit,
		RuleMeta:   ruleMeta,
	})
	if err != nil {
		if errors.Is(err, guajibet.ErrInsufficient) {
			return schemeGuajiBetMeta{}, member.ErrInsufficientFunds
		}
		if errors.Is(err, guajibet.ErrPeriodClosed) {
			return schemeGuajiBetMeta{}, guajibet.ErrPeriodClosed
		}
		return schemeGuajiBetMeta{}, err
	}
	returned := strings.TrimSpace(betRes.Periods)
	if returned == "" {
		return schemeGuajiBetMeta{}, fmt.Errorf("%w: upstream did not return periods", guajibet.ErrPlaceRejected)
	}

	orderNo := fmt.Sprintf("BO%d%d", inst.MemberID, time.Now().UnixMilli())
	outLottery := pgtype.Text{String: gameID, Valid: gameID != ""}
	outPlay := pgtype.Text{String: ruleID, Valid: ruleID != ""}
	payload, err := NormalizeBetPayload(BetPayload{
		PlayTemplate: cfg.Play.PlayTemplate,
		TypeID:       cfg.Play.PlayTypeID,
		SubID:        cfg.Play.SubPlayID,
		BetMode:      cfg.Play.BetMode,
		GroupContent: betContent,
		PlayMethod:   playMethodForPayload(cfg.PlayTypeLabel, subPlay.Label),
	})
	if err != nil {
		return schemeGuajiBetMeta{}, err
	}
	_, err = qtx.InsertBetOrder(ctx, sqlcdb.InsertBetOrderParams{
		OrderNo:             orderNo,
		MemberID:            inst.MemberID,
		LotteryCode:         inst.LotteryCode,
		LotteryName:         inst.LotteryLabel,
		LotteryCategory:     lotteryCategoryForCode(inst.LotteryCode),
		IssueNo:             returned,
		Amount:              numericFromFloat(amount),
		PlayMethod:          pgtype.Text{String: playMethodForPayload(cfg.PlayTypeLabel, subPlay.Label), Valid: true},
		BetPayload:          payload,
		OutboundLotteryCode: outLottery,
		OutboundPlayCode:    outPlay,
		GuajiAccountID:      pgtype.Int8{Int64: betRes.GuajiAccountID, Valid: betRes.GuajiAccountID != 0},
		ThirdPartyBetID:     pgtype.Text{String: betRes.ThirdPartyBetID, Valid: betRes.ThirdPartyBetID != ""},
		Currency:            pgtype.Text{String: betRes.Currency, Valid: betRes.Currency != ""},
	})
	if err != nil {
		return schemeGuajiBetMeta{}, err
	}
	if w.guajiBets != nil {
		if err := w.guajiBets.MirrorBetDebitLedger(ctx, qtx, inst.MemberID, orderNo, amount, betRes.GuajiAccountID, betRes.Currency); err != nil {
			return schemeGuajiBetMeta{}, err
		}
	}
	return schemeGuajiBetMeta{
		OrderNo:         orderNo,
		ThirdPartyBetID: strings.TrimSpace(betRes.ThirdPartyBetID),
		Periods:         returned,
		Amount:          amount,
	}, nil
}

func textVal(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// playMethodForPayload 合并大类+子玩法中文名，供结算识别（如「大小单双 五星和值大小」）。
func playMethodForPayload(playTypeLabel, subLabel string) string {
	play := strings.TrimSpace(playTypeLabel)
	sub := strings.TrimSpace(subLabel)
	switch {
	case play == "" && sub == "":
		return ""
	case play == "":
		return sub
	case sub == "":
		return play
	case strings.Contains(play, sub) || strings.Contains(sub, play):
		if len(sub) >= len(play) {
			return sub
		}
		return play
	default:
		return play + " " + sub
	}
}

func resolveOutboundPlayCode(ctx context.Context, q *sqlcdb.Queries, cfg parsedSchemeConfig, template string) (string, sqlcdb.GetSubPlayRow, error) {
	tpl := strings.TrimSpace(cfg.Play.PlayTemplate)
	if tpl == "" {
		tpl = strings.TrimSpace(template)
	}
	typeID := strings.TrimSpace(cfg.Play.PlayTypeID)
	// 优先 catalog 数字 subId（如 "120"）；语义 zhixuan_fs 仅作回退。
	subID := strings.TrimSpace(cfg.Play.CatalogSubID)
	if subID == "" {
		subID = strings.TrimSpace(cfg.Play.SubPlayID)
	}
	betMode := strings.TrimSpace(cfg.Play.BetMode)
	if subID == "" {
		switch {
		case typeID == "dingwei" || betMode == "dingwei":
			subID = "dingwei"
			if betMode == "" {
				betMode = "dingwei"
			}
		case betMode != "" && !isBetUnitArtifact(betMode):
			subID = betMode
		}
	}
	if tpl == "" || typeID == "" || subID == "" {
		return "", sqlcdb.GetSubPlayRow{}, fmt.Errorf("resolve rule_id: missing play template/type/sub")
	}
	sub, err := lookupSubPlay(ctx, q, tpl, typeID, subID, betMode, cfg.Play.PositionIdx)
	if err != nil {
		return "", sqlcdb.GetSubPlayRow{}, fmt.Errorf("resolve rule_id: %w", err)
	}
	ruleID := resolveGuajiRuleIDFromSubPlay(sub)
	if ruleID == "" {
		outbound := textVal(sub.OutboundPlayCode)
		return "", sub, fmt.Errorf("%w: %s/%s/%s outbound=%q", errGuajiRuleIDMissing, tpl, typeID, sub.SubID, outbound)
	}
	return ruleID, sub, nil
}

// isNumericBetModeArtifact 已废弃，请用 isBetUnitArtifact。
func isNumericBetModeArtifact(betMode string) bool {
	return isBetUnitArtifact(betMode)
}

func lotteryCategoryForCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	switch {
	case strings.Contains(code, "pk10"), strings.Contains(code, "feiting"):
		return "pk10"
	case strings.Contains(code, "syxw"):
		return "x5"
	case strings.Contains(code, "k3"):
		return "k3"
	default:
		return "ssc"
	}
}

func pauseInstanceForInsufficientFunds(ctx context.Context, qtx *sqlcdb.Queries, instanceID string) error {
	_, err := qtx.PauseSchemeInstanceByWorker(ctx, sqlcdb.PauseSchemeInstanceByWorkerParams{
		ID:           instanceID,
		StatusReason: StatusReasonInsufficientFunds,
	})
	return err
}

func pauseInstanceForBetFailed(ctx context.Context, qtx *sqlcdb.Queries, instanceID, detail string) error {
	detail = normalizeBetFailedDetail(detail)
	_, err := qtx.PauseSchemeInstanceByWorker(ctx, sqlcdb.PauseSchemeInstanceByWorkerParams{
		ID:           instanceID,
		StatusReason: StatusReasonBetFailed,
		Column3:      detail,
	})
	return err
}
