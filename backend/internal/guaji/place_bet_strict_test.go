package guaji_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
)

func TestPlaceLottBetStrictResolvesOneExactOrderWithoutSecondPost(t *testing.T) {
	const period = "115202608190001"
	c, capture := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{
				"id": 999991, "game_id": 29, "periods": period, "bet_amount": 12, "currency": 3,
				"bet_contents": []any{map[string]any{
					"rule_id": "99", "bet_content": ",,9,9,9", "amount_unit": 2,
					"bets_nums": 2, "multiple": 3, "solo": false, "bet_amount": 12,
				}},
			},
			map[string]any{
				"id": 999992, "game_id": 29, "periods": period, "bet_amount": 12, "currency": 3,
				"bet_contents": []any{map[string]any{
					"rule_id": "13", "bet_content": ",,1,3,5", "amount_unit": 2,
					"bets_nums": 2, "multiple": 3, "solo": false, "bet_amount": 12,
				}},
			},
		}},
	)
	result, err := c.PlaceLottBetStrict(context.Background(), wireToken, sampleReq())
	if err != nil {
		t.Fatalf("strict placement did not resolve the unique exact order: %v", err)
	}
	if result.ThirdPartyBetID != "999992" {
		t.Fatalf("third-party order id=%q, want 999992", result.ThirdPartyBetID)
	}
	if got := capture.calls.Load(); got != 1 {
		t.Fatalf("placement POST calls=%d, want exactly 1", got)
	}
	if got := capture.listCalls.Load(); got != 1 {
		t.Fatalf("order-list GET calls=%d, want 1", got)
	}
}

func TestPlaceLottBetStrictRejectsTwoExactOrdersAsAmbiguous(t *testing.T) {
	const period = "115202608190002"
	exactContent := func() []any {
		return []any{map[string]any{
			"rule_id": "13", "bet_content": ",,1,3,5", "amount_unit": 2,
			"bets_nums": 2, "multiple": 3, "solo": false, "bet_amount": 12,
		}}
	}
	c, capture := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{"id": 999993, "game_id": 29, "periods": period, "bet_amount": 12, "currency": 3, "bet_contents": exactContent()},
			map[string]any{"id": 999994, "game_id": 29, "periods": period, "bet_amount": 12, "currency": 3, "bet_contents": exactContent()},
		}},
	)
	result, err := c.PlaceLottBetStrict(context.Background(), wireToken, sampleReq())
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("strict placement accepted ambiguous exact orders: result=%+v err=%v", result, err)
	}
	if got := capture.calls.Load(); got != 1 {
		t.Fatalf("placement POST calls=%d, want exactly 1", got)
	}
}

func TestPlaceLottBetStrictRejectsDifferentMultiple(t *testing.T) {
	const period = "115202608190003"
	c, capture := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{
				"id": 999995, "game_id": 29, "periods": period, "bet_amount": 12, "currency": 3,
				"bet_contents": []any{map[string]any{
					"rule_id": "13", "bet_content": ",,1,3,5", "amount_unit": 2,
					"bets_nums": 2, "multiple": 2, "solo": false, "bet_amount": 12,
				}},
			},
		}},
	)
	if result, err := c.PlaceLottBetStrict(context.Background(), wireToken, sampleReq()); err == nil {
		t.Fatalf("strict placement accepted an order with a different multiple: %+v", result)
	}
	if got := capture.calls.Load(); got != 1 {
		t.Fatalf("placement POST calls=%d, want exactly 1", got)
	}
}

func TestPlaceLottBetStrictSupportsNestedBetContentShape(t *testing.T) {
	const period = "115202608190004"
	c, _ := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{
				"id": 999996, "game_id": 29, "periods": period, "bet_amount": 12, "currency": 3,
				"bet_content": map[string]any{"bet_content": map[string]any{
					"rule_id": "13", "bet_content": ",,1,3,5", "amount_unit": 2,
					"bets_nums": 2, "multiple": 3, "solo": false, "bet_amount": 12,
				}},
			},
		}},
	)
	result, err := c.PlaceLottBetStrict(context.Background(), wireToken, sampleReq())
	if err != nil || result.ThirdPartyBetID != "999996" {
		t.Fatalf("nested exact order was not resolved: result=%+v err=%v", result, err)
	}
}

func TestPlaceLottBetStrictIgnoresProviderContentOrder(t *testing.T) {
	const period = "115202608190006"
	req := sampleReq()
	second := req.BetContents[0]
	second.RuleID = "14"
	second.BetContent = ",,2,4,6"
	req.BetContents = append(req.BetContents, second)
	c, _ := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{
				"id": 999999, "game_id": 29, "periods": period, "bet_amount": 24, "currency": 3,
				"bet_contents": []any{webBetContent(second), webBetContent(req.BetContents[0])},
			},
		}},
	)
	result, err := c.PlaceLottBetStrict(context.Background(), wireToken, req)
	if err != nil || result.ThirdPartyBetID != "999999" {
		t.Fatalf("reordered exact contents were not resolved: result=%+v err=%v", result, err)
	}
}

func TestPlaceLottBetStrictDoesNotDoubleCountDualProviderShapes(t *testing.T) {
	const period = "115202608190007"
	content := sampleReq().BetContents[0]
	c, _ := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{
				"id": 1000000, "game_id": 29, "periods": period, "bet_amount": 12, "currency": 3,
				"bet_contents": []any{webBetContent(content)},
				"bet_content":  map[string]any{"bet_content": webBetContent(content)},
			},
		}},
	)
	result, err := c.PlaceLottBetStrict(context.Background(), wireToken, sampleReq())
	if err != nil || result.ThirdPartyBetID != "1000000" {
		t.Fatalf("dual-shape exact order was not resolved: result=%+v err=%v", result, err)
	}
}

func TestPlaceLottBetStrictCoalescesConcurrentOrderListReads(t *testing.T) {
	const period = "115202608190005"
	secondReq := sampleReq()
	secondReq.BetContents[0].RuleID = "14"
	secondReq.BetContents[0].BetContent = ",,2,4,6"
	c, capture := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			exactWebBetRow(999997, period, sampleReq().BetContents[0]),
			exactWebBetRow(999998, period, secondReq.BetContents[0]),
		}},
	)
	capture.listDelayNanos.Store(int64(75 * time.Millisecond))

	start := make(chan struct{})
	var wg sync.WaitGroup
	type outcome struct {
		id  string
		err error
	}
	results := make(chan outcome, 2)
	for _, req := range []guaji.LottBetRequest{sampleReq(), secondReq} {
		wg.Add(1)
		go func(request guaji.LottBetRequest) {
			defer wg.Done()
			<-start
			result, err := c.PlaceLottBetStrict(context.Background(), wireToken, request)
			if result == nil {
				results <- outcome{err: err}
				return
			}
			results <- outcome{id: result.ThirdPartyBetID, err: err}
		}(req)
	}
	close(start)
	wg.Wait()
	close(results)

	seen := map[string]bool{}
	for result := range results {
		if result.err != nil {
			t.Fatalf("strict concurrent placement: %v", result.err)
		}
		seen[result.id] = true
	}
	if !seen["999997"] || !seen["999998"] {
		t.Fatalf("resolved order ids=%v, want both exact orders", seen)
	}
	if got := capture.calls.Load(); got != 2 {
		t.Fatalf("placement POST calls=%d, want one per request", got)
	}
	if got := capture.listCalls.Load(); got != 1 {
		t.Fatalf("concurrent order-list GET calls=%d, want 1 shared read", got)
	}
}

func TestPlaceLottBetStrictCancelOneSharedListWaiterDoesNotCancelAnother(t *testing.T) {
	const period = "115202608190008"
	secondReq := sampleReq()
	secondReq.BetContents[0].RuleID = "14"
	secondReq.BetContents[0].BetContent = ",,2,4,6"
	c, capture := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			exactWebBetRow(1000001, period, sampleReq().BetContents[0]),
			exactWebBetRow(1000002, period, secondReq.BetContents[0]),
		}},
	)
	capture.listDelayNanos.Store(int64(150 * time.Millisecond))

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type outcome struct {
		name string
		id   string
		err  error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	go func() {
		<-start
		result, err := c.PlaceLottBetStrict(cancelCtx, wireToken, sampleReq())
		if result == nil {
			results <- outcome{name: "cancelled", err: err}
			return
		}
		results <- outcome{name: "cancelled", id: result.ThirdPartyBetID, err: err}
	}()
	go func() {
		<-start
		result, err := c.PlaceLottBetStrict(context.Background(), wireToken, secondReq)
		if result == nil {
			results <- outcome{name: "survivor", err: err}
			return
		}
		results <- outcome{name: "survivor", id: result.ThirdPartyBetID, err: err}
	}()
	close(start)

	deadline := time.Now().Add(time.Second)
	for (capture.calls.Load() < 2 || capture.listCalls.Load() < 1) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if capture.calls.Load() != 2 || capture.listCalls.Load() != 1 {
		t.Fatalf("shared lookup did not start: POST=%d GET=%d", capture.calls.Load(), capture.listCalls.Load())
	}
	cancel()

	got := map[string]outcome{}
	for range 2 {
		result := <-results
		got[result.name] = result
	}
	if !errors.Is(got["cancelled"].err, context.Canceled) {
		t.Fatalf("cancelled waiter err=%v, want context.Canceled", got["cancelled"].err)
	}
	if got["survivor"].err != nil || got["survivor"].id != "1000002" {
		t.Fatalf("surviving waiter result=%+v", got["survivor"])
	}
	if capture.listCalls.Load() != 1 {
		t.Fatalf("shared order-list GET calls=%d, want 1", capture.listCalls.Load())
	}
}

func exactWebBetRow(id int64, period string, content guaji.LottBetContent) map[string]any {
	return map[string]any{
		"id": id, "game_id": 29, "periods": period, "bet_amount": content.BetAmount, "currency": 3,
		"bet_contents": []any{webBetContent(content)},
	}
}

func webBetContent(content guaji.LottBetContent) map[string]any {
	return map[string]any{
		"rule_id": content.RuleID, "bet_content": content.BetContent, "amount_unit": content.AmountUnit,
		"bets_nums": content.BetsNums, "multiple": content.Multiple, "solo": content.Solo, "bet_amount": content.BetAmount,
	}
}
