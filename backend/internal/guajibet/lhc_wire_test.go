package guajibet

import "testing"

func TestCountLHCBetNums_RenyiDuipengCountsCartesianProduct(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "284", "二全中任意对碰", "连码", nil, "284")
	meta.ForcedBetMode = "renyi_dp"
	content := "01,02,03,04,05,06,07,08|09,10"
	if wire := FormatBetContentForRule(meta, content); wire != content {
		t.Fatalf("wire=%q want %q", wire, content)
	}
	if n := CountBetNums(meta, content); n != 16 {
		t.Fatalf("bets=%d want 16", n)
	}
}

func TestSampleLHCTuotouContent_numberErquanzhong(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "280", "拖头", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中拖头"}`), "280")
	got := sampleLHCTuotouContent(meta)
	if got != "01|02,03" {
		t.Fatalf("sample=%q want 01|02,03", got)
	}
	wire := FormatBetContentForRule(meta, got)
	if n := countLHCBetNums(meta, wire); n != 2 {
		t.Fatalf("bets=%d want 2", n)
	}
}

func TestFormatLHCWsDuipengWire(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "282", "尾数对碰", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中尾数对碰"}`), "282")
	want := "10,20,30,40|01,11,21,31,41"
	if w := FormatBetContentForRule(meta, "0,1"); w != want {
		t.Fatalf("flat wire=%q want %s", w, want)
	}
	if w := FormatBetContentForRule(meta, "0|1"); w != want {
		t.Fatalf("bar wire=%q want %s", w, want)
	}
	if w := FormatBetContentForRule(meta, want); w != want {
		t.Fatalf("number wire=%q want idempotent %s", w, want)
	}
	if n := countLHCBetNums(meta, "0|1"); n != 20 {
		t.Fatalf("0|1 bets=%d want 20", n)
	}
	if n := countLHCBetNums(meta, "1|2"); n != 25 {
		t.Fatalf("1|2 bets=%d want 25", n)
	}
	meta.ForcedBetMode = "ws_dp"
	meta2 := ParseRuleMeta("lhc_std", "g003", "282", "", "", nil, "282")
	meta2.ForcedBetMode = "ws_dp"
	if w := FormatBetContentForRule(meta2, "0|1"); w != want {
		t.Fatalf("forced mode wire=%q want %s", w, want)
	}
	if got := SampleGroupContent(meta); FormatBetContentForRule(meta, got) != want && got != want {
		t.Fatalf("sample=%q want %s", got, want)
	}
}

func TestFormatLHCSxDuipengWire(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "281", "生肖对碰", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中生肖对碰"}`), "281")
	wantHorseSnake := "01,13,25,37,49|02,14,26,38"
	if w := FormatBetContentForRule(meta, "马,蛇"); w != wantHorseSnake {
		t.Fatalf("flat wire=%q want %s", w, wantHorseSnake)
	}
	if w := FormatBetContentForRule(meta, "马|蛇"); w != wantHorseSnake {
		t.Fatalf("bar wire=%q want %s", w, wantHorseSnake)
	}
	// 已是号码列表：幂等
	if w := FormatBetContentForRule(meta, wantHorseSnake); w != wantHorseSnake {
		t.Fatalf("number wire=%q want idempotent %s", w, wantHorseSnake)
	}
	if n := countLHCBetNums(meta, "马,蛇"); n != 20 {
		t.Fatalf("马|蛇 bets=%d want 20", n)
	}
	if n := countLHCBetNums(meta, "蛇|龙"); n != 16 {
		t.Fatalf("蛇|龙 bets=%d want 16", n)
	}
	if n := countLHCBetNums(meta, wantHorseSnake); n != 20 {
		t.Fatalf("expanded bets=%d want 20", n)
	}
	if got := SampleGroupContent(meta); FormatBetContentForRule(meta, got) != wantHorseSnake && got != wantHorseSnake {
		t.Fatalf("sample=%q want %s", got, wantHorseSnake)
	}
}

// TestFormatLHCSxDuipengWire_def1786204366339 复现定码二全中生肖对碰：
// 方案落库「马|兔」；若原样下单会被第三方拒「投注数字不合规」；
// 须展开为号码列表且 bets=20（2026-08-09 bet-probe 实测：马|兔 拒 / 展开 过）。
func TestFormatLHCSxDuipengWire_def1786204366339(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "281", "二全中生肖对碰", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiRuleId":"281","guajiFullName":"二全中生肖对碰"}`), "281")
	meta.ForcedBetMode = "sx_dp"
	want := "01,13,25,37,49|04,16,28,40"
	if w := FormatBetContentForRule(meta, "马|兔"); w != want {
		t.Fatalf("wire=%q want %s", w, want)
	}
	if n := CountBetNums(meta, "马|兔"); n != 20 {
		t.Fatalf("bets=%d want 20", n)
	}
	// 仅靠 ForcedBetMode / rule id，label 为空也须展开
	meta2 := ParseRuleMeta("lhc_std", "g003", "281", "", "", nil, "281")
	meta2.ForcedBetMode = "sx_dp"
	if w := FormatBetContentForRule(meta2, "马|兔"); w != want {
		t.Fatalf("forced mode wire=%q want %s", w, want)
	}
}

func TestFormatLHCTuotouWire_flatErquanzhong(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "280", "拖头", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中拖头"}`), "280")
	// 扁选落库 → 下单 胆|拖
	if w := FormatBetContentForRule(meta, "7,13,25"); w != "07|13,25" {
		t.Fatalf("flat wire=%q want 07|13,25", w)
	}
	if w := FormatBetContentForRule(meta, "07,13,25"); w != "07|13,25" {
		t.Fatalf("flat wire=%q want 07|13,25", w)
	}
	// 已是胆|拖：幂等补零
	if w := FormatBetContentForRule(meta, "7|13,25"); w != "07|13,25" {
		t.Fatalf("bar wire=%q want 07|13,25", w)
	}
	if once := FormatBetContentForRule(meta, "07,13,25"); FormatBetContentForRule(meta, once) != once {
		t.Fatalf("wire not idempotent: %q", once)
	}
	if n := countLHCBetNums(meta, "07,13,25"); n != 2 {
		t.Fatalf("flat bets=%d want 2", n)
	}
	if n := countLHCBetNums(meta, "07|13,25"); n != 2 {
		t.Fatalf("bar bets=%d want 2", n)
	}
}

func TestSampleLHCTuotouContent_zodiacErxiao(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g011", "319", "拖头", "生肖连",
		[]byte(`{"guajiTeam":"二肖中","guajiGroup":"生肖连","guajiFullName":"生肖连二肖中拖头"}`), "319")
	mode := inferLHCBetMode(meta)
	if mode != "tuotou" {
		t.Fatalf("mode=%q want tuotou", mode)
	}
	got := sampleLHCTuotouContent(meta)
	if got != "鼠|牛" {
		t.Fatalf("sample=%q want 鼠|牛", got)
	}
	wire := FormatBetContentForRule(meta, got)
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("wire=%q bets=%d want 1 min=%d", wire, n, lhcTeamMinPick(meta))
	}
	if got := MatrixSkipReason(meta); got != "" {
		t.Fatalf("319 should not skip: %q", got)
	}
}

func TestSampleLHCTuotouContent_zodiacSixiao(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g011", "323", "拖头", "生肖连",
		[]byte(`{"guajiTeam":"四肖中","guajiGroup":"生肖连","guajiFullName":"生肖连四肖中拖头"}`), "323")
	got := sampleLHCTuotouContent(meta)
	if got != "鼠|牛,虎,兔" {
		t.Fatalf("sample=%q want 鼠|牛,虎,兔", got)
	}
	wire := FormatBetContentForRule(meta, got)
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("wire=%q bets=%d want 1", wire, n)
	}
}

func TestSampleLHCTuotouContent_buzhong6(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g013", "349", "拖头", "全不中",
		[]byte(`{"guajiTeam":"6不中","guajiGroup":"全不中","guajiFullName":"全不中6不中拖头"}`), "349")
	got := sampleLHCTuotouContent(meta)
	if got != "01|02,03,04,05,06" {
		t.Fatalf("sample=%q want 01|02,03,04,05,06", got)
	}
}

func TestSampleLHCTuotouContent_buzhong5(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g013", "347", "拖头", "全不中",
		[]byte(`{"guajiTeam":"5不中","guajiGroup":"全不中","guajiFullName":"全不中5不中拖头"}`), "347")
	got := sampleLHCTuotouContent(meta)
	if got != "01|02,03,04,05" {
		t.Fatalf("sample=%q want 01|02,03,04,05", got)
	}
	wire := FormatBetContentForRule(meta, got)
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("wire=%q bets=%d want 1", wire, n)
	}
}

func TestSampleLHCTuotouContent_tepingzhong3(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g015", "380", "拖头", "特平中",
		[]byte(`{"guajiTeam":"三粒任中","guajiGroup":"特平中","guajiFullName":"特平中三粒任中拖头"}`), "380")
	got := sampleLHCTuotouContent(meta)
	if got != "01|02,03" {
		t.Fatalf("sample=%q want 01|02,03", got)
	}
	wire := FormatBetContentForRule(meta, got)
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("wire=%q bets=%d want 1", wire, n)
	}
}

func TestSampleLHCTuotouContent_tepingzhong4(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g015", "382", "拖头", "特平中",
		[]byte(`{"guajiTeam":"四粒任中","guajiGroup":"特平中","guajiFullName":"特平中四粒任中拖头"}`), "382")
	got := sampleLHCTuotouContent(meta)
	if got != "01|02,03,04" {
		t.Fatalf("sample=%q want 01|02,03,04", got)
	}
}

func TestSampleLHCTuotouContent_tailErwai(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g012", "335", "拖头", "尾数连",
		[]byte(`{"guajiTeam":"二尾中","guajiGroup":"尾数连","guajiFullName":"尾数连二尾中拖头"}`), "335")
	got := sampleLHCTuotouContent(meta)
	if got != "0尾|1尾" {
		t.Fatalf("sample=%q want 0尾|1尾", got)
	}
}

func TestSampleLHCFushiContent_zodiacErxiao(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g011", "318", "复式", "生肖连",
		[]byte(`{"guajiTeam":"二肖中","guajiGroup":"生肖连","guajiFullName":"生肖连二肖中复式"}`), "318")
	got := sampleLHCFushiContent(meta)
	if got != "鼠,牛" {
		t.Fatalf("sample=%q want 鼠,牛", got)
	}
	wire := FormatBetContentForRule(meta, got)
	if wire != "鼠,牛" {
		t.Fatalf("wire=%q want 鼠,牛", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
	if got := MatrixSkipReason(meta); got != "" {
		t.Fatalf("318 should not skip: %q", got)
	}
}

func TestSampleLHCFushiContent_tailErwai(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g012", "334", "复式", "尾数连",
		[]byte(`{"guajiTeam":"二尾中","guajiGroup":"尾数连","guajiFullName":"尾数连二尾中复式"}`), "334")
	got := sampleLHCFushiContent(meta)
	if got != "0尾,1尾" {
		t.Fatalf("sample=%q want 0尾,1尾", got)
	}
	wire := FormatBetContentForRule(meta, got)
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
}

func TestSampleLHCFushiContent_tepingzhongSan(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g015", "379", "复式", "特平中",
		[]byte(`{"guajiTeam":"三粒任中","guajiGroup":"特平中","guajiFullName":"特平中三粒任中"}`), "379")
	got := sampleLHCFushiContent(meta)
	if got != "01,02,03" {
		t.Fatalf("sample=%q want 01,02,03", got)
	}
	if n := CountBetNums(meta, FormatBetContentForRule(meta, got)); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
}

func TestSampleLHCGroupContent_duipeng(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "280", "尾数对碰", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中尾数对碰"}`), "280")
	got := SampleGroupContent(meta)
	want := "10,20,30,40|01,11,21,31,41"
	if got != want {
		t.Fatalf("sample=%q want %q", got, want)
	}
	if got := MatrixSkipReason(meta); got != "" {
		t.Fatalf("280 should not skip: %q", got)
	}
}

func TestSampleLHCSwDuipengContent(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "283", "生尾对碰", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中生尾对碰"}`), "283")
	got := SampleGroupContent(meta)
	want := "07,19,31,43|10,20,30,40"
	if got != want {
		t.Fatalf("sample=%q want %q", got, want)
	}
	wire := FormatBetContentForRule(meta, got)
	if n := CountBetNums(meta, wire); n != 16 {
		t.Fatalf("bets=%d want 16 wire=%q", n, wire)
	}
	if got := MatrixSkipReason(meta); got != "" {
		t.Fatalf("283 should not skip: %q", got)
	}
}

func TestFormatLHCSwDuipengWire(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "283", "生尾对碰", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中生尾对碰"}`), "283")
	meta.ForcedBetMode = "sw_dp"
	got := FormatBetContentForRule(meta, "马|0")
	want := "01,13,25,37,49|10,20,30,40"
	if got != want {
		t.Fatalf("wire=%q want %q", got, want)
	}
	if n := CountBetNums(meta, "马|0"); n != 20 {
		t.Fatalf("马|0 bets=%d want 20", n)
	}
	if n := CountBetNums(meta, "0|马"); n != 20 {
		t.Fatalf("0|马 bets=%d want 20", n)
	}
	// 狗∩5尾={45}：展开仍含两侧 45，注数须 4×5−1=19（bets=20 第三方拒）
	wireDog5 := FormatBetContentForRule(meta, "狗|5")
	wantDog5 := "09,21,33,45|05,15,25,35,45"
	if wireDog5 != wantDog5 {
		t.Fatalf("狗|5 wire=%q want %q", wireDog5, wantDog5)
	}
	if n := CountBetNums(meta, "狗|5"); n != 19 {
		t.Fatalf("狗|5 bets=%d want 19", n)
	}
	if n := CountBetNums(meta, wireDog5); n != 19 {
		t.Fatalf("expanded 狗|5 bets=%d want 19", n)
	}
	if n := CountBetNums(meta, "马|1"); n != 24 {
		t.Fatalf("马|1 bets=%d want 24 (∩01)", n)
	}
}

func TestFormatLHCTemaZongxiaoQimaWire(t *testing.T) {
	tema := ParseRuleMeta("lhc_std", "g001", "385", "特码A", "特码", nil, "385")
	if w := FormatBetContentForRule(tema, "07"); w != "07||" {
		t.Fatalf("tema wire=%q", w)
	}
	if w := FormatBetContentForRule(tema, "7,13"); w != "07,13||" {
		t.Fatalf("tema multi wire=%q", w)
	}
	if w := FormatBetContentForRule(tema, "大,红波,7,13"); w != "07,13|大|红波" {
		t.Fatalf("tema attrs wire=%q", w)
	}
	if w := FormatBetContentForRule(tema, "07||,13||"); w != "07,13||" {
		t.Fatalf("tema legacy multi wire=%q", w)
	}
	// 00 非法：下单会「投注数字不合规」，wire 须滤掉
	if w := FormatBetContentForRule(tema, "00,01,02|大|红波"); w != "01,02|大|红波" {
		t.Fatalf("tema strip 00 wire=%q", w)
	}
	zx := ParseRuleMeta("lhc_std", "g005", "301", "总肖", "生肖", nil, "301")
	if w := FormatBetContentForRule(zx, "2,5"); w != "二肖,五肖" {
		t.Fatalf("zongxiao wire=%q", w)
	}
	if w := FormatBetContentForRule(zx, "二肖,五肖"); w != "二肖,五肖" {
		t.Fatalf("zongxiao wire=%q", w)
	}
	if w := FormatBetContentForRule(zx, "0,12"); w != "" {
		t.Fatalf("invalid zongxiao should filter out, got %q", w)
	}
	qm := ParseRuleMeta("lhc_std", "qima", "qima", "七码", "七码", nil, "313")
	if w := FormatBetContentForRule(qm, "双1"); w != "双1" {
		t.Fatalf("qima wire=%q", w)
	}
	if w := FormatBetContentForRule(qm, "双1,单0,invalid"); w != "双1,单0" {
		t.Fatalf("qima filter wire=%q", w)
	}
}

func TestFormatLHCTematouweiWire(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g006", "307", "特码头尾", "特码头尾",
		[]byte(`{"guajiTeam":"特码头尾","guajiGroup":"特码头尾"}`), "307")
	if got := SampleGroupContent(meta); got != "0|1" {
		t.Fatalf("sample=%q want 0|1", got)
	}
	if got := MatrixSkipReason(meta); got != "" {
		t.Fatalf("307 should not skip: %q", got)
	}
	for _, tc := range []struct {
		in, wire string
		bets     int
	}{
		{"头0,尾1", "0|1", 2},
		{"头0", "0|", 1},
		{"尾1", "|1", 1},
		{"0|1", "0|1", 2},
	} {
		wire := FormatBetContentForRule(meta, tc.in)
		if wire != tc.wire {
			t.Fatalf("in=%q wire=%q want %q", tc.in, wire, tc.wire)
		}
		if n := CountBetNums(meta, wire); n != tc.bets {
			t.Fatalf("in=%q bets=%d want %d wire=%q", tc.in, n, tc.bets, wire)
		}
	}
}

func TestSampleLHCJiayeWeishu(t *testing.T) {
	jiaye := ParseRuleMeta("lhc_std", "g007", "309", "家野", "五行家野",
		[]byte(`{"guajiTeam":"五行家野","guajiGroup":"五行家野"}`), "309")
	if SampleGroupContent(jiaye) != "家禽" {
		t.Fatalf("jiaye sample")
	}
	if MatrixSkipReason(jiaye) != "" {
		t.Fatal("309 should not skip")
	}
	wei := ParseRuleMeta("lhc_std", "g010", "316", "尾数", "一肖尾数",
		[]byte(`{"guajiTeam":"尾数","guajiGroup":"一肖尾数"}`), "316")
	if SampleGroupContent(wei) != "0尾" {
		t.Fatalf("weishu sample=%q", SampleGroupContent(wei))
	}
	if MatrixSkipReason(wei) != "" {
		t.Fatal("316 should not skip")
	}
}
