package schemes

import (
	"context"
	"errors"
	"testing"

	"caipiao/backend/internal/guajibet"
)

type stubAuthBalance struct {
	healthy bool
	amount  float64
	ok      bool
	err     error
}

func (s stubAuthBalance) HasHealthyAuthForMember(context.Context, string) (bool, error) {
	return s.healthy, nil
}

func (s stubAuthBalance) PrimaryUsableBalance(context.Context, string) (float64, bool, error) {
	return s.amount, s.ok, s.err
}

func (s stubAuthBalance) UsableBalance(_ context.Context, _ string, _ string) (float64, bool, error) {
	return s.amount, s.ok, s.err
}

func TestEnsureUsableBalanceForStart(t *testing.T) {
	svc := &Service{}

	t.Run("nil checker skips", func(t *testing.T) {
		if err := svc.ensureUsableBalanceForStart(context.Background(), "u1", 10, "USDT"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("guaji disabled skips", func(t *testing.T) {
		svc.authChecker = stubAuthBalance{ok: false}
		if err := svc.ensureUsableBalanceForStart(context.Background(), "u1", 10, "USDT"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("enough balance", func(t *testing.T) {
		svc.authChecker = stubAuthBalance{amount: 10, ok: true}
		if err := svc.ensureUsableBalanceForStart(context.Background(), "u1", 10, "USDT"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("insufficient", func(t *testing.T) {
		svc.authChecker = stubAuthBalance{amount: 1.99, ok: true}
		err := svc.ensureUsableBalanceForStart(context.Background(), "u1", 2, "USDT")
		if !errors.Is(err, ErrStartInsufficientFunds) {
			t.Fatalf("got %v, want ErrStartInsufficientFunds", err)
		}
	})

	t.Run("propagates auth error", func(t *testing.T) {
		svc.authChecker = stubAuthBalance{err: guajibet.ErrNoActiveAuth}
		err := svc.ensureUsableBalanceForStart(context.Background(), "u1", 2, "USDT")
		if !errors.Is(err, guajibet.ErrNoActiveAuth) {
			t.Fatalf("got %v, want ErrNoActiveAuth", err)
		}
	})
}
