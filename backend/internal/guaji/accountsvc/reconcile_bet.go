package accountsvc

import (
	"context"
	"fmt"
	"strings"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
)

var _ guajibet.AcceptanceResolver = (*Service)(nil)
var _ guajibet.AcceptanceBatchResolver = (*Service)(nil)

// ResolveAcceptedBet performs a read-only exact lookup for a previously frozen
// request. It never retries PlaceRealBet and therefore cannot duplicate a bet.
func (s *Service) ResolveAcceptedBet(ctx context.Context, memberAccount string, req guajibet.Request) (guajibet.Result, error) {
	lookups := s.ResolveAcceptedBets(ctx, memberAccount, []guajibet.Request{req})
	return lookups[0].Result, lookups[0].Err
}

// ResolveAcceptedBets shares one provider order-list read across all frozen
// requests for the same member account.
func (s *Service) ResolveAcceptedBets(ctx context.Context, memberAccount string, requests []guajibet.Request) []guajibet.AcceptanceLookup {
	lookups := make([]guajibet.AcceptanceLookup, len(requests))
	if len(requests) == 0 {
		return lookups
	}
	if s == nil || s.guaji == nil || !s.guaji.Enabled() {
		return failedAcceptanceLookups(lookups, guajibet.ErrPlaceRejected)
	}
	memberID, err := s.memberID(ctx, memberAccount)
	if err != nil {
		return failedAcceptanceLookups(lookups, err)
	}
	account, err := s.getActiveRow(ctx, memberID)
	if err != nil {
		if isNoRows(err) {
			return failedAcceptanceLookups(lookups, guajibet.ErrNoActiveAuth)
		}
		return failedAcceptanceLookups(lookups, err)
	}
	if !s.tokenHealthy(account) {
		return failedAcceptanceLookups(lookups, guajibet.ErrNoActiveAuth)
	}
	token, err := guaji.DecryptSecret(s.credKey, account.accessTokenEnc.String)
	if err != nil || strings.TrimSpace(token) == "" {
		return failedAcceptanceLookups(lookups, guajibet.ErrTokenInvalid)
	}
	queries := make([]guaji.ExactLottBetQuery, len(requests))
	currencies := make([]string, len(requests))
	valid := make([]bool, len(requests))
	for i, req := range requests {
		currency := strings.TrimSpace(req.Currency)
		if currency == "" {
			currency, err = s.primaryCurrency(ctx, memberID)
			if err != nil {
				lookups[i].Err = err
				continue
			}
		}
		currency = guaji.NormalizeCurrency(currency)
		wireRequest, buildErr := s.buildReconcileWireRequest(ctx, currency, req)
		if buildErr != nil {
			lookups[i].Err = buildErr
			continue
		}
		queries[i] = guaji.ExactLottBetQuery{Request: wireRequest, Periods: req.IssueNo}
		currencies[i] = currency
		valid[i] = true
	}
	found, findErrs := s.guaji.FindExactLottBets(ctx, token, queries)
	for i, req := range requests {
		if !valid[i] {
			continue
		}
		if findErrs[i] != nil {
			lookups[i].Err = findErrs[i]
			continue
		}
		lookups[i].Result = guajibet.Result{
			GuajiAccountID:  account.id,
			ThirdPartyBetID: found[i].ThirdPartyBetID,
			Periods:         found[i].Periods,
			Currency:        currencies[i],
			Amount:          acceptedFinancialAmount(req.Amount, found[i].Amount),
		}
	}
	return lookups
}

func failedAcceptanceLookups(lookups []guajibet.AcceptanceLookup, err error) []guajibet.AcceptanceLookup {
	for i := range lookups {
		lookups[i].Err = err
	}
	return lookups
}

func (s *Service) buildReconcileWireRequest(ctx context.Context, currency string, req guajibet.Request) (guaji.LottBetRequest, error) {
	gameID, err := s.resolveGameID(ctx, req.LotteryCode, req.GameID)
	if err != nil {
		return guaji.LottBetRequest{}, err
	}
	if !guajibet.IsNumericGuajiRuleID(req.RuleID) {
		return guaji.LottBetRequest{}, fmt.Errorf("rule_id=%q must be a numeric provider id", req.RuleID)
	}
	multiplier := req.Multiplier
	if multiplier <= 0 {
		multiplier = 1
	}
	unit := req.AmountUnit
	if unit <= 0 {
		unit = 2
	}
	content := req.Content
	if strings.TrimSpace(req.RuleMeta.PlayTemplate) != "" {
		if formatted := guajibet.FormatBetContentForRule(req.RuleMeta, content); strings.TrimSpace(formatted) != "" {
			content = formatted
		}
	}
	betsNums := req.BetsNums
	if resolved := guajibet.ResolveBetsNums(req.RuleMeta, content, req.Amount, unit, multiplier); resolved > 0 {
		betsNums = resolved
	}
	if betsNums <= 0 || guajibet.IsFushiBaoziZeroBet(req.RuleMeta, content) {
		return guaji.LottBetRequest{}, guajibet.ErrZeroBets
	}
	item := lottBetContentForRequest(req.RuleMeta, content, unit, betsNums, multiplier)
	item.RuleID = req.RuleID
	return guaji.LottBetRequest{
		AutoType:    "platform",
		BetContents: []guaji.LottBetContent{item},
		GameID:      gameID,
		Currency:    guaji.CurrencyCode(currency),
		BetMultiple: []guaji.LottBetMultipleOuter{},
	}, nil
}
