-- name: ListPublishedAnnouncements :many
SELECT a.id, a.title, a.published_at,
       EXISTS (
           SELECT 1 FROM member_announcement_reads r
           WHERE r.member_id = sqlc.arg(member_id) AND r.announcement_id = a.id
       ) AS is_read
FROM cms_announcements a
WHERE a.status = 'published'
ORDER BY a.pinned DESC, a.published_at DESC NULLS LAST, a.id DESC;

-- name: GetPublishedAnnouncement :one
SELECT id, title, published_at, body_html
FROM cms_announcements
WHERE id = $1 AND status = 'published';

-- name: UpsertAnnouncementRead :exec
INSERT INTO member_announcement_reads (member_id, announcement_id, read_at)
VALUES ($1, $2, now())
ON CONFLICT (member_id, announcement_id) DO UPDATE SET read_at = EXCLUDED.read_at;

-- name: ListFaqArticles :many
SELECT id, title, sort
FROM cms_faq_articles
ORDER BY sort ASC, id ASC;

-- name: GetFaqArticle :one
SELECT id, title, body_html
FROM cms_faq_articles
WHERE id = $1;

-- name: ListHelpArticles :many
SELECT id, title, sort, body_html
FROM cms_help_articles
ORDER BY sort ASC, id ASC;

-- name: InsertMemberFeedback :one
INSERT INTO member_feedback (member_id, subject, content)
VALUES ($1, $2, $3)
RETURNING id, subject, content, created_at;

-- name: ListAnnouncementsAdmin :many
SELECT id, title, status, published_at, body_html, pinned
FROM cms_announcements
ORDER BY pinned DESC, published_at DESC NULLS LAST, id DESC;

-- name: UpsertAnnouncementAdmin :one
INSERT INTO cms_announcements (id, title, status, published_at, body_html, pinned)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status,
    published_at = EXCLUDED.published_at,
    body_html = EXCLUDED.body_html,
    pinned = CASE WHEN EXCLUDED.status = 'draft' THEN false ELSE cms_announcements.pinned END
RETURNING id, title, status, published_at, body_html, pinned;

-- name: ClearAnnouncementPinsAdmin :exec
UPDATE cms_announcements SET pinned = false WHERE pinned = true;

-- name: SetAnnouncementPinnedAdmin :one
UPDATE cms_announcements
SET pinned = $2
WHERE id = $1
RETURNING id, title, status, published_at, body_html, pinned;

-- name: DeleteAnnouncementAdmin :exec
DELETE FROM cms_announcements WHERE id = $1;

-- name: ListFaqArticlesAdmin :many
SELECT id, title, sort, body_html
FROM cms_faq_articles
ORDER BY sort ASC, id ASC;

-- name: UpsertFaqArticleAdmin :one
INSERT INTO cms_faq_articles (id, title, sort, body_html)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    sort = EXCLUDED.sort,
    body_html = EXCLUDED.body_html
RETURNING id, title, sort, body_html;

-- name: DeleteFaqArticleAdmin :exec
DELETE FROM cms_faq_articles WHERE id = $1;

-- name: ListHelpArticlesAdmin :many
SELECT id, title, sort, body_html
FROM cms_help_articles
ORDER BY sort ASC, id ASC;

-- name: UpsertHelpArticleAdmin :one
INSERT INTO cms_help_articles (id, title, sort, body_html)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    sort = EXCLUDED.sort,
    body_html = EXCLUDED.body_html
RETURNING id, title, sort, body_html;

-- name: DeleteHelpArticleAdmin :exec
DELETE FROM cms_help_articles WHERE id = $1;

-- name: ListLobbySlotsAdmin :many
SELECT id, slot_key, title, brief, sort, enabled
FROM cms_lobby_slots
ORDER BY sort ASC, id ASC;

-- name: ListLobbySlotsPublic :many
SELECT slot_key, title, brief, sort
FROM cms_lobby_slots
WHERE enabled = true
ORDER BY sort ASC, id ASC;

-- name: UpsertLobbySlotAdmin :one
INSERT INTO cms_lobby_slots (id, slot_key, title, brief, sort, enabled)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    slot_key = EXCLUDED.slot_key,
    title = EXCLUDED.title,
    brief = EXCLUDED.brief,
    sort = EXCLUDED.sort,
    enabled = EXCLUDED.enabled,
    updated_at = now()
RETURNING id, slot_key, title, brief, sort, enabled;

-- name: GetSiteBrandAdmin :one
SELECT site_name, logo_url, tagline
FROM cms_site_brand
WHERE id = 'default';

-- name: GetSiteBrandPublic :one
SELECT site_name, logo_url, tagline
FROM cms_site_brand
WHERE id = 'default';

-- name: UpsertSiteBrandAdmin :one
INSERT INTO cms_site_brand (id, site_name, logo_url, tagline)
VALUES ('default', $1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
    site_name = EXCLUDED.site_name,
    logo_url = EXCLUDED.logo_url,
    tagline = EXCLUDED.tagline,
    updated_at = now()
RETURNING site_name, logo_url, tagline;

-- name: GetPromoChannelAdmin :one
SELECT default_material, invite_code_required
FROM cms_promo_channel
WHERE id = 'default';

-- name: UpsertPromoChannelAdmin :one
INSERT INTO cms_promo_channel (id, default_material, invite_code_required)
VALUES ('default', $1, $2)
ON CONFLICT (id) DO UPDATE SET
    default_material = EXCLUDED.default_material,
    invite_code_required = EXCLUDED.invite_code_required,
    updated_at = now()
RETURNING default_material, invite_code_required;

-- name: CountBannersAdmin :one
SELECT COUNT(*)::bigint
FROM cms_banners b
WHERE (
    sqlc.narg(enabled_filter)::boolean IS NULL
    OR b.enabled = sqlc.narg(enabled_filter)::boolean
)
AND (
    sqlc.narg(created_from)::timestamptz IS NULL
    OR b.created_at >= sqlc.narg(created_from)::timestamptz
)
AND (
    sqlc.narg(created_to)::timestamptz IS NULL
    OR b.created_at <= sqlc.narg(created_to)::timestamptz
);

-- name: ListBannersAdmin :many
SELECT id, remark, image_url, link_url, sort, enabled, created_at, updated_at
FROM cms_banners b
WHERE (
    sqlc.narg(enabled_filter)::boolean IS NULL
    OR b.enabled = sqlc.narg(enabled_filter)::boolean
)
AND (
    sqlc.narg(created_from)::timestamptz IS NULL
    OR b.created_at >= sqlc.narg(created_from)::timestamptz
)
AND (
    sqlc.narg(created_to)::timestamptz IS NULL
    OR b.created_at <= sqlc.narg(created_to)::timestamptz
)
ORDER BY sort ASC, created_at DESC, id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: UpsertBannerAdmin :one
INSERT INTO cms_banners (id, remark, image_url, link_url, sort, enabled)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    remark = EXCLUDED.remark,
    image_url = EXCLUDED.image_url,
    link_url = EXCLUDED.link_url,
    sort = EXCLUDED.sort,
    enabled = EXCLUDED.enabled,
    updated_at = now()
RETURNING id, remark, image_url, link_url, sort, enabled, created_at, updated_at;

-- name: SetBannerEnabledAdmin :one
UPDATE cms_banners
SET enabled = $2, updated_at = now()
WHERE id = $1
RETURNING id, remark, image_url, link_url, sort, enabled, created_at, updated_at;

-- name: DeleteBannerAdmin :exec
DELETE FROM cms_banners WHERE id = $1;

-- name: ListBannersPublic :many
SELECT id, image_url, link_url, sort
FROM cms_banners
WHERE enabled = true
ORDER BY sort ASC, id ASC;
