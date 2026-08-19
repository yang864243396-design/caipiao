package guaji

import "context"

type strictBetPlacementContextKey struct{}

// PlaceLottBetStrict performs one placement request and requires that response
// to carry a unique order id. It does not run the ambiguous list lookup by
// period and amount.
func (c *Client) PlaceLottBetStrict(ctx context.Context, accessToken string, req LottBetRequest) (*LottBetResult, error) {
	return c.PlaceLottBet(context.WithValue(ctx, strictBetPlacementContextKey{}, true), accessToken, req)
}

func allowAmbiguousBetLookup(ctx context.Context) bool {
	strict, _ := ctx.Value(strictBetPlacementContextKey{}).(bool)
	return !strict
}
