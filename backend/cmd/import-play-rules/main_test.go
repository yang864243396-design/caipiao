package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestImportRejectsAmbiguousExcelRuleName(t *testing.T) {
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

	_, err := ImportFile(path, []CatalogCandidate{
		{RuleID: "1", FullName: "前三直选复式", TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"},
		{RuleID: "1", FullName: "前三直选复式", TemplateCode: "fast_ssc_std", TypeID: "g001", SubID: "1"},
	})
	if !errors.Is(err, ErrAmbiguousRuleMatch) {
		t.Fatalf("error = %v, want %v", err, ErrAmbiguousRuleMatch)
	}
}
