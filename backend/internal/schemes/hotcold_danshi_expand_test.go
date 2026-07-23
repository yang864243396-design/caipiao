package schemes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHotColdQian3DanshiPoolExpandsBeforeBet(t *testing.T) {
	raw := []byte(`{
		"subId":"2","typeId":"g001","betMode":"danshi","runTypeId":"hot_cold_warm",
		"subPlayId":"2","playTypeId":"g001","playTemplate":"ssc_std",
		"hotColdWarm":{"pool":["1,9","1,9","1,9"],"strategy":"keep","totalPeriods":20},
		"schemeGroups":["1,9","1,9","1,9"]
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if cfg.Play.BetMode != "danshi" {
		t.Fatalf("betMode=%q want danshi; play=%+v", cfg.Play.BetMode, cfg.Play)
	}
	if cfg.Play.SegmentLen != 3 {
		t.Fatalf("segmentLen=%d want 3", cfg.Play.SegmentLen)
	}
	// 无出号类型时回退 pool 拼接
	content := ""
	if cfg.HotCold != nil && len(cfg.HotCold.Pool) > 0 {
		content = strings.Join(cfg.HotCold.Pool, "\n")
	}
	expanded := normalizeZhixuanDanshiContent(cfg.Play, content)
	if expanded == content || !strings.Contains(expanded, "111") {
		t.Fatalf("expand failed: in=%q out=%q", content, expanded)
	}
	payload, err := NormalizeBetPayload(BetPayload{
		PlayTemplate: cfg.Play.PlayTemplate,
		TypeID:       cfg.Play.PlayTypeID,
		SubID:        cfg.Play.SubPlayID,
		BetMode:      cfg.Play.BetMode,
		PlayMethod:   "前三码",
		GroupContent: expanded,
	})
	if err != nil {
		t.Fatalf("NormalizeBetPayload: %v (content=%q)", err, expanded)
	}
	var p BetPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.GroupContent != expanded {
		t.Fatalf("payload content=%q want %q", p.GroupContent, expanded)
	}
}

func TestNormalizeBetPayloadExpandsPositionPoolInline(t *testing.T) {
	_, err := NormalizeBetPayload(BetPayload{
		PlayTemplate: "ssc_std",
		TypeID:       "g001",
		SubID:        "2",
		BetMode:      "danshi",
		PlayMethod:   "前三码",
		GroupContent: "1,9\n1,9\n1,9",
	})
	if err != nil {
		t.Fatalf("should expand then validate: %v", err)
	}
}
