package content

import (
	"context"
	"strings"
	"unicode/utf8"

	"caipiao/backend/internal/db/sqlcdb"
)

type CustomerServiceAgent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TgLink    string `json:"tgLink"`
	WorkHours string `json:"workHours"`
	Sort      int    `json:"sort"`
	Enabled   bool   `json:"enabled"`
}

func (s *Service) ListCustomerServiceAgents(ctx context.Context) ([]CustomerServiceAgent, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListCustomerServiceAgents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CustomerServiceAgent, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCustomerServiceAgent(row.ID, row.Name, row.TgLink, row.WorkHours, int(row.Sort), row.Enabled))
	}
	return out, nil
}

func (s *Service) ListEnabledCustomerServiceAgents(ctx context.Context) ([]CustomerServiceAgent, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListEnabledCustomerServiceAgents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CustomerServiceAgent, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCustomerServiceAgent(row.ID, row.Name, row.TgLink, row.WorkHours, int(row.Sort), true))
	}
	return out, nil
}

func (s *Service) SaveCustomerServiceAgent(ctx context.Context, in CustomerServiceAgent) (CustomerServiceAgent, error) {
	if s == nil || s.q == nil {
		return CustomerServiceAgent{}, ErrUnavailable
	}
	id := strings.TrimSpace(in.ID)
	name := strings.TrimSpace(in.Name)
	tgLink := strings.TrimSpace(in.TgLink)
	workHours := strings.TrimSpace(in.WorkHours)
	if id == "" {
		return CustomerServiceAgent{}, ErrInvalid
	}
	if name == "" || utf8.RuneCountInString(name) > 128 {
		return CustomerServiceAgent{}, ErrInvalid
	}
	if tgLink == "" || utf8.RuneCountInString(tgLink) > 512 {
		return CustomerServiceAgent{}, ErrInvalid
	}
	if utf8.RuneCountInString(workHours) > 256 {
		return CustomerServiceAgent{}, ErrInvalid
	}
	row, err := s.q.UpsertCustomerServiceAgent(ctx, sqlcdb.UpsertCustomerServiceAgentParams{
		ID:        id,
		Name:      name,
		TgLink:    tgLink,
		WorkHours: workHours,
		Sort:      int32(in.Sort),
		Enabled:   in.Enabled,
	})
	if err != nil {
		return CustomerServiceAgent{}, err
	}
	return mapCustomerServiceAgent(row.ID, row.Name, row.TgLink, row.WorkHours, int(row.Sort), row.Enabled), nil
}

func (s *Service) DeleteCustomerServiceAgent(ctx context.Context, id string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}
	return s.q.DeleteCustomerServiceAgent(ctx, id)
}

func mapCustomerServiceAgent(id, name, tgLink, workHours string, sort int, enabled bool) CustomerServiceAgent {
	return CustomerServiceAgent{
		ID:        id,
		Name:      name,
		TgLink:    tgLink,
		WorkHours: workHours,
		Sort:      sort,
		Enabled:   enabled,
	}
}
