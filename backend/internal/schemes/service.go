package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"

	"caipiao/backend/internal/guaji/periodsync"
)

var ErrUnavailable = errors.New("schemes service unavailable")

type Service struct {
	q           *sqlcdb.Queries
	pool        *db.Pool
	periodSync  *periodsync.Syncer
	authChecker memberAuthChecker
}

// memberAuthChecker 由 accountsvc 注入：开启真实投注前校验授权与第三方可用余额。
type memberAuthChecker interface {
	HasHealthyAuthForMember(ctx context.Context, memberAccount string) (bool, error)
	// PrimaryUsableBalance 拉取启用授权的主币种可用余额；对接关闭时返回 ok=false。
	PrimaryUsableBalance(ctx context.Context, memberAccount string) (amount float64, ok bool, err error)
	// UsableBalance 按币种拉取可用余额；对接关闭时返回 ok=false。
	UsableBalance(ctx context.Context, memberAccount, currency string) (amount float64, ok bool, err error)
}

func NewService(pool *db.Pool, periodSync *periodsync.Syncer) *Service {
	if pool == nil {
		return nil
	}
	return &Service{q: sqlcdb.New(pool), pool: pool, periodSync: periodSync}
}

func (s *Service) SetMemberAuthChecker(c memberAuthChecker) {
	if s == nil {
		return
	}
	s.authChecker = c
}

type ShareSnapshot struct {
	ID           string                 `json:"id"`
	Kind         string                 `json:"kind"`
	SchemeName   string                 `json:"schemeName"`
	LotteryCode  string                 `json:"lotteryCode"`
	LotteryLabel string                 `json:"lotteryLabel,omitempty"`
	PlayMethod   string                 `json:"playMethod,omitempty"`
	FundYuan     float64                `json:"fundYuan,omitempty"`
	Config       map[string]interface{} `json:"config"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
}

type PageMeta struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

type ShareCatalogResult struct {
	Items []ShareSnapshot `json:"items"`
	Page  PageMeta        `json:"page"`
}

type ShareCatalogQuery struct {
	Keyword string
	Cursor  string
	Limit   int
}

func (s *Service) ShareCatalog(ctx context.Context, q ShareCatalogQuery) (ShareCatalogResult, error) {
	if s == nil || s.q == nil {
		return ShareCatalogResult{}, ErrUnavailable
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	kw := pgtype.Text{}
	if q.Keyword != "" {
		kw = pgtype.Text{String: q.Keyword, Valid: true}
	}
	cursor := pgtype.Text{}
	if q.Cursor != "" {
		cursor = pgtype.Text{String: q.Cursor, Valid: true}
	}

	rows, err := s.q.ListSchemeShareSnapshots(ctx, sqlcdb.ListSchemeShareSnapshotsParams{
		Keyword:  kw,
		Cursor:   cursor,
		RowLimit: int32(limit + 1),
	})
	if err != nil {
		return ShareCatalogResult{}, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	items := make([]ShareSnapshot, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapShareSnapshotRow(row))
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].ID
	}

	return ShareCatalogResult{
		Items: items,
		Page: PageMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func mapShareSnapshotRow(row sqlcdb.SchemeShareSnapshot) ShareSnapshot {
	cfg := map[string]interface{}{}
	if len(row.Config) > 0 {
		_ = json.Unmarshal(row.Config, &cfg)
	}
	return ShareSnapshot{
		ID:           row.ID,
		Kind:         row.Kind,
		SchemeName:   row.SchemeName,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: row.LotteryLabel,
		PlayMethod:   row.PlayMethod,
		FundYuan:     numericToFloat(row.FundYuan),
		Config:       cfg,
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func numericToFloat(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return math.Round(f.Float64*100) / 100
}
