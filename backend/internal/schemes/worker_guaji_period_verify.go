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
	if guajiShortPeriodSharedSnapshotsAllowDirectPlaceAt(lotteryCode, targetPeriod, now) {
		return false
	}
	closeAt, ok := lottery.PeriodsBetCloseAt(lotteryCode, now)
	if !ok {
		return true
	}
	return closeAt.Sub(now.UTC()) <= guajiPrePlaceVerificationLead
}

func guajiShortPeriodSharedSnapshotsAllowDirectPlaceAt(lotteryCode, targetPeriod string, now time.Time) bool {
	ps, periodsOK := lottery.PeriodsScheduleFor(lotteryCode)
	state, stateOK := lottery.PeriodStateFor(lotteryCode)
	if !periodsOK || !stateOK || !guajiBetSnapshotFreshAt(ps, now) {
		return false
	}
	if state.IntervalSec <= 0 || state.IntervalSec > 15 {
		return false
	}
	targetPeriod = strings.TrimSpace(targetPeriod)
	if targetPeriod == "" ||
		strings.TrimSpace(ps.CurrentPeriod) != targetPeriod ||
		strings.TrimSpace(state.NextIssue) != targetPeriod {
		return false
	}
	if ps.CloseAt.UTC().Sub(now.UTC()) <= guajiPlaceCloseSafety {
		return false
	}
	return guajiShortPeriodWSWindowAllowsAt(lotteryCode, targetPeriod, now)
}

// guajiFinalPeriodSafetyAllows keeps the current period and close-window
// checks mandatory. A successful member-level verification may only replace
// the shared draw websocket phase check, which can lag on very short games.
func guajiFinalPeriodSafetyAllows(lotteryCode, targetPeriod string, now time.Time, upstreamVerified bool) bool {
	if !guajiBetPeriodMatches(lotteryCode, targetPeriod) ||
		!guajiBetPeriodHasSafeWindowAt(lotteryCode, now) {
		return false
	}
	return upstreamVerified || guajiShortPeriodWSWindowAllowsAt(lotteryCode, targetPeriod, now)
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
