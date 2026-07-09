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
