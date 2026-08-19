package guaji_test

import (
	"context"
	"testing"
)

func TestPlaceLottBetStrictDoesNotGuessOrderByPeriodAndAmount(t *testing.T) {
	const period = "115202608190001"
	c, _ := fakeGuaji(t,
		map[string]any{"code": 201, "periods": period},
		map[string]any{"code": 201, "data": []any{
			map[string]any{"id": 999991, "game_id": 29, "periods": period, "bet_amount": 12},
		}},
	)
	if result, err := c.PlaceLottBetStrict(context.Background(), wireToken, sampleReq()); err == nil {
		t.Fatalf("strict placement guessed an ambiguous order: %+v", result)
	}
}
