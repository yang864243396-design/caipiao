package audit

import (
	"context"
	"errors"
	"strings"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

var ErrUnavailable = errors.New("audit service unavailable")

type Entry struct {
	ID     string `json:"id"`
	Time   string `json:"time"`
	Actor  string `json:"actor"`
	Action string `json:"action"`
	IP     string `json:"ip"`
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

func (s *Service) List(ctx context.Context, limit int) ([]Entry, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.q.ListAdminAuditLogs(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapEntry(row))
	}
	return out, nil
}

func (s *Service) Append(ctx context.Context, actor, action, ip string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	actor = strings.TrimSpace(actor)
	action = strings.TrimSpace(action)
	if actor == "" {
		actor = "admin"
	}
	if action == "" {
		return nil
	}
	if ip == "" {
		ip = "127.0.0.1"
	}
	_, err := s.q.InsertAdminAuditLog(ctx, sqlcdb.InsertAdminAuditLogParams{
		Actor:  actor,
		Action: action,
		Ip:     ip,
	})
	return err
}

func mapEntry(row sqlcdb.AdminAuditLog) Entry {
	return Entry{
		ID:     row.ID,
		Time:   formatAuditTime(row.CreatedAt.Time),
		Actor:  row.Actor,
		Action: row.Action,
		IP:     row.Ip,
	}
}

func formatAuditTime(ts time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return ts.In(loc).Format("2006-01-02 15:04")
}
