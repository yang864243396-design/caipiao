package schemes

import (
	"encoding/csv"
	"fmt"
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

	balls := []string{"3", "12", "25", "33", "41", "7", "49"}
	var count int
	var zeroUnits []string
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
		label := strings.TrimSpace(row[3])
		count++

		inferred := inferLHCBetMode(typeID, subID)
		if inferred == "" {
			t.Fatalf("empty betMode for %s/%s", typeID, subID)
		}
		rule := resolveLHCPlayRule(typeID, subID, betMode)
		if rule.PlayTemplate != "lhc_std" {
			t.Fatalf("template for %s/%s: %+v", typeID, subID, rule)
		}
		ev, ok := evaluateLHCByBetMode(rule, balls, "01,13,49")
		if !ok {
			t.Fatalf("evaluateLHCByBetMode not handled: %s/%s betMode=%s inferred=%s", typeID, subID, betMode, inferred)
		}
		// 此前这里只看 ok、把 ev 丢掉了，等于只验证「验奖分支能处理」，不验证算出来是几注。
		//
		// 注意这条断言比 ssc/p4 那两份弱：实测喂完全非法的内容（"99,100,零"）
		// 82 个子玩法照样全部返回正注数——LHC 验奖只数 token，不查号池。
		// 号池闭包由 TestLHCContentPoolClosure 单独盯，别把这条当成号池覆盖。
		if ev.BetUnits <= 0 {
			zeroUnits = append(zeroUnits, fmt.Sprintf("%s/%s %s betMode=%s inferred=%s",
				typeID, subID, label, betMode, inferred))
		}
	}
	if len(zeroUnits) > 0 {
		t.Errorf("验奖算出零注的子玩法 %d 个：\n  %s",
			len(zeroUnits), strings.Join(zeroUnits, "\n  "))
	}
	if count != 82 {
		t.Fatalf("want 82 lhc_std sub plays, got %d", count)
	}
}

// TestLHCContentPoolClosure 六合彩号码必须落在 1-49 内。
//
// 验奖那条路只数 token 不查号池（见 TestLHCSubPlayCoverage 的注释），
// 所以号池闭包只能靠合法投注空间这一层兜住。这条测的就是那一层还在不在。
func TestLHCContentPoolClosure(t *testing.T) {
	u := playNumberPool(playRule{PlayTemplate: "lhc_std"})
	if len(u) != 49 || u[0] != "1" || u[48] != "49" {
		t.Fatalf("六合彩号池 = %d 个 [%s..%s]，期望 1-49", len(u), u[0], u[len(u)-1])
	}

	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"lhc_tema","subId":"tema","betMode":"tema"}`)
	cases := []struct {
		name    string
		content string
		wantBad bool
	}{
		{name: "池内号码", content: "01,13,49"},
		{name: "超出上界的 50", content: "01,50", wantBad: true},
		{name: "超出上界的 99", content: "99", wantBad: true},
		{name: "下界外的 0", content: "0,13", wantBad: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := ValidateSchemeBetContent("custom", cfg, tc.content, 0)
			if tc.wantBad && len(vs) == 0 {
				t.Fatalf("内容 %q 越出 1-49，应被判违规", tc.content)
			}
			if !tc.wantBad && len(vs) > 0 {
				t.Fatalf("内容 %q 合法却被判违规：%s", tc.content, vs[0].Detail)
			}
		})
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
