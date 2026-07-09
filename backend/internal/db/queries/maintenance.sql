-- name: GetMaintenanceAdmin :one
SELECT enabled, popup_announcement_id, title, message
FROM cms_maintenance
WHERE id = 'default';

-- name: UpsertMaintenanceAdmin :one
INSERT INTO cms_maintenance (id, enabled, popup_announcement_id, title, message)
VALUES ('default', $1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    popup_announcement_id = EXCLUDED.popup_announcement_id,
    title = EXCLUDED.title,
    message = EXCLUDED.message,
    updated_at = now()
RETURNING enabled, popup_announcement_id, title, message;

-- name: GetPublishedAnnouncementBrief :one
SELECT id, title, body_html
FROM cms_announcements
WHERE id = $1 AND status = 'published';
