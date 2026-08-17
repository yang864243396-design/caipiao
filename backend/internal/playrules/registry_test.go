package playrules

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
	"github.com/jackc/pgx/v5/pgtype"
)

func testSpec(t *testing.T, mode string) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(EvaluationSpec{Mode: mode, NumberMin: 0, NumberMax: 9})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRegistryReturnsLotteryOverrideBeforeDefault(t *testing.T) {
	registry, err := NewRegistry([]PublishedSpec{
		{Locator: Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"}, RuleVersion: 1, EvaluatorVersion: 1, EvaluatorKey: "ssc.direct", EvaluationSpec: testSpec(t, "direct"), StrategyEnabled: true},
		{Locator: Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1", LotteryCode: "tron_ffc_3s"}, RuleVersion: 2, EvaluatorVersion: 1, EvaluatorKey: "ssc.direct", EvaluationSpec: testSpec(t, "direct"), StrategyEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := registry.Resolve(Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1", LotteryCode: "tron_ffc_3s"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.RuleVersion != 2 {
		t.Fatalf("rule version = %d, want lottery override version 2", snapshot.RuleVersion)
	}
}

func TestRegistryRejectsDisabledAndUnknownEvaluatorRules(t *testing.T) {
	_, err := NewRegistry([]PublishedSpec{{
		Locator: Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"}, RuleVersion: 1, EvaluatorVersion: 1,
		EvaluatorKey: "unknown.script", EvaluationSpec: testSpec(t, "direct"), StrategyEnabled: true,
	}})
	if err == nil {
		t.Fatal("unknown evaluator must be rejected")
	}
	_, err = NewRegistry([]PublishedSpec{{
		Locator: Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"}, RuleVersion: 1, EvaluatorVersion: 1,
		EvaluatorKey: "ssc.direct", EvaluationSpec: json.RawMessage(`{"mode":"direct","numberMin":0,"numberMax":9,"unsafe":"ignored"}`), StrategyEnabled: true,
	}})
	if err == nil {
		t.Fatal("unknown evaluation spec field must be rejected")
	}

	registry, err := NewRegistry([]PublishedSpec{{
		Locator: Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "2"}, RuleVersion: 1, EvaluatorVersion: 1,
		EvaluatorKey: "ssc.direct", EvaluationSpec: testSpec(t, "direct"), StrategyEnabled: false,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve(Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "2"}); err != ErrRuleUnavailable {
		t.Fatalf("disabled rule error = %v, want %v", err, ErrRuleUnavailable)
	}
}

func TestSnapshotHashChangesWithRuleVersion(t *testing.T) {
	base := PublishedSpec{Locator: Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"}, EvaluatorVersion: 1, EvaluatorKey: "ssc.direct", EvaluationSpec: testSpec(t, "direct"), StrategyEnabled: true}
	base.RuleVersion = 1
	first, err := NewRegistry([]PublishedSpec{base})
	if err != nil {
		t.Fatal(err)
	}
	base.RuleVersion = 2
	second, err := NewRegistry([]PublishedSpec{base})
	if err != nil {
		t.Fatal(err)
	}
	one, _ := first.Resolve(base.Locator)
	two, _ := second.Resolve(base.Locator)
	if one.ContentHash == two.ContentHash {
		t.Fatal("rule version change must change snapshot hash")
	}
}

type fakePublishedSpecReader struct {
	rows []sqlcdb.PlayRuleSpec
	err  error
}

func (r fakePublishedSpecReader) ListEnabledPlayRuleSpecs(context.Context) ([]sqlcdb.PlayRuleSpec, error) {
	return r.rows, r.err
}

func TestRegistryStoreKeepsLastKnownGoodSnapshotWhenReloadFails(t *testing.T) {
	initial, err := NewRegistry([]PublishedSpec{{
		Locator: Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"}, RuleVersion: 1, EvaluatorVersion: 1,
		EvaluatorKey: "ssc.direct", EvaluationSpec: testSpec(t, "direct"), StrategyEnabled: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	store := NewRegistryStore(initial)
	reader := fakePublishedSpecReader{rows: []sqlcdb.PlayRuleSpec{{
		TemplateCode: "ssc_std", TypeID: "g001", SubID: "1", RuleVersion: 2, EvaluatorVersion: 1,
		EvaluatorKey: "ssc.direct", EvaluationSpec: testSpec(t, "direct"), StrategyEnabled: true,
		LotteryCode: pgtype.Text{},
	}}}
	if err := store.Reload(context.Background(), reader); err != nil {
		t.Fatal(err)
	}
	reader.err = errors.New("database unavailable")
	if err := store.Reload(context.Background(), reader); err == nil {
		t.Fatal("reload must report database error")
	}

	snapshot, err := store.Resolve(Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.RuleVersion != 2 {
		t.Fatalf("cached rule version = %d, want last successful version 2", snapshot.RuleVersion)
	}
}
