package accountsvc

import (
	"errors"
	"testing"
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

func TestEnsureActiveAuthUnavailable(t *testing.T) {
	t.Parallel()
	var s *Service
	if err := s.EnsureActiveAuth(nil, "x"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v want ErrUnavailable", err)
	}
}
