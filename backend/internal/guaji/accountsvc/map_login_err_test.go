package accountsvc

import (
	"errors"
	"testing"

	"caipiao/backend/internal/guaji"
)

func TestMapLoginErr(t *testing.T) {
	t.Parallel()

	if got := mapLoginErr(&guaji.APIError{Code: 400, Message: "bad"}); !errors.Is(got, ErrInvalidCredentials) {
		t.Fatalf("APIError => %v", got)
	}

	timeout := errors.New(`guaji http POST /auth/login: Post "https://www.v6hs1.com/auth/login": context deadline exceeded`)
	if got := mapLoginErr(timeout); !errors.Is(got, ErrGuajiUpstream) {
		t.Fatalf("timeout => %v", got)
	}

	dns := errors.New(`guaji ws dial: dial tcp: lookup www.v6hs1.com: i/o timeout`)
	if got := mapLoginErr(dns); !errors.Is(got, ErrGuajiUpstream) {
		t.Fatalf("dns timeout => %v", got)
	}

	mis := guaji.ErrMisconfigured("GUAJI_AUTH_BASE 未配置（正式环境必填）")
	if got := mapLoginErr(mis); !errors.Is(got, ErrGuajiUpstream) {
		t.Fatalf("misconfigured => %v", got)
	}
}
