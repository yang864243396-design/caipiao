package member

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
	"caipiao/backend/internal/ws"
)

var (
	ErrNotFound = errors.New("member not found")
)

type Service struct {
	q    *sqlcdb.Queries
	pool *db.Pool
	hub  *ws.Hub
}

func NewService(pool *db.Pool, hub *ws.Hub) *Service {
	if pool == nil {
		return nil
	}
	return &Service{
		q:    sqlcdb.New(pool),
		pool: pool,
		hub:  hub,
	}
}

// Profile 会员资料；余额一律来自第三方（/client/guaji/balance），平台不维护自有金额。
type Profile struct {
	MemberId    int64  `json:"memberId"`
	Account     string `json:"account"`
	DisplayName string `json:"displayName"`
	Currency    string `json:"currency"`
}

type Wallet struct {
	Balance        float64 `json:"balance"`
	FrozenBalance  float64 `json:"frozenBalance"`
	AvailableBalance float64 `json:"availableBalance"`
	Currency       string  `json:"currency"`
}

type LedgerItem struct {
	Time         string  `json:"time"`
	Type         string  `json:"type"`
	TypeCode     string  `json:"typeCode"`
	OrderID      string  `json:"orderId"`
	Delta        float64 `json:"delta"`
	BalanceAfter float64 `json:"balanceAfter"`
	LedgerNo     string  `json:"ledgerNo"`
}

type LedgerResult struct {
	Items      []LedgerItem `json:"items"`
	NextCursor string       `json:"nextCursor,omitempty"`
}

type LedgerQuery struct {
	DateFrom string
	DateTo   string
	Type     string
	OrderNo  string
	Cursor   string
	Limit    int
}

type MemberRef struct {
	ID      int64
	Account string
}

func (s *Service) GetByAccount(ctx context.Context, account string) (MemberRef, error) {
	row, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MemberRef{}, ErrNotFound
		}
		return MemberRef{}, err
	}
	return MemberRef{ID: row.ID, Account: row.Account}, nil
}

func (s *Service) Profile(ctx context.Context, account string) (Profile, error) {
	row, err := s.q.GetMemberProfileByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, ErrNotFound
		}
		return Profile{}, err
	}
	return Profile{
		MemberId:    row.ID,
		Account:     row.Account,
		DisplayName: row.DisplayName,
		Currency:    row.Currency,
	}, nil
}

func (s *Service) Wallet(ctx context.Context, account string) (Wallet, error) {
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Wallet{}, ErrNotFound
		}
		return Wallet{}, err
	}
	w, err := s.q.GetMemberWalletByMemberID(ctx, m.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Wallet{}, ErrNotFound
		}
		return Wallet{}, err
	}
	balance := roundMoney(w.Balance)
	frozen := roundMoney(w.FrozenBalance)
	return Wallet{
		Balance:          balance,
		FrozenBalance:    frozen,
		AvailableBalance: roundMoney(balance - frozen),
		Currency:         w.Currency,
	}, nil
}

func (s *Service) Ledger(ctx context.Context, account string, q LedgerQuery) (LedgerResult, error) {
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerResult{}, ErrNotFound
		}
		return LedgerResult{}, err
	}

	timeFrom, timeTo, err := timeutil.ParseDateRange(q.DateFrom, q.DateTo)
	if err != nil {
		return LedgerResult{}, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	txnType := mapLedgerFilterType(q.Type)
	if q.Type != "" && q.Type != "all" && txnType.Valid && txnType.String == "__invalid__" {
		return LedgerResult{}, fmt.Errorf("%w: type 参数无效", ErrInvalidQuery)
	}
	orderRef := pgtype.Text{}
	if q.OrderNo != "" {
		orderRef = pgtype.Text{String: q.OrderNo, Valid: true}
	}

	var rows []sqlcdb.ListWalletLedgerRow
	if q.Cursor == "" {
		rows, err = s.q.ListWalletLedger(ctx, sqlcdb.ListWalletLedgerParams{
			MemberID: m.ID,
			TimeFrom: pgtype.Timestamptz{Time: timeFrom, Valid: true},
			TimeTo:   pgtype.Timestamptz{Time: timeTo, Valid: true},
			TxnType:  txnType,
			OrderRef: orderRef,
			RowLimit: int32(limit + 1),
		})
	} else {
		anchor, aerr := s.q.GetWalletLedgerCursorAnchor(ctx, sqlcdb.GetWalletLedgerCursorAnchorParams{
			MemberID: m.ID,
			LedgerNo: q.Cursor,
		})
		if aerr != nil {
			if errors.Is(aerr, pgx.ErrNoRows) {
				return LedgerResult{}, ErrInvalidQuery
			}
			return LedgerResult{}, aerr
		}
		cursorRows, lerr := s.q.ListWalletLedgerAfterCursor(ctx, sqlcdb.ListWalletLedgerAfterCursorParams{
			MemberID:   m.ID,
			TimeFrom:   pgtype.Timestamptz{Time: timeFrom, Valid: true},
			TimeTo:     pgtype.Timestamptz{Time: timeTo, Valid: true},
			TxnType:    txnType,
			OrderRef:   orderRef,
			CursorTime: anchor.CreatedAt,
			CursorID:   anchor.ID,
			RowLimit:   int32(limit + 1),
		})
		if lerr != nil {
			return LedgerResult{}, lerr
		}
		rows = make([]sqlcdb.ListWalletLedgerRow, len(cursorRows))
		for i, r := range cursorRows {
			rows[i] = ledgerRowFromCursor(r)
		}
	}
	if err != nil {
		return LedgerResult{}, err
	}

	nextCursor := ""
	if len(rows) > limit {
		nextCursor = rows[limit-1].LedgerNo
		rows = rows[:limit]
	}

	items := make([]LedgerItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, LedgerItem{
			Time:         timeutil.FormatISO(row.CreatedAt.Time),
			Type:         txnTypeLabel(row.TxnType),
			TypeCode:     row.TxnType,
			OrderID:      row.OrderRef,
			Delta:        roundMoney(row.DeltaAmount),
			BalanceAfter: roundMoney(row.BalanceAfter),
			LedgerNo:     row.LedgerNo,
		})
	}

	return LedgerResult{Items: items, NextCursor: nextCursor}, nil
}

var ErrInvalidQuery = errors.New("invalid ledger query")

func mapLedgerFilterType(raw string) pgtype.Text {
	switch raw {
	case "", "all":
		return pgtype.Text{}
	case "deposit":
		return pgtype.Text{String: "deposit", Valid: true}
	case "withdraw":
		return pgtype.Text{String: "withdraw", Valid: true}
	case "bet":
		return pgtype.Text{String: "bet_debit", Valid: true}
	case "payout":
		return pgtype.Text{String: "payout", Valid: true}
	case "adjust":
		return pgtype.Text{String: "adjust", Valid: true}
	case "withdraw_freeze":
		return pgtype.Text{String: "withdraw_freeze", Valid: true}
	default:
		return pgtype.Text{String: "__invalid__", Valid: true}
	}
}

func txnTypeLabel(code string) string {
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

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func ledgerRowFromCursor(r sqlcdb.ListWalletLedgerAfterCursorRow) sqlcdb.ListWalletLedgerRow {
	return sqlcdb.ListWalletLedgerRow{
		ID:           r.ID,
		LedgerNo:     r.LedgerNo,
		TxnType:      r.TxnType,
		DeltaAmount:  r.DeltaAmount,
		BalanceAfter: r.BalanceAfter,
		OrderRef:     r.OrderRef,
		CreatedAt:    r.CreatedAt,
	}
}
