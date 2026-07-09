package sqlcdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const countBannersAdmin = `-- name: CountBannersAdmin :one
SELECT COUNT(*)::bigint
FROM cms_banners b
WHERE (
    $1::boolean IS NULL
    OR b.enabled = $1::boolean
)
AND (
    $2::timestamptz IS NULL
    OR b.created_at >= $2::timestamptz
)
AND (
    $3::timestamptz IS NULL
    OR b.created_at <= $3::timestamptz
)
`

type CountBannersAdminParams struct {
	EnabledFilter pgtype.Bool        `json:"enabled_filter"`
	CreatedFrom   pgtype.Timestamptz `json:"created_from"`
	CreatedTo     pgtype.Timestamptz `json:"created_to"`
}

func (q *Queries) CountBannersAdmin(ctx context.Context, arg CountBannersAdminParams) (int64, error) {
	row := q.db.QueryRow(ctx, countBannersAdmin, arg.EnabledFilter, arg.CreatedFrom, arg.CreatedTo)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const listBannersAdmin = `-- name: ListBannersAdmin :many
SELECT id, remark, image_url, link_url, sort, enabled, created_at, updated_at
FROM cms_banners b
WHERE (
    $1::boolean IS NULL
    OR b.enabled = $1::boolean
)
AND (
    $2::timestamptz IS NULL
    OR b.created_at >= $2::timestamptz
)
AND (
    $3::timestamptz IS NULL
    OR b.created_at <= $3::timestamptz
)
ORDER BY sort ASC, created_at DESC, id DESC
LIMIT $4 OFFSET $5
`

type ListBannersAdminParams struct {
	EnabledFilter pgtype.Bool        `json:"enabled_filter"`
	CreatedFrom   pgtype.Timestamptz `json:"created_from"`
	CreatedTo     pgtype.Timestamptz `json:"created_to"`
	RowLimit      int32              `json:"row_limit"`
	RowOffset     int32              `json:"row_offset"`
}

type ListBannersAdminRow struct {
	ID        string             `json:"id"`
	Remark    string             `json:"remark"`
	ImageUrl  string             `json:"image_url"`
	LinkUrl   string             `json:"link_url"`
	Sort      int32              `json:"sort"`
	Enabled   bool               `json:"enabled"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

func (q *Queries) ListBannersAdmin(ctx context.Context, arg ListBannersAdminParams) ([]ListBannersAdminRow, error) {
	rows, err := q.db.Query(ctx, listBannersAdmin,
		arg.EnabledFilter, arg.CreatedFrom, arg.CreatedTo, arg.RowLimit, arg.RowOffset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListBannersAdminRow{}
	for rows.Next() {
		var i ListBannersAdminRow
		if err := rows.Scan(
			&i.ID, &i.Remark, &i.ImageUrl, &i.LinkUrl, &i.Sort, &i.Enabled, &i.CreatedAt, &i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const upsertBannerAdmin = `-- name: UpsertBannerAdmin :one
INSERT INTO cms_banners (id, remark, image_url, link_url, sort, enabled)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    remark = EXCLUDED.remark,
    image_url = EXCLUDED.image_url,
    link_url = EXCLUDED.link_url,
    sort = EXCLUDED.sort,
    enabled = EXCLUDED.enabled,
    updated_at = now()
RETURNING id, remark, image_url, link_url, sort, enabled, created_at, updated_at
`

type UpsertBannerAdminParams struct {
	ID       string `json:"id"`
	Remark   string `json:"remark"`
	ImageUrl string `json:"image_url"`
	LinkUrl  string `json:"link_url"`
	Sort     int32  `json:"sort"`
	Enabled  bool   `json:"enabled"`
}

type UpsertBannerAdminRow = ListBannersAdminRow

func (q *Queries) UpsertBannerAdmin(ctx context.Context, arg UpsertBannerAdminParams) (UpsertBannerAdminRow, error) {
	row := q.db.QueryRow(ctx, upsertBannerAdmin, arg.ID, arg.Remark, arg.ImageUrl, arg.LinkUrl, arg.Sort, arg.Enabled)
	var i UpsertBannerAdminRow
	err := row.Scan(&i.ID, &i.Remark, &i.ImageUrl, &i.LinkUrl, &i.Sort, &i.Enabled, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const setBannerEnabledAdmin = `-- name: SetBannerEnabledAdmin :one
UPDATE cms_banners
SET enabled = $2, updated_at = now()
WHERE id = $1
RETURNING id, remark, image_url, link_url, sort, enabled, created_at, updated_at
`

type SetBannerEnabledAdminParams struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type SetBannerEnabledAdminRow = ListBannersAdminRow

func (q *Queries) SetBannerEnabledAdmin(ctx context.Context, arg SetBannerEnabledAdminParams) (SetBannerEnabledAdminRow, error) {
	row := q.db.QueryRow(ctx, setBannerEnabledAdmin, arg.ID, arg.Enabled)
	var i SetBannerEnabledAdminRow
	err := row.Scan(&i.ID, &i.Remark, &i.ImageUrl, &i.LinkUrl, &i.Sort, &i.Enabled, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

const deleteBannerAdmin = `-- name: DeleteBannerAdmin :exec
DELETE FROM cms_banners WHERE id = $1
`

func (q *Queries) DeleteBannerAdmin(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, deleteBannerAdmin, id)
	return err
}

const listBannersPublic = `-- name: ListBannersPublic :many
SELECT id, image_url, link_url, sort
FROM cms_banners
WHERE enabled = true
ORDER BY sort ASC, id ASC
`

type ListBannersPublicRow struct {
	ID       string `json:"id"`
	ImageUrl string `json:"image_url"`
	LinkUrl  string `json:"link_url"`
	Sort     int32  `json:"sort"`
}

func (q *Queries) ListBannersPublic(ctx context.Context) ([]ListBannersPublicRow, error) {
	rows, err := q.db.Query(ctx, listBannersPublic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListBannersPublicRow{}
	for rows.Next() {
		var i ListBannersPublicRow
		if err := rows.Scan(&i.ID, &i.ImageUrl, &i.LinkUrl, &i.Sort); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
