package guaji

import "testing"

func TestIsIgnoredDrawGame(t *testing.T) {
	cases := []struct {
		id, name string
		want     bool
	}{
		{"fc3d", "福彩3D", true},
		{"pl35", "排列35", true},
		{"fc_pl3d", "福彩排列3D", true},
		{"lottery_log033", "波场极速彩", false},
		{"tron_lhc", "波场六合彩", false},
	}
	for _, c := range cases {
		if got := IsIgnoredDrawGame(c.id, c.name); got != c.want {
			t.Errorf("IsIgnoredDrawGame(%q,%q)=%v want %v", c.id, c.name, got, c.want)
		}
	}
}

// 真实抓包样例（wss://hash.iyes.dev/ws）：一条 lottery_v2_broadcast 含多彩种线 + 多玩法号码。
func TestParseDrawEventsRealBroadcast(t *testing.T) {
	raw := []byte(`{"send":true,"message":{"type":"lottery_v2_broadcast","block_num":83441446,"created":"2026-06-09T08:25:30+00:00","last5_num":"94819","last11_5_num":"06,04,08,01,09","last_pk10_num":"03,06,01,04,08,05,10,09,02,07","last_k3_num":"2,3,5","lhc_num":"49,21,18,04,39,36,34","lottery_log101":{"periods":"10113906900723","next_periods":"10113906900724"},"lottery_log033":{"periods":"105202606091971","next_periods":"105202606091972"}}}`)
	events := ParseDrawEvents(raw)
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	byKey := map[string]DrawEvent{}
	for _, e := range events {
		byKey[e.GameKey] = e
	}
	log033, ok := byKey["lottery_log033"]
	if !ok || log033.Periods != "105202606091971" {
		t.Fatalf("log033 %+v", log033)
	}
	// ssc_std 取 last5_num 连写拆 5 位
	if balls := log033.Balls.BallsFor("ssc_std"); len(balls) != 5 || balls[0] != "9" {
		t.Fatalf("ssc balls %+v", balls)
	}
	// syxw_std 取 last11_5_num
	if balls := log033.Balls.BallsFor("syxw_std"); len(balls) != 5 || balls[0] != "06" {
		t.Fatalf("syxw balls %+v", balls)
	}
	// lhc_std 取 lhc_num 7 个
	if balls := log033.Balls.BallsFor("lhc_std"); len(balls) != 7 {
		t.Fatalf("lhc balls %+v", balls)
	}
	// pk10_std 取 last_pk10_num 10 个
	if balls := log033.Balls.BallsFor("pk10_std"); len(balls) != 10 {
		t.Fatalf("pk10 balls %+v", balls)
	}
	// k3_std 取 last_k3_num 3 个，和 10
	if balls := log033.Balls.BallsFor("k3_std"); len(balls) != 3 || SumBalls(balls) != 10 {
		t.Fatalf("k3 balls %+v", balls)
	}
}

func TestParseDrawEventsSkipsHeartbeat(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(`{"message":{"type":"block-new","block_num":"1","now":"x"}}`),
		[]byte(`{"message":{"type":"long_dragon_update"}}`),
		[]byte(`{"message":{"type":"fc3d_lottery_v2_broadcast","last_fc3d_num":"6,5,8","fc3d_lottery_log":{"periods":"42438"}}}`),
		[]byte(`{"type":"ping"}`),
	} {
		if events := ParseDrawEvents(raw); len(events) != 0 {
			t.Fatalf("expected skip for %s, got %d", raw, len(events))
		}
	}
}
