package guajibet

import "testing"

func TestSampleLHCTuotouContent_numberErquanzhong(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "278", "拖头", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中拖头"}`), "278")
	got := sampleLHCTuotouContent(meta)
	if got != "01|02,03" {
		t.Fatalf("sample=%q want 01|02,03", got)
	}
	wire := FormatBetContentForRule(meta, got)
	if n := countLHCBetNums(meta, wire); n != 2 {
		t.Fatalf("bets=%d want 2", n)
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
	if got != "01|02" {
		t.Fatalf("sample=%q want 01|02", got)
	}
	if got := MatrixSkipReason(meta); got != "" {
		t.Fatalf("280 should not skip: %q", got)
	}
}

func TestSampleLHCSwDuipengContent(t *testing.T) {
	meta := ParseRuleMeta("lhc_std", "g003", "281", "生尾对碰", "连码",
		[]byte(`{"guajiTeam":"二全中","guajiGroup":"连码","guajiFullName":"二全中生尾对碰"}`), "281")
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
		t.Fatalf("281 should not skip: %q", got)
	}
}

func TestFormatLHCTemaZongxiaoQimaWire(t *testing.T) {
	tema := ParseRuleMeta("lhc_std", "g001", "385", "特码A", "特码", nil, "385")
	if w := FormatBetContentForRule(tema, "07"); w != "07||" {
		t.Fatalf("tema wire=%q", w)
	}
	if w := FormatBetContentForRule(tema, "7,13"); w != "07||,13||" {
		t.Fatalf("tema multi wire=%q", w)
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
