package main

import "testing"

func TestParseOperationRequiresExplicitCutoverConfirmation(t *testing.T) {
	if _, err := parseOperation([]string{"cutover"}); err == nil {
		t.Fatal("cutover without confirmation must fail")
	}
	op, err := parseOperation([]string{"cutover", "--confirm-cutover"})
	if err != nil {
		t.Fatal(err)
	}
	if op.name != "cutover" {
		t.Fatalf("operation = %q", op.name)
	}
}

func TestParseOperationRequiresExplicitRollbackConfirmation(t *testing.T) {
	if _, err := parseOperation([]string{"rollback"}); err == nil {
		t.Fatal("rollback without confirmation must fail")
	}
	if _, err := parseOperation([]string{"rollback", "--confirm-rollback"}); err != nil {
		t.Fatal(err)
	}
}

func TestParseOperationValidatesBatchSize(t *testing.T) {
	op, err := parseOperation([]string{"backfill", "--batch", "7500"})
	if err != nil {
		t.Fatal(err)
	}
	if op.batch != 7500 {
		t.Fatalf("batch = %d", op.batch)
	}
	for _, args := range [][]string{
		{"backfill", "--batch", "0"},
		{"backfill", "--batch", "100001"},
		{"unknown"},
	} {
		if _, err := parseOperation(args); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
}
