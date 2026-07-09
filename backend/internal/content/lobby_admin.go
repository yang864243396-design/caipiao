package content

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"caipiao/backend/internal/db/sqlcdb"
)

var ErrDuplicateSlotKey = errors.New("slotKey already exists")

type AdminLobbySlot struct {
	ID      string `json:"id"`
	SlotKey string `json:"slotKey"`
	Title   string `json:"title"`
	Brief   string `json:"brief"`
	Sort    int    `json:"sort"`
	Enabled bool   `json:"enabled"`
}

type PublicLobbySlot struct {
	SlotKey string `json:"slotKey"`
	Title   string `json:"title"`
	Brief   string `json:"brief"`
	Sort    int    `json:"sort"`
}

func (s *Service) AdminListLobbySlots(ctx context.Context) ([]AdminLobbySlot, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListLobbySlotsAdmin(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminLobbySlot, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapLobbySlotRow(row.ID, row.SlotKey, row.Title, row.Brief, row.Sort, row.Enabled))
	}
	return out, nil
}

func (s *Service) AdminSaveLobbySlot(ctx context.Context, in AdminLobbySlot) (AdminLobbySlot, error) {
	if s == nil || s.q == nil {
		return AdminLobbySlot{}, ErrUnavailable
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = fmt.Sprintf("L_%d", time.Now().UnixNano())
	}
	slotKey := strings.TrimSpace(in.SlotKey)
	title := strings.TrimSpace(in.Title)
	if slotKey == "" || title == "" {
		return AdminLobbySlot{}, ErrInvalid
	}
	sort := in.Sort
	if sort < 0 {
		sort = 0
	}
	if sort > 999 {
		sort = 999
	}

	row, err := s.q.UpsertLobbySlotAdmin(ctx, sqlcdb.UpsertLobbySlotAdminParams{
		ID: id, SlotKey: slotKey, Title: title, Brief: in.Brief,
		Sort: int32(sort), Enabled: in.Enabled,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return AdminLobbySlot{}, ErrDuplicateSlotKey
		}
		return AdminLobbySlot{}, err
	}
	return mapLobbySlotRow(row.ID, row.SlotKey, row.Title, row.Brief, row.Sort, row.Enabled), nil
}

func (s *Service) PublicLobbySlots(ctx context.Context) ([]PublicLobbySlot, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListLobbySlotsPublic(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PublicLobbySlot, 0, len(rows))
	for _, row := range rows {
		out = append(out, PublicLobbySlot{
			SlotKey: row.SlotKey,
			Title:   row.Title,
			Brief:   row.Brief,
			Sort:    int(row.Sort),
		})
	}
	return out, nil
}

func mapLobbySlotRow(id, slotKey, title, brief string, sort int32, enabled bool) AdminLobbySlot {
	return AdminLobbySlot{
		ID: id, SlotKey: slotKey, Title: title, Brief: brief,
		Sort: int(sort), Enabled: enabled,
	}
}
