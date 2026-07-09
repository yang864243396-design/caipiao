package content

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

var (
	ErrUnavailable = errors.New("content service unavailable")
	ErrNotFound    = errors.New("content not found")
	ErrInvalid     = errors.New("invalid content request")
)

type Service struct {
	q *sqlcdb.Queries
}

func NewService(pool *db.Pool) *Service {
	if pool == nil {
		return nil
	}
	return &Service{q: sqlcdb.New(pool)}
}

type AnnouncementListItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  string `json:"date"`
	Read  bool   `json:"read"`
}

type AnnouncementDetail struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Date      string `json:"date"`
	BodyHtml  string `json:"bodyHtml"`
	Read      bool   `json:"read"`
}

type FaqListItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type FaqDetail struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	BodyHtml string `json:"bodyHtml"`
}

type HelpArticle struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Sort     int    `json:"sort"`
	BodyHtml string `json:"bodyHtml"`
}

type FeedbackInput struct {
	Subject string
	Content string
}

type FeedbackResult struct {
	ID        int64  `json:"id"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"createdAt"`
}

func (s *Service) ListAnnouncements(ctx context.Context, memberID int64) ([]AnnouncementListItem, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListPublishedAnnouncements(ctx, memberID)
	if err != nil {
		return nil, err
	}
	out := make([]AnnouncementListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, AnnouncementListItem{
			ID:    row.ID,
			Title: row.Title,
			Date:  formatDateSlash(row.PublishedAt),
			Read:  row.IsRead,
		})
	}
	return out, nil
}

func (s *Service) GetAnnouncement(ctx context.Context, memberID int64, id string) (AnnouncementDetail, error) {
	if s == nil || s.q == nil {
		return AnnouncementDetail{}, ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return AnnouncementDetail{}, ErrInvalid
	}
	row, err := s.q.GetPublishedAnnouncement(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AnnouncementDetail{}, ErrNotFound
		}
		return AnnouncementDetail{}, err
	}
	if err := s.q.UpsertAnnouncementRead(ctx, sqlcdb.UpsertAnnouncementReadParams{
		MemberID: memberID, AnnouncementID: id,
	}); err != nil {
		return AnnouncementDetail{}, err
	}
	return AnnouncementDetail{
		ID:       row.ID,
		Title:    row.Title,
		Date:     formatDateSlash(row.PublishedAt),
		BodyHtml: row.BodyHtml,
		Read:     true,
	}, nil
}

func (s *Service) ListFaq(ctx context.Context) ([]FaqListItem, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListFaqArticles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]FaqListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, FaqListItem{ID: row.ID, Title: row.Title})
	}
	return out, nil
}

func (s *Service) GetFaq(ctx context.Context, id string) (FaqDetail, error) {
	if s == nil || s.q == nil {
		return FaqDetail{}, ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return FaqDetail{}, ErrInvalid
	}
	row, err := s.q.GetFaqArticle(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FaqDetail{}, ErrNotFound
		}
		return FaqDetail{}, err
	}
	return FaqDetail{ID: row.ID, Title: row.Title, BodyHtml: row.BodyHtml}, nil
}

func (s *Service) ListHelp(ctx context.Context) ([]HelpArticle, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListHelpArticles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]HelpArticle, 0, len(rows))
	for _, row := range rows {
		out = append(out, HelpArticle{
			ID: row.ID, Title: row.Title, Sort: int(row.Sort), BodyHtml: row.BodyHtml,
		})
	}
	return out, nil
}

func (s *Service) SubmitFeedback(ctx context.Context, memberID int64, in FeedbackInput) (FeedbackResult, error) {
	if s == nil || s.q == nil {
		return FeedbackResult{}, ErrUnavailable
	}
	subject := strings.TrimSpace(in.Subject)
	content := strings.TrimSpace(in.Content)
	if subject == "" || content == "" {
		return FeedbackResult{}, ErrInvalid
	}
	if utf8.RuneCountInString(subject) > 80 {
		return FeedbackResult{}, ErrInvalid
	}
	if utf8.RuneCountInString(content) > 500 {
		return FeedbackResult{}, ErrInvalid
	}
	row, err := s.q.InsertMemberFeedback(ctx, sqlcdb.InsertMemberFeedbackParams{
		MemberID: memberID, Subject: subject, Content: content,
	})
	if err != nil {
		return FeedbackResult{}, err
	}
	return FeedbackResult{
		ID:        row.ID,
		Subject:   row.Subject,
		CreatedAt: timeutil.FormatISO(row.CreatedAt.Time),
	}, nil
}

func formatDateSlash(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return ts.Time.In(loc).Format("2006/01/02")
}
