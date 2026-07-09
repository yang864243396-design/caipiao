package ordersadmin

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

var ErrUnavailable = errors.New("orders admin service unavailable")

type ListQuery struct {
	Keyword  string
	Page     int
	PageSize int
}

type BetListQuery struct {
	IssueNo       string
	MemberAccount string
	SchemeName    string
	LotteryCode   string
	Page          int
	PageSize      int
}

type ChaseListQuery struct {
	ChaseNo       string
	MemberAccount string
	Status        string
	LotteryCode   string
	Page          int
	PageSize      int
}

type ListResult[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

type BetRow struct {
	OrderNo      string  `json:"orderNo"`
	IssueNo      string  `json:"issueNo"`
	Member       string  `json:"member"`
	Lottery      string  `json:"lottery"`
	SchemeName   string  `json:"schemeName"`
	Amount       float64 `json:"amount"`
	PayoutAmount float64 `json:"payoutAmount"`
	ResultStatus string  `json:"resultStatus"`
	Created      string  `json:"created"`
}

type ChaseRow struct {
	ChaseNo     string  `json:"chaseNo"`
	Member      string  `json:"member"`
	Lottery     string  `json:"lottery"`
	TotalIssues int32   `json:"totalIssues"`
	DoneIssues  int32   `json:"doneIssues"`
	PeriodsLeft int32   `json:"periodsLeft"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	Created     string  `json:"created"`
}

type LedgerRow struct {
	ID      string  `json:"id"`
	Member  string  `json:"member"`
	Type    string  `json:"type"`
	Amount  float64 `json:"amount"`
	Created string  `json:"created"`
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

func (s *Service) ListBets(ctx context.Context, q BetListQuery) (ListResult[BetRow], error) {
	if s == nil || s.q == nil {
		return ListResult[BetRow]{}, ErrUnavailable
	}
	_, pageSize, offset := normalizeBetPage(q)
	filter := sqlcdb.CountAdminBetOrdersParams{
		IssueNo:       pgText(q.IssueNo),
		MemberAccount: pgText(q.MemberAccount),
		SchemeName:    pgText(q.SchemeName),
		LotteryCode:   pgText(q.LotteryCode),
	}

	total, err := s.q.CountAdminBetOrders(ctx, filter)
	if err != nil {
		return ListResult[BetRow]{}, err
	}
	rows, err := s.q.ListAdminBetOrders(ctx, sqlcdb.ListAdminBetOrdersParams{
		IssueNo:       filter.IssueNo,
		MemberAccount: filter.MemberAccount,
		SchemeName:    filter.SchemeName,
		LotteryCode:   filter.LotteryCode,
		RowLimit:      int32(pageSize),
		RowOffset:     int32(offset),
	})
	if err != nil {
		return ListResult[BetRow]{}, err
	}
	items := make([]BetRow, 0, len(rows))
	for _, row := range rows {
		schemeName := strings.TrimSpace(row.SchemeName)
		if schemeName == "" {
			schemeName = "—"
		}
		orderNo := strings.TrimSpace(row.ThirdPartyBetID)
		if orderNo == "" {
			orderNo = "—"
		}
		items = append(items, BetRow{
			OrderNo:      orderNo,
			IssueNo:      strings.TrimSpace(row.IssueNo),
			Member:       strings.TrimSpace(row.Account),
			Lottery:      row.LotteryName,
			SchemeName:   schemeName,
			Amount:       roundMoney(row.Amount),
			PayoutAmount: roundMoney(row.PayoutAmount),
			ResultStatus: betResultStatus(row.Status),
			Created:      timeutil.FormatISO(row.PlacedAt.Time),
		})
	}
	return ListResult[BetRow]{Items: items, Total: total}, nil
}

func (s *Service) ListChases(ctx context.Context, q ChaseListQuery) (ListResult[ChaseRow], error) {
	if s == nil || s.q == nil {
		return ListResult[ChaseRow]{}, ErrUnavailable
	}
	_, pageSize, offset := normalizeChasePage(q)
	filter := sqlcdb.CountAdminChaseOrdersParams{
		ChaseNo:       pgText(q.ChaseNo),
		MemberAccount: pgText(q.MemberAccount),
		Status:        pgText(q.Status),
		LotteryCode:   pgText(q.LotteryCode),
	}

	total, err := s.q.CountAdminChaseOrders(ctx, filter)
	if err != nil {
		return ListResult[ChaseRow]{}, err
	}
	rows, err := s.q.ListAdminChaseOrders(ctx, sqlcdb.ListAdminChaseOrdersParams{
		ChaseNo:       filter.ChaseNo,
		MemberAccount: filter.MemberAccount,
		Status:        filter.Status,
		LotteryCode:   filter.LotteryCode,
		RowLimit:      int32(pageSize),
		RowOffset:     int32(offset),
	})
	if err != nil {
		return ListResult[ChaseRow]{}, err
	}
	items := make([]ChaseRow, 0, len(rows))
	for _, row := range rows {
		left := row.TotalIssues - row.DoneIssues
		if left < 0 {
			left = 0
		}
		items = append(items, ChaseRow{
			ChaseNo:     row.ChaseNo,
			Member:      strings.TrimSpace(row.Account),
			Lottery:     row.LotteryName,
			TotalIssues: row.TotalIssues,
			DoneIssues:  row.DoneIssues,
			PeriodsLeft: left,
			Amount:      roundMoney(row.Amount),
			Status:      chaseResultStatus(row.Status),
			Created:     timeutil.FormatISO(row.StartedAt.Time),
		})
	}
	return ListResult[ChaseRow]{Items: items, Total: total}, nil
}

func (s *Service) ListLedger(ctx context.Context, q ListQuery) (ListResult[LedgerRow], error) {
	if s == nil || s.q == nil {
		return ListResult[LedgerRow]{}, ErrUnavailable
	}
	_, pageSize, offset := normalizePage(q)
	kw := pgText(q.Keyword)

	total, err := s.q.CountAdminLedgerEntries(ctx, kw)
	if err != nil {
		return ListResult[LedgerRow]{}, err
	}
	rows, err := s.q.ListAdminLedgerEntries(ctx, sqlcdb.ListAdminLedgerEntriesParams{
		Keyword:   kw,
		RowLimit:  int32(pageSize),
		RowOffset: int32(offset),
	})
	if err != nil {
		return ListResult[LedgerRow]{}, err
	}
	items := make([]LedgerRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, LedgerRow{
			ID:      row.LedgerNo,
			Member:  row.DisplayName,
			Type:    ledgerTypeLabel(row.TxnType),
			Amount:  roundMoney(row.DeltaAmount),
			Created: timeutil.FormatISO(row.CreatedAt.Time),
		})
	}
	return ListResult[LedgerRow]{Items: items, Total: total}, nil
}

func normalizePage(q ListQuery) (page, pageSize, offset int) {
	page = q.Page
	if page < 1 {
		page = 1
	}
	pageSize = q.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset = (page - 1) * pageSize
	return page, pageSize, offset
}

func normalizeBetPage(q BetListQuery) (page, pageSize, offset int) {
	page = q.Page
	if page < 1 {
		page = 1
	}
	pageSize = q.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset = (page - 1) * pageSize
	return page, pageSize, offset
}

func normalizeChasePage(q ChaseListQuery) (page, pageSize, offset int) {
	page = q.Page
	if page < 1 {
		page = 1
	}
	pageSize = q.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset = (page - 1) * pageSize
	return page, pageSize, offset
}

func pgText(v string) pgtype.Text {
	v = strings.TrimSpace(v)
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func roundMoney(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func betResultStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "win":
		return "hit"
	case "lose":
		return "miss"
	case "pending":
		return "pending"
	case "cancel":
		return "cancel"
	default:
		return status
	}
}

func chaseResultStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "running":
		return "running"
	case "completed":
		return "completed"
	case "cancelled":
		return "cancelled"
	default:
		return status
	}
}

func ledgerTypeLabel(code string) string {
	switch code {
	case "deposit":
		return "入款"
	case "withdraw":
		return "出款"
	case "bet_debit":
		return "投注扣款"
	case "payout":
		return "派奖"
	case "withdraw_freeze":
		return "提现冻结"
	case "adjust":
		return "调账"
	default:
		return code
	}
}
