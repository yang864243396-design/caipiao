package guaji

import (
	"encoding/json"
	"testing"
)

func TestParseLottBetResultTopLevelPeriods(t *testing.T) {
	raw := []byte(`{"account":{},"periods":"115202606160196","code":201,"message":"下注成功"}`)
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	res := parseLottBetResult(env, raw)
	if res.Periods != "115202606160196" {
		t.Fatalf("periods=%q", res.Periods)
	}
}

func TestWebBetToSettlementWin(t *testing.T) {
	s := webBetToSettlement(&WebBetRecord{
		ID:           1,
		BetAmount:    10,
		NetAmount:    4.31,
		PayoutAmount: 14.31,
		Settled:      true,
	})
	if !s.Settled || s.Status != "win" || s.Pnl != 4.31 {
		t.Fatalf("settlement=%+v", s)
	}
}

func TestWebBetToSettlementLose(t *testing.T) {
	s := webBetToSettlement(&WebBetRecord{
		ID:        2,
		BetAmount: 10,
		NetAmount: -10,
		Settled:   true,
	})
	if s.Status != "lose" || s.Pnl != -10 {
		t.Fatalf("settlement=%+v", s)
	}
}

func TestWebBetToSettlementTieRefundIsLose(t *testing.T) {
	// 龙虎和局退本：net=0、payout=本金 → 应记挂，不能因 payout>0 判赢
	s := webBetToSettlement(&WebBetRecord{
		ID:           3,
		BetAmount:    4,
		NetAmount:    0,
		PayoutAmount: 4,
		Settled:      true,
	})
	if s.Status != "lose" || s.Pnl != 0 {
		t.Fatalf("tie refund settlement=%+v", s)
	}
}

func TestDecodeWebBetListStringCode(t *testing.T) {
	raw := []byte(`{"data":[{"id":398515,"game_id":29,"periods":"p1","bet_amount":10,"settled":true}],"code":"0"}`)
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env.Code.Int() != 0 {
		t.Fatalf("code=%d", env.Code.Int())
	}
	items, err := decodeWebBetList(env.Data, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != 398515 {
		t.Fatalf("items=%+v", items)
	}
}

func TestDecodeWebBetListDataArray(t *testing.T) {
	raw := []byte(`{"data":[{"id":398515,"game_id":29,"periods":"p1","bet_amount":10,"settled":true}]}`)
	items, err := decodeWebBetList(json.RawMessage(`[{"id":398515,"game_id":29,"periods":"p1","bet_amount":10,"settled":true}]`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != 398515 {
		t.Fatalf("items=%+v", items)
	}
}
