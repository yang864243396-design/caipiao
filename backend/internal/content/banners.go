package content

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

type AdminBanner struct {
	ID        string `json:"id"`
	Remark    string `json:"remark"`
	ImageUrl  string `json:"imageUrl"`
	LinkUrl   string `json:"linkUrl"`
	Sort      int    `json:"sort"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type AdminBannerListQuery struct {
	Page        int
	PageSize    int
	Enabled     *bool
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type AdminBannerListResult struct {
	Items []AdminBanner `json:"items"`
	Total int64         `json:"total"`
}

type PublicBanner struct {
	ID       string `json:"id"`
	ImageUrl string `json:"imageUrl"`
	LinkUrl  string `json:"linkUrl"`
	Sort     int    `json:"sort"`
}

type SaveBannerInput struct {
	ID       string
	Remark   string
	ImageUrl string
	LinkUrl  string
	Sort     int
	Enabled  bool
}

func (s *Service) AdminListBanners(ctx context.Context, q AdminBannerListQuery) (AdminBannerListResult, error) {
	if s == nil || s.q == nil {
		return AdminBannerListResult{}, ErrUnavailable
	}
	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	filterParams := bannerFilterParams(q.Enabled, q.CreatedFrom, q.CreatedTo)

	total, err := s.q.CountBannersAdmin(ctx, filterParams.count)
	if err != nil {
		return AdminBannerListResult{}, err
	}
	rows, err := s.q.ListBannersAdmin(ctx, sqlcdb.ListBannersAdminParams{
		EnabledFilter: filterParams.enabled,
		CreatedFrom:   filterParams.createdFrom,
		CreatedTo:     filterParams.createdTo,
		RowLimit:      int32(pageSize),
		RowOffset:     int32(offset),
	})
	if err != nil {
		return AdminBannerListResult{}, err
	}
	items := make([]AdminBanner, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapBannerAdminRow(row))
	}
	return AdminBannerListResult{Items: items, Total: total}, nil
}

func (s *Service) AdminSaveBanner(ctx context.Context, in SaveBannerInput) (AdminBanner, error) {
	if s == nil || s.q == nil {
		return AdminBanner{}, ErrUnavailable
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = fmt.Sprintf("BN_%d", time.Now().UnixNano())
	}
	imageUrl := strings.TrimSpace(in.ImageUrl)
	if imageUrl == "" {
		return AdminBanner{}, ErrInvalid
	}
	sort := in.Sort
	if sort < 0 {
		sort = 0
	}
	if sort > 9999 {
		sort = 9999
	}
	row, err := s.q.UpsertBannerAdmin(ctx, sqlcdb.UpsertBannerAdminParams{
		ID: id, Remark: strings.TrimSpace(in.Remark), ImageUrl: imageUrl,
		LinkUrl: normalizeBannerLink(in.LinkUrl), Sort: int32(sort), Enabled: in.Enabled,
	})
	if err != nil {
		return AdminBanner{}, err
	}
	return mapBannerAdminRow(row), nil
}

func (s *Service) AdminSetBannerEnabled(ctx context.Context, id string, enabled bool) (AdminBanner, error) {
	if s == nil || s.q == nil {
		return AdminBanner{}, ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return AdminBanner{}, ErrInvalid
	}
	row, err := s.q.SetBannerEnabledAdmin(ctx, sqlcdb.SetBannerEnabledAdminParams{ID: id, Enabled: enabled})
	if err != nil {
		if err == pgx.ErrNoRows {
			return AdminBanner{}, ErrNotFound
		}
		return AdminBanner{}, err
	}
	return mapBannerAdminRow(row), nil
}

func (s *Service) AdminDeleteBanner(ctx context.Context, id string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	return s.q.DeleteBannerAdmin(ctx, id)
}

func (s *Service) PublicBanners(ctx context.Context) ([]PublicBanner, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListBannersPublic(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PublicBanner, 0, len(rows))
	for _, row := range rows {
		out = append(out, PublicBanner{
			ID: row.ID, ImageUrl: row.ImageUrl, LinkUrl: row.LinkUrl, Sort: int(row.Sort),
		})
	}
	return out, nil
}

type bannerFilterBundle struct {
	count       sqlcdb.CountBannersAdminParams
	enabled     pgtype.Bool
	createdFrom pgtype.Timestamptz
	createdTo   pgtype.Timestamptz
}

func bannerFilterParams(enabled *bool, createdFrom, createdTo *time.Time) bannerFilterBundle {
	var enabledFilter pgtype.Bool
	if enabled != nil {
		enabledFilter = pgtype.Bool{Bool: *enabled, Valid: true}
	}
	var from pgtype.Timestamptz
	if createdFrom != nil {
		from = pgtype.Timestamptz{Time: createdFrom.UTC(), Valid: true}
	}
	var to pgtype.Timestamptz
	if createdTo != nil {
		to = pgtype.Timestamptz{Time: createdTo.UTC(), Valid: true}
	}
	countParams := sqlcdb.CountBannersAdminParams{
		EnabledFilter: enabledFilter,
		CreatedFrom:   from,
		CreatedTo:     to,
	}
	return bannerFilterBundle{
		count: countParams, enabled: enabledFilter, createdFrom: from, createdTo: to,
	}
}

func mapBannerAdminRow(row sqlcdb.ListBannersAdminRow) AdminBanner {
	return AdminBanner{
		ID: row.ID, Remark: row.Remark, ImageUrl: row.ImageUrl, LinkUrl: row.LinkUrl,
		Sort: int(row.Sort), Enabled: row.Enabled,
		CreatedAt: timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt: timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func normalizeBannerLink(raw string) string {
	link := strings.TrimSpace(raw)
	if link == "" {
		return ""
	}
	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		link = "https://" + link
	}
	return link
}
