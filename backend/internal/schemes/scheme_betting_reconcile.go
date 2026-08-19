package schemes

import (
	"context"
	"errors"

	"caipiao/backend/internal/schemebetting"
)

func (w *Worker) ResolveUnknownEventBet(ctx context.Context, outboxID int64, actor, reason string, resolution schemebetting.UnknownResolution) error {
	if w == nil || w.unknownResolver == nil {
		return errors.New("scheme betting reconciliation is unavailable")
	}
	return w.unknownResolver.ResolveUnknownEventBet(ctx, outboxID, actor, reason, resolution)
}
