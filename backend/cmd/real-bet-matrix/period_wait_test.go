package main

import (
	"context"
	"testing"
	"time"

	"caipiao/backend/internal/guajibet"
)

func TestIsTransientPeriodErr(t *testing.T) {
	if !isTransientPeriodErr(guajibet.ErrPeriodClosed) {
		t.Fatal("ErrPeriodClosed should be transient")
	}
	if isTransientPeriodErr(nil) {
		t.Fatal("nil should not be transient")
	}
}

func TestSleepCtx_cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepCtx(ctx, time.Second); err == nil {
		t.Fatal("expected cancel error")
	}
}
