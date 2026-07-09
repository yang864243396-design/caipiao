package guajibet

import "testing"

func TestFormatBetContent_sscDingweiWan(t *testing.T) {
	got := FormatBetContent("ssc_std", "dingwei", "定位胆 · 万位", 0, "3,9")
	if got != "39,,,," {
		t.Fatalf("got %q want 39,,,,", got)
	}
}

func TestFormatBetContent_sscDingweiGe(t *testing.T) {
	got := FormatBetContent("ssc_std", "dingwei", "定位胆 · 个位", 4, "1,3,5,7,9")
	if got != ",,,,13579" {
		t.Fatalf("got %q want ,,,,13579", got)
	}
}

func TestFormatBetContent_sscDingweiShiTrailingComma(t *testing.T) {
	got := FormatBetContent("ssc_std", "dingwei", "定位胆 · 十位", 3, "1,3,5,7,9")
	if got != ",,,13579," {
		t.Fatalf("got %q want ,,,13579,", got)
	}
}

func TestCountDingweiBetsNums(t *testing.T) {
	if n := CountDingweiBetsNums("4,,,,"); n != 1 {
		t.Fatalf("4,,,, = %d want 1", n)
	}
	if n := CountDingweiBetsNums("39,,,,"); n != 2 {
		t.Fatalf("39,,,, = %d want 2", n)
	}
	if NeedsSoloBet("4,,,,") || NeedsSoloBet("39,,,,") {
		t.Fatal("v6hs1 定位胆不应 solo")
	}
}

func TestFormatBetContent_sscDingweiFromPlayMethod(t *testing.T) {
	got := FormatBetContent("ssc_std", "", "定位胆 · 百位", -1, "0,8")
	if got != ",,08,," {
		t.Fatalf("got %q want ,,08,,", got)
	}
}

func TestFormatBetContent_sscDingweiMultiline(t *testing.T) {
	got := FormatBetContent("ssc_std", "dingwei", "定位胆", 0, "1,3,5\n2,4\n\n\n7,8")
	if got != "135,24,,,78" {
		t.Fatalf("got %q want 135,24,,,78", got)
	}
}

func TestCountDingweiBetsNums_nonDingweiWire(t *testing.T) {
	if n := CountDingweiBetsNums("123"); n != 0 {
		t.Fatalf("plain danshi content should not use dingwei counter, got %d", n)
	}
	if n := CountDingweiBetsNums("1,3|4,5|6,7"); n != 0 {
		t.Fatalf("pipe content should not use dingwei counter, got %d", n)
	}
}

func TestCountDingweiBetsNums_multiline(t *testing.T) {
	if n := CountDingweiBetsNums("135,24,,,78"); n != 7 {
		t.Fatalf("135,24,,,78 = %d want 7", n)
	}
}

func TestFormatBetContent_nonDingweiPassthrough(t *testing.T) {
	raw := "1,2,3"
	if got := FormatBetContent("ssc_std", "fushi", "前三直选复式", 0, raw); got != raw {
		t.Fatalf("got %q want passthrough", got)
	}
}
