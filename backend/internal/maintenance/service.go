package maintenance

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

var (
	ErrUnavailable = errors.New("maintenance service unavailable")
	ErrInvalid     = errors.New("invalid maintenance request")
)

type PopupAnnouncement struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	BodyHTML string `json:"bodyHtml"`
}

type State struct {
	Enabled             bool               `json:"enabled"`
	PopupAnnouncementID string             `json:"popupAnnouncementId,omitempty"`
	Title               string             `json:"title,omitempty"`
	Message             string             `json:"message,omitempty"`
	PopupAnnouncement   *PopupAnnouncement `json:"popupAnnouncement,omitempty"`
}

type AdminState struct {
	Enabled             bool   `json:"enabled"`
	PopupAnnouncementID string `json:"popupAnnouncementId,omitempty"`
	Title               string `json:"title,omitempty"`
	Message             string `json:"message,omitempty"`
}

type Service struct {
	q *sqlcdb.Queries
}

func NewService(pool *db.Pool) *Service {
	if pool == nil {
		return nil
	}
	return &Service{q: sqlcdb.New(pool)}
}

func (s *Service) AdminGet(ctx context.Context) (AdminState, error) {
	if s == nil || s.q == nil {
		return AdminState{}, ErrUnavailable
	}
	row, err := s.q.GetMaintenanceAdmin(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminState{}, nil
		}
		return AdminState{}, err
	}
	return mapAdminRow(row.Enabled, row.PopupAnnouncementID, row.Title, row.Message), nil
}

func (s *Service) AdminSave(ctx context.Context, in AdminState) (AdminState, error) {
	if s == nil || s.q == nil {
		return AdminState{}, ErrUnavailable
	}
	row, err := s.q.UpsertMaintenanceAdmin(ctx, sqlcdb.UpsertMaintenanceAdminParams{
		Enabled:             in.Enabled,
		PopupAnnouncementID: textParam(in.PopupAnnouncementID),
		Title:               strings.TrimSpace(in.Title),
		Message:             strings.TrimSpace(in.Message),
	})
	if err != nil {
		return AdminState{}, err
	}
	return mapAdminRow(row.Enabled, row.PopupAnnouncementID, row.Title, row.Message), nil
}

func (s *Service) PublicGet(ctx context.Context) (State, error) {
	if s == nil || s.q == nil {
		return State{}, ErrUnavailable
	}
	row, err := s.q.GetMaintenanceAdmin(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return State{}, nil
		}
		return State{}, err
	}
	out := State{
		Enabled:             row.Enabled,
		PopupAnnouncementID: textVal(row.PopupAnnouncementID),
		Title:               row.Title,
		Message:             row.Message,
	}
	popupID := textVal(row.PopupAnnouncementID)
	if popupID == "" {
		return out, nil
	}
	ann, err := s.q.GetPublishedAnnouncementBrief(ctx, popupID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return State{}, err
		}
		return out, nil
	}
	out.PopupAnnouncement = &PopupAnnouncement{
		ID: ann.ID, Title: ann.Title, BodyHTML: ann.BodyHtml,
	}
	return out, nil
}

func mapAdminRow(enabled bool, popupID pgtype.Text, title, message string) AdminState {
	return AdminState{
		Enabled:             enabled,
		PopupAnnouncementID: textVal(popupID),
		Title:               title,
		Message:             message,
	}
}

func textVal(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func textParam(raw string) pgtype.Text {
	s := strings.TrimSpace(raw)
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
