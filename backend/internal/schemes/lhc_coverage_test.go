package schemes

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLHCSubPlayCoverage(t *testing.T) {
	root := findRepoRoot(t)
	csvPath := filepath.Join(root, "backend", "docs", "seeds", "sub_plays.csv")
	f, err := os.Open(csvPath)
	if err != nil {
		t.Skipf("sub_plays.csv not found: %v", err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	var count int
	for i, row := range rows {
		if i == 0 || len(row) < 4 {
			continue
		}
		if strings.TrimSpace(row[0]) != "lhc_std" {
			continue
		}
		typeID := strings.TrimSpace(row[1])
		subID := strings.TrimSpace(row[2])
		betMode := strings.TrimSpace(row[5])
		count++

		inferred := inferLHCBetMode(typeID, subID)
		if inferred == "" {
			t.Fatalf("empty betMode for %s/%s", typeID, subID)
		}
		rule := resolveLHCPlayRule(typeID, subID, betMode)
		if rule.PlayTemplate != "lhc_std" {
			t.Fatalf("template for %s/%s: %+v", typeID, subID, rule)
		}
		balls := []string{"3", "12", "25", "33", "41", "7", "49"}
		ev, ok := evaluateLHCByBetMode(rule, balls, "01,13,49")
		if !ok {
			t.Fatalf("evaluateLHCByBetMode not handled: %s/%s betMode=%s inferred=%s", typeID, subID, betMode, inferred)
		}
		_ = ev
	}
	if count != 82 {
		t.Fatalf("want 82 lhc_std sub plays, got %d", count)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "backend", "docs", "seeds", "sub_plays.csv")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
