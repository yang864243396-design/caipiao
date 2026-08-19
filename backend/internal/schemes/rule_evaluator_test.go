package schemes

import (
	"encoding/json"
	"errors"
	"testing"

	"caipiao/backend/internal/playrules"
)

func frozenSnapshot(t *testing.T, template, evaluator string, spec map[string]any) playrules.Snapshot {
	t.Helper()
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	return playrules.Snapshot{
		Locator:      playrules.Locator{TemplateCode: template, TypeID: "g001", SubID: "1"},
		EvaluatorKey: evaluator, EvaluationSpec: raw,
	}
}

func TestFrozenSSCDirectRuleMatchesKnownSample(t *testing.T) {
	snapshot := frozenSnapshot(t, "ssc_std", "ssc.direct", map[string]any{
		"mode": "direct", "numberMin": 0, "numberMax": 9,
		"segmentStart": 0, "segmentLen": 3, "betMode": "zhixuan_fs",
	})
	hit, err := evaluateFrozenRule(snapshot, playRule{}, []string{"1", "2", "3", "4", "5"}, "1\n2\n3", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !hit.Hit || hit.BetUnits != 1 {
		t.Fatalf("hit evaluation = %+v, want one winning unit", hit)
	}
	miss, err := evaluateFrozenRule(snapshot, playRule{}, []string{"1", "2", "3", "4", "5"}, "1\n2\n4", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if miss.Hit {
		t.Fatalf("miss evaluation = %+v, want miss", miss)
	}
}

func TestFrozenFastSSCHashTailBigSmallUsesFinalDigit(t *testing.T) {
	snapshot := playrules.Snapshot{
		Locator:        playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g017", SubID: "390"},
		EvaluatorKey:   "ssc.attribute",
		EvaluationSpec: []byte(`{"mode":"attribute","numberMin":0,"numberMax":9,"segmentStart":0,"segmentLen":5,"betMode":"daxiao","catalogSubId":"390"}`),
	}
	small, err := evaluateFrozenRule(snapshot, playRule{}, []string{"5", "5", "6", "5", "3"}, "小", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !small.Hit || small.BetUnits != 1 {
		t.Fatalf("small evaluation = %+v, want one hit from final digit 3", small)
	}
	large, err := evaluateFrozenRule(snapshot, playRule{}, []string{"5", "5", "6", "5", "3"}, "大", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if large.Hit {
		t.Fatalf("large evaluation = %+v, want miss from final digit 3", large)
	}
}

func TestFrozenFastSSCHashTailOddEvenUsesFinalDigit(t *testing.T) {
	snapshot := playrules.Snapshot{
		Locator:        playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g017", SubID: "387"},
		EvaluatorKey:   "ssc.attribute",
		EvaluationSpec: []byte(`{"mode":"attribute","numberMin":0,"numberMax":9,"segmentStart":0,"segmentLen":5,"betMode":"danshuang","catalogSubId":"387"}`),
	}
	odd, err := evaluateFrozenRule(snapshot, playRule{}, []string{"5", "5", "6", "5", "3"}, "单", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !odd.Hit || odd.BetUnits != 1 {
		t.Fatalf("odd evaluation = %+v, want one hit from final digit 3", odd)
	}
	even, err := evaluateFrozenRule(snapshot, playRule{}, []string{"5", "5", "6", "5", "3"}, "双", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if even.Hit {
		t.Fatalf("even evaluation = %+v, want miss from final digit 3", even)
	}
}

func TestFrozenLHCGuoguanRulePreservesEmptyPositions(t *testing.T) {
	snapshot := frozenSnapshot(t, "lhc_std", "lhc.guoguan", map[string]any{
		"mode": "guoguan", "numberMin": 1, "numberMax": 49, "betMode": "guoguan",
	})
	evaluation, err := evaluateFrozenRule(snapshot, playRule{}, []string{"30", "11", "21", "28", "05", "42", "49"}, "大,单,,大,,双", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !evaluation.Hit || evaluation.BetUnits != 1 {
		t.Fatalf("evaluation = %+v, want hit while preserving empty positions", evaluation)
	}
}

func TestUnknownFrozenEvaluatorNeverFallsBackToLabelHeuristics(t *testing.T) {
	snapshot := frozenSnapshot(t, "ssc_std", "unknown.script", map[string]any{
		"mode": "direct", "numberMin": 0, "numberMax": 9,
	})
	_, err := evaluateFrozenRule(snapshot, playRule{PlayTemplate: "ssc_std", BetMode: "zhixuan_fs", SegmentLen: 3}, []string{"1", "2", "3"}, "1\n2\n3", false, "")
	if !errors.Is(err, ErrStrategyRuleUnavailable) {
		t.Fatalf("error = %v, want %v", err, ErrStrategyRuleUnavailable)
	}
}

func TestWorkerResolvesPublishedRuleForLotterySpecificScheme(t *testing.T) {
	spec := []byte(`{"mode":"direct","numberMin":0,"numberMax":9,"segmentStart":0,"segmentLen":3,"betMode":"zhixuan_fs"}`)
	registry, err := playrules.NewRegistry([]playrules.PublishedSpec{
		{Locator: playrules.Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"}, RuleVersion: 1, EvaluatorVersion: 1, EvaluatorKey: "ssc.direct", EvaluationSpec: spec, StrategyEnabled: true},
		{Locator: playrules.Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1", LotteryCode: "tron_ffc_3s"}, RuleVersion: 2, EvaluatorVersion: 1, EvaluatorKey: "ssc.direct", EvaluationSpec: spec, StrategyEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	worker := &Worker{}
	worker.SetRuleRegistry(playrules.NewRegistryStore(registry))

	snapshot, ok := worker.resolvePublishedRule("tron_ffc_3s", playRule{PlayTemplate: "ssc_std", PlayTypeID: "g001", SubPlayID: "1"})
	if !ok || snapshot.RuleVersion != 2 {
		t.Fatalf("snapshot = %+v, ok=%v; want lottery override version 2", snapshot, ok)
	}
}

func TestWorkerResolvesPublishedRuleByCatalogSubIDBeforeSemanticMode(t *testing.T) {
	registry, err := playrules.NewRegistry([]playrules.PublishedSpec{
		{Locator: playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g017", SubID: "390"}, RuleVersion: 7, EvaluatorVersion: 1, EvaluatorKey: "ssc.attribute", EvaluationSpec: []byte(`{"mode":"attribute","catalogSubId":"390"}`), StrategyEnabled: true},
		{Locator: playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g017", SubID: "daxiao"}, RuleVersion: 3, EvaluatorVersion: 1, EvaluatorKey: "ssc.attribute", EvaluationSpec: []byte(`{"mode":"attribute","catalogSubId":"legacy"}`), StrategyEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	worker := &Worker{ruleRegistry: playrules.NewRegistryStore(registry)}
	got, ok := worker.resolvePublishedRule("tron_ffc_3s", playRule{PlayTemplate: "fast_ssc_std", PlayTypeID: "g017", CatalogSubID: "390", SubPlayID: "daxiao"})
	if !ok || got.Locator.SubID != "390" || got.RuleVersion != 7 {
		t.Fatalf("snapshot=%+v ok=%v, want catalogue rule 390 version 7", got, ok)
	}
}

func TestWorkerResolvesPublishedRuleBySemanticFallbackWithoutCatalogID(t *testing.T) {
	registry, err := playrules.NewRegistry([]playrules.PublishedSpec{
		{Locator: playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g017", SubID: "daxiao"}, RuleVersion: 3, EvaluatorVersion: 1, EvaluatorKey: "ssc.attribute", EvaluationSpec: []byte(`{"mode":"attribute","catalogSubId":"390"}`), StrategyEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	worker := &Worker{ruleRegistry: playrules.NewRegistryStore(registry)}
	got, ok := worker.resolvePublishedRule("tron_ffc_3s", playRule{PlayTemplate: "fast_ssc_std", PlayTypeID: "g017", SubPlayID: "daxiao"})
	if !ok || got.Locator.SubID != "daxiao" || got.RuleVersion != 3 {
		t.Fatalf("snapshot=%+v ok=%v, want semantic fallback daxiao version 3", got, ok)
	}
}

func TestSimSettlementUsesFrozenPublishedRuleWhenAvailable(t *testing.T) {
	snapshot := frozenSnapshot(t, "ssc_std", "ssc.direct", map[string]any{
		"mode": "direct", "numberMin": 0, "numberMax": 9,
		"segmentStart": 0, "segmentLen": 3, "betMode": "zhixuan_fs",
	})
	settlement, ok, err := decideSimSettlementWithSnapshot(
		"custom", []byte(`{}`), "tron_ffc_3s", "1\n2\n3", 0,
		[]string{"1", "2", "3", "4", "5"}, 1, &snapshot,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !settlement.Hit || settlement.Status != "hit" {
		t.Fatalf("settlement = %+v, ok=%v; want frozen-rule hit", settlement, ok)
	}
}
