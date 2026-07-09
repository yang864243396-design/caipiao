package member

import (
	"testing"

	"caipiao/backend/internal/timeutil"
)

func TestMapLedgerFilterType(t *testing.T) {
	if mapLedgerFilterType("bet").String != "bet_debit" {
		t.Fatalf("bet -> bet_debit")
	}
	if mapLedgerFilterType("all").Valid {
		t.Fatalf("all should be null filter")
	}
}

func TestParseDateRangeDefaultToday(t *testing.T) {
	from, to, err := timeutil.ParseDateRange("", "")
	if err != nil {
		t.Fatal(err)
	}
	if !to.After(from) {
		t.Fatalf("range invalid")
	}
}

func TestTxnTypeLabel(t *testing.T) {
	if txnTypeLabel("payout") != "派奖" {
		t.Fatal("label mismatch")
	}
}

func TestRoundMoney(t *testing.T) {
	if roundMoney(12888.666) != 12888.67 {
		t.Fatalf("got %v", roundMoney(12888.666))
	}
}
