package guajibet

import "context"

// SingleAttemptPlacer is the production outbox boundary. Implementations must
// perform no automatic retry after the provider placement request starts.
type SingleAttemptPlacer interface {
	Enabled() bool
	PlaceRealBetOnce(ctx context.Context, memberAccount string, req Request) (Result, error)
}
