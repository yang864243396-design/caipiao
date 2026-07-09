package lookback

import "testing"

func TestAppliesTo(t *testing.T) {
	j := Settings{Judgment: JudgmentIndividual, ApplyFormal: true}
	if !AppliesTo(j, false) || AppliesTo(j, true) {
		t.Fatal("applyFormal only")
	}
	j = Settings{Judgment: JudgmentIndividual, ApplySim: true}
	if AppliesTo(j, false) || !AppliesTo(j, true) {
		t.Fatal("applySim only")
	}
	j = Settings{Judgment: JudgmentIndividual, ApplyFormal: true, ApplySim: true}
	if !AppliesTo(j, false) || !AppliesTo(j, true) {
		t.Fatal("both channels")
	}
	if AppliesTo(Settings{Judgment: JudgmentIndividual}, false) {
		t.Fatal("none selected")
	}
	if AppliesTo(Settings{ApplyFormal: true}, false) {
		t.Fatal("judgment none")
	}
}
