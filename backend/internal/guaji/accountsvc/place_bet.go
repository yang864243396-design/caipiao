package accountsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
)

// roundLottBetAmount 计算第三方 bet_amount（unit×注数×倍数）。
// 请求中的 amount_unit、bets_nums、multiple、bet_amount 必须严格相乘一致；
// 因此保留系统投注单位的三位小数，不能在请求前截为两位。
// 账本/展示的实际扣款金额仍由 schemes.TruncateBetAmount 单独按两位处理。
func roundLottBetAmount(unit float64, betsNums, mult int) float64 {
	if unit <= 0 {
		unit = 2
	}
	if betsNums <= 0 {
		betsNums = 1
	}
	if mult <= 0 {
		mult = 1
	}
	return math.Round(unit*float64(betsNums)*float64(mult)*1000+1e-9) / 1000
}

func lottBetContentForRequest(meta guajibet.RuleMeta, content string, unit float64, betsNums, mult int) guaji.LottBetContent {
	item := guaji.LottBetContent{
		BetContent: content,
		AmountUnit: unit,
		BetsNums:   betsNums,
		Multiple:   mult,
		BetAmount:  roundLottBetAmount(unit, betsNums, mult),
		Solo:       guajibet.ResolveSolo(meta, content, betsNums),
	}
	if meta.PlayTemplate == "lhc_std" && guajibet.InferBetMode(meta) == "renyi_dp" && betsNums > 1 && mult > 1 {
		single := roundLottBetAmount(unit, 1, mult)
		item.SingleBetAmount = &single
	}
	return item
}

// Enabled 报告第三方对接是否启用（guajibet.Placer）。
func (s *Service) Enabled() bool {
	return s != nil && s.guaji != nil && s.guaji.Enabled()
}

func mapAuthErrToBet(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNoActiveAccount), errors.Is(err, ErrAccountNotFound):
		return guajibet.ErrNoActiveAuth
	case errors.Is(err, ErrTokenInvalid), errors.Is(err, ErrReauthNeedsBind):
		return guajibet.ErrTokenInvalid
	default:
		return guajibet.ErrTokenInvalid
	}
}

// PlaceRealBet 用会员启用授权下注（guajibet.Placer 实现，T4）：
// 取启用授权 Token → web_bets/lott 接单。
// token 失效时自动重新授权最多 3 次；成功则继续下注，失败则返回 ErrTokenInvalid（由 worker 按原逻辑停方案）。
func (s *Service) PlaceRealBet(ctx context.Context, memberAccount string, req guajibet.Request) (guajibet.Result, error) {
	if s == nil || s.guaji == nil || !s.guaji.Enabled() {
		return guajibet.Result{}, guajibet.ErrPlaceRejected
	}
	m, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return guajibet.Result{}, err
	}
	currency := strings.TrimSpace(req.Currency)
	if currency != "" {
		currency = guaji.NormalizeCurrency(currency)
	} else {
		currency, err = s.primaryCurrency(ctx, m)
		if err != nil {
			return guajibet.Result{}, err
		}
	}

	row, err := s.getActiveRow(ctx, m)
	if err != nil {
		if isNoRows(err) {
			return guajibet.Result{}, guajibet.ErrNoActiveAuth
		}
		return guajibet.Result{}, err
	}
	if !s.tokenHealthy(row) {
		if err := s.EnsureActiveAuth(ctx, memberAccount); err != nil {
			return guajibet.Result{}, mapAuthErrToBet(err)
		}
		row, err = s.getActiveRow(ctx, m)
		if err != nil {
			if isNoRows(err) {
				return guajibet.Result{}, guajibet.ErrNoActiveAuth
			}
			return guajibet.Result{}, err
		}
		if !s.tokenHealthy(row) {
			return guajibet.Result{}, guajibet.ErrTokenInvalid
		}
	}

	res, err := s.placeRealBetWithRow(ctx, memberAccount, m, currency, row, req)
	if !errors.Is(err, guajibet.ErrTokenInvalid) {
		return res, err
	}
	// 下注过程中令牌被上游判无效：强制自动授权最多 3 次后再试一单。
	if ensureErr := s.ensureActiveAuth(ctx, memberAccount, true); ensureErr != nil {
		return guajibet.Result{}, mapAuthErrToBet(ensureErr)
	}
	row, err = s.getActiveRow(ctx, m)
	if err != nil {
		if isNoRows(err) {
			return guajibet.Result{}, guajibet.ErrNoActiveAuth
		}
		return guajibet.Result{}, err
	}
	if !s.tokenHealthy(row) {
		return guajibet.Result{}, guajibet.ErrTokenInvalid
	}
	return s.placeRealBetWithRow(ctx, memberAccount, m, currency, row, req)
}

func (s *Service) placeRealBetWithRow(
	ctx context.Context,
	memberAccount string,
	memberID int64,
	currency string,
	row row,
	req guajibet.Request,
) (guajibet.Result, error) {
	token, err := guaji.DecryptSecret(s.credKey, row.accessTokenEnc.String)
	if err != nil {
		return guajibet.Result{}, err
	}
	if strings.TrimSpace(token) == "" {
		return guajibet.Result{}, guajibet.ErrNoActiveAuth
	}

	gameID, err := s.resolveGameID(ctx, req.LotteryCode, req.GameID)
	if err != nil {
		return guajibet.Result{}, fmt.Errorf("%w: %w", guajibet.ErrPlaceRejected, err)
	}
	if !guajibet.IsNumericGuajiRuleID(req.RuleID) {
		return guajibet.Result{}, fmt.Errorf("%w: rule_id=%q 须为第三方数字 ID，请执行 guaji-rules-sync", guajibet.ErrPlaceRejected, req.RuleID)
	}
	mult := req.Multiplier
	if mult <= 0 {
		mult = 1
	}
	unit := req.AmountUnit
	if unit <= 0 {
		unit = 2
	}
	// 再 format 一次（幂等）：防止冷热按位换行号池未转 wire 时 Count=1 + solo=true → 单挑参数错误
	content := req.Content
	if strings.TrimSpace(req.RuleMeta.PlayTemplate) != "" {
		if formatted := guajibet.FormatBetContentForRule(req.RuleMeta, req.Content); strings.TrimSpace(formatted) != "" {
			content = formatted
		}
	}
	if guajibet.IsFushiBaoziZeroBet(req.RuleMeta, content) {
		return guajibet.Result{}, fmt.Errorf("%w: %w", guajibet.ErrPlaceRejected, guajibet.ErrZeroBets)
	}
	betsNums := req.BetsNums
	if n := guajibet.ResolveBetsNums(req.RuleMeta, content, req.Amount, unit, mult); n > 0 {
		betsNums = n
	}
	// 组六不足 3 码等计 0 注：勿回落成 1，否则会带着非法 content 去撞「单挑参数错误」
	if betsNums <= 0 {
		return guajibet.Result{}, fmt.Errorf("%w: %w", guajibet.ErrPlaceRejected, guajibet.ErrZeroBets)
	}
	item := lottBetContentForRequest(req.RuleMeta, content, unit, betsNums, mult)
	item.RuleID = req.RuleID
	betRes, err := s.guaji.PlaceLottBet(ctx, token, guaji.LottBetRequest{
		AutoType:    "platform",
		BetContents: []guaji.LottBetContent{item},
		GameID:      gameID,
		Currency:    guaji.CurrencyCode(currency),
		BetMultiple: []guaji.LottBetMultipleOuter{},
	})
	if err != nil {
		if businessErr := placeLottBetBusinessError(err); businessErr != nil {
			return guajibet.Result{}, businessErr
		}
		fault := guaji.ClassifyUpstreamError(err)
		if fault.IsTokenInvalid {
			_ = s.markTokenError(ctx, memberID, row.id, row.accessTokenEnc.String, fault.UserMessage)
			return guajibet.Result{}, guajibet.ErrTokenInvalid
		}
		slog.Warn("guaji place bet rejected", "member", memberAccount, "gameId", gameID, "ruleId", req.RuleID, "issue", req.IssueNo, "content", content, "betsNums", betsNums, "solo", item.Solo, "amount", item.BetAmount, "err", err)
		return guajibet.Result{}, fmt.Errorf("%w: %w", guajibet.ErrPlaceRejected, err)
	}
	periods := strings.TrimSpace(betRes.Periods)
	if periods == "" {
		return guajibet.Result{}, fmt.Errorf("%w: upstream did not return periods", guajibet.ErrPlaceRejected)
	}
	expected := strings.TrimSpace(req.IssueNo)
	if expected != "" && periods != expected {
		slog.Warn("guaji place bet period mismatch, trust upstream periods",
			"gameId", gameID, "expected", expected, "got", periods)
	}
	if strings.TrimSpace(betRes.ThirdPartyBetID) == "" {
		return guajibet.Result{}, guajibet.ErrPlaceRejected
	}

	return guajibet.Result{
		GuajiAccountID:  row.id,
		ThirdPartyBetID: betRes.ThirdPartyBetID,
		Periods:         periods,
		Currency:        currency,
	}, nil
}

func placeLottBetBusinessError(err error) error {
	switch {
	case guaji.IsInsufficientBalanceError(err):
		return guajibet.ErrInsufficient
	case guaji.IsPeriodClosedError(err):
		return guajibet.ErrPeriodClosed
	default:
		return nil
	}
}
