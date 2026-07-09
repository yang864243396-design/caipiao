package reports

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

var (
	ErrUnavailable  = errors.New("reports service unavailable")
	ErrInvalidQuery = errors.New("invalid reports query")
)

type Query struct {
	DateFrom    string
	DateTo      string
	LotteryCode string
}

type LotteryStatSummary struct {
	EffectiveBetYuan float64 `json:"effectiveBetYuan"`
	PayoutYuan       float64 `json:"payoutYuan"`
	DateFrom         string  `json:"dateFrom"`
	DateTo           string  `json:"dateTo"`
}

type LotteryStatRow struct {
	Lottery          string  `json:"lottery"`
	BetCount         int64   `json:"betCount"`
	EffectiveBetYuan float64 `json:"effectiveBetYuan"`
	MemberPnlYuan    float64 `json:"memberPnlYuan"`
}

type LotteryStatResult struct {
	Summary LotteryStatSummary `json:"summary"`
	Items   []LotteryStatRow   `json:"items"`
}

type PnlSummary struct {
	PlatformPnlYuan float64 `json:"platformPnlYuan"`
	ValidBetYuan    float64 `json:"validBetYuan"`
	DateFrom        string  `json:"dateFrom"`
	DateTo          string  `json:"dateTo"`
}

type PnlDailyRow struct {
	Period          string  `json:"period"`
	ValidBetYuan    float64 `json:"validBetYuan"`
	PlatformPnlYuan float64 `json:"platformPnlYuan"`
}

type PnlReportResult struct {
	Summary PnlSummary    `json:"summary"`
	Items   []PnlDailyRow `json:"items"`
}

// DailyLotterySummary 合并报表顶部总计
type DailyLotterySummary struct {
	BetCount        int64   `json:"betCount"`
	BetAmountYuan   float64 `json:"betAmountYuan"`
	PlatformPnlYuan float64 `json:"platformPnlYuan"`
	DateFrom        string  `json:"dateFrom"`
	DateTo          string  `json:"dateTo"`
}

// DailyLotteryRow 每天×每彩种一行
type DailyLotteryRow struct {
	Date            string  `json:"date"`
	LotteryCode     string  `json:"lotteryCode"`
	Lottery         string  `json:"lottery"`
	BetCount        int64   `json:"betCount"`
	BetAmountYuan   float64 `json:"betAmountYuan"`
	PlatformPnlYuan float64 `json:"platformPnlYuan"`
}

type DailyLotteryReportResult struct {
	Summary DailyLotterySummary `json:"summary"`
	Items   []DailyLotteryRow   `json:"items"`
}

type Service struct {
	q *sqlcdb.Queries
}

func NewService(pool *db.Pool) *Service {
	if pool == nil {
		return nil
	}
	return &Service{q: sqlcdb.New(pool)}
}

func (s *Service) AdminLotteryStat(ctx context.Context, q Query) (LotteryStatResult, error) {
	if s == nil || s.q == nil {
		return LotteryStatResult{}, ErrUnavailable
	}
	fromTime, toTime, labels, err := parseRange(q)
	if err != nil {
		return LotteryStatResult{}, err
	}
	from := pgtype.Timestamptz{Time: fromTime, Valid: true}
	to := pgtype.Timestamptz{Time: toTime, Valid: true}
	sum, err := s.q.AdminLotteryStatSummary(ctx, sqlcdb.AdminLotteryStatSummaryParams{PlacedAt: from, PlacedAt_2: to})
	if err != nil {
		return LotteryStatResult{}, err
	}
	rows, err := s.q.AdminLotteryStatByLottery(ctx, sqlcdb.AdminLotteryStatByLotteryParams{PlacedAt: from, PlacedAt_2: to})
	if err != nil {
		return LotteryStatResult{}, err
	}
	items := make([]LotteryStatRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, LotteryStatRow{
			Lottery:          row.LotteryName,
			BetCount:         row.BetCount,
			EffectiveBetYuan: roundMoney(row.EffectiveBet),
			MemberPnlYuan:    roundMoney(row.MemberPnl),
		})
	}
	return LotteryStatResult{
		Summary: LotteryStatSummary{
			EffectiveBetYuan: roundMoney(sum.EffectiveBet),
			PayoutYuan:       roundMoney(sum.Payout),
			DateFrom:         labels.from,
			DateTo:           labels.to,
		},
		Items: items,
	}, nil
}

func (s *Service) AdminPnlReport(ctx context.Context, q Query) (PnlReportResult, error) {
	if s == nil || s.q == nil {
		return PnlReportResult{}, ErrUnavailable
	}
	fromTime, toTime, labels, err := parseRange(q)
	if err != nil {
		return PnlReportResult{}, err
	}
	from := pgtype.Timestamptz{Time: fromTime, Valid: true}
	to := pgtype.Timestamptz{Time: toTime, Valid: true}
	sum, err := s.q.AdminPnlReportSummary(ctx, sqlcdb.AdminPnlReportSummaryParams{PlacedAt: from, PlacedAt_2: to})
	if err != nil {
		return PnlReportResult{}, err
	}
	rows, err := s.q.AdminPnlReportDaily(ctx, sqlcdb.AdminPnlReportDailyParams{PlacedAt: from, PlacedAt_2: to})
	if err != nil {
		return PnlReportResult{}, err
	}
	items := make([]PnlDailyRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, PnlDailyRow{
			Period:          formatDate(row.StatDate),
			ValidBetYuan:    roundMoney(row.ValidBet),
			PlatformPnlYuan: roundMoney(row.PlatformPnl),
		})
	}
	return PnlReportResult{
		Summary: PnlSummary{
			PlatformPnlYuan: roundMoney(sum.PlatformPnl),
			ValidBetYuan:    roundMoney(sum.ValidBet),
			DateFrom:        labels.from,
			DateTo:          labels.to,
		},
		Items: items,
	}, nil
}

// AdminDailyLotteryReport 合并「彩种统计 + 盈亏报表」：按天×彩种输出笔数、投注金额、平台盈亏，并附总计。
func (s *Service) AdminDailyLotteryReport(ctx context.Context, q Query) (DailyLotteryReportResult, error) {
	if s == nil || s.q == nil {
		return DailyLotteryReportResult{}, ErrUnavailable
	}
	fromTime, toTime, labels, err := parseRange(q)
	if err != nil {
		return DailyLotteryReportResult{}, err
	}
	from := pgtype.Timestamptz{Time: fromTime, Valid: true}
	to := pgtype.Timestamptz{Time: toTime, Valid: true}
	lotteryCode := strings.TrimSpace(q.LotteryCode)

	sum, err := s.q.AdminDailyLotterySummary(ctx, sqlcdb.AdminDailyLotterySummaryParams{
		PlacedAt: from, PlacedAt_2: to, LotteryCode: lotteryCode,
	})
	if err != nil {
		return DailyLotteryReportResult{}, err
	}
	rows, err := s.q.AdminDailyLotteryReport(ctx, sqlcdb.AdminDailyLotteryReportParams{
		PlacedAt: from, PlacedAt_2: to, LotteryCode: lotteryCode,
	})
	if err != nil {
		return DailyLotteryReportResult{}, err
	}
	items := make([]DailyLotteryRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, DailyLotteryRow{
			Date:            formatDate(row.StatDate),
			LotteryCode:     row.LotteryCode,
			Lottery:         row.LotteryName,
			BetCount:        row.BetCount,
			BetAmountYuan:   roundMoney(row.ValidBet),
			PlatformPnlYuan: roundMoney(row.PlatformPnl),
		})
	}
	return DailyLotteryReportResult{
		Summary: DailyLotterySummary{
			BetCount:        sum.BetCount,
			BetAmountYuan:   roundMoney(sum.ValidBet),
			PlatformPnlYuan: roundMoney(sum.PlatformPnl),
			DateFrom:        labels.from,
			DateTo:          labels.to,
		},
		Items: items,
	}, nil
}

type rangeLabels struct {
	from string
	to   string
}

func parseRange(q Query) (time.Time, time.Time, rangeLabels, error) {
	from, to, err := timeutil.ParseDateRange(q.DateFrom, q.DateTo)
	if err != nil {
		return time.Time{}, time.Time{}, rangeLabels{}, ErrInvalidQuery
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	endDay := to.In(loc).Add(-24 * time.Hour)
	startDay := from.In(loc)
	if q.DateFrom == "" && q.DateTo == "" {
		return from, to, rangeLabels{
			from: startDay.Format("2006-01-02"),
			to:   startDay.Format("2006-01-02"),
		}, nil
	}
	return from, to, rangeLabels{
		from: startDay.Format("2006-01-02"),
		to:   endDay.Format("2006-01-02"),
	}, nil
}

func formatDate(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
