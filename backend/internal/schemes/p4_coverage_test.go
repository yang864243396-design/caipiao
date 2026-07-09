package schemes

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestP4SubPlayCoverage(t *testing.T) {
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

	want := map[string]int{
		"syxw_std": 30,
		"pk10_std": 36,
		"k3_std":   9,
		"pc28_std": 8,
	}
	got := map[string]int{}
	ballsByTemplate := map[string][]string{
		"syxw_std": {"01", "03", "05", "07", "09"},
		"pk10_std": {"3", "7", "1", "9", "5", "2", "8", "4", "6", "10"},
		"k3_std":   {"2", "4", "6"},
		"pc28_std": {"3", "5", "7"},
	}

	for i, row := range rows {
		if i == 0 || len(row) < 6 {
			continue
		}
		template := strings.TrimSpace(row[0])
		if _, ok := want[template]; !ok {
			continue
		}
		typeID := strings.TrimSpace(row[1])
		subID := strings.TrimSpace(row[2])
		betMode := strings.TrimSpace(row[5])
		got[template]++

		cfg := map[string]interface{}{
			"playTemplate": template,
			"typeId":       typeID,
			"subId":        subID,
			"betMode":      betMode,
		}
		rule, ok := resolveCatalogPlayRule(cfg)
		if !ok {
			t.Fatalf("resolveCatalogPlayRule failed: %s/%s", typeID, subID)
		}
		if rule.PlayTemplate != template {
			t.Fatalf("template mismatch for %s/%s: %+v", typeID, subID, rule)
		}
		balls := ballsByTemplate[template]
		ev := evaluatePlayHit(rule, balls, sampleP4Content(rule), false, "", rule.PositionIdx)
		if ev.BetUnits <= 0 {
			t.Fatalf("zero bet units for %s/%s", typeID, subID)
		}
	}

	for template, n := range want {
		if got[template] != n {
			t.Fatalf("%s: want %d sub plays, got %d", template, n, got[template])
		}
	}
}

func sampleP4Content(rule playRule) string {
	switch rule.BetMode {
	case "longhu", "daxiao", "danshuang", "dxds", "teshu", "longhubao":
		return "大,单"
	case "hezhi":
		return "12,15"
	case "lianhao", "sanlian":
		return "三连"
	case "tonghao", "butong", "dantiao", "shoudong":
		return "2,4,6"
	case "danshi":
		if rule.PlayTypeID == "tonghao" {
			return "2|4"
		}
		return "020406"
	default:
		if rule.PlayTypeID == "renxuan_fs" || rule.PlayTypeID == "renxuan_ds" {
			return "01,03,05"
		}
		if rule.SubPlayID == "zhixuan_ds" {
			return strings.Repeat("0", rule.SegmentLen)
		}
		return "01,03,05"
	}
}
