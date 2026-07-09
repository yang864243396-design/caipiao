package member

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

var ErrUnavailable = errors.New("member service unavailable")

type AdminGuajiBalances struct {
	USDT float64 `json:"usdt"`
	TRX  float64 `json:"trx"`
	CNY  float64 `json:"cny"`
}

type AdminMemberRow struct {
	ID            string             `json:"id"`
	Account       string             `json:"account"`
	DisplayName   string             `json:"displayName"`
	Status        string             `json:"status"`
	GuajiBalances AdminGuajiBalances `json:"guajiBalances"`
	BalanceYuan   float64            `json:"balanceYuan,omitempty"`
	RegisteredAt  string             `json:"registeredAt"`
	LastLoginAt   string             `json:"lastLoginAt"`
}

type AdminMemberListResult struct {
	Items []AdminMemberRow `json:"items"`
	Total int64            `json:"total"`
}

type AdminMemberListQuery struct {
	Keyword     string
	SearchField string
	Page        int
	PageSize    int
}

func ParseMemberID(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty member id")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrNotFound
	}
	return id, nil
}

func (s *Service) AdminListMembers(ctx context.Context, q AdminMemberListQuery) (AdminMemberListResult, error) {
	if s == nil || s.q == nil {
		return AdminMemberListResult{}, ErrUnavailable
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
	kw := pgText(q.Keyword)
	searchField := normalizeMemberSearchField(q.SearchField)

	total, err := s.q.CountAdminMembers(ctx, sqlcdb.CountAdminMembersParams{
		Keyword:     kw,
		SearchField: searchField,
	})
	if err != nil {
		return AdminMemberListResult{}, err
	}

	offset := (page - 1) * pageSize
	rows, err := s.q.ListAdminMembers(ctx, sqlcdb.ListAdminMembersParams{
		Keyword:     kw,
		SearchField: searchField,
		RowLimit:    int32(pageSize),
		RowOffset:   int32(offset),
	})
	if err != nil {
		return AdminMemberListResult{}, err
	}

	items := make([]AdminMemberRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAdminMemberRow(row))
	}
	return AdminMemberListResult{Items: items, Total: total}, nil
}

func (s *Service) AdminGetMember(ctx context.Context, memberID int64) (AdminMemberRow, error) {
	if s == nil || s.q == nil {
		return AdminMemberRow{}, ErrUnavailable
	}
	if memberID <= 0 {
		return AdminMemberRow{}, ErrNotFound
	}
	row, err := s.q.GetMemberByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminMemberRow{}, ErrNotFound
		}
		return AdminMemberRow{}, err
	}
	return mapAdminMemberDetailRow(row), nil
}

func mapAdminMemberRow(row sqlcdb.ListAdminMembersRow) AdminMemberRow {
	return AdminMemberRow{
		ID:           formatMemberID(row.ID),
		Account:      row.Account,
		DisplayName:  row.DisplayName,
		Status:       statusLabel(row.Status),
		RegisteredAt: timeutil.FormatISO(row.RegisteredAt.Time),
		LastLoginAt:  formatLastLogin(row.LastLoginAt),
	}
}

func mapAdminMemberDetailRow(row sqlcdb.GetMemberByIDRow) AdminMemberRow {
	return AdminMemberRow{
		ID:           formatMemberID(row.ID),
		Account:      row.Account,
		DisplayName:  row.DisplayName,
		Status:       statusLabel(row.Status),
		BalanceYuan:  roundMoney(row.Balance),
		RegisteredAt: timeutil.FormatISO(row.RegisteredAt.Time),
		LastLoginAt:  formatLastLogin(row.LastLoginAt),
	}
}

func formatMemberID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func normalizeMemberSearchField(raw string) string {
	switch strings.TrimSpace(raw) {
	case "guajiAccount", "guaji":
		return "guajiAccount"
	case "id":
		return "id"
	default:
		return "account"
	}
}

func statusLabel(status string) string {
	if status == "frozen" {
		return "冻结"
	}
	return "正常"
}

func formatLastLogin(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return timeutil.FormatISO(ts.Time)
}

func pgText(v string) pgtype.Text {
	v = strings.TrimSpace(v)
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}
