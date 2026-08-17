// Package playrules keeps the published, database-backed play-rule contract
// small enough for the hot strategy path to resolve in memory.
package playrules

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrRuleUnavailable = errors.New("play rule unavailable")

// Locator identifies a play in the existing catalogue. LotteryCode is empty
// for the catalogue default and set only for a lottery-specific override.
type Locator struct {
	TemplateCode string
	TypeID       string
	SubID        string
	LotteryCode  string
}

// EvaluationSpec is deliberately declarative. Evaluator implementations are
// selected by EvaluatorKey; no executable expression is stored in the database.
type EvaluationSpec struct {
	Mode         string `json:"mode"`
	NumberMin    int    `json:"numberMin"`
	NumberMax    int    `json:"numberMax"`
	SegmentStart int    `json:"segmentStart,omitempty"`
	SegmentLen   int    `json:"segmentLen,omitempty"`
	PositionIdx  int    `json:"positionIdx,omitempty"`
	BetMode      string `json:"betMode,omitempty"`
	CatalogSubID string `json:"catalogSubId,omitempty"`
	HezhiZuxuan  bool   `json:"hezhiZuxuan,omitempty"`
}

// PublishedSpec is the current published rule read from play_rule_specs.
type PublishedSpec struct {
	Locator          Locator
	RuleVersion      int
	EvaluatorVersion int
	EvaluatorKey     string
	EvaluationSpec   json.RawMessage
	StrategyEnabled  bool
}

// Snapshot is frozen with an accepted bet so future rule releases cannot
// change how a historical period is evaluated.
type Snapshot struct {
	Locator          Locator
	RuleVersion      int
	EvaluatorVersion int
	EvaluatorKey     string
	EvaluationSpec   json.RawMessage
	StrategyEnabled  bool
	ContentHash      string
}

var knownEvaluatorKeys = map[string]struct{}{
	"ssc.direct":    {},
	"ssc.group":     {},
	"ssc.sum":       {},
	"ssc.attribute": {},
	"lhc.guoguan":   {},
	"lhc.duipeng":   {},
	"lhc.attribute": {},
	"pk10.direct":   {},
	"syxw.renxuan":  {},
	"k3.standard":   {},
	"pc28.standard": {},
}

func normalizeLocator(locator Locator) Locator {
	return Locator{
		TemplateCode: strings.TrimSpace(locator.TemplateCode),
		TypeID:       strings.TrimSpace(locator.TypeID),
		SubID:        strings.TrimSpace(locator.SubID),
		LotteryCode:  strings.TrimSpace(locator.LotteryCode),
	}
}

func snapshotFromPublished(row PublishedSpec) (Snapshot, error) {
	locator := normalizeLocator(row.Locator)
	if locator.TemplateCode == "" || locator.TypeID == "" || locator.SubID == "" {
		return Snapshot{}, fmt.Errorf("rule locator is incomplete")
	}
	if row.RuleVersion < 1 || row.EvaluatorVersion < 1 {
		return Snapshot{}, fmt.Errorf("rule %s/%s/%s has invalid version", locator.TemplateCode, locator.TypeID, locator.SubID)
	}
	evaluatorKey := strings.TrimSpace(row.EvaluatorKey)
	if _, ok := knownEvaluatorKeys[evaluatorKey]; !ok {
		return Snapshot{}, fmt.Errorf("rule %s/%s/%s uses unsupported evaluator %q", locator.TemplateCode, locator.TypeID, locator.SubID, evaluatorKey)
	}

	var spec EvaluationSpec
	decoder := json.NewDecoder(bytes.NewReader(row.EvaluationSpec))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return Snapshot{}, fmt.Errorf("rule %s/%s/%s has invalid evaluation spec: %w", locator.TemplateCode, locator.TypeID, locator.SubID, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Snapshot{}, fmt.Errorf("rule %s/%s/%s has invalid evaluation spec trailing data", locator.TemplateCode, locator.TypeID, locator.SubID)
	}
	if strings.TrimSpace(spec.Mode) == "" || spec.NumberMin > spec.NumberMax {
		return Snapshot{}, fmt.Errorf("rule %s/%s/%s has incomplete evaluation spec", locator.TemplateCode, locator.TypeID, locator.SubID)
	}
	canonicalSpec, err := json.Marshal(spec)
	if err != nil {
		return Snapshot{}, fmt.Errorf("canonicalize rule %s/%s/%s: %w", locator.TemplateCode, locator.TypeID, locator.SubID, err)
	}

	content := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d\x00%d\x00%s\x00%s", locator.TemplateCode, locator.TypeID, locator.SubID, locator.LotteryCode, row.RuleVersion, row.EvaluatorVersion, evaluatorKey, canonicalSpec)
	hash := sha256.Sum256([]byte(content))
	return Snapshot{
		Locator:          locator,
		RuleVersion:      row.RuleVersion,
		EvaluatorVersion: row.EvaluatorVersion,
		EvaluatorKey:     evaluatorKey,
		EvaluationSpec:   append(json.RawMessage(nil), canonicalSpec...),
		StrategyEnabled:  row.StrategyEnabled,
		ContentHash:      hex.EncodeToString(hash[:]),
	}, nil
}
