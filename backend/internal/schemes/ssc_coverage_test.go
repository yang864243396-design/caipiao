package schemes

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSCSubPlayCoverage(t *testing.T) {
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

	balls := []string{"1", "3", "5", "7", "9"}
	var count int
	for i, row := range rows {
		if i == 0 || len(row) < 6 {
			continue
		}
		if strings.TrimSpace(row[0]) != "ssc_std" {
			continue
		}
		typeID := strings.TrimSpace(row[1])
		subID := strings.TrimSpace(row[2])
		betMode := strings.TrimSpace(row[5])
		count++

		cfg := map[string]interface{}{
			"playTemplate": "ssc_std",
			"typeId":       typeID,
			"subId":        subID,
			"betMode":      betMode,
		}
		rule, ok := resolveCatalogPlayRule(cfg)
		if !ok {
			t.Fatalf("resolveCatalogPlayRule failed: %s/%s", typeID, subID)
		}
		if rule.PlayTemplate != "ssc_std" {
			t.Fatalf("template for %s/%s: %+v", typeID, subID, rule)
		}
		ev := evaluatePlayHit(rule, balls, sampleSSCContent(rule), false, "", rule.PositionIdx)
		if ev.BetUnits <= 0 {
			t.Fatalf("zero bet units for %s/%s betMode=%s sub=%s", typeID, subID, betMode, rule.SubPlayID)
		}
	}
	if count != 175 {
		t.Fatalf("want 175 ssc_std sub plays, got %d", count)
	}
}

func sampleSSCContent(rule playRule) string {
	switch rule.BetMode {
	case "longhu", "longhuhe":
		return "龙,虎"
	case "hezhi":
		return "12,15"
	case "kuadu":
		return "3,5"
	case "budingwei":
		return "1,3,5,7"
	case "dxds", "daxiao", "danshuang":
		return "大,单"
	case "zu3", "zu6":
		return "1,3,5"
	case "zuhe":
		n := rule.SegmentLen
		if n <= 0 {
			n = 3
		}
		return strings.TrimSuffix(strings.Repeat("1,3\n", n), "\n")
	case "baodan":
		return "1,3"
	case "hunhe":
		return "123"
	case "weishu":
		return "1,3,5"
	case "teshu":
		// 五星趣味号（如一帆风顺）走数字选号，不能使用常规特殊号文本。
		if isWuxingQuweiDigitPlay(rule) {
			return "6"
		}
		return "顺子"
	case "zu4":
		// 组选4为「三重号,单号」双区格式，扁平号码池会算成 0 注。
		return "1,2"
	case "zu24", "zu12", "zu60", "zu30", "zu120", "zu5", "zu10", "zu20":
		return "1,2,3,4,5"
	default:
		if rule.SubPlayID == "zhixuan_ds" || rule.BetMode == "danshi" {
			n := rule.SegmentLen
			if n <= 0 {
				n = 1
			}
			return strings.Repeat("1", n)
		}
		if rule.PlayTypeID == "renxuan" {
			return "1,3,5,7"
		}
		return "1,3,7"
	}
}

func TestSampleSSCContentUsesDigitsForWuxingQuwei(t *testing.T) {
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g015",
		SubPlayID:    "wuxing_yifan",
		BetMode:      "teshu",
	}
	if got := sampleSSCContent(rule); got != "6" {
		t.Fatalf("sampleSSCContent() = %q, want numeric content for 五星趣味", got)
	}
}
