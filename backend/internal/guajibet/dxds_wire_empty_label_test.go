package guajibet_test

import (
	"testing"

	"caipiao/backend/internal/guajibet"
)

// 线上实例：config 仅有 typeId=g016/subId=266、无中文 playMethod 时，
// wire 须为「双,单」，不能落成「双单」被第三方拒「投注不合规」。
func TestFormatBetContentForRule_dxdsHou2EmptyLabel(t *testing.T) {
	meta := guajibet.ParseRuleMeta("ssc_std", "g016", "266", "", "", nil, "266")
	if mode := guajibet.InferBetMode(meta); mode != "dxds" {
		t.Fatalf("mode=%q want dxds", mode)
	}
	wire := guajibet.FormatBetContentForRule(meta, "双\n单")
	if wire != "双,单" {
		t.Fatalf("wire=%q want 双,单", wire)
	}
	if guajibet.NeedsSoloForRule(meta, wire) {
		t.Fatal("后二大小单双不应 solo")
	}
	if n := guajibet.CountBetNums(meta, wire); n != 1 {
		t.Fatalf("betsNums=%d want 1", n)
	}
}
