package schemes

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

// AdminBetHistoryResult 管理端「投注与盈亏」抽屉数据（执行+账务合并为一行）。
type AdminBetHistoryResult struct {
	InstanceID string               `json:"instanceId"`
	SchemeName string               `json:"schemeName"`
	SimBet     bool                 `json:"simBet"`
	Days       int                  `json:"days"`
	DateFrom   string               `json:"dateFrom"`
	DateTo     string               `json:"dateTo"`
	Items      []AdminBetHistoryItem `json:"items"`
}

type AdminBetHistoryItem struct {
	ID               string  `json:"id"`
	SchemeInstanceID string  `json:"schemeInstanceId"`
	Time             string  `json:"time"`
	BetAt            string  `json:"betAt"`
	SchemeName       string  `json:"schemeName"`
	Numbers          string  `json:"numbers"`
	Period           string  `json:"period"`
	Draw             string  `json:"draw"`
	PlayMethod       string  `json:"playMethod"`
	Multiplier       string  `json:"multiplier"`
	Round            string  `json:"round"`
	Amount           string  `json:"amount"`
	ProfitLoss       float64 `json:"profitLoss"`
	Status           string  `json:"status"`
	// Win 仅已结算有值：true=中，false=挂；待开奖/撤单为 null。
	Win *bool `json:"win"`
}

func (s *Service) AdminBetHistory(ctx context.Context, instanceID string, days int) (AdminBetHistoryResult, error) {
	if s == nil || s.q == nil {
		return AdminBetHistoryResult{}, ErrUnavailable
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return AdminBetHistoryResult{}, ErrInstanceNotFound
	}
	if days <= 0 {
		days = 30
	}
	if days > 90 {
		days = 90
	}

	inst, err := s.q.GetSchemeInstanceByID(ctx, instanceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminBetHistoryResult{}, ErrInstanceNotFound
		}
		return AdminBetHistoryResult{}, err
	}

	dateFrom, dateTo, since, until := timeutil.NaturalDaysMeta(days)
	rows, err := s.q.ListCloudBetRecordsByScheme(ctx, sqlcdb.ListCloudBetRecordsBySchemeParams{
		MemberID: inst.MemberID,
		SimBet:   inst.SimBet,
		SchemeID: instanceID,
		SinceAt:  pgtype.Timestamptz{Time: since, Valid: true},
		UntilAt:  pgtype.Timestamptz{Time: until, Valid: true},
	})
	if err != nil {
		return AdminBetHistoryResult{}, err
	}

	drawCache := map[string]string{}
	items := make([]AdminBetHistoryItem, 0, len(rows))
	for _, r := range rows {
		statusLabel := mapAdminBetStatusLabel(r.Status)
		settled := statusLabel == "已结算"
		var win *bool
		if settled {
			w := isAdminBetWin(r.Status)
			win = &w
		}
		draw := ""
		if settled {
			draw = s.lookupAdminDrawBalls(ctx, inst.LotteryCode, r.PeriodNo, r.ThirdPartyPeriod, drawCache)
		}
		items = append(items, AdminBetHistoryItem{
			ID:               r.RecordNo,
			SchemeInstanceID: instanceID,
			Time:             formatAdminBetTime(r.PlacedAt),
			BetAt:            formatAdminBetAt(r.PlacedAt),
			SchemeName:       r.SchemeName,
			Numbers:          strings.TrimSpace(r.BetContent),
			Period:           adminBetDisplayPeriod(r),
			Draw:             draw,
			PlayMethod:       strings.TrimSpace(r.PlayType),
			Multiplier:       formatAdminMultiplier(r.Multiplier),
			Round:            formatAdminRound(r.RoundLabel),
			Amount:           formatAdminAmount(r.Amount),
			ProfitLoss:       roundAdminMoney(r.Pnl),
			Status:           statusLabel,
			Win:              win,
		})
	}

	return AdminBetHistoryResult{
		InstanceID: instanceID,
		SchemeName: inst.SchemeName,
		SimBet:     inst.SimBet,
		Days:       days,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Items:      items,
	}, nil
}

func (s *Service) lookupAdminDrawBalls(
	ctx context.Context,
	lotteryCode, periodNo string,
	thirdParty pgtype.Text,
	cache map[string]string,
) string {
	candidates := []string{strings.TrimSpace(periodNo)}
	if thirdParty.Valid {
		if p := strings.TrimSpace(thirdParty.String); p != "" && p != candidates[0] {
			candidates = append(candidates, p)
		}
	}
	for _, issue := range candidates {
		if issue == "" {
			continue
		}
		if v, ok := cache[issue]; ok {
			return v
		}
		row, err := s.q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
			LotteryCode: lotteryCode,
			IssueNo:     issue,
		})
		if err != nil {
			cache[issue] = ""
			continue
		}
		balls := sqlcdb.ParseDrawBalls(row.Balls)
		if len(balls) == 0 {
			cache[issue] = ""
			continue
		}
		joined := strings.Join(balls, " ")
		cache[issue] = joined
		return joined
	}
	return ""
}

func adminBetDisplayPeriod(r sqlcdb.ListCloudBetRecordsBySchemeRow) string {
	if r.ThirdPartyPeriod.Valid {
		if p := strings.TrimSpace(r.ThirdPartyPeriod.String); p != "" {
			return p
		}
	}
	return strings.TrimSpace(r.PeriodNo)
}

func formatAdminBetAt(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return timeutil.FormatDisplayCST(ts.Time)
}

func formatAdminBetTime(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.In(timeutil.PlatformLocation()).Format("15:04:05")
}

func mapAdminBetStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "hit", "miss", "won", "lost":
		return "已结算"
	case "cancelled":
		return "已撤单"
	case "pending":
		return "待开奖"
	default:
		if status == "" {
			return "待开奖"
		}
		return status
	}
}

func isAdminBetWin(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "hit", "won":
		return true
	default:
		return false
	}
}

func formatAdminMultiplier(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "1"
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		n := int(math.Round(f))
		if n <= 0 {
			return "1"
		}
		return strconv.Itoa(n)
	}
	if i := strings.IndexByte(raw, '.'); i > 0 {
		if head := strings.TrimSpace(raw[:i]); head != "" {
			return head
		}
	}
	return raw
}

func formatAdminRound(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "1"
	}
	if slash := strings.IndexByte(raw, '/'); slash > 0 {
		if head := strings.TrimSpace(raw[:slash]); head != "" {
			return head
		}
	}
	return raw
}

func formatAdminAmount(v float64) string {
	return strconv.FormatFloat(roundAdminMoney(v), 'f', 2, 64)
}

func roundAdminMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
