-- name: ListLotteryCatalog :many
SELECT
    code,
    display_name,
    category_code,
    play_template,
    ball_count,
    draw_interval,
    sort_order,
    on_sale,
    sale_status,
    outbound_lottery_code
FROM lottery_catalog
ORDER BY sort_order ASC, code ASC;

-- name: ListLotteryCatalogOnSale :many
SELECT
    code,
    display_name,
    category_code,
    play_template,
    ball_count,
    draw_interval,
    sort_order,
    on_sale,
    sale_status,
    outbound_lottery_code
FROM lottery_catalog
WHERE sale_status = 'on_sale'
ORDER BY sort_order ASC, code ASC;

-- name: GetLotteryCatalogByCode :one
SELECT
    code,
    display_name,
    category_code,
    play_template,
    ball_count,
    draw_interval,
    sort_order,
    on_sale,
    sale_status,
    outbound_lottery_code
FROM lottery_catalog
WHERE code = $1;

-- name: CountLotteryCatalogWithTemplate :one
SELECT COUNT(*)::int AS count
FROM lottery_catalog
WHERE play_template IS NOT NULL AND play_template <> '';

-- name: SetLotteryCatalogMaintenance :one
UPDATE lottery_catalog
SET
    sale_status = 'maintenance'::lottery_sale_status,
    on_sale = false,
    updated_at = now()
WHERE code = $1
  AND sale_status = 'on_sale'::lottery_sale_status
RETURNING
    code,
    display_name,
    category_code,
    play_template,
    ball_count,
    draw_interval,
    sort_order,
    on_sale,
    sale_status,
    outbound_lottery_code;

-- name: PatchLotteryCatalogMaintenance :one
UPDATE lottery_catalog
SET
    display_name = $2,
    outbound_lottery_code = $3,
    sort_order = $4,
    sale_status = $5::lottery_sale_status,
    on_sale = ($5 = 'on_sale'::lottery_sale_status),
    updated_at = now()
WHERE code = $1
  AND sale_status = 'maintenance'::lottery_sale_status
RETURNING
    code,
    display_name,
    category_code,
    play_template,
    ball_count,
    draw_interval,
    sort_order,
    on_sale,
    sale_status,
    outbound_lottery_code;

-- name: UpdateSchemeDefinitionsLotteryLabel :exec
UPDATE scheme_definitions
SET lottery_label = $2, updated_at = now()
WHERE lottery_code = $1;

-- name: UpdateSchemeInstancesLotteryLabel :exec
UPDATE scheme_instances
SET lottery_label = $2, updated_at = now()
WHERE lottery_code = $1;

-- name: UpdateSchemeShareSnapshotsLotteryLabel :exec
UPDATE scheme_share_snapshots
SET lottery_label = $2, updated_at = now()
WHERE lottery_code = $1;

-- name: SyncLotteryCatalogGuaji :one
UPDATE lottery_catalog
SET
    display_name = $2,
    outbound_lottery_code = $3,
    updated_at = now()
WHERE code = $1
RETURNING
    code,
    display_name,
    category_code,
    play_template,
    ball_count,
    draw_interval,
    sort_order,
    on_sale,
    sale_status,
    outbound_lottery_code;
