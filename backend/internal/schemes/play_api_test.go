package schemes

import (
	"encoding/json"
	"testing"
)

func TestBuildBetPayloadDeterministic(t *testing.T) {
	a := BuildBetPayload("定位胆万位", "BO1001")
	b := BuildBetPayload("定位胆万位", "BO1001")
	if string(a) != string(b) {
		t.Fatalf("payload not deterministic: %s vs %s", a, b)
	}
}

func TestEvaluateBetPayloadDingwei(t *testing.T) {
	payload := BuildBetPayload("定位胆万位", "BO-eval-1")
	balls := []string{"3", "9", "2", "7", "5"}
	hit, odds := EvaluateBetPayload(payload, balls)
	if odds != oddsDingwei {
		t.Fatalf("odds=%v", odds)
	}
	_ = hit
}

func TestEvaluateBetPayloadClientPicks(t *testing.T) {
	raw, err := NormalizeBetPayload(BetPayload{
		PlayMethod:   "定位胆万位",
		GroupContent: "3,9",
	})
	if err != nil {
		t.Fatal(err)
	}
	balls := []string{"3", "9", "2", "7", "5"}
	hit, _ := EvaluateBetPayload(raw, balls)
	if !hit {
		t.Fatal("expected hit on 万位 3")
	}
}

func TestConfigFromPlayMethodSixing(t *testing.T) {
	cfg := configFromPlayMethod("后四直选复式")
	if cfg["playTypeId"] != "sixing" || cfg["subPlayId"] != "zhixuan_fs" {
		t.Fatalf("cfg=%v", cfg)
	}
}

func TestConfigFromPlayMethodDingweiWan(t *testing.T) {
	cfg := configFromPlayMethod("定位胆万位")
	if cfg["playTypeId"] != "dingwei" || cfg["subPlayId"] != "dingwei_wan" {
		t.Fatalf("cfg=%v", cfg)
	}
}

func TestNormalizeBetPayload(t *testing.T) {
	raw, err := NormalizeBetPayload(BetPayload{
		PlayMethod:   "定位胆万位",
		GroupContent: "1,3,7",
	})
	if err != nil {
		t.Fatal(err)
	}
	var p BetPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.GroupContent != "1,3,7" {
		t.Fatalf("content=%q", p.GroupContent)
	}
}

func TestNormalizeBetPayloadEmpty(t *testing.T) {
	_, err := NormalizeBetPayload(BetPayload{PlayMethod: "定位胆万位"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnsureBetPayloadUsesClientContent(t *testing.T) {
	client, _ := json.Marshal(BetPayload{PlayMethod: "定位胆万位", GroupContent: "2,5,8"})
	out := EnsureBetPayload(client, "定位胆万位", "BO1")
	var p BetPayload
	_ = json.Unmarshal(out, &p)
	if p.GroupContent != "2,5,8" {
		t.Fatalf("got %q", p.GroupContent)
	}
}

func TestCalcOrderPnL(t *testing.T) {
	pnl := CalcOrderPnL(2, true, oddsDingwei)
	if pnl != 18 {
		t.Fatalf("pnl=%v", pnl)
	}
	if CalcOrderPnL(2, false, oddsDingwei) != -2 {
		t.Fatal("miss pnl")
	}
}
