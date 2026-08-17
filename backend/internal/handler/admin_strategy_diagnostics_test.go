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
