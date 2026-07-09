package content

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

type AdminAnnouncement struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"publishedAt"`
	BodyHtml    string  `json:"bodyHtml"`
	Pinned      bool    `json:"pinned"`
}

type AdminFaqArticle struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Sort     int    `json:"sort"`
	BodyHtml string `json:"bodyHtml"`
}

type AdminHelpArticle struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Sort     int    `json:"sort"`
	BodyHtml string `json:"bodyHtml"`
}

func (s *Service) AdminListAnnouncements(ctx context.Context) ([]AdminAnnouncement, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListAnnouncementsAdmin(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminAnnouncement, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAdminAnnouncementFromList(row))
	}
	return out, nil
}

func (s *Service) AdminSaveAnnouncement(ctx context.Context, in AdminAnnouncement) (AdminAnnouncement, error) {
	if s == nil || s.q == nil {
		return AdminAnnouncement{}, ErrUnavailable
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = fmt.Sprintf("ANN_%d", time.Now().UnixNano())
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return AdminAnnouncement{}, ErrInvalid
	}
	status := mapStatusToDB(in.Status)
	publishedAt := pgtype.Timestamptz{Valid: false}
	if status == "published" {
		if in.PublishedAt != nil && strings.TrimSpace(*in.PublishedAt) != "" {
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*in.PublishedAt)); err == nil {
				publishedAt = pgtype.Timestamptz{Time: t, Valid: true}
			}
		}
		if !publishedAt.Valid {
			publishedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
		}
	}
	row, err := s.q.UpsertAnnouncementAdmin(ctx, sqlcdb.UpsertAnnouncementAdminParams{
		ID: id, Title: title, Status: status, PublishedAt: publishedAt, BodyHtml: in.BodyHtml, Pinned: false,
	})
	if err != nil {
		return AdminAnnouncement{}, err
	}
	return mapAdminAnnouncementFromUpsert(row), nil
}

func (s *Service) AdminSetAnnouncementPinned(ctx context.Context, id string, pinned bool) (AdminAnnouncement, error) {
	if s == nil || s.q == nil {
		return AdminAnnouncement{}, ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return AdminAnnouncement{}, ErrInvalid
	}
	if pinned {
		rows, err := s.q.ListAnnouncementsAdmin(ctx)
		if err != nil {
			return AdminAnnouncement{}, err
		}
		var target *sqlcdb.ListAnnouncementsAdminRow
		for i := range rows {
			if rows[i].ID == id {
				target = &rows[i]
				break
			}
		}
		if target == nil {
			return AdminAnnouncement{}, ErrNotFound
		}
		if target.Status != "published" {
			return AdminAnnouncement{}, ErrInvalid
		}
	}
	if pinned {
		if err := s.q.ClearAnnouncementPinsAdmin(ctx); err != nil {
			return AdminAnnouncement{}, err
		}
	}
	row, err := s.q.SetAnnouncementPinnedAdmin(ctx, sqlcdb.SetAnnouncementPinnedAdminParams{ID: id, Pinned: pinned})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminAnnouncement{}, ErrNotFound
		}
		return AdminAnnouncement{}, err
	}
	return mapAdminAnnouncementFromUpsert(row), nil
}

func (s *Service) AdminDeleteAnnouncement(ctx context.Context, id string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	return s.q.DeleteAnnouncementAdmin(ctx, id)
}

func (s *Service) AdminListFaqArticles(ctx context.Context) ([]AdminFaqArticle, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListFaqArticlesAdmin(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminFaqArticle, 0, len(rows))
	for _, row := range rows {
		out = append(out, AdminFaqArticle{
			ID: row.ID, Title: row.Title, Sort: int(row.Sort), BodyHtml: row.BodyHtml,
		})
	}
	return out, nil
}

func (s *Service) AdminSaveFaqArticle(ctx context.Context, in AdminFaqArticle) (AdminFaqArticle, error) {
	if s == nil || s.q == nil {
		return AdminFaqArticle{}, ErrUnavailable
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = fmt.Sprintf("FAQ_%d", time.Now().UnixNano())
	}
	if strings.TrimSpace(in.Title) == "" {
		return AdminFaqArticle{}, ErrInvalid
	}
	sort := int32(in.Sort)
	if sort <= 0 {
		rows, err := s.q.ListFaqArticlesAdmin(ctx)
		if err != nil {
			return AdminFaqArticle{}, err
		}
		maxSort := int32(0)
		for _, row := range rows {
			if row.Sort > maxSort {
				maxSort = row.Sort
			}
		}
		sort = maxSort + 1
	}
	row, err := s.q.UpsertFaqArticleAdmin(ctx, sqlcdb.UpsertFaqArticleAdminParams{
		ID: id, Title: in.Title, Sort: sort, BodyHtml: in.BodyHtml,
	})
	if err != nil {
		return AdminFaqArticle{}, err
	}
	return AdminFaqArticle{
		ID: row.ID, Title: row.Title, Sort: int(row.Sort), BodyHtml: row.BodyHtml,
	}, nil
}

func (s *Service) AdminDeleteFaqArticle(ctx context.Context, id string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	return s.q.DeleteFaqArticleAdmin(ctx, id)
}

func (s *Service) AdminListHelpArticles(ctx context.Context) ([]AdminHelpArticle, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListHelpArticlesAdmin(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminHelpArticle, 0, len(rows))
	for _, row := range rows {
		out = append(out, AdminHelpArticle{
			ID: row.ID, Title: row.Title, Sort: int(row.Sort), BodyHtml: row.BodyHtml,
		})
	}
	return out, nil
}

func (s *Service) AdminSaveHelpArticle(ctx context.Context, in AdminHelpArticle) (AdminHelpArticle, error) {
	if s == nil || s.q == nil {
		return AdminHelpArticle{}, ErrUnavailable
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = fmt.Sprintf("HP_%d", time.Now().UnixNano())
	}
	if strings.TrimSpace(in.Title) == "" {
		return AdminHelpArticle{}, ErrInvalid
	}
	row, err := s.q.UpsertHelpArticleAdmin(ctx, sqlcdb.UpsertHelpArticleAdminParams{
		ID: id, Title: in.Title, Sort: int32(in.Sort), BodyHtml: in.BodyHtml,
	})
	if err != nil {
		return AdminHelpArticle{}, err
	}
	return AdminHelpArticle{
		ID: row.ID, Title: row.Title, Sort: int(row.Sort), BodyHtml: row.BodyHtml,
	}, nil
}

func (s *Service) AdminDeleteHelpArticle(ctx context.Context, id string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	return s.q.DeleteHelpArticleAdmin(ctx, id)
}

func mapStatusToDB(status string) string {
	if status == "已发布" || strings.EqualFold(status, "published") {
		return "published"
	}
	return "draft"
}

func mapStatusFromDB(status string) string {
	if status == "published" {
		return "已发布"
	}
	return "草稿"
}

func mapAdminAnnouncementFromList(row sqlcdb.ListAnnouncementsAdminRow) AdminAnnouncement {
	return mapAdminAnnouncementFields(row.ID, row.Title, row.Status, row.PublishedAt, row.BodyHtml, row.Pinned)
}

func mapAdminAnnouncementFromUpsert(row sqlcdb.UpsertAnnouncementAdminRow) AdminAnnouncement {
	return mapAdminAnnouncementFields(row.ID, row.Title, row.Status, row.PublishedAt, row.BodyHtml, row.Pinned)
}

func mapAdminAnnouncementFields(id, title, status string, publishedAt pgtype.Timestamptz, bodyHtml string, pinned bool) AdminAnnouncement {
	var publishedAtOut *string
	if publishedAt.Valid {
		iso := publishedAt.Time.UTC().Format(time.RFC3339)
		publishedAtOut = &iso
	}
	return AdminAnnouncement{
		ID: id, Title: title, Status: mapStatusFromDB(status),
		PublishedAt: publishedAtOut, BodyHtml: bodyHtml, Pinned: pinned,
	}
}
