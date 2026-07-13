package schemes

import (
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestLookupSubPlayFromRows_g006Dingwei(t *testing.T) {
	rows := []sqlcdb.GetSubPlayRow{
		guajiSubPlayRow("g006", "13", "定位胆", 1),
		guajiSubPlayRow("g006", "14", "直选复式", 2),
	}
	got, err := lookupSubPlayFromRows("ssc_std", rows, "g006", "dingwei", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.SubID != "13" {
		t.Fatalf("sub_id = %q want 13", got.SubID)
	}
}

func TestLookupSubPlayFromRows_legacyDingweiWan(t *testing.T) {
	rows := []sqlcdb.GetSubPlayRow{
		legacyDingweiRow("dingwei_wan", "一星定位胆 · 万位", 1),
		legacyDingweiRow("dingwei_ge", "一星定位胆 · 个位", 5),
	}
	got, err := lookupSubPlayFromRows("ssc_std", rows, "dingwei", "dingwei", "dingwei", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.SubID != "dingwei_wan" {
		t.Fatalf("sub_id = %q want dingwei_wan", got.SubID)
	}
}

func TestLookupSubPlayFromRows_g006DingweiByPosition(t *testing.T) {
	rows := []sqlcdb.GetSubPlayRow{
		guajiSubPlayRow("g006", "13", "定位胆 · 万位", 1),
		guajiSubPlayRow("g006", "14", "定位胆 · 个位", 5),
	}
	got, err := lookupSubPlayFromRows("ssc_std", rows, "g006", "dingwei", "", 4)
	if err != nil {
		t.Fatal(err)
	}
	if got.SubID != "14" {
		t.Fatalf("sub_id = %q want 14", got.SubID)
	}
}

func TestLookupSubPlayFromRows_zhixuanFsVsZuxuan(t *testing.T) {
	rows := []sqlcdb.GetSubPlayRow{
		guajiSubPlayRow("g008", "119", "前后二组选复式", 1),
		guajiSubPlayRow("g008", "120", "前后二直选复式", 2),
	}
	got, err := lookupSubPlayFromRows("ssc_std", rows, "g008", "zhixuan_fs", "fushi", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.SubID != "120" {
		t.Fatalf("sub_id = %q want 120 (直选复式)", got.SubID)
	}
	gotZu, err := lookupSubPlayFromRows("ssc_std", rows, "g008", "zuxuan_fs", "zuxuan_fs", 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotZu.SubID != "119" {
		t.Fatalf("sub_id = %q want 119 (组选复式)", gotZu.SubID)
	}
}

func TestLookupSubPlayFromRows_zhixuanDs(t *testing.T) {
	rows := []sqlcdb.GetSubPlayRow{
		guajiSubPlayRow("g001", "1", "前三直选复式", 1),
		guajiSubPlayRow("g001", "2", "前三直选单式", 2),
		guajiSubPlayRow("g001", "5", "前三组选复式", 3),
	}
	got, err := lookupSubPlayFromRows("ssc_std", rows, "g001", "zhixuan_ds", "danshi", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.SubID != "2" {
		t.Fatalf("sub_id = %q want 2", got.SubID)
	}
}

func guajiSubPlayRow(typeID, subID, label string, sort int32) sqlcdb.GetSubPlayRow {
	seg, _ := json.Marshal(map[string]string{
		"guajiGroup":  "一星",
		"guajiRuleId": subID,
	})
	return sqlcdb.GetSubPlayRow{
		TemplateCode:     "ssc_std",
		TypeID:           typeID,
		SubID:            subID,
		Label:            label,
		SortOrder:        sort,
		SegmentRule:      seg,
		OutboundPlayCode: testPgText(subID),
		Enabled:          true,
	}
}

func legacyDingweiRow(subID, label string, sort int32) sqlcdb.GetSubPlayRow {
	return sqlcdb.GetSubPlayRow{
		TemplateCode:     "ssc_std",
		TypeID:           "dingwei",
		SubID:            subID,
		Label:            label,
		SortOrder:        sort,
		BetMode:          testPgText("dingwei"),
		OutboundPlayCode: testPgText("ssc_std:dingwei:" + subID),
		Enabled:          true,
	}
}

func testPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
