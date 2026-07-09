package member

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

const fundRecordsDefaultLimit = 20

type FundRecordItem struct {
	ID           string  `json:"id"`
	SchemeName   string  `json:"schemeName"`
	Currency     string  `json:"currency"`
	Amount       float64 `json:"amount"`
	Time         string  `json:"time"`
	FlowType     string  `json:"flowType"`
	FlowTypeCode string  `json:"flowTypeCode"`
	BalanceAfter float64 `json:"balanceAfter"`
	LedgerNo     string  `json:"ledgerNo"`
}

type FundRecordsPage struct {
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type FundRecordsResult struct {
	Items []FundRecordItem `json:"items"`
	Page  FundRecordsPage  `json:"page"`
}

type FundRecordsQuery struct {
	DateFrom string
	DateTo   string
	FlowType string
	Currency string
	Cursor   string
	Limit    int
}

type AdminFundRecordsQuery struct {
	DateFrom string
	DateTo   string
	FlowType string
	Currency string
	Page     int
	PageSize int
}

type AdminSiteFundRecordsQuery struct {
	DateFrom      string
	DateTo        string
	FlowType      string
	Currency      string
	MemberAccount string
	LedgerNo      string
	Page          int
	PageSize      int
}

type AdminSiteFundRecordItem struct {
	FundRecordItem
	Member string `json:"member"`
}

type AdminSiteFundRecordsResult struct {
	Items    []AdminSiteFundRecordItem `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
}

type AdminFundRecordsResult struct {
	Items    []FundRecordItem `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

func (s *Service) FundRecords(ctx context.Context, account string, q FundRecordsQuery) (FundRecordsResult, error) {
	if s == nil || s.q == nil {
		return FundRecordsResult{}, ErrUnavailable
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FundRecordsResult{}, ErrNotFound
		}
		return FundRecordsResult{}, err
	}
	return s.FundRecordsForMemberID(ctx, m.ID, q)
}

// FundRecordsForMemberID 与 Client /client/funds/records 同源：启用授权下的 bet_debit/payout 镜像流水。
func (s *Service) FundRecordsForMemberID(ctx context.Context, memberID int64, q FundRecordsQuery) (FundRecordsResult, error) {
	if s == nil || s.q == nil {
		return FundRecordsResult{}, ErrUnavailable
	}
	if memberID <= 0 {
		return FundRecordsResult{}, ErrNotFound
	}
	if _, err := s.q.GetMemberByID(ctx, memberID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FundRecordsResult{}, ErrNotFound
		}
		return FundRecordsResult{}, err
	}

	timeFrom, timeTo, err := timeutil.ParseDateRange(q.DateFrom, q.DateTo)
	if err != nil {
		return FundRecordsResult{}, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	flowDir, err := mapFundFlowDir(q.FlowType)
	if err != nil {
		return FundRecordsResult{}, err
	}
	currency, err := mapFundCurrency(q.Currency)
	if err != nil {
		return FundRecordsResult{}, err
	}

	limit := q.Limit
	if limit <= 0 {
		limit = fundRecordsDefaultLimit
	}
	if limit > 100 {
		limit = 100
	}

	guajiID, err := LookupActiveGuajiAccountID(ctx, s.q, memberID)
	if err != nil {
		return FundRecordsResult{}, err
	}
	if !guajiID.Valid {
		return emptyFundRecordsResult(), nil
	}

	var rows []fundRecordRow
	if q.Cursor == "" {
		listRows, lerr := s.q.ListMemberFundRecords(ctx, sqlcdb.ListMemberFundRecordsParams{
			MemberID:       memberID,
			GuajiAccountID: guajiID,
			TimeFrom:       pgtype.Timestamptz{Time: timeFrom, Valid: true},
			TimeTo:         pgtype.Timestamptz{Time: timeTo, Valid: true},
			FlowDir:        flowDir,
			Currency:       currency,
			RowLimit:       int32(limit + 1),
		})
		if lerr != nil {
			return FundRecordsResult{}, lerr
		}
		rows = fundRowsFromList(listRows)
	} else {
		anchor, aerr := s.q.GetWalletLedgerCursorAnchor(ctx, sqlcdb.GetWalletLedgerCursorAnchorParams{
			MemberID: memberID,
			LedgerNo: q.Cursor,
		})
		if aerr != nil {
			if errors.Is(aerr, pgx.ErrNoRows) {
				return FundRecordsResult{}, ErrInvalidQuery
			}
			return FundRecordsResult{}, aerr
		}
		cursorRows, lerr := s.q.ListMemberFundRecordsAfterCursor(ctx, sqlcdb.ListMemberFundRecordsAfterCursorParams{
			MemberID:       memberID,
			GuajiAccountID: guajiID,
			TimeFrom:       pgtype.Timestamptz{Time: timeFrom, Valid: true},
			TimeTo:         pgtype.Timestamptz{Time: timeTo, Valid: true},
			FlowDir:        flowDir,
			Currency:       currency,
			CursorTime:     anchor.CreatedAt,
			CursorID:       anchor.ID,
			RowLimit:       int32(limit + 1),
		})
		if lerr != nil {
			return FundRecordsResult{}, lerr
		}
		rows = fundRowsFromCursor(cursorRows)
	}

	nextCursor := ""
	hasMore := len(rows) > limit
	if hasMore {
		nextCursor = rows[limit-1].LedgerNo
		rows = rows[:limit]
	}

	items := make([]FundRecordItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapFundRecordRow(row))
	}

	return FundRecordsResult{
		Items: items,
		Page: FundRecordsPage{
			HasMore:    hasMore,
			NextCursor: nextCursor,
		},
	}, nil
}

// AdminFundRecordsForMemberID Admin 分页查询资金记录（与 Client 同源筛选，offset 分页）。
func (s *Service) AdminFundRecordsForMemberID(ctx context.Context, memberID int64, q AdminFundRecordsQuery) (AdminFundRecordsResult, error) {
	empty := AdminFundRecordsResult{Items: []FundRecordItem{}, Page: 1, PageSize: 10}
	if s == nil || s.q == nil {
		return empty, ErrUnavailable
	}
	if memberID <= 0 {
		return empty, ErrNotFound
	}
	if _, err := s.q.GetMemberByID(ctx, memberID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return empty, ErrNotFound
		}
		return empty, err
	}

	timeFrom, timeTo, err := timeutil.ParseDateRange(q.DateFrom, q.DateTo)
	if err != nil {
		return empty, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}
	flowDir, err := mapFundFlowDir(q.FlowType)
	if err != nil {
		return empty, err
	}
	currency, err := mapFundCurrency(q.Currency)
	if err != nil {
		return empty, err
	}

	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	guajiID, err := LookupActiveGuajiAccountID(ctx, s.q, memberID)
	if err != nil {
		return empty, err
	}
	if !guajiID.Valid {
		return AdminFundRecordsResult{Items: []FundRecordItem{}, Total: 0, Page: page, PageSize: pageSize}, nil
	}

	filter := sqlcdb.CountMemberFundRecordsParams{
		MemberID:       memberID,
		GuajiAccountID: guajiID,
		TimeFrom:       pgtype.Timestamptz{Time: timeFrom, Valid: true},
		TimeTo:         pgtype.Timestamptz{Time: timeTo, Valid: true},
		FlowDir:        flowDir,
		Currency:       currency,
	}
	total, err := s.q.CountMemberFundRecords(ctx, filter)
	if err != nil {
		return empty, err
	}

	offset := (page - 1) * pageSize
	listRows, err := s.q.ListMemberFundRecordsPaged(ctx, sqlcdb.ListMemberFundRecordsPagedParams{
		MemberID:       memberID,
		GuajiAccountID: guajiID,
		TimeFrom:       pgtype.Timestamptz{Time: timeFrom, Valid: true},
		TimeTo:         pgtype.Timestamptz{Time: timeTo, Valid: true},
		FlowDir:        flowDir,
		Currency:       currency,
		RowLimit:       int32(pageSize),
		RowOffset:      int32(offset),
	})
	if err != nil {
		return empty, err
	}

	items := make([]FundRecordItem, 0, len(listRows))
	for _, row := range fundRowsFromPaged(listRows) {
		items = append(items, mapFundRecordRow(row))
	}
	return AdminFundRecordsResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// AdminFundRecords 全站钱包流水（与 Client 同源：bet_debit / payout 镜像）。
func (s *Service) AdminFundRecords(ctx context.Context, q AdminSiteFundRecordsQuery) (AdminSiteFundRecordsResult, error) {
	empty := AdminSiteFundRecordsResult{Items: []AdminSiteFundRecordItem{}, Page: 1, PageSize: 10}
	if s == nil || s.q == nil {
		return empty, ErrUnavailable
	}

	timeFrom, timeTo, err := timeutil.ParseDateRange(q.DateFrom, q.DateTo)
	if err != nil {
		return empty, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}
	flowDir, err := mapFundFlowDir(q.FlowType)
	if err != nil {
		return empty, err
	}
	currency, err := mapFundCurrency(q.Currency)
	if err != nil {
		return empty, err
	}

	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filter := sqlcdb.CountAdminFundRecordsParams{
		TimeFrom:       pgtype.Timestamptz{Time: timeFrom, Valid: true},
		TimeTo:         pgtype.Timestamptz{Time: timeTo, Valid: true},
		MemberAccount:  pgTextOptional(q.MemberAccount),
		LedgerNo:       pgTextOptional(q.LedgerNo),
		FlowDir:        flowDir,
		Currency:       currency,
	}
	total, err := s.q.CountAdminFundRecords(ctx, filter)
	if err != nil {
		return empty, err
	}

	offset := (page - 1) * pageSize
	listRows, err := s.q.ListAdminFundRecordsPaged(ctx, sqlcdb.ListAdminFundRecordsPagedParams{
		TimeFrom:      filter.TimeFrom,
		TimeTo:        filter.TimeTo,
		MemberAccount: filter.MemberAccount,
		LedgerNo:      filter.LedgerNo,
		FlowDir:       filter.FlowDir,
		Currency:      filter.Currency,
		RowLimit:      int32(pageSize),
		RowOffset:     int32(offset),
	})
	if err != nil {
		return empty, err
	}

	items := make([]AdminSiteFundRecordItem, 0, len(listRows))
	for _, row := range listRows {
		base := mapFundRecordRow(fundRecordRow{
			ID:           row.ID,
			LedgerNo:     row.LedgerNo,
			TxnType:      row.TxnType,
			DeltaAmount:  row.DeltaAmount,
			BalanceAfter: row.BalanceAfter,
			Currency:     row.Currency,
			SchemeName:   row.SchemeName,
			CreatedAt:    row.CreatedAt,
		})
		items = append(items, AdminSiteFundRecordItem{
			FundRecordItem: base,
			Member:         strings.TrimSpace(row.Account),
		})
	}
	return AdminSiteFundRecordsResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

type fundRecordRow struct {
	ID           int64
	LedgerNo     string
	TxnType      string
	DeltaAmount  float64
	BalanceAfter float64
	Currency     string
	SchemeName   string
	CreatedAt    pgtype.Timestamptz
}

func fundRowsFromList(rows []sqlcdb.ListMemberFundRecordsRow) []fundRecordRow {
	out := make([]fundRecordRow, len(rows))
	for i, r := range rows {
		out[i] = fundRecordRow{
			ID:           r.ID,
			LedgerNo:     r.LedgerNo,
			TxnType:      r.TxnType,
			DeltaAmount:  r.DeltaAmount,
			BalanceAfter: r.BalanceAfter,
			Currency:     r.Currency,
			SchemeName:   r.SchemeName,
			CreatedAt:    r.CreatedAt,
		}
	}
	return out
}

func fundRowsFromCursor(rows []sqlcdb.ListMemberFundRecordsAfterCursorRow) []fundRecordRow {
	out := make([]fundRecordRow, len(rows))
	for i, r := range rows {
		out[i] = fundRecordRow{
			ID:           r.ID,
			LedgerNo:     r.LedgerNo,
			TxnType:      r.TxnType,
			DeltaAmount:  r.DeltaAmount,
			BalanceAfter: r.BalanceAfter,
			Currency:     r.Currency,
			SchemeName:   r.SchemeName,
			CreatedAt:    r.CreatedAt,
		}
	}
	return out
}

func fundRowsFromPaged(rows []sqlcdb.ListMemberFundRecordsPagedRow) []fundRecordRow {
	out := make([]fundRecordRow, len(rows))
	for i, r := range rows {
		out[i] = fundRecordRow{
			ID:           r.ID,
			LedgerNo:     r.LedgerNo,
			TxnType:      r.TxnType,
			DeltaAmount:  r.DeltaAmount,
			BalanceAfter: r.BalanceAfter,
			Currency:     r.Currency,
			SchemeName:   r.SchemeName,
			CreatedAt:    r.CreatedAt,
		}
	}
	return out
}

func mapFundRecordRow(row fundRecordRow) FundRecordItem {
	delta := RoundMoney(row.DeltaAmount)
	flowCode := "expense"
	flowLabel := "支出"
	if delta > 0 {
		flowCode = "income"
		flowLabel = "收入"
	}
	schemeName := strings.TrimSpace(row.SchemeName)
	if schemeName == "" {
		schemeName = "—"
	}
	return FundRecordItem{
		ID:           row.LedgerNo,
		SchemeName:   schemeName,
		Currency:     row.Currency,
		Amount:       delta,
		Time:         timeutil.FormatISO(row.CreatedAt.Time),
		FlowType:     flowLabel,
		FlowTypeCode: flowCode,
		BalanceAfter: RoundMoney(row.BalanceAfter),
		LedgerNo:     row.LedgerNo,
	}
}

func mapFundFlowDir(raw string) (pgtype.Text, error) {
	switch strings.TrimSpace(raw) {
	case "", "all":
		return pgtype.Text{}, nil
	case "income", "expense":
		return pgtype.Text{String: raw, Valid: true}, nil
	default:
		return pgtype.Text{}, fmt.Errorf("%w: flowType 参数无效", ErrInvalidQuery)
	}
}

func mapFundCurrency(raw string) (pgtype.Text, error) {
	switch strings.TrimSpace(raw) {
	case "", "all":
		return pgtype.Text{}, nil
	case "USDT", "TRX", "CNY":
		return pgtype.Text{String: raw, Valid: true}, nil
	default:
		return pgtype.Text{}, fmt.Errorf("%w: currency 参数无效", ErrInvalidQuery)
	}
}

func emptyFundRecordsResult() FundRecordsResult {
	return FundRecordsResult{
		Items: []FundRecordItem{},
		Page:  FundRecordsPage{HasMore: false},
	}
}

func pgTextOptional(v string) pgtype.Text {
	v = strings.TrimSpace(v)
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}
