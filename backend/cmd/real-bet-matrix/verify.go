package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
)

type verifyOutcome struct {
	Status string // ok | mismatch | not_found | skipped
	Detail string
}

func verifyThirdPartyBet(
	ctx context.Context,
	client *guaji.Client,
	accounts *accountsvc.Service,
	account, betID, gameID, issueNo, ruleID string,
	expectedAmount float64,
) verifyOutcome {
	betID = strings.TrimSpace(betID)
	if betID == "" {
		return verifyOutcome{Status: "skipped", Detail: "无 thirdPartyBetId"}
	}
	if client == nil || accounts == nil {
		return verifyOutcome{Status: "skipped", Detail: "guaji 未启用"}
	}
	token, err := accounts.MemberAccessToken(ctx, account)
	if err != nil {
		return verifyOutcome{Status: "not_found", Detail: fmt.Sprintf("token: %v", err)}
	}
	raw, err := client.GetWebBetRaw(ctx, token, betID)
	if err != nil {
		return verifyOutcome{Status: "not_found", Detail: err.Error()}
	}
	var mismatches []string
	if wantGame, err := strconv.Atoi(strings.TrimSpace(gameID)); err == nil && wantGame > 0 {
		if got := intNum(raw["game_id"]); got != int64(wantGame) {
			mismatches = append(mismatches, fmt.Sprintf("game_id want=%d got=%d", wantGame, got))
		}
	}
	if issue := strings.TrimSpace(issueNo); issue != "" {
		if got := strings.TrimSpace(fmt.Sprint(raw["periods"])); got != "" && got != issue {
			mismatches = append(mismatches, fmt.Sprintf("periods want=%q got=%q", issue, got))
		}
	}
	if expectedAmount > 0 {
		gotAmt := floatNum(raw["bet_amount"])
		if math.Abs(gotAmt-expectedAmount) > 0.02 {
			mismatches = append(mismatches, fmt.Sprintf("bet_amount want=%.2f got=%.2f", expectedAmount, gotAmt))
		}
	}
	if rule := strings.TrimSpace(ruleID); rule != "" {
		if gotRule := extractRuleIDFromRaw(raw); gotRule != "" && gotRule != rule {
			mismatches = append(mismatches, fmt.Sprintf("rule_id want=%q got=%q", rule, gotRule))
		}
	}
	if len(mismatches) > 0 {
		return verifyOutcome{Status: "mismatch", Detail: strings.Join(mismatches, "; ")}
	}
	return verifyOutcome{Status: "ok", Detail: "第三方注单一致"}
}

func extractRuleIDFromRaw(raw map[string]any) string {
	contents, ok := raw["bet_contents"].([]any)
	if !ok || len(contents) == 0 {
		return ""
	}
	first, ok := contents[0].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(first["rule_id"]))
}

func intNum(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}

func floatNum(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	default:
		return 0
	}
}
