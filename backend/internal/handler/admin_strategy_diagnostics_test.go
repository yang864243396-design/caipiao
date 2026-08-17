package handler

import "testing"

func boolPtr(v bool) *bool { return &v }

func TestStrategyReconciliationStatusUsesHitDifferenceNotPnl(t *testing.T) {
	if got := strategyReconciliationStatus("completed", boolPtr(true), boolPtr(false)); got != "mismatch" {
		t.Fatalf("status=%q, want mismatch when provider and frozen-rule hits differ", got)
	}
	if got := strategyReconciliationStatus("completed", boolPtr(true), boolPtr(true)); got != "completed" {
		t.Fatalf("status=%q, want completed when hits match", got)
	}
}

func TestStrategyPipelineStatusExposesMissingRuleSnapshot(t *testing.T) {
	if got := strategyPipelineStatus(true, false, ""); got != "missing_rule_snapshot" {
		t.Fatalf("status=%q, want missing_rule_snapshot", got)
	}
	if got := strategyPipelineStatus(false, false, ""); got != "awaiting_draw" {
		t.Fatalf("status=%q, want awaiting_draw", got)
	}
	if got := strategyPipelineStatus(true, true, ""); got != "awaiting_evaluation" {
		t.Fatalf("status=%q, want awaiting_evaluation", got)
	}
}
