package schemes

import (
	"testing"

	"caipiao/backend/internal/playrules"
)

func TestCompileCatalogRuleBuildsExecutableSSCDirectSpec(t *testing.T) {
	compiled, err := CompileCatalogRule(CatalogRuleCompileInput{
		Locator: playrules.Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"},
		BetMode: "fushi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.EvaluatorKey != "ssc.direct" {
		t.Fatalf("evaluator = %q, want ssc.direct", compiled.EvaluatorKey)
	}
	if compiled.Spec.Mode != "fushi" || compiled.Spec.BetMode != "fushi" || compiled.Spec.SegmentStart != 0 || compiled.Spec.SegmentLen != 3 {
		t.Fatalf("spec = %+v, want three-position direct fushi spec", compiled.Spec)
	}
	if compiled.Spec.NumberMin != 0 || compiled.Spec.NumberMax != 9 || compiled.Spec.CatalogSubID != "1" {
		t.Fatalf("spec = %+v, want SSC number range and catalog sub id", compiled.Spec)
	}
}

func TestCompileCatalogRuleBuildsExecutableLHCGuoguanSpec(t *testing.T) {
	compiled, err := CompileCatalogRule(CatalogRuleCompileInput{
		Locator: playrules.Locator{TemplateCode: "lhc_std", TypeID: "guoguan", SubID: "guoguan"},
		BetMode: "guoguan",
	})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.EvaluatorKey != "lhc.guoguan" || compiled.Spec.BetMode != "guoguan" {
		t.Fatalf("compiled = %+v, want lhc guoguan", compiled)
	}
	if compiled.Spec.NumberMin != 1 || compiled.Spec.NumberMax != 49 {
		t.Fatalf("spec = %+v, want LHC number range", compiled.Spec)
	}
}

func TestCompileCatalogRuleUsesPlayMethodWhenCatalogBetModeIsEmpty(t *testing.T) {
	compiled, err := CompileCatalogRule(CatalogRuleCompileInput{
		Locator:    playrules.Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"},
		PlayMethod: "前三直选复式",
	})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Spec.BetMode != "fushi" {
		t.Fatalf("bet mode = %q, want fushi inferred from play method", compiled.Spec.BetMode)
	}
}

func TestEvaluateCatalogRuleUsesCompiledSpec(t *testing.T) {
	evaluation, err := EvaluateCatalogRule(CatalogRuleCompileInput{
		Locator: playrules.Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"},
		BetMode: "fushi",
	}, []string{"1", "2", "3", "4", "5"}, "1\n2\n3")
	if err != nil {
		t.Fatal(err)
	}
	if !evaluation.Hit || evaluation.BetUnits != 1 {
		t.Fatalf("evaluation = %+v, want one winning unit", evaluation)
	}
}
