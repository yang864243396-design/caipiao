package schemes

import "context"

type StrategyLeaseFence struct {
	ShardNo int32
	Owner   string
	Epoch   int64
}

type strategyLeaseFenceContextKey struct{}

func WithStrategyLeaseFence(ctx context.Context, fence StrategyLeaseFence) context.Context {
	return context.WithValue(ctx, strategyLeaseFenceContextKey{}, fence)
}

func strategyLeaseFenceFromContext(ctx context.Context) (StrategyLeaseFence, bool) {
	fence, ok := ctx.Value(strategyLeaseFenceContextKey{}).(StrategyLeaseFence)
	return fence, ok
}
