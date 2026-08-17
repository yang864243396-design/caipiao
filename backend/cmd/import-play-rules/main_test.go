package main

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestImportExpandsEquivalentTemplatesForAnExactRuleName(t *testing.T) {
	file := excelize.NewFile()
	sheet := file.GetSheetName(0)
	if err := file.SetSheetRow(sheet, "A1", &[]any{"ID", "", "", "", "规则全名", "描述", "示例", "号码区间", "全注数", "可中注数", "", "赔率", "", "", "", "单挑注数"}); err != nil {
		t.Fatal(err)
	}
	if err := file.SetSheetRow(sheet, "A2", &[]any{"1", "", "", "", "前三直选复式", "前三直选", "0,1,2", "0-9", "1000", "1", "", "1000", "", "", "", "10"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rules.xlsx")
	if err := file.SaveAs(path); err != nil {
		t.Fatal(err)
	}

	drafts, err := ImportFile(path, []CatalogCandidate{
		{RuleID: "1", FullName: "前三直选复式", TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"},
		{RuleID: "1", FullName: "前三直选复式", TemplateCode: "fast_ssc_std", TypeID: "g001", SubID: "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(drafts) != 2 {
		t.Fatalf("drafts = %d, want 2", len(drafts))
	}
}

func TestMatchCandidatesRejectsUniqueRuleIDWhenWorkbookVersionDiffers(t *testing.T) {
	rule := ExcelRule{RuleID: "447", FullName: "个位大小单双"}
	candidates := []CatalogCandidate{
		{RuleID: "447", FullName: "一星个位大小单双", TemplateCode: "ssc_std", TypeID: "g016", SubID: "447"},
	}

	matches := matchCandidates(rule, candidates)
	if len(matches) != 0 {
		t.Fatalf("matches = %#v, want no match without an exact name", matches)
	}
}

func TestMatchCandidatesUsesExactNameWhenWorkbookRuleIDWasRenumbered(t *testing.T) {
	rule := ExcelRule{RuleID: "436", FullName: "前三直选复式"}
	candidates := []CatalogCandidate{
		{RuleID: "1", FullName: "前三直选复式", TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"},
	}

	matches := matchCandidates(rule, candidates)
	if len(matches) != 1 || matches[0].RuleID != "1" {
		t.Fatalf("matches = %#v, want the exact-name catalogue candidate", matches)
	}
}

func TestImportFilePartialKeepsResolvedDraftsWhenOtherRowsNeedReview(t *testing.T) {
	file := excelize.NewFile()
	sheet := file.GetSheetName(0)
	if err := file.SetSheetRow(sheet, "A1", &[]any{"ID", "", "", "", "规则全名"}); err != nil {
		t.Fatal(err)
	}
	if err := file.SetSheetRow(sheet, "A2", &[]any{"1", "", "", "", "前三直选复式"}); err != nil {
		t.Fatal(err)
	}
	if err := file.SetSheetRow(sheet, "A3", &[]any{"999", "", "", "", "旧版未知玩法"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "partial-rules.xlsx")
	if err := file.SaveAs(path); err != nil {
		t.Fatal(err)
	}

	report, err := ImportFilePartial(path, []CatalogCandidate{
		{RuleID: "1", FullName: "前三直选复式", TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Drafts) != 1 || len(report.Unresolved) != 1 || len(report.Ambiguous) != 0 {
		t.Fatalf("report = %#v, want one draft and one unresolved row", report)
	}
}
