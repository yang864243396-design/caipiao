package guajibet

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestProbeWireSamples(t *testing.T) {
	cases := []struct {
		typeID, sub, label, team string
	}{
		{"g011", "75", "直选单式", "任选二"},
		{"g011", "79", "组选和值", "任选二"},
		{"g011", "81", "直选单式", "任选三"},
		{"g011", "83", "组三复式", "任选三"},
		{"g011", "84", "组三单式", "任选三"},
		{"g011", "86", "组六单式", "任选三"},
		{"g011", "87", "混合组选", "任选三"},
		{"g011", "88", "组选和值", "任选三"},
		{"g011", "141", "直选复式", "任选四"},
	}
	for _, c := range cases {
		seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": c.team})
		meta := ParseRuleMeta("ssc_std", c.typeID, c.sub, c.label, "任选", seg, c.sub)
		content := SampleGroupContent(meta)
		wire := FormatBetContentForRule(meta, content)
		bets := ResolveBetsNums(meta, wire, 0, 2, 1)
		solo := ResolveSolo(meta, wire, bets)
		fmt.Printf("rule=%s mode=%s content=%q wire=%q bets=%d solo=%v needs=%v\n", c.sub, InferBetMode(meta), content, wire, bets, solo, NeedsSoloForRule(meta, wire))
	}
}

func TestNeedsSolo_renxuanZu3Fs(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选三"})
	meta := ParseRuleMeta("ssc_std", "g011", "83", "组三复式", "任选", seg, "83")
	content := SampleGroupContent(meta)
	wire := FormatBetContentForRule(meta, content)
	if bets := ResolveBetsNums(meta, wire, 0, 2, 1); bets != 2 {
		t.Fatalf("bets=%d want 2 content=%q wire=%q", bets, content, wire)
	}
	if !ResolveSolo(meta, wire, 2) {
		t.Fatalf("zu3 2注应 solo=true wire=%q", wire)
	}
}

func TestNeedsSolo_renxuanHunhe(t *testing.T) {
	seg, _ := json.Marshal(map[string]string{"guajiGroup": "任选", "guajiTeam": "任选三"})
	meta := ParseRuleMeta("ssc_std", "g011", "87", "混合组选", "任选", seg, "87")
	wire := "万千个|112"
	if !isRenxuanMeta(meta) {
		t.Fatal("not renxuan")
	}
	if mode := InferBetMode(meta); mode != "hunhe" {
		t.Fatalf("mode=%q want hunhe", mode)
	}
	if n := CountBetNums(meta, wire); n != 1 {
		t.Fatalf("bets=%d want 1", n)
	}
	if !renxuanNeedsSoloTrue(meta, "hunhe", 1) {
		t.Fatal("renxuanNeedsSoloTrue false")
	}
	if !NeedsSoloForRule(meta, wire) {
		t.Fatal("NeedsSoloForRule false")
	}
}
