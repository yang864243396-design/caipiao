package schemes

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestPreviewBettingExecutions_fixedRotate(t *testing.T) {
	raw := []byte(`{
		"runTypeId":"fixed_rotate",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"playMethod":"定位胆万位",
		"schemeGroups":["1,3,7","2,4,6"]
	}`)
	draws := []sqlcdb.ListLotteryDrawsRow{
		{IssueNo: "100", PeriodShort: "100", Balls: []byte(`["1","2","3","4","5"]`)},
		{IssueNo: "101", PeriodShort: "101", Balls: []byte(`["2","4","6","8","0"]`)},
	}
	rows := PreviewBettingExecutions("测试方案", "seed", "custom", raw, draws)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows got %d", len(rows))
	}
	if rows[0].Scheme != "测试方案" {
		t.Fatalf("scheme=%q", rows[0].Scheme)
	}
	if rows[0].Numbers == "—" || rows[0].Numbers == "" {
		t.Fatalf("numbers empty")
	}
}

func TestPreviewBettingExecutions_longThirdPartyPeriod(t *testing.T) {
	raw := []byte(`{
		"runTypeId":"fixed_rotate",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"schemeGroups":["1,3,7"]
	}`)
	draws := []sqlcdb.ListLotteryDrawsRow{
		{
			IssueNo:     "105202606091971",
			PeriodShort: "105202606091971",
			Balls:       []byte(`["1","2","3","4","5"]`),
		},
		{
			IssueNo:     "105202606091972",
			PeriodShort: "105202606091972",
			Balls:       []byte(`["2","4","6","8","0"]`),
		},
	}
	rows := PreviewBettingExecutions("测试", "seed", "custom", raw, draws)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows got %d", len(rows))
	}
	if rows[0].Period != "972" {
		t.Fatalf("period=%q want 972", rows[0].Period)
	}
	if rows[0].Time != "972-973" {
		t.Fatalf("time=%q want 972-973", rows[0].Time)
	}
	if rows[1].Period != "971" {
		t.Fatalf("period=%q want 971", rows[1].Period)
	}
}

func TestThirdPartyPeriodDisplay(t *testing.T) {
	if got := ThirdPartyPeriodDisplay("105202606091971"); got != "202606091971" {
		t.Fatalf("got %q", got)
	}
	if got := ThirdPartyPeriodDisplay("20231103032"); got != "31103032" {
		t.Fatalf("got %q", got)
	}
	if got := ThirdPartyPeriodDisplay("971"); got != "971" {
		t.Fatalf("got %q", got)
	}
}

func TestThirdPartyPeriodShort(t *testing.T) {
	if got := thirdPartyPeriodShort("20231103032"); got != "032" {
		t.Fatalf("got %q", got)
	}
	if got := thirdPartyPeriodShort("971"); got != "971" {
		t.Fatalf("got %q", got)
	}
}

func TestPreviewBettingExecutions_generatesContentWhenMissing(t *testing.T) {
	raw := []byte(`{"playTypeId":"dingwei","subPlayId":"dingwei_wan","playMethod":"定位胆万位"}`)
	draws := []sqlcdb.ListLotteryDrawsRow{
		{IssueNo: "200", PeriodShort: "200", Balls: []byte(`["3","1","9","2","5"]`)},
	}
	rows := PreviewBettingExecutions("太乙后二", "snap-1", "custom", raw, draws)
	if len(rows) != 1 {
		t.Fatalf("want 1 row got %d", len(rows))
	}
	if rows[0].Numbers == "—" {
		t.Fatal("expected generated numbers")
	}
}
