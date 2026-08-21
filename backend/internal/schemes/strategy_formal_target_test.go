package schemes

import (
	"errors"
	"testing"
)

func TestFormalProviderTargetFailuresAreRetryable(t *testing.T) {
	if !errors.Is(formalProviderTargetError(false, nil), errNoFreshFormalProviderTarget) {
		t.Fatal("a temporarily missing provider target must remain retryable")
	}
	deadlineErr := errors.New("no safe dispatch window")
	if !errors.Is(formalProviderTargetError(true, deadlineErr), errUnsafeFormalProviderTarget) {
		t.Fatal("an exhausted provider window must remain retryable for the next current period")
	}
	if err := formalProviderTargetError(true, nil); err != nil {
		t.Fatalf("available target returned %v", err)
	}
}
