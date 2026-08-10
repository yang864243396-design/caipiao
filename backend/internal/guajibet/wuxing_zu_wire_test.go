package guajibet

import "testing"

func TestWuxingZu60Wire(t *testing.T) {
	meta := ParseRuleMeta("ssc_std", "g015", "157", "组选60", "五星", nil, "157")
	content := SampleGroupContent(meta)
	if content != "1,234" {
		t.Fatalf("sample=%q want 1,234", content)
	}
	wire := FormatBetContentForRule(meta, content)
	if wire != "1,234" {
		t.Fatalf("wire=%q want 1,234", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
	if n := countWuxingZu60BetNums("0,1234"); n != 4 {
		t.Fatalf("pool bets=%d want 4", n)
	}
	// 多重号：12,345 → 2；跨区重叠 1,1234 → C(3,3)=1
	if n := countWuxingZu60BetNums("12,345"); n != 2 {
		t.Fatalf("12,345 bets=%d want 2", n)
	}
	if n := countWuxingZu60BetNums("1,1234"); n != 1 {
		t.Fatalf("1,1234 bets=%d want 1", n)
	}
	if n := countWuxingZu60BetNums("12,3456"); n != 8 {
		t.Fatalf("12,3456 bets=%d want 8", n)
	}
}

func TestWuxingZu30Wire(t *testing.T) {
	meta := ParseRuleMeta("fast_ssc_std", "g015", "158", "组选30", "五星", nil, "158")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "123,1" {
		t.Fatalf("wire=%q want 123,1", wire)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
	if n := countWuxingZu30BetNums("123,45"); n != 6 {
		t.Fatalf("123,45 bets=%d want 6", n)
	}
	if n := countWuxingZu30BetNums("1234,5"); n != 6 {
		t.Fatalf("1234,5 bets=%d want 6", n)
	}
	if n := countWuxingZu30BetNums("1234,56"); n != 12 {
		t.Fatalf("1234,56 bets=%d want 12", n)
	}
	if n := countWuxingZu30BetNums("12,3"); n != 0 {
		t.Fatalf("12,3 bets=%d want 0 (二重须≥3)", n)
	}
	if n := countWuxingZu30BetNums("123,12"); n != 2 {
		t.Fatalf("123,12 bets=%d want 2", n)
	}
}

func TestWuxingZu20Wire(t *testing.T) {
	meta := ParseRuleMeta("fast_ssc_std", "g015", "159", "组选20", "五星", nil, "159")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "12,34" {
		t.Fatalf("wire=%q want 12,34", wire)
	}
	if n := CountBetNums(meta, wire); n != 2 {
		t.Fatalf("bets=%d want 2", n)
	}
	// 个数相同且各≥2：12,34→2；123,345→7；123,456→9；1234,5678→24
	if n := countWuxingZu20BetNums("12,34"); n != 2 {
		t.Fatalf("12,34 bets=%d want 2", n)
	}
	if n := countWuxingZu20BetNums("123,345"); n != 7 {
		t.Fatalf("123,345 bets=%d want 7", n)
	}
	if n := countWuxingZu20BetNums("123,456"); n != 9 {
		t.Fatalf("123,456 bets=%d want 9", n)
	}
	if n := countWuxingZu20BetNums("1234,5678"); n != 24 {
		t.Fatalf("1234,5678 bets=%d want 24", n)
	}
	// 各 1、个数不同：非法
	if n := countWuxingZu20BetNums("12,345"); n != 0 {
		t.Fatalf("12,345 bets=%d want 0 (须个数相同)", n)
	}
	if n := countWuxingZu20BetNums("1,2"); n != 0 {
		t.Fatalf("1,2 bets=%d want 0 (须各≥2)", n)
	}
}

func TestWuxingZu10And5Wire(t *testing.T) {
	meta10 := ParseRuleMeta("fast_ssc_std", "g015", "160", "组选10", "五星", nil, "160")
	wire10 := FormatBetContentForRule(meta10, SampleGroupContent(meta10))
	if wire10 != "1,2" {
		t.Fatalf("zu10 wire=%q want 1,2", wire10)
	}
	if n := CountBetNums(meta10, wire10); n != 1 {
		t.Fatalf("zu10 bets=%d want 1", n)
	}
	if n := CountBetNums(meta10, "12,34"); n != 4 {
		t.Fatalf("zu10 12,34 bets=%d want 4", n)
	}
	if n := CountBetNums(meta10, "12,3"); n != 2 {
		t.Fatalf("zu10 12,3 bets=%d want 2", n)
	}
	// 旧五码池仍可出站计注
	if n := CountBetNums(meta10, "0,12345"); n != 5 {
		t.Fatalf("zu10 legacy bets=%d want 5", n)
	}

	meta5 := ParseRuleMeta("fast_ssc_std", "g015", "161", "组选5", "五星", nil, "161")
	wire5 := FormatBetContentForRule(meta5, SampleGroupContent(meta5))
	if wire5 != "1,2" {
		t.Fatalf("zu5 wire=%q want 1,2", wire5)
	}
	if n := CountBetNums(meta5, wire5); n != 1 {
		t.Fatalf("zu5 bets=%d want 1", n)
	}
	if n := CountBetNums(meta5, "12,34"); n != 4 {
		t.Fatalf("zu5 12,34 bets=%d want 4", n)
	}
}

func TestMatrixSkipReason_wuxingZu(t *testing.T) {
	meta158 := ParseRuleMeta("ssc_std", "g015", "158", "组选30", "五星", nil, "158")
	if got := MatrixSkipReason(meta158); got != "" {
		t.Fatalf("unexpected skip: %q", got)
	}
}

func TestFormatZu12Zu4_flatDigits(t *testing.T) {
	meta12 := ParseRuleMeta("ssc_std", "g013", "131", "组选12", "四星", nil, "131")
	wire12 := FormatBetContentForRule(meta12, "1,2,3,4")
	if wire12 != "12,34" {
		t.Fatalf("zu12 flat→wire=%q want 12,34", wire12)
	}
	if n := CountBetNums(meta12, wire12); n != 2 {
		t.Fatalf("zu12 bets=%d want 2", n)
	}
	// 跨区重叠原样出站；按二重分别计注
	if got := FormatBetContentForRule(meta12, "2,123"); got != "2,123" {
		t.Fatalf("zu12 2,123→wire=%q want 2,123", got)
	}
	if n := CountBetNums(meta12, "2,123"); n != 1 {
		t.Fatalf("zu12 2,123 bets=%d want 1", n)
	}
	if got := FormatBetContentForRule(meta12, "23,123"); got != "23,123" {
		t.Fatalf("zu12 23,123→wire=%q want 23,123", got)
	}
	if n := CountBetNums(meta12, "23,123"); n != 2 {
		t.Fatalf("zu12 23,123 bets=%d want 2", n)
	}

	meta4 := ParseRuleMeta("ssc_std", "g013", "133", "组选4", "四星", nil, "133")
	wire4 := FormatBetContentForRule(meta4, "1,2,3,4")
	if wire4 != "1,234" {
		t.Fatalf("zu4 flat→wire=%q want 1,234", wire4)
	}
	if n := CountBetNums(meta4, wire4); n != 3 {
		t.Fatalf("zu4 1,234 bets=%d want 3", n)
	}
	if got := FormatBetContentForRule(meta4, "12,34"); got != "12,34" {
		t.Fatalf("zu4 12,34→wire=%q want 12,34", got)
	}
	if n := CountBetNums(meta4, "12,34"); n != 4 {
		t.Fatalf("zu4 12,34 bets=%d want 4", n)
	}
	if n := CountBetNums(meta4, "1,2"); n != 1 {
		t.Fatalf("zu4 1,2 bets=%d want 1", n)
	}
	if n := CountBetNums(meta4, "1,12"); n != 1 {
		t.Fatalf("zu4 1,12 bets=%d want 1", n)
	}
	if n := CountBetNums(meta4, "1,1"); n != 0 {
		t.Fatalf("zu4 1,1 bets=%d want 0", n)
	}
}

func TestWuxingZu_coerceFlatDigits(t *testing.T) {
	meta60 := ParseRuleMeta("ssc_std", "g015", "157", "组选60", "五星", nil, "157")
	wire60 := FormatBetContentForRule(meta60, "0,1,2,3,4")
	if wire60 != "0,1234" {
		t.Fatalf("zu60 flat→wire=%q want 0,1234", wire60)
	}
	if n := CountBetNums(meta60, wire60); n != 4 {
		t.Fatalf("zu60 bets=%d want 4", n)
	}

	meta30 := ParseRuleMeta("ssc_std", "g015", "158", "组选30", "五星", nil, "158")
	wire30 := FormatBetContentForRule(meta30, "1,2,3,4,5")
	if wire30 != "123,45" {
		t.Fatalf("zu30 flat→wire=%q want 123,45", wire30)
	}
	if n := CountBetNums(meta30, wire30); n != 6 {
		t.Fatalf("zu30 flat bets=%d want 6", n)
	}
	wire30b := FormatBetContentForRule(meta30, "123,1")
	if wire30b != "123,1" {
		t.Fatalf("zu30 123,1→wire=%q want 123,1", wire30b)
	}

	meta20 := ParseRuleMeta("ssc_std", "g015", "159", "组选20", "五星", nil, "159")
	wire20 := FormatBetContentForRule(meta20, "1,2,3,4,5,6")
	if wire20 != "123,456" {
		t.Fatalf("zu20 flat→wire=%q want 123,456", wire20)
	}
	if n := CountBetNums(meta20, wire20); n != 9 {
		t.Fatalf("zu20 flat bets=%d want 9", n)
	}

	meta10 := ParseRuleMeta("ssc_std", "g015", "160", "组选10", "五星", nil, "160")
	wire10 := FormatBetContentForRule(meta10, "1,2,3")
	if wire10 != "1,23" {
		t.Fatalf("zu10 flat→wire=%q want 1,23", wire10)
	}
	if n := CountBetNums(meta10, wire10); n != 2 {
		t.Fatalf("zu10 1,23 bets=%d want 2", n)
	}
}
