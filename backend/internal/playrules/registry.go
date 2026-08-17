package playrules

import "fmt"

type registryKey struct {
	templateCode string
	typeID       string
	subID        string
	lotteryCode  string
}

func keyFor(locator Locator) registryKey {
	locator = normalizeLocator(locator)
	return registryKey{locator.TemplateCode, locator.TypeID, locator.SubID, locator.LotteryCode}
}

// Registry is immutable after construction so the worker can resolve a rule
// without a database lookup or mutex on every strategy tick.
type Registry struct {
	rules map[registryKey]Snapshot
}

func NewRegistry(rows []PublishedSpec) (*Registry, error) {
	rules := make(map[registryKey]Snapshot, len(rows))
	for _, row := range rows {
		snapshot, err := snapshotFromPublished(row)
		if err != nil {
			return nil, err
		}
		key := keyFor(snapshot.Locator)
		if _, exists := rules[key]; exists {
			return nil, fmt.Errorf("duplicate published rule %s/%s/%s for lottery %q", key.templateCode, key.typeID, key.subID, key.lotteryCode)
		}
		rules[key] = snapshot
	}
	return &Registry{rules: rules}, nil
}

// Resolve selects an explicit lottery override first, then the catalogue
// default. A disabled override intentionally blocks fallback for that lottery.
func (r *Registry) Resolve(locator Locator) (Snapshot, error) {
	if r == nil {
		return Snapshot{}, ErrRuleUnavailable
	}
	exact := keyFor(locator)
	if exact.lotteryCode != "" {
		if snapshot, ok := r.rules[exact]; ok {
			if !snapshot.StrategyEnabled {
				return Snapshot{}, ErrRuleUnavailable
			}
			return snapshot, nil
		}
	}
	defaultKey := exact
	defaultKey.lotteryCode = ""
	snapshot, ok := r.rules[defaultKey]
	if !ok || !snapshot.StrategyEnabled {
		return Snapshot{}, ErrRuleUnavailable
	}
	return snapshot, nil
}
