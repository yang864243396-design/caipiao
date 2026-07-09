package historysync

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHistoryAPIPathCoversSeedOnSale(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	csvPath := filepath.Join(filepath.Dir(file), "..", "..", "..", "docs", "seeds", "lottery_catalog.csv")
	f, err := os.Open(csvPath)
	if err != nil {
		t.Skip("lottery_catalog.csv not found:", err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatal("empty csv")
	}
	header := rows[0]
	codeIdx, onSaleIdx := -1, -1
	for i, h := range header {
		h = strings.TrimPrefix(h, "\ufeff")
		switch h {
		case "code":
			codeIdx = i
		case "on_sale":
			onSaleIdx = i
		}
	}
	if codeIdx < 0 || onSaleIdx < 0 {
		t.Fatal("csv missing code/on_sale columns")
	}

	var onSale []string
	for _, row := range rows[1:] {
		if len(row) <= onSaleIdx {
			continue
		}
		if row[onSaleIdx] != "true" {
			continue
		}
		onSale = append(onSale, row[codeIdx])
	}
	if len(onSale) == 0 {
		t.Fatal("no on_sale rows in seed csv")
	}

	var missing []string
	for _, code := range onSale {
		if HistoryAPIPathForCode(code) == "" {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("seed on_sale=%d missing history API for: %v", len(onSale), missing)
	}
	t.Logf("seed on_sale=%d all covered", len(onSale))
}
