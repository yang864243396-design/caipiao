package accountsvc

import (
	"context"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
)

type strictSingleAttemptContextKey struct{}

// PlaceRealBetOnce permits at most one POST to the provider. Authentication may
// be refreshed before the POST, but a token error from the placement response
// is returned without submitting the bet again.
func (s *Service) PlaceRealBetOnce(ctx context.Context, memberAccount string, req guajibet.Request) (guajibet.Result, error) {
	return s.PlaceRealBet(context.WithValue(ctx, strictSingleAttemptContextKey{}, true), memberAccount, req)
}

func isStrictSingleAttempt(ctx context.Context) bool {
	strict, _ := ctx.Value(strictSingleAttemptContextKey{}).(bool)
	return strict
}

func (s *Service) placeLottBetForContext(ctx context.Context, token string, req guaji.LottBetRequest) (*guaji.LottBetResult, error) {
	if isStrictSingleAttempt(ctx) {
		return s.guaji.PlaceLottBetStrict(ctx, token, req)
	}
	return s.guaji.PlaceLottBet(ctx, token, req)
}
