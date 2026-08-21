package guaji

import "context"

type strictBetPlacementContextKey struct{}

// PlaceLottBetStrict performs exactly one placement POST. If the successful
// response omits its id, it may resolve one unique order from the read-only
// list using the complete request fingerprint; it never guesses by period and
// amount alone.
func (c *Client) PlaceLottBetStrict(ctx context.Context, accessToken string, req LottBetRequest) (*LottBetResult, error) {
	return c.PlaceLottBet(context.WithValue(ctx, strictBetPlacementContextKey{}, true), accessToken, req)
}

func allowAmbiguousBetLookup(ctx context.Context) bool {
	strict, _ := ctx.Value(strictBetPlacementContextKey{}).(bool)
	return !strict
}
