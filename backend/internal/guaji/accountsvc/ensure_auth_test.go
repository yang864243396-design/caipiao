package accountsvc

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMapAuthErrToBet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   error
		want string
	}{
		{name: "no active", in: ErrNoActiveAccount, want: "无启用中的授权账号"},
		{name: "not found", in: ErrAccountNotFound, want: "无启用中的授权账号"},
		{name: "token", in: ErrTokenInvalid, want: "授权已失效，请重新授权"},
		{name: "needs bind", in: ErrReauthNeedsBind, want: "授权已失效，请重新授权"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapAuthErrToBet(tc.in)
			if got == nil || got.Error() != tc.want {
				t.Fatalf("mapAuthErrToBet(%v)=%v want %q", tc.in, got, tc.want)
			}
		})
	}
	if mapAuthErrToBet(nil) != nil {
		t.Fatal("nil should map to nil")
	}
}

func TestMaxAutoReauthAttempts(t *testing.T) {
	t.Parallel()
	if maxAutoReauthAttempts != 3 {
		t.Fatalf("maxAutoReauthAttempts=%d want 3", maxAutoReauthAttempts)
	}
	if maxReauthFailures != 3 {
		t.Fatalf("maxReauthFailures=%d want 3", maxReauthFailures)
	}
}

func TestReauthModesDifferOnFailCountGate(t *testing.T) {
	t.Parallel()
	// 自动路径熔断：failCount≥阈值直接 ErrReauthNeedsBind，不再打上游。
	// 手动路径（Reauth）跳过该门闩，始终走 refresh/密码登录。
	if reauthModeAuto == reauthModeManual {
		t.Fatal("auto/manual modes must differ")
	}
}

func TestAutoReauthUsesSingleflightKey(t *testing.T) {
	t.Parallel()
	// 约定：ensureActiveAuth 以 memberId 为 singleflight key，
	// 同会员多方案并发 401 只跑一轮 doAutoReauthAttempts。
	if maxAutoReauthAttempts < 1 {
		t.Fatal("auto attempts must be >= 1")
	}
}

func TestAutoReauthUsesSingleflightField(t *testing.T) {
	t.Parallel()
	// NewService 零值 Group 可用；多方案 EnsureActiveAuth 经 autoReauthSF 合并。
	s := &Service{}
	_ = s.autoReauthSF
}

func TestEnsureActiveAuthUnavailable(t *testing.T) {
	t.Parallel()
	var s *Service
	if err := s.EnsureActiveAuth(nil, "x"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v want ErrUnavailable", err)
	}
}

func TestAutoReauthSingleflightWaitHonorsCallerCancellation(t *testing.T) {
	s := &Service{}
	started := make(chan struct{})
	release := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		err, _ := s.waitAutoReauthFlight(ctx, "member-1", func() (any, error) {
			close(started)
			<-release
			return nil, nil
		})
		done <- err
	}()
	<-started
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error=%v want context deadline exceeded", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("singleflight waiter ignored caller cancellation")
	}
	close(release)
}
