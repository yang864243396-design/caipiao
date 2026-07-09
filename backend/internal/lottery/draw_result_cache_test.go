package lottery

import (
	"testing"
	"time"
)

func TestDrawResultCache(t *testing.T) {
	t.Parallel()
	code := "tron_ffc_1m_test"
	StoreDrawResult(code, "100", []string{"1", "2", "3", "4", "5"}, time.Now())
	if _, ok := DrawResultForIssue(code, "100"); !ok {
		t.Fatal("expected cached draw for issue 100")
	}
	StoreDrawResult(code, "101", []string{"9", "8", "7", "6", "5"}, time.Now())
	got, ok := DrawResultForIssue(code, "101")
	if !ok || got.IssueNo != "101" {
		t.Fatalf("want issue 101, got %+v ok=%v", got, ok)
	}
	if _, ok := DrawResultForIssue(code, "100"); ok {
		t.Fatal("stale issue should not match after newer cache")
	}
	// 旧期号不应覆盖较新缓存
	StoreDrawResult(code, "100", []string{"0", "0", "0", "0", "0"}, time.Now())
	if got, ok := DrawResultForIssue(code, "101"); !ok || got.Balls[0] != "9" {
		t.Fatalf("older issue must not overwrite newer cache: %+v", got)
	}
}
