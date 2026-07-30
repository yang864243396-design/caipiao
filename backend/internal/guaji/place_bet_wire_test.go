package guaji_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
)

// 正式盘下注是唯一会真花钱的链路，没法拿真第三方跑自动化。
// 但「发出去的请求长什么样」「回来的各种响应怎么处理」完全可以测：
// 把 HTTPBase 指向一个自己控制的假服务器，整条链路照常走一遍。
//
// 极速彩那个 bug 就是发错了 game_id——注单成功下出去了，只是下到了别的彩种上。
// 在此之前从没有一条测试断言过我们究竟发了什么，所以它能潜伏那么久。

const wireToken = "tok-wire-test"

// betCapture 记录假第三方收到的下单请求。
type betCapture struct {
	calls   atomic.Int32
	rawBody atomic.Value // string
	authHdr atomic.Value // string
}

func (c *betCapture) body(t *testing.T) map[string]any {
	t.Helper()
	s, _ := c.rawBody.Load().(string)
	if s == "" {
		t.Fatal("假第三方没收到任何下单请求体")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("解析请求体: %v (raw=%s)", err, s)
	}
	return m
}

func (c *betCapture) rawJSON() string {
	s, _ := c.rawBody.Load().(string)
	return s
}

// fakeGuaji 起一个假的第三方。placeResp 是 /api/web_bets/lott 的响应，
// listResp 是回退查单用的 /api/web_bets/ 列表响应。
func fakeGuaji(t *testing.T, placeResp, listResp any) (*guaji.Client, *betCapture) {
	t.Helper()
	cap := &betCapture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/web_bets/lott":
			cap.calls.Add(1)
			cap.authHdr.Store(r.Header.Get("Authorization"))
			raw, _ := io.ReadAll(r.Body)
			cap.rawBody.Store(string(raw))
			_ = json.NewEncoder(w).Encode(placeResp)
		case strings.HasPrefix(r.URL.Path, "/api/web_bets/"):
			if listResp == nil {
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 201, "data": []any{}})
				return
			}
			_ = json.NewEncoder(w).Encode(listResp)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := guaji.NewClient(guaji.Config{
		Enabled:     true,
		HTTPBase:    srv.URL,
		AuthBase:    srv.URL,
		WSBase:      "wss://example.test",
		HTTPTimeout: 5 * time.Second,
	})
	return c, cap
}

// sampleReq 一笔典型的正式盘请求：时时彩后三直选，2 注、每注 2 元、倍数 3。
func sampleReq() guaji.LottBetRequest {
	return guaji.LottBetRequest{
		GameID:   29,
		Currency: 3,
		AutoType: "platform",
		BetContents: []guaji.LottBetContent{{
			RuleID:     "13",
			BetContent: ",,1,3,5",
			AmountUnit: 2,
			BetsNums:   2,
			Multiple:   3,
			BetAmount:  12,
		}},
	}
}

// TestPlaceLottBetSendsExpectedWireFormat 发出去的请求体必须与接口文档 §11 一致。
func TestPlaceLottBetSendsExpectedWireFormat(t *testing.T) {
	c, cap := fakeGuaji(t, map[string]any{
		"code": 201, "message": "下注成功",
		"data": map[string]any{"id": 398515, "periods": "115202606160196"},
	}, nil)

	if _, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq()); err != nil {
		t.Fatalf("下单: %v", err)
	}

	body := cap.body(t)
	if got := body["game_id"]; got != float64(29) {
		t.Errorf("game_id = %v，期望 29——发错 game_id 会把注下到别的彩种上", got)
	}
	if got := body["currency"]; got != float64(3) {
		t.Errorf("currency = %v，期望 3(cny)", got)
	}
	if got := body["auto_type"]; got != "platform" {
		t.Errorf("auto_type = %v", got)
	}

	contents, ok := body["bet_contents"].([]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("bet_contents = %v，期望 1 条", body["bet_contents"])
	}
	item := contents[0].(map[string]any)
	for _, tc := range []struct {
		field string
		want  any
	}{
		{"rule_id", "13"},
		{"bet_content", ",,1,3,5"},
		{"amount_unit", float64(2)},
		{"bets_nums", float64(2)},
		{"multiple", float64(3)},
		{"bet_amount", float64(12)},
		{"solo", false},
	} {
		if got := item[tc.field]; got != tc.want {
			t.Errorf("bet_contents[0].%s = %v，期望 %v", tc.field, got, tc.want)
		}
	}

	if got, _ := cap.authHdr.Load().(string); got != "Bearer "+wireToken {
		t.Errorf("Authorization = %q", got)
	}
}

// TestPlaceLottBetFillsDefaults 未加倍时 bet_multiple 必须是 []，不能是 null。
func TestPlaceLottBetFillsDefaults(t *testing.T) {
	c, cap := fakeGuaji(t, map[string]any{
		"code": 201, "data": map[string]any{"id": 1},
	}, nil)

	req := sampleReq()
	req.BetMultiple = nil // 不加倍
	req.AutoType = ""     // 不指定来源
	if _, err := c.PlaceLottBet(context.Background(), wireToken, req); err != nil {
		t.Fatalf("下单: %v", err)
	}

	raw := cap.rawJSON()
	if !strings.Contains(raw, `"bet_multiple":[]`) {
		t.Errorf("bet_multiple 未序列化为空数组，实际报文: %s", raw)
	}
	if strings.Contains(raw, `"bet_multiple":null`) {
		t.Error("bet_multiple 发成了 null——第三方按数组解析会直接拒单")
	}
	if got := cap.body(t)["auto_type"]; got != "platform" {
		t.Errorf("auto_type 缺省值 = %v，期望 platform", got)
	}
}

// TestPlaceLottBetGuardsNeverReachUpstream 参数不合法时必须在本地就拦下。
//
// 这条是安全线：守卫若放行，发出去的就是一笔真实的错误注单，
// 花的是真钱，且没有撤单接口可以挽回。
func TestPlaceLottBetGuardsNeverReachUpstream(t *testing.T) {
	for _, tc := range []struct {
		name    string
		disable bool
		token   string
		mutate  func(*guaji.LottBetRequest)
	}{
		{name: "开关关闭", disable: true, token: wireToken},
		{name: "空 token", token: ""},
		{name: "game_id 为 0", token: wireToken, mutate: func(r *guaji.LottBetRequest) { r.GameID = 0 }},
		{name: "投注内容为空", token: wireToken, mutate: func(r *guaji.LottBetRequest) { r.BetContents = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cap := fakeGuaji(t, map[string]any{"code": 201, "data": map[string]any{"id": 1}}, nil)
			if tc.disable {
				c = guaji.NewClient(guaji.Config{Enabled: false, HTTPBase: "http://127.0.0.1:1"})
			}
			req := sampleReq()
			if tc.mutate != nil {
				tc.mutate(&req)
			}
			if _, err := c.PlaceLottBet(context.Background(), tc.token, req); err == nil {
				t.Fatal("参数不合法却下单成功")
			}
			if n := cap.calls.Load(); n != 0 {
				t.Errorf("向第三方发出了 %d 次请求——不合法的请求绝不能触达上游", n)
			}
		})
	}
}

// TestPlaceLottBetUpstreamRejection 上游拒单要如实往上抛。
func TestPlaceLottBetUpstreamRejection(t *testing.T) {
	// 40055 是线上日志里最常见的一个：超出时间、投注失败
	c, _ := fakeGuaji(t, map[string]any{
		"code": 40055, "message": "超出时间,投注失败",
	}, nil)

	_, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq())
	if err == nil {
		t.Fatal("上游拒单却当成功返回——这会记下一笔并不存在的注单")
	}
	if !strings.Contains(err.Error(), "40055") && !strings.Contains(err.Error(), "超出时间") {
		t.Errorf("错误未透出上游原因: %v", err)
	}
}

// TestPlaceLottBetFallsBackToListLookup 上游只回 periods 不回 id 时，回列表里按期号找。
func TestPlaceLottBetFallsBackToListLookup(t *testing.T) {
	const period = "115202606160196"
	c, _ := fakeGuaji(t,
		map[string]any{"code": 201, "message": "下注成功", "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{"id": 398515, "game_id": 29, "periods": period, "bet_amount": 12},
		}},
	)

	res, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq())
	if err != nil {
		t.Fatalf("下单: %v", err)
	}
	if res.ThirdPartyBetID != "398515" {
		t.Errorf("注单号 = %q，期望从列表回查到 398515", res.ThirdPartyBetID)
	}
	if res.Periods != period {
		t.Errorf("期号 = %q", res.Periods)
	}
}

// TestPlaceLottBetLookupIgnoresOtherGame 回查时期号相同但 game_id 不同的注单不能认。
//
// 不同彩种的期号会撞号，认错了就会把别的彩种的注单号记到这笔上，
// 之后的派奖同步全部对错对象。
func TestPlaceLottBetLookupIgnoresOtherGame(t *testing.T) {
	const period = "115202606160196"
	c, _ := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			// 同期号但属于另一个彩种
			map[string]any{"id": 111111, "game_id": 31, "periods": period, "bet_amount": 12},
		}},
	)

	res, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq())
	if err == nil {
		t.Fatalf("认下了别的彩种的注单: %+v", res)
	}
}

// TestPlaceLottBetLookupIgnoresOtherAmount 金额对不上的注单也不能认。
func TestPlaceLottBetLookupIgnoresOtherAmount(t *testing.T) {
	const period = "115202606160196"
	c, _ := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{"id": 222222, "game_id": 29, "periods": period, "bet_amount": 999},
		}},
	)

	if res, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq()); err == nil {
		t.Fatalf("认下了金额对不上的注单: %+v", res)
	}
}

// TestPlaceLottBetRejectsWhenNoIDResolvable 既无 id 又无 periods 时必须报错。
// 悄悄返回一个空注单号，后续派奖同步就永远找不到这笔。
func TestPlaceLottBetRejectsWhenNoIDResolvable(t *testing.T) {
	c, _ := fakeGuaji(t, map[string]any{"code": 201, "message": "下注成功"}, nil)

	_, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq())
	if err == nil {
		t.Fatal("没拿到注单号却当成功——这笔注单之后永远对不上账")
	}
	if !strings.Contains(err.Error(), "bet id") {
		t.Errorf("错误信息未说明缺注单号: %v", err)
	}
}

// TestPlaceLottBetAcceptsNumericDataID 上游把 id 回成数字放在 data 里也要能认。
//
// LottBetResult.ThirdPartyBetID 是 string，数字 id 反序列化到 string 会失败，
// 这个字段就被静默丢掉。丢掉之后会退化成按期号回列表扫描（每次 1.2 秒起），
// 扫不到就对一笔「其实已经下出去了」的注单返回错误——钱花了、本地却没有记录，
// 而且平台没有撤单接口，事后无从挽回。
func TestPlaceLottBetAcceptsNumericDataID(t *testing.T) {
	// 故意不给列表回查任何结果：能拿到注单号就只可能来自 data.id 本身
	c, _ := fakeGuaji(t, map[string]any{
		"code": 201, "message": "下注成功",
		"data": map[string]any{"id": 398515, "periods": "115202606160196"},
	}, nil)

	res, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq())
	if err != nil {
		t.Fatalf("data.id 为数字时下单失败: %v", err)
	}
	if res.ThirdPartyBetID != "398515" {
		t.Errorf("注单号 = %q，期望 398515", res.ThirdPartyBetID)
	}
}

// TestPlaceLottBetAcceptsTopLevelID 线上实测的常见形态：id 与 periods 都在顶层。
func TestPlaceLottBetAcceptsTopLevelID(t *testing.T) {
	c, _ := fakeGuaji(t, map[string]any{
		"code": 201, "message": "下注成功",
		"id": 398517, "periods": "115202606160197",
	}, nil)

	res, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq())
	if err != nil {
		t.Fatalf("下单: %v", err)
	}
	if res.ThirdPartyBetID != "398517" || res.Periods != "115202606160197" {
		t.Errorf("结果 = %+v", res)
	}
}

// TestPlaceLottBetAcceptsStringID 上游把 id 回成字符串也要能认。
func TestPlaceLottBetAcceptsStringID(t *testing.T) {
	c, _ := fakeGuaji(t, map[string]any{
		"code": 201, "data": map[string]any{"id": "398516"},
	}, nil)

	res, err := c.PlaceLottBet(context.Background(), wireToken, sampleReq())
	if err != nil {
		t.Fatalf("下单: %v", err)
	}
	if res.ThirdPartyBetID != "398516" {
		t.Errorf("注单号 = %q，期望 398516", res.ThirdPartyBetID)
	}
}
