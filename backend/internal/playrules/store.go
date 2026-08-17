package playrules

import (
	"context"
	"fmt"
	"sync/atomic"

	"caipiao/backend/internal/db/sqlcdb"
)

// PublishedSpecReader is deliberately the small query surface needed by the
// hot-path cache. sqlcdb.Queries implements it directly.
type PublishedSpecReader interface {
	ListEnabledPlayRuleSpecs(context.Context) ([]sqlcdb.PlayRuleSpec, error)
}

// RegistryStore atomically swaps complete registries. A failed reload leaves
// the last verified snapshot available to running strategy workers.
type RegistryStore struct {
	registry atomic.Pointer[Registry]
}

func NewRegistryStore(initial *Registry) *RegistryStore {
	if initial == nil {
		initial, _ = NewRegistry(nil)
	}
	store := &RegistryStore{}
	store.registry.Store(initial)
	return store
}

func (s *RegistryStore) Resolve(locator Locator) (Snapshot, error) {
	if s == nil {
		return Snapshot{}, ErrRuleUnavailable
	}
	registry := s.registry.Load()
	if registry == nil {
		return Snapshot{}, ErrRuleUnavailable
	}
	return registry.Resolve(locator)
}

// Reload validates every enabled database rule before publishing it to the
// in-memory cache. It never swaps a partially valid registry.
func (s *RegistryStore) Reload(ctx context.Context, reader PublishedSpecReader) error {
	if s == nil {
		return fmt.Errorf("play rule registry store is nil")
	}
	if reader == nil {
		return fmt.Errorf("play rule reader is nil")
	}
	rows, err := reader.ListEnabledPlayRuleSpecs(ctx)
	if err != nil {
		return fmt.Errorf("list enabled play rules: %w", err)
	}
	published := make([]PublishedSpec, 0, len(rows))
	for _, row := range rows {
		lotteryCode := ""
		if row.LotteryCode.Valid {
			lotteryCode = row.LotteryCode.String
		}
		published = append(published, PublishedSpec{
			Locator: Locator{
				TemplateCode: row.TemplateCode,
				TypeID:       row.TypeID,
				SubID:        row.SubID,
				LotteryCode:  lotteryCode,
			},
			RuleVersion:      int(row.RuleVersion),
			EvaluatorVersion: int(row.EvaluatorVersion),
			EvaluatorKey:     row.EvaluatorKey,
			EvaluationSpec:   append([]byte(nil), row.EvaluationSpec...),
			StrategyEnabled:  row.StrategyEnabled,
		})
	}
	registry, err := NewRegistry(published)
	if err != nil {
		return fmt.Errorf("validate enabled play rules: %w", err)
	}
	s.registry.Store(registry)
	return nil
}
