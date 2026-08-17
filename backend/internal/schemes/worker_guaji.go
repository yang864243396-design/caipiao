package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/member"
)

// guajiBetPlacer 与 guajibet.Placer 对齐；accountsvc.Service 实现该接口。
type guajiBetPlacer = guajibet.Placer

// errGuajiPlacePreflight 表示尚未调用第三方 Place（可安全释放占位重试）。
// 一旦进入 PlaceRealBet，失败一律保留占位，避免「上游已接单、本地删占位」同期限连打。
var errGuajiPlacePreflight = errors.New("guaji place preflight")

func preflightPlaceErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", errGuajiPlacePreflight, err)
}

type schemeGuajiBetMeta struct {
	OrderNo         string
	ThirdPartyBetID string
	Periods         string
	Amount          float64 // 与第三方 bets_nums×单位×倍数对齐后的实扣金额
	BetsNums        int     // wire 口径注数；供 cloud_bet_records.bet_units
	PlayType        string  // 大类 · 子玩法展示文案
	GroupContent    string  // 实际 groupContent，回写 cloud.bet_content
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
		return schemeGuajiBetMeta{}, preflightPlaceErr(errors.New("guaji disabled"))
	}
	var account string
	if err := w.pool.QueryRow(ctx, `SELECT account FROM members WHERE id = $1`, inst.MemberID).Scan(&account); err != nil {
		return schemeGuajiBetMeta{}, preflightPlaceErr(err)
	}
	account = strings.TrimSpace(account)
	if account == "" {
		return schemeGuajiBetMeta{}, preflightPlaceErr(member.ErrNotFound)
	}

	cat, err := qtx.GetLotteryCatalogByCode(ctx, inst.LotteryCode)
	if err != nil {
		return schemeGuajiBetMeta{}, preflightPlaceErr(err)
	}
	gameID := strings.TrimSpace(textVal(cat.OutboundLotteryCode))
	if gameID == "" {
		gameID = inst.LotteryCode
	}
	ruleID, subPlay, err := resolveOutboundPlayCode(ctx, qtx, cfg, textVal(cat.PlayTemplate))
	if err != nil {
		return schemeGuajiBetMeta{}, preflightPlaceErr(err)
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
	// 方案 betMode 优先：防文案推断把「组选和值」打成「组选复式」→ 单和值 content=1 计 0 注。
	if bm := strings.TrimSpace(cfg.Play.BetMode); bm != "" {
		ruleMeta.ForcedBetMode = bm
	}
	// 冷热按位号池先展开为单式整注，并在调用第三方之前完成本端校验落库载荷。
	// 避免「先下到第三方、后 Normalize 失败」造成对账孤儿单。
	normalizedContent := normalizeZhixuanDanshiContent(cfg.Play, betContent)
	payload, err := NormalizeBetPayload(BetPayload{
		PlayTemplate: cfg.Play.PlayTemplate,
		TypeID:       cfg.Play.PlayTypeID,
		SubID:        cfg.Play.SubPlayID,
		BetMode:      cfg.Play.BetMode,
		GroupContent: normalizedContent,
		PlayMethod:   playMethodForPayload(cfg.PlayTypeLabel, subPlay.Label),
	})
	if err != nil {
		return schemeGuajiBetMeta{}, preflightPlaceErr(err)
	}
	var normalizedPayload BetPayload
	if err := json.Unmarshal(payload, &normalizedPayload); err == nil && strings.TrimSpace(normalizedPayload.GroupContent) != "" {
		normalizedContent = normalizedPayload.GroupContent
	}
	guajiContent := guajibet.FormatBetContentForRule(ruleMeta, normalizedContent)
	if guajibet.IsFushiBaoziZeroBet(ruleMeta, guajiContent) {
		return schemeGuajiBetMeta{}, preflightPlaceErr(fmt.Errorf("%w: %w", guajibet.ErrPlaceRejected, guajibet.ErrZeroBets))
	}
	betsNums = guajibet.ResolveBetsNums(ruleMeta, guajiContent, amount, amountUnit, multInt)
	if betsNums <= 0 {
		// 组选号池不足等：0 注，勿带非法 content 撞第三方「投注数字不合规」
		return schemeGuajiBetMeta{}, preflightPlaceErr(fmt.Errorf("%w: %w", guajibet.ErrPlaceRejected, guajibet.ErrZeroBets))
	}
	// 按玩法独立上限拦一层（随机出号侧会重抽；此处防其它运行模式/缓存漏网）。
	if max := maxBetUnitsForPlay(cfg.Play); max > 0 && betsNums > max {
		return schemeGuajiBetMeta{}, preflightPlaceErr(fmt.Errorf("%w: %w", guajibet.ErrPlaceRejected, errMaxBetUnitsExceeded(max)))
	}
	// 本端 evaluate 注数偶发偏少（如单式未按逗号切分）；以 wire 注数为准同步金额，避免账本少扣、对账差一倍。
	amount = calcBetAmount(betsNums, float64(multInt), amountUnit)
	// 第三方单次金额上限：本端预检，避免撞 guaji 40053。
	if betAmountExceedsMax(amount) {
		return schemeGuajiBetMeta{}, preflightPlaceErr(fmt.Errorf("%w: %w", guajibet.ErrPlaceRejected, errMaxBetAmountExceeded(cfg.Currency)))
	}

	periodNo := strings.TrimSpace(draw.IssueNo)
	if !guajiBetPeriodMatches(inst.LotteryCode, periodNo) {
		return schemeGuajiBetMeta{}, preflightPlaceErr(fmt.Errorf("%w: period %s not current guaji open issue", guajibet.ErrPeriodClosed, periodNo))
	}

	if slotErr := w.acquirePlaceSlot(ctx); slotErr != nil {
		return schemeGuajiBetMeta{}, preflightPlaceErr(slotErr)
	}
	defer w.releasePlaceSlot()

	placeReq := guajibet.Request{
		LotteryCode: inst.LotteryCode,
		GameID:      gameID,
		RuleID:      ruleID,
		IssueNo:     draw.IssueNo,
		Content:     guajiContent,
		PlayMethod:  cfg.PlayTypeLabel,
		Amount:      amount,
		Multiplier:  multInt,
		BetsNums:    betsNums,
		AmountUnit:  amountUnit,
		Currency:    cfg.Currency,
		RuleMeta:    ruleMeta,
	}
	var betRes guajibet.Result
	var placeErr error
	for attempt := 1; attempt <= guajiPlaceSafeRetryAttempts; attempt++ {
		// A wait for a shared place slot or a retry backoff can consume the final
		// part of a very short period. Recheck immediately before every request;
		// never let the upstream roll this bet into the next issue.
		if !guajiBetPeriodMatches(inst.LotteryCode, periodNo) || !guajiBetPeriodHasSafeWindowAt(inst.LotteryCode, time.Now()) {
			err := fmt.Errorf("%w: period %s is too close to close", guajibet.ErrPeriodClosed, periodNo)
			if attempt == 1 {
				return schemeGuajiBetMeta{}, preflightPlaceErr(err)
			}
			// An earlier attempt may already have reached the upstream. Keep its
			// period claim rather than risking a duplicate by releasing it.
			return schemeGuajiBetMeta{}, err
		}
		betRes, placeErr = w.guajiBets.PlaceRealBet(ctx, account, placeReq)
		if placeErr == nil {
			break
		}
		if errors.Is(placeErr, guajibet.ErrInsufficient) {
			return schemeGuajiBetMeta{}, member.ErrInsufficientFunds
		}
		if errors.Is(placeErr, guajibet.ErrPeriodClosed) {
			return schemeGuajiBetMeta{}, guajibet.ErrPeriodClosed
		}
		if attempt < guajiPlaceSafeRetryAttempts && guaji.IsSafeImmediateRetryError(placeErr) {
			slog.Warn("scheme worker place bet safe-retry",
				"instanceId", inst.ID, "attempt", attempt, "err", placeErr)
			select {
			case <-ctx.Done():
				return schemeGuajiBetMeta{}, ctx.Err()
			case <-time.After(guajiPlaceSafeRetryBackoff):
			}
			continue
		}
		return schemeGuajiBetMeta{}, placeErr
	}
	if placeErr != nil {
		return schemeGuajiBetMeta{}, placeErr
	}
	returned := strings.TrimSpace(betRes.Periods)
	if returned == "" {
		return schemeGuajiBetMeta{}, fmt.Errorf("%w: upstream did not return periods", guajibet.ErrPlaceRejected)
	}

	meta := schemeGuajiBetMeta{
		ThirdPartyBetID: strings.TrimSpace(betRes.ThirdPartyBetID),
		Periods:         returned,
		Amount:          amount,
		BetsNums:        betsNums,
		PlayType:        cloudPlayTypeLabel(cfg.PlayTypeLabel, subPlay.Label),
		GroupContent:    normalizedContent,
	}

	// 并发下单同一毫秒会撞 uq_bet_orders_order_no；带实例尾缀 + 纳秒降低冲突。
	orderNo := fmt.Sprintf("BO%d%d%s", inst.MemberID, time.Now().UnixNano(), shortIDSuffix(inst.ID))
	outLottery := pgtype.Text{String: gameID, Valid: gameID != ""}
	outPlay := pgtype.Text{String: ruleID, Valid: ruleID != ""}
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
		// 第三方已接单：绝不可把错误抛给上层去「删占位重投」，否则同期限连打。
		slog.Warn("scheme worker bet_orders insert failed after upstream accept",
			"instanceId", inst.ID, "period", returned, "thirdPartyBetId", meta.ThirdPartyBetID, "err", err)
		return meta, nil
	}
	meta.OrderNo = orderNo
	if w.guajiBets != nil {
		if err := w.guajiBets.MirrorBetDebitLedger(ctx, qtx, inst.MemberID, orderNo, amount, betRes.GuajiAccountID, betRes.Currency); err != nil {
			slog.Warn("scheme worker ledger mirror failed after upstream accept",
				"instanceId", inst.ID, "period", returned, "orderNo", orderNo, "err", err)
			return meta, nil
		}
	}
	return meta, nil
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

// shortIDSuffix 取实例 id 尾部数字，缩短注单号碰撞面。
func shortIDSuffix(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "0"
	}
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] < '0' || id[i] > '9' {
			tail := id[i+1:]
			if tail == "" {
				return "0"
			}
			if len(tail) > 6 {
				return tail[len(tail)-6:]
			}
			return tail
		}
	}
	if len(id) > 6 {
		return id[len(id)-6:]
	}
	return id
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
	if refuseRandomDrawMaxUnitsPause(ctx, qtx, instanceID, StatusReasonBetFailed, detail) {
		slog.Info("scheme worker refuse pause: random_draw over max",
			"instanceId", instanceID, "detail", detail)
		return nil
	}
	_, err := qtx.PauseSchemeInstanceByWorker(ctx, sqlcdb.PauseSchemeInstanceByWorkerParams{
		ID:           instanceID,
		StatusReason: StatusReasonBetFailed,
		Column3:      detail,
	})
	return err
}
