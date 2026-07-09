package games

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestCompareIssueNo_numeric(t *testing.T) {
	if compareIssueNo("1014017600297", "1014017600299") >= 0 {
		t.Fatal("expected 297 < 299")
	}
	if compareIssueNo("1014017600299", "1014017600299") != 0 {
		t.Fatal("expected equal")
	}
}

func TestMaxIssueNo(t *testing.T) {
	got := maxIssueNo("1014017600297", "1014017600299", "1014017600298")
	if got != "1014017600299" {
		t.Fatalf("got %q want max", got)
	}
}

func TestPrevIssueNo(t *testing.T) {
	if prevIssueNo("1014017600300") != "1014017600299" {
		t.Fatal("prev issue mismatch")
	}
}

func TestFilterDrawsBeforeOpenPeriod(t *testing.T) {
	draws := []sqlcdb.ListLotteryDrawsRow{
		{IssueNo: "1014017600299"},
		{IssueNo: "1014017600298"},
		{IssueNo: "1014017600297"},
	}
	filtered := filterDrawsBeforeOpenPeriod(draws, "1014017600299", "1014017600297")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(filtered))
	}
	if filtered[0].IssueNo != "1014017600298" || filtered[1].IssueNo != "1014017600297" {
		t.Fatalf("unexpected rows: %+v", filtered)
	}
}
