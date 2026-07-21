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
}

func TestWuxingZu30Wire(t *testing.T) {
	meta := ParseRuleMeta("fast_ssc_std", "g015", "158", "组选30", "五星", nil, "158")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "123,45" {
		t.Fatalf("wire=%q want 123,45", wire)
	}
	if n := CountBetNums(meta, wire); n != 6 {
		t.Fatalf("bets=%d want 6", n)
	}
}

func TestWuxingZu20Wire(t *testing.T) {
	meta := ParseRuleMeta("fast_ssc_std", "g015", "159", "组选20", "五星", nil, "159")
	wire := FormatBetContentForRule(meta, SampleGroupContent(meta))
	if wire != "12,345" {
		t.Fatalf("wire=%q want 12,345", wire)
	}
	if n := CountBetNums(meta, wire); n != 2 {
		t.Fatalf("bets=%d want 2", n)
	}
}

func TestWuxingZu10And5Wire(t *testing.T) {
	meta10 := ParseRuleMeta("fast_ssc_std", "g015", "160", "组选10", "五星", nil, "160")
	wire10 := FormatBetContentForRule(meta10, SampleGroupContent(meta10))
	if wire10 != "0,12345" {
		t.Fatalf("zu10 wire=%q want 0,12345", wire10)
	}
	if n := CountBetNums(meta10, wire10); n != 5 {
		t.Fatalf("zu10 bets=%d want 5", n)
	}

	meta5 := ParseRuleMeta("fast_ssc_std", "g015", "161", "组选5", "五星", nil, "161")
	wire5 := FormatBetContentForRule(meta5, SampleGroupContent(meta5))
	if wire5 != "0,12345" {
		t.Fatalf("zu5 wire=%q want 0,12345", wire5)
	}
	if n := CountBetNums(meta5, wire5); n != 5 {
		t.Fatalf("zu5 bets=%d want 5", n)
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

	meta4 := ParseRuleMeta("ssc_std", "g013", "133", "组选4", "四星", nil, "133")
	wire4 := FormatBetContentForRule(meta4, "1,2,3,4")
	if wire4 != "1,2" {
		t.Fatalf("zu4 flat→wire=%q want 1,2", wire4)
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

	meta20 := ParseRuleMeta("ssc_std", "g015", "159", "组选20", "五星", nil, "159")
	wire20 := FormatBetContentForRule(meta20, "1,2,3,4,5")
	if wire20 != "12,345" {
		t.Fatalf("zu20 flat→wire=%q want 12,345", wire20)
	}

	meta10 := ParseRuleMeta("ssc_std", "g015", "160", "组选10", "五星", nil, "160")
	wire10 := FormatBetContentForRule(meta10, "1,2,3,4,5")
	if wire10 != "0,12345" {
		t.Fatalf("zu10 flat→wire=%q want 0,12345", wire10)
	}
}
