package bets

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/timeutil"
)

var ErrInvalidQuery = errors.New("invalid bet query")

const maxBetQueryDays = 3

type Service struct {
	members *member.Service
	q       *sqlcdb.Queries
}

func NewService(pool *db.Pool, members *member.Service) *Service {
	if pool == nil || members == nil {
		return nil
	}
	return &Service{members: members, q: sqlcdb.New(pool)}
}

type Item struct {
	Time         string  `json:"time"`
	Game         string  `json:"game"`
	OrderID      string  `json:"orderId"`
	Amount       float64 `json:"amount"`
	ReturnAmount float64 `json:"returnAmount"`
	Status       string  `json:"status"`
}

type PageMeta struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

// CurrencySummary 投注记录筛选结果按币种汇总。
type CurrencySummary struct {
	Currency    string  `json:"currency"`
	OrderCount  int64   `json:"orderCount"`
	ValidAmount float64 `json:"validAmount"`
	Pnl         float64 `json:"pnl"`
}

type Result struct {
	Items   []Item            `json:"items"`
	Page    PageMeta          `json:"page"`
	Summary []CurrencySummary `json:"summary,omitempty"`
}

type Query struct {
	Account            string
	DateFrom           string
	DateTo             string
	GameCode           string
	SchemeDefinitionID string
	OrderNo            string
	Currency           string
	Cursor             string
	Limit              int
}

func (s *Service) List(ctx context.Context, q Query) (Result, error) {
	m, err := s.members.GetByAccount(ctx, q.Account)
	if err != nil {
		return Result{}, err
	}

	timeFrom, timeTo, err := timeutil.ParseDateRange(q.DateFrom, q.DateTo)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}
	if err := validateQuerySpan(q.DateFrom, q.DateTo); err != nil {
		return Result{}, err
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	orderNoRaw := strings.TrimSpace(q.OrderNo)
	orderNo := pgtype.Text{}
	if orderNoRaw != "" {
		orderNo = pgtype.Text{String: orderNoRaw, Valid: true}
		// 注单号检索：在允许的最大跨度内向前扩窗，避免默认「仅今天」漏掉刚删方案的历史单。
		timeFrom = expandRangeStartForOrderNo(timeFrom, timeTo, maxBetQueryDays)
	}

	currency, err := mapCurrencyFilter(q.Currency)
	if err != nil {
		return Result{}, err
	}

	schemeDefID := strings.TrimSpace(q.SchemeDefinitionID)
	// 注单号优先走「全部方案」路径：已删除方案的 instance/definition 不在，按 definition 联表会漏单。
	if orderNoRaw != "" {
		schemeDefID = "all"
	}
	lotteryCode := mapLotteryCodeFilter(q.GameCode)
	if q.GameCode != "" && q.GameCode != "all" && lotteryCode.Valid && lotteryCode.String == "__invalid__" {
		return Result{}, fmt.Errorf("%w: gameCode 参数无效", ErrInvalidQuery)
	}

	var (
		result Result
		errList error
	)
	switch {
	case schemeDefID != "" && schemeDefID != "all":
		result, errList = s.listBySchemeDefinition(ctx, m.ID, schemeDefID, timeFrom, timeTo, orderNo, currency, q.Cursor, limit)
	case schemeDefID == "all":
		code := ""
		if lotteryCode.Valid {
			code = lotteryCode.String
		}
		result, errList = s.listByLottery(ctx, m.ID, code, timeFrom, timeTo, orderNo, currency, q.Cursor, limit)
	default:
		params := sqlcdb.ListBetOrdersParams{
			MemberID:    m.ID,
			TimeFrom:    pgtype.Timestamptz{Time: timeFrom, Valid: true},
			TimeTo:      pgtype.Timestamptz{Time: timeTo, Valid: true},
			LotteryCode: lotteryCode,
			OrderNo:     orderNo,
			RowLimit:    int32(limit + 1),
		}
		result, errList = s.listBetOrders(ctx, params, currency, q.Cursor, limit)
	}
	if errList != nil {
		return Result{}, errList
	}

	// 首页带币种汇总（不受列表币种筛选项限制，始终返回 USDT/TRX/CNY 三行）
	if q.Cursor == "" {
		summary, serr := s.summarizeByCurrency(ctx, m.ID, schemeDefID, lotteryCode, timeFrom, timeTo, orderNo)
		if serr != nil {
			return Result{}, serr
		}
		result.Summary = summary
	}
	return result, nil
}

func (s *Service) summarizeByCurrency(
	ctx context.Context,
	memberID int64,
	schemeDefID string,
	lotteryCode pgtype.Text,
	timeFrom, timeTo time.Time,
	orderNo pgtype.Text,
) ([]CurrencySummary, error) {
	since := pgtype.Timestamptz{Time: timeFrom, Valid: true}
	until := pgtype.Timestamptz{Time: timeTo, Valid: true}

	var (
		rows []sqlcdb.CloudBetCurrencySummaryRow
		err  error
	)
	if schemeDefID == "" {
		rows, err = s.q.SummarizeBetOrdersByCurrencyEx(ctx, memberID, since, until, lotteryCode, orderNo)
	} else {
		guajiID, gerr := member.LookupActiveGuajiAccountID(ctx, s.q, memberID)
		if gerr != nil {
			return nil, gerr
		}
		defID := ""
		if schemeDefID != "all" {
			defID = schemeDefID
		}
		lotCode := ""
		if lotteryCode.Valid {
			lotCode = lotteryCode.String
		}
		rows, err = s.q.SummarizeCloudBetRecordsByCurrencyEx(ctx, memberID, defID, lotCode, since, until, orderNo, guajiID)
	}
	if err != nil {
		return nil, err
	}

	byCur := make(map[string]sqlcdb.CloudBetCurrencySummaryRow, len(rows))
	for _, row := range rows {
		cur := strings.ToUpper(strings.TrimSpace(row.Currency))
		if cur == "" {
			continue
		}
		byCur[cur] = row
	}

	out := make([]CurrencySummary, 0, 3)
	for _, cur := range []string{"USDT", "TRX", "CNY"} {
		row := byCur[cur]
		out = append(out, CurrencySummary{
			Currency:    cur,
			OrderCount:  row.OrderCount,
			ValidAmount: roundMoney(row.ValidAmount),
			Pnl:         roundMoney(row.Pnl),
		})
	}
	return out, nil
}

func (s *Service) listBySchemeDefinition(
	ctx context.Context,
	memberID int64,
	definitionID string,
	timeFrom, timeTo time.Time,
	orderNo pgtype.Text,
	currency pgtype.Text,
	cursor string,
	limit int,
) (Result, error) {
	_, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: memberID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Result{}, fmt.Errorf("%w: 方案不存在", ErrInvalidQuery)
		}
		return Result{}, err
	}

	guajiID, err := member.LookupActiveGuajiAccountID(ctx, s.q, memberID)
	if err != nil {
		return Result{}, err
	}

	baseParams := sqlcdb.ListCloudBetRecordsByDefinitionParams{
		MemberID:       memberID,
		DefinitionID:   definitionID,
		SinceAt:        pgtype.Timestamptz{Time: timeFrom, Valid: true},
		UntilAt:        pgtype.Timestamptz{Time: timeTo, Valid: true},
		OrderNo:        orderNo,
		GuajiAccountID: guajiID,
		RowLimit:       int32(limit + 1),
	}

	var rows []sqlcdb.CloudBetListRow
	if cursor == "" {
		rows, err = s.q.ListCloudBetRecordsByDefinitionEx(ctx, baseParams, currency)
	} else {
		anchor, aerr := s.q.GetCloudBetRecordCursorAnchor(ctx, sqlcdb.GetCloudBetRecordCursorAnchorParams{
			MemberID: memberID,
			RecordNo: cursor,
		})
		if aerr != nil {
			if errors.Is(aerr, pgx.ErrNoRows) {
				return Result{}, ErrInvalidQuery
			}
			return Result{}, aerr
		}
		rows, err = s.q.ListCloudBetRecordsByDefinitionAfterCursorEx(ctx, sqlcdb.ListCloudBetRecordsByDefinitionAfterCursorParams{
			MemberID:       baseParams.MemberID,
			DefinitionID:   baseParams.DefinitionID,
			SinceAt:        baseParams.SinceAt,
			UntilAt:        baseParams.UntilAt,
			OrderNo:        baseParams.OrderNo,
			GuajiAccountID: baseParams.GuajiAccountID,
			CursorTime:     anchor.PlacedAt,
			CursorID:       anchor.ID,
			RowLimit:       baseParams.RowLimit,
		}, currency)
	}
	if err != nil {
		return Result{}, err
	}

	nextCursor := ""
	hasMore := len(rows) > limit
	if hasMore {
		nextCursor = rows[limit-1].RecordNo
		rows = rows[:limit]
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, itemFromCloudRow(
			timeutil.FormatDisplayCST(row.PlacedAt.Time),
			formatGameColumn(row.LotteryLabel, row.SchemeName),
			displayBetOrderID(row.RecordNo, row.ThirdPartyBetID),
			row.Amount,
			row.Pnl,
			row.Status,
		))
	}

	return Result{
		Items: items,
		Page: PageMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Service) listByLottery(
	ctx context.Context,
	memberID int64,
	lotteryCode string,
	timeFrom, timeTo time.Time,
	orderNo pgtype.Text,
	currency pgtype.Text,
	cursor string,
	limit int,
) (Result, error) {
	guajiID, err := member.LookupActiveGuajiAccountID(ctx, s.q, memberID)
	if err != nil {
		return Result{}, err
	}

	baseParams := sqlcdb.ListCloudBetRecordsByLotteryParams{
		MemberID:       memberID,
		LotteryCode:    lotteryCode,
		SinceAt:        pgtype.Timestamptz{Time: timeFrom, Valid: true},
		UntilAt:        pgtype.Timestamptz{Time: timeTo, Valid: true},
		OrderNo:        orderNo,
		GuajiAccountID: guajiID,
		RowLimit:       int32(limit + 1),
	}

	var rows []sqlcdb.CloudBetListRow
	if cursor == "" {
		rows, err = s.q.ListCloudBetRecordsByLotteryEx(ctx, baseParams, currency)
	} else {
		anchor, aerr := s.q.GetCloudBetRecordCursorAnchor(ctx, sqlcdb.GetCloudBetRecordCursorAnchorParams{
			MemberID: memberID,
			RecordNo: cursor,
		})
		if aerr != nil {
			if errors.Is(aerr, pgx.ErrNoRows) {
				return Result{}, ErrInvalidQuery
			}
			return Result{}, aerr
		}
		rows, err = s.q.ListCloudBetRecordsByLotteryAfterCursorEx(ctx, sqlcdb.ListCloudBetRecordsByLotteryAfterCursorParams{
			MemberID:       baseParams.MemberID,
			LotteryCode:    baseParams.LotteryCode,
			SinceAt:        baseParams.SinceAt,
			UntilAt:        baseParams.UntilAt,
			OrderNo:        baseParams.OrderNo,
			GuajiAccountID: baseParams.GuajiAccountID,
			CursorTime:     anchor.PlacedAt,
			CursorID:       anchor.ID,
			RowLimit:       baseParams.RowLimit,
		}, currency)
	}
	if err != nil {
		return Result{}, err
	}

	nextCursor := ""
	hasMore := len(rows) > limit
	if hasMore {
		nextCursor = rows[limit-1].RecordNo
		rows = rows[:limit]
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, itemFromCloudRow(
			timeutil.FormatDisplayCST(row.PlacedAt.Time),
			formatGameColumn(row.LotteryLabel, row.SchemeName),
			displayBetOrderID(row.RecordNo, row.ThirdPartyBetID),
			row.Amount,
			row.Pnl,
			row.Status,
		))
	}

	return Result{
		Items: items,
		Page: PageMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Service) listBetOrders(ctx context.Context, params sqlcdb.ListBetOrdersParams, currency pgtype.Text, cursor string, limit int) (Result, error) {
	var rows []sqlcdb.BetOrderListRow
	var err error
	if cursor == "" {
		rows, err = s.q.ListBetOrdersEx(ctx, params, currency)
	} else {
		anchor, aerr := s.q.GetBetOrderCursorAnchor(ctx, sqlcdb.GetBetOrderCursorAnchorParams{
			MemberID: params.MemberID,
			OrderNo:  cursor,
		})
		if aerr != nil {
			if errors.Is(aerr, pgx.ErrNoRows) {
				return Result{}, ErrInvalidQuery
			}
			return Result{}, aerr
		}
		rows, err = s.q.ListBetOrdersAfterCursorEx(ctx, sqlcdb.ListBetOrdersAfterCursorParams{
			MemberID:        params.MemberID,
			TimeFrom:        params.TimeFrom,
			TimeTo:          params.TimeTo,
			Status:          params.Status,
			LotteryCategory: params.LotteryCategory,
			LotteryCode:     params.LotteryCode,
			OrderNo:         params.OrderNo,
			CursorTime:      anchor.PlacedAt,
			CursorID:        anchor.ID,
			RowLimit:        int32(limit + 1),
		}, currency)
	}
	if err != nil {
		return Result{}, err
	}

	nextCursor := ""
	hasMore := len(rows) > limit
	if hasMore {
		nextCursor = rows[limit-1].OrderNo
		rows = rows[:limit]
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, Item{
			Time:         timeutil.FormatDisplayCST(row.PlacedAt.Time),
			Game:         row.LotteryName,
			OrderID:      displayBetOrderID(row.OrderNo, row.ThirdPartyBetID),
			Amount:       roundMoney(row.Amount),
			ReturnAmount: orderReturnAmount(row.Amount, row.Pnl, row.Status),
			Status:       statusLabel(row.Status),
		})
	}

	return Result{
		Items: items,
		Page: PageMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func mapLotteryCodeFilter(raw string) pgtype.Text {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "all" {
		return pgtype.Text{}
	}
	if len(raw) > 64 {
		return pgtype.Text{String: "__invalid__", Valid: true}
	}
	return pgtype.Text{String: raw, Valid: true}
}

func mapCurrencyFilter(raw string) (pgtype.Text, error) {
	raw = strings.TrimSpace(strings.ToUpper(raw))
	if raw == "" || raw == "ALL" {
		return pgtype.Text{}, nil
	}
	switch raw {
	case "USDT", "TRX", "CNY":
		return pgtype.Text{String: raw, Valid: true}, nil
	default:
		return pgtype.Text{}, fmt.Errorf("%w: currency 参数无效", ErrInvalidQuery)
	}
}

func cloudStatusLabel(code string) string {
	switch code {
	case "hit":
		return "已中奖"
	case "miss":
		return "未中奖"
	case "pending":
		return "未开奖"
	default:
		return code
	}
}

func formatGameColumn(lotteryLabel, schemeName string) string {
	lotteryLabel = strings.TrimSpace(lotteryLabel)
	schemeName = strings.TrimSpace(schemeName)
	switch {
	case lotteryLabel != "" && schemeName != "":
		return lotteryLabel + " - " + schemeName
	case lotteryLabel != "":
		return lotteryLabel
	default:
		return schemeName
	}
}

func statusLabel(code string) string {
	switch code {
	case "pending":
		return "未开奖"
	case "win":
		return "已中奖"
	case "lose":
		return "未中奖"
	case "cancel":
		return "已撤单"
	default:
		return code
	}
}

func displayBetOrderID(fallback string, thirdParty pgtype.Text) string {
	if thirdParty.Valid {
		if id := strings.TrimSpace(thirdParty.String); id != "" {
			return id
		}
	}
	return fallback
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func itemFromCloudRow(time, game, orderID string, amount, pnl float64, statusCode string) Item {
	return Item{
		Time:         time,
		Game:         game,
		OrderID:      orderID,
		Amount:       roundMoney(amount),
		ReturnAmount: cloudReturnAmount(amount, pnl, statusCode),
		Status:       cloudStatusLabel(statusCode),
	}
}

func cloudReturnAmount(amount, pnl float64, status string) float64 {
	if status == "hit" {
		return roundMoney(amount + pnl)
	}
	return 0
}

func orderReturnAmount(amount, pnl float64, status string) float64 {
	if status == "win" {
		return roundMoney(amount + pnl)
	}
	return 0
}

func validateQuerySpan(from, to string) error {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(from), loc)
	if err != nil {
		return fmt.Errorf("%w: dateFrom 格式须为 YYYY-MM-DD", ErrInvalidQuery)
	}
	endDay, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(to), loc)
	if err != nil {
		return fmt.Errorf("%w: dateTo 格式须为 YYYY-MM-DD", ErrInvalidQuery)
	}
	if endDay.Before(start) {
		return fmt.Errorf("%w: dateTo 不能早于 dateFrom", ErrInvalidQuery)
	}
	span := int(endDay.Sub(start).Hours()/24) + 1
	if span > maxBetQueryDays {
		return fmt.Errorf("%w: 查询区间最多连续 %d 天", ErrInvalidQuery, maxBetQueryDays)
	}
	return nil
}

// expandRangeStartForOrderNo 将查询起点前推至「结束时刻往前 maxDays 天」，
// 仍不超过 maxBetQueryDays（timeTo 为半开区间上界）。
func expandRangeStartForOrderNo(from, to time.Time, maxDays int) time.Time {
	if maxDays < 1 {
		return from
	}
	wantFrom := to.Add(-time.Duration(maxDays) * 24 * time.Hour)
	if from.After(wantFrom) {
		return wantFrom
	}
	return from
}
