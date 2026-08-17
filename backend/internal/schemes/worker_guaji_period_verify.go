package schemes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/lottery"
)

const guajiPrePlaceVerificationLead = 2 * guajiPlaceCloseSafety
const guajiPrePlaceVerificationTimeout = 900 * time.Millisecond
const guajiVerifiedPlaceSafety = guajiPlaceCloseSafety + 300*time.Millisecond

// guajiPrePlaceVerificationNeeded preserves the normal centralized cache path
// for healthy, well-before-close periods. Only a stale, mismatched, or
// near-close snapshot needs a member-token upstream confirmation.
func guajiPrePlaceVerificationNeeded(lotteryCode, targetPeriod string, now time.Time) bool {
	ps, ok := lottery.PeriodsScheduleFor(lotteryCode)
	if !ok || !guajiBetSnapshotFreshAt(ps, now) {
		return true
	}
	if strings.TrimSpace(ps.CurrentPeriod) != strings.TrimSpace(targetPeriod) {
		return true
	}
	closeAt, ok := lottery.PeriodsBetCloseAt(lotteryCode, now)
	if !ok {
		return true
	}
	return closeAt.Sub(now.UTC()) <= guajiPrePlaceVerificationLead
}

func (w *Worker) verifyGuajiPeriodBeforePlace(ctx context.Context, lotteryCode, memberAccount, targetPeriod string) error {
	if w == nil || w.periodVerifier == nil {
		return nil
	}
	verifyCtx, cancel := context.WithTimeout(ctx, guajiPrePlaceVerificationTimeout)
	defer cancel()
	period, closeAt, err := w.periodVerifier.VerifyOpenPeriodForMember(verifyCtx, lotteryCode, memberAccount)
	if err != nil {
		// A failed confirmation must never fall back to a possibly stale local
		// period. Treat it as a safe skip; the period claim is released before
		// PlaceRealBet has been called.
		return fmt.Errorf("%w: upstream period confirmation failed: %v", guajibet.ErrPeriodClosed, err)
	}
	period = strings.TrimSpace(period)
	targetPeriod = strings.TrimSpace(targetPeriod)
	if period == "" || period != targetPeriod {
		return fmt.Errorf("%w: upstream open period=%s target=%s", guajibet.ErrPeriodClosed, period, targetPeriod)
	}
	if closeAt.IsZero() || closeAt.UTC().Sub(time.Now().UTC()) <= guajiVerifiedPlaceSafety {
		return fmt.Errorf("%w: upstream period %s is too close to close", guajibet.ErrPeriodClosed, targetPeriod)
	}
	return nil
}
