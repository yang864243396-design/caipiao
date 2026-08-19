package schemes

import (
	"context"
	"testing"
	"time"
)

func TestStrategyProcessorNotifyDrawUsesExactScope(t *testing.T) {
	called := make(chan [2]string, 1)
	processor := &StrategyProcessor{recoverScopedFn: func(_ context.Context, lotteryCode, periodNo string) error {
		called <- [2]string{lotteryCode, periodNo}
		return nil
	}}
	processor.NotifyDraw(context.Background(), " lottery-X ", " period-Z ")
	defer processor.Close()
	select {
	case got := <-called:
		if got != [2]string{"lottery-X", "period-Z"} {
			t.Fatalf("scope=%q/%q", got[0], got[1])
		}
	case <-time.After(time.Second):
		t.Fatal("scoped recovery was not called")
	}
}
