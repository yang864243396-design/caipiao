package schemes

import (
	"errors"
	"testing"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
)

func TestShouldDeferGuajiPlaceError(t *testing.T) {
	cases := []struct {
		err    error
		defer_ bool
	}{
		{errors.New("guaji http POST /api/web_bets/lott: context deadline exceeded"), true},
		{errors.New("guaji http POST /x: connection refused"), true},
		{errors.New(`guaji http POST /x: status 502 body={}`), true},
		{guajibet.ErrPeriodClosed, false},
		{guajibet.ErrTokenInvalid, false},
		{guajibet.ErrInsufficient, false},
		{&guaji.APIError{Code: 40000, Message: "余额不足"}, false},
	}
	for _, c := range cases {
		got := guaji.IsRetryableTransportError(c.err)
		if got != c.defer_ {
			t.Fatalf("%v => defer=%v want %v", c.err, got, c.defer_)
		}
	}
}

func TestClampSchemeWorkerPlaceConcurrency(t *testing.T) {
	if got := clampSchemeWorkerPlaceConcurrency(-1); got != defaultSchemeWorkerPlaceConcurrency {
		t.Fatalf("neg => %d", got)
	}
	if got := clampSchemeWorkerPlaceConcurrency(0); got != 0 {
		t.Fatalf("0 => %d want unlimited", got)
	}
	if got := clampSchemeWorkerPlaceConcurrency(8); got != 8 {
		t.Fatalf("8 => %d", got)
	}
	if got := clampSchemeWorkerPlaceConcurrency(maxSchemeWorkerPlaceConcurrency + 9); got != maxSchemeWorkerPlaceConcurrency {
		t.Fatalf("over max => %d", got)
	}
}

func TestSetPlaceConcurrency(t *testing.T) {
	w := &Worker{}
	w.SetPlaceConcurrency(8)
	if w.placeSem == nil || cap(w.placeSem) != 8 {
		t.Fatalf("placeSem cap=%v", w.placeSem)
	}
	w.SetPlaceConcurrency(0)
	if w.placeSem != nil {
		t.Fatal("0 should disable place sem")
	}
}
