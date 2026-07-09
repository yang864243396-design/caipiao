// 从 real-bet-matrix JSONL 生成 Markdown 测试报告。
// go run ./cmd/real-bet-matrix-report [-out ../docs/real-bet-matrix-test-report.md]
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type row struct {
	LotteryCode     string  `json:"lotteryCode"`
	TypeID          string  `json:"typeId"`
	SubID           string  `json:"subId"`
	Label           string  `json:"label"`
	RuleID          string  `json:"ruleId"`
	Status          string  `json:"status"`
	Error           string  `json:"error"`
	VerifyStatus    string  `json:"verifyStatus"`
	VerifyDetail    string  `json:"verifyDetail"`
	ThirdPartyBetID string  `json:"thirdPartyBetId"`
	PeriodWaits     int     `json:"periodWaits"`
	Amount          float64 `json:"amount"`
}

func main() {
	outDir := flag.String("dir", "data/real-bet-matrix/by-lottery", "分彩种 JSONL 目录")
	merged := flag.String("merged", "data/real-bet-matrix/all-results.jsonl", "合并 JSONL")
	outPath := flag.String("out", "../docs/real-bet-matrix-test-report.md", "报告输出路径")
	account := flag.String("account", "vs8888", "测试账号")
	unit := flag.Float64("unit", 2, "单注金额")
	note := flag.String("note", "", "报告顶部备注")
	flag.Parse()

	rows, err := loadRows(*merged, *outDir)
	if err != nil {
		fmt.Println("load:", err)
		os.Exit(1)
	}

	const expectedTotal = 4059
	ok, skip, fail := 0, 0, 0
	verifyOk, verifyMismatch, verifyNotFound, verifySkipped := 0, 0, 0, 0
	byLottery := map[string][]row{}
	skipReason := map[string]int{}
	failReason := map[string]int{}
	for _, r := range rows {
		byLottery[r.LotteryCode] = append(byLottery[r.LotteryCode], r)
		switch r.Status {
		case "ok":
			ok++
		case "skip":
			skip++
			skipReason[r.Error]++
		case "fail":
			fail++
			failReason[r.Error]++
		}
		switch strings.TrimSpace(r.VerifyStatus) {
		case "ok":
			verifyOk++
		case "mismatch":
			verifyMismatch++
		case "not_found":
			verifyNotFound++
		case "skipped":
			verifySkipped++
		}
	}
	total := 0
	for _, items := range byLottery {
		total += len(items)
	}
	targetTotal := expectedTotal
	passRate := 0.0
	if total > 0 {
		passRate = float64(ok) * 100 / float64(total)
	}

	codes := make([]string, 0, len(byLottery))
	for c := range byLottery {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	var b strings.Builder
	w := func(s string) { b.WriteString(s); b.WriteByte('\n') }
	w("# 全彩种全玩法真实下单矩阵测试报告")
	w("")
	w("> 自动生成于 " + time.Now().Format("2006-01-02 15:04:05"))
	if strings.TrimSpace(*note) != "" {
		w("> " + strings.TrimSpace(*note))
	}
	w("")
	w("## 1. 测试概览")
	w("")
	w("| 指标 | 数值 |")
	w("|------|------|")
	w(fmt.Sprintf("| 矩阵总行数（在测） | %d / %d |", total, targetTotal))
	w(fmt.Sprintf("| 全量设计规模 | %d |", expectedTotal))
	w(fmt.Sprintf("| 成功 (ok) | %d |", ok))
	w(fmt.Sprintf("| 跳过 (skip) | %d |", skip))
	w(fmt.Sprintf("| 失败 (fail) | %d |", fail))
	w(fmt.Sprintf("| 通过率 (ok/在测总数) | %.2f%% |", passRate))
	w(fmt.Sprintf("| 测试账号 | %s |", *account))
	w(fmt.Sprintf("| 单注金额 (unit) | %.0f 元 |", *unit))
	w(fmt.Sprintf("| 第三方对账 ok | %d |", verifyOk))
	w(fmt.Sprintf("| 第三方对账 mismatch | %d |", verifyMismatch))
	w(fmt.Sprintf("| 第三方对账 not_found | %d |", verifyNotFound))
	w("")
	w("## 2. 按彩种汇总")
	w("")
	w("| 彩种 | 总数 | ok | skip | fail | 通过率 | 状态 |")
	w("|------|------|----|------|------|--------|------|")
	for _, code := range codes {
		items := byLottery[code]
		nOk, nSkip, nFail := 0, 0, 0
		for _, it := range items {
			switch it.Status {
			case "ok":
				nOk++
			case "skip":
				nSkip++
			case "fail":
				nFail++
			}
		}
		nAll := len(items)
		rate := 0.0
		if nAll > 0 {
			rate = float64(nOk) * 100 / float64(nAll)
		}
		state := "incomplete"
		switch {
		case nFail > 0:
			state = "fail"
		case nOk == nAll:
			state = "done"
		case nOk > 0:
			state = "partial"
		case nSkip == nAll:
			state = "all-skip"
		}
		w(fmt.Sprintf("| %s | %d | %d | %d | %d | %.1f%% | %s |", code, nAll, nOk, nSkip, nFail, rate, state))
	}

	w("")
	w("## 3. Skip 原因分布")
	w("")
	writeReasonTable(&b, skipReason)

	w("")
	w("## 4. 失败原因分布")
	w("")
	writeReasonTable(&b, failReason)

	w("")
	w("## 5. 第三方投注记录对账")
	w("")
	if verifyOk+verifyMismatch+verifyNotFound+verifySkipped == 0 {
		w("本次结果未含对账字段（可能为旧版 jsonl）。")
	} else {
		w("| 状态 | 次数 | 说明 |")
		w("|------|------|------|")
		w(fmt.Sprintf("| ok | %d | game_id / periods / bet_amount / rule_id 与 web_bets 一致 |", verifyOk))
		w(fmt.Sprintf("| mismatch | %d | 字段不一致 |", verifyMismatch))
		w(fmt.Sprintf("| not_found | %d | 第三方列表未找到注单 |", verifyNotFound))
		w(fmt.Sprintf("| skipped | %d | 未启用对账或无 thirdPartyBetId |", verifySkipped))
		if verifyMismatch+verifyNotFound > 0 {
			w("")
			w("### 对账异常明细（最多 50 条）")
			w("")
			w("| 彩种 | type/sub | rule | verify | 详情 |")
			w("|------|----------|------|--------|------|")
			shown := 0
			for _, r := range rows {
				if r.VerifyStatus != "mismatch" && r.VerifyStatus != "not_found" {
					continue
				}
				if shown >= 50 {
					break
				}
				w(fmt.Sprintf("| %s | %s/%s | %s | %s | %s |",
					r.LotteryCode, r.TypeID, r.SubID, r.RuleID, r.VerifyStatus, mdEsc(r.VerifyDetail)))
				shown++
			}
		}
	}

	w("")
	w("## 6. 失败明细（最多 200 条）")
	w("")
	if fail == 0 {
		w("无。")
	} else {
		w("| 彩种 | type/sub | 玩法 | rule | 错误 |")
		w("|------|----------|------|------|------|")
		shown := 0
		for _, r := range rows {
			if r.Status != "fail" {
				continue
			}
			if shown >= 200 {
				break
			}
			w(fmt.Sprintf("| %s | %s/%s | %s | %s | %s |",
				r.LotteryCode, r.TypeID, r.SubID, mdEsc(r.Label), r.RuleID, mdEsc(r.Error)))
			shown++
		}
		if fail > 200 {
			w("")
			w(fmt.Sprintf("> 另有 %d 条失败，见 `data/real-bet-matrix/all-results.jsonl`。", fail-200))
		}
	}

	w("")
	w("## 7. 结论")
	w("")
	if total < targetTotal {
		w(fmt.Sprintf("- **进度**：在测矩阵尚未跑完（%d / %d），以下为当前快照。", total, targetTotal))
	}
	switch {
	case fail == 0 && ok > 0 && skip == 0 && verifyMismatch == 0 && verifyNotFound == 0 && total >= targetTotal:
		w("- **结论**：全矩阵真实下单成功，第三方对账通过。")
	case fail == 0 && ok > 0 && skip == 0 && verifyMismatch == 0 && verifyNotFound == 0:
		w("- **结论**：在测项全部下单成功，第三方对账通过。")
	case fail == 0 && ok > 0 && verifyMismatch == 0 && verifyNotFound == 0:
		w("- **结论**：无编码/下单失败；剩余 skip 为固定 wire 待确认项。")
	case verifyMismatch > 0 || verifyNotFound > 0:
		w("- **结论**：存在第三方对账异常，见第 6 节。")
	case fail > 0:
		w("- **结论**：存在 fail，需按第 4-5 节修复 guajibet 编码或 solo/bets_nums 后重跑失败项。")
	default:
		w("- **结论**：尚无成功下单，请检查 GUAJI_ENABLED、会员挂账 token、期号同步。")
	}

	w("")
	w("## 8. 原始数据")
	w("")
	w("- 分彩种：`backend/data/real-bet-matrix/by-lottery/*.jsonl`")
	w("- 合并：`backend/data/real-bet-matrix/all-results.jsonl`")
	w("- 批次汇总：`backend/data/real-bet-matrix/batch-summary.jsonl`")
	w("- 跑批日志：`backend/data/real-bet-matrix/batch-run.log`")

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Println("mkdir:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, []byte(b.String()), 0o644); err != nil {
		fmt.Println("write:", err)
		os.Exit(1)
	}
	fmt.Printf("report -> %s (rows=%d ok=%d skip=%d fail=%d verify_ok=%d verify_bad=%d)\n",
		*outPath, total, ok, skip, fail, verifyOk, verifyMismatch+verifyNotFound)
}

func loadRows(merged, dir string) ([]row, error) {
	if st, err := os.Stat(merged); err == nil && st.Size() > 0 {
		return readJSONL(merged)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []row
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		part, err := readJSONL(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

func readJSONL(path string) ([]row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []row
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, sc.Err()
}

func writeReasonTable(b *strings.Builder, counts map[string]int) {
	if len(counts) == 0 {
		b.WriteString("无。\n")
		return
	}
	type kv struct {
		k string
		v int
	}
	list := make([]kv, 0, len(counts))
	for k, v := range counts {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].v == list[j].v {
			return list[i].k < list[j].k
		}
		return list[i].v > list[j].v
	})
	b.WriteString("| 次数 | 原因 |\n|------|------|\n")
	for _, it := range list {
		fmt.Fprintf(b, "| %d | %s |\n", it.v, mdEsc(it.k))
	}
}

func mdEsc(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
