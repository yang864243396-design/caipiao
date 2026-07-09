package copyhall

import "testing"

func TestAdminBoardSlotsFromDB_empty(t *testing.T) {
	slots := adminBoardSlotsFromDB(nil)
	if len(slots) != 10 {
		t.Fatalf("len=%d want 10", len(slots))
	}
	for i, slot := range slots {
		if slot.Rank != i+1 {
			t.Fatalf("rank[%d]=%d", i, slot.Rank)
		}
		if slot.SchemeID != "" {
			t.Fatalf("slot[%d] should be empty, got schemeId=%q", i, slot.SchemeID)
		}
	}
}

func TestValidateAdminSaveSlots_allowsEmpty(t *testing.T) {
	in := make([]RankSlot, 10)
	for i := range in {
		in[i] = RankSlot{Rank: i + 1}
	}
	in[0] = RankSlot{Rank: 1, SchemeID: "sch_1", SchemeName: "方案A", PlayMethod: "定位胆万位", LotteryCode: "tron_ffc_1m"}

	out, err := validateAdminSaveSlots(in)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].SchemeID != "sch_1" {
		t.Fatalf("slot0=%q", out[0].SchemeID)
	}
	if out[1].SchemeID != "" {
		t.Fatalf("slot1 should stay empty")
	}
}

// 重复方案不再整板报错：保留第一个名次，其余去重清空，保证「更换方案」可保存。
func TestValidateAdminSaveSlots_dedupesDuplicateScheme(t *testing.T) {
	in := make([]RankSlot, 10)
	for i := range in {
		in[i] = RankSlot{Rank: i + 1}
	}
	in[0] = RankSlot{Rank: 1, SchemeID: "snap_1", SchemeName: "方案A", PlayMethod: "定位胆万位", LotteryCode: "tron_ffc_1m"}
	in[1] = RankSlot{Rank: 2, SchemeID: "snap_1", SchemeName: "方案A", PlayMethod: "定位胆万位", LotteryCode: "tron_ffc_1m"}

	out, err := validateAdminSaveSlots(in)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].SchemeID != "snap_1" {
		t.Fatalf("slot0 should keep first occurrence, got %q", out[0].SchemeID)
	}
	if out[1].SchemeID != "" {
		t.Fatalf("slot1 duplicate should be cleared, got %q", out[1].SchemeID)
	}
}
