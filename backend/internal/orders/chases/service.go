package chases

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/timeutil"
)

var ErrInvalidQuery = errors.New("invalid chase query")

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
	ChaseNo      string  `json:"chaseNo"`
	TotalIssues  int32   `json:"totalIssues"`
	DoneIssues   int32   `json:"doneIssues"`
	Amount       float64 `json:"amount"`
	Status       string  `json:"status"`
}

type PageMeta struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

type Result struct {
	Items []Item   `json:"items"`
	Page  PageMeta `json:"page"`
}

type Query struct {
	Account  string
	DateFrom string
	DateTo   string
	GameCode string
	Cursor   string
	Limit    int
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

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	category := mapGameCodeFilter(q.GameCode)
	if q.GameCode != "" && q.GameCode != "all" && category.Valid && category.String == "__invalid__" {
		return Result{}, fmt.Errorf("%w: gameCode 参数无效", ErrInvalidQuery)
	}

	params := sqlcdb.ListChaseOrdersParams{
		MemberID:        m.ID,
		TimeFrom:        pgtype.Timestamptz{Time: timeFrom, Valid: true},
		TimeTo:          pgtype.Timestamptz{Time: timeTo, Valid: true},
		LotteryCategory: category,
		RowLimit:        int32(limit + 1),
	}

	var rows []sqlcdb.ListChaseOrdersRow
	if q.Cursor == "" {
		rows, err = s.q.ListChaseOrders(ctx, params)
	} else {
		anchor, aerr := s.q.GetChaseOrderCursorAnchor(ctx, sqlcdb.GetChaseOrderCursorAnchorParams{
			MemberID: m.ID,
			ChaseNo:  q.Cursor,
		})
		if aerr != nil {
			if errors.Is(aerr, pgx.ErrNoRows) {
				return Result{}, ErrInvalidQuery
			}
			return Result{}, aerr
		}
		cursorRows, lerr := s.q.ListChaseOrdersAfterCursor(ctx, sqlcdb.ListChaseOrdersAfterCursorParams{
			MemberID:        m.ID,
			TimeFrom:        params.TimeFrom,
			TimeTo:          params.TimeTo,
			LotteryCategory: params.LotteryCategory,
			CursorTime:      anchor.StartedAt,
			CursorID:        anchor.ID,
			RowLimit:        int32(limit + 1),
		})
		if lerr != nil {
			return Result{}, lerr
		}
		rows = make([]sqlcdb.ListChaseOrdersRow, len(cursorRows))
		for i, r := range cursorRows {
			rows[i] = chaseRowFromCursor(r)
		}
	}
	if err != nil {
		return Result{}, err
	}

	nextCursor := ""
	hasMore := len(rows) > limit
	if hasMore {
		nextCursor = rows[limit-1].ChaseNo
		rows = rows[:limit]
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, Item{
			Time:        timeutil.FormatDisplayCST(row.StartedAt.Time),
			Game:        row.LotteryName,
			ChaseNo:     row.ChaseNo,
			TotalIssues: row.TotalIssues,
			DoneIssues:  row.DoneIssues,
			Amount:      roundMoney(row.Amount),
			Status:      statusLabel(row.Status),
		})
	}

	return Result{
		Items: items,
		Page:  PageMeta{NextCursor: nextCursor, HasMore: hasMore},
	}, nil
}

func mapGameCodeFilter(raw string) pgtype.Text {
	switch raw {
	case "", "all":
		return pgtype.Text{}
	case "ssc", "pk10", "k3", "x5":
		return pgtype.Text{String: raw, Valid: true}
	default:
		return pgtype.Text{String: "__invalid__", Valid: true}
	}
}

func statusLabel(code string) string {
	switch code {
	case "running":
		return "追号中"
	case "completed":
		return "已完成"
	case "cancelled":
		return "已取消"
	default:
		return code
	}
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func chaseRowFromCursor(r sqlcdb.ListChaseOrdersAfterCursorRow) sqlcdb.ListChaseOrdersRow {
	return sqlcdb.ListChaseOrdersRow{
		ID:              r.ID,
		ChaseNo:         r.ChaseNo,
		LotteryName:     r.LotteryName,
		LotteryCategory: r.LotteryCategory,
		TotalIssues:     r.TotalIssues,
		DoneIssues:      r.DoneIssues,
		Amount:          r.Amount,
		Status:          r.Status,
		StartedAt:       r.StartedAt,
	}
}
