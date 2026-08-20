package guaji

import "context"

// RequestProgress is an in-memory snapshot of one third-party HTTP request.
// It intentionally carries no request body, token, or member identity.
type RequestProgress struct {
	Operation      string
	Phase          string
	RequestWritten bool
}

type requestProgressObserverKey struct{}

// WithRequestProgressObserver installs a lightweight request progress callback.
// The callback may be invoked from net/http trace goroutines and must be safe
// for concurrent use.
func WithRequestProgressObserver(ctx context.Context, observer func(RequestProgress)) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if observer == nil {
		return ctx
	}
	return context.WithValue(ctx, requestProgressObserverKey{}, observer)
}

// ReportRequestProgress publishes a trace update to the observer carried by ctx.
// It is used by HTTP adapters and test transports; it performs no I/O.
func ReportRequestProgress(ctx context.Context, progress RequestProgress) {
	if ctx == nil {
		return
	}
	observer, _ := ctx.Value(requestProgressObserverKey{}).(func(RequestProgress))
	if observer != nil {
		observer(progress)
	}
}
