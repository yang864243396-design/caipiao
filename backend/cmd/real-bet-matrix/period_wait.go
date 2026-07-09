package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"caipiao/backend/internal/guaji/periodsync"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/lottery"
)

const periodPollInterval = 500 * time.Millisecond

func refreshOpenIssue(ctx context.Context, sync *periodsync.Syncer, account, lotteryCode string) (string, bool) {
	if lottery.GuajiPeriodsNotProvided(lotteryCode) {
		return "", true
	}
	if sync != nil {
		_ = sync.EnsureFreshIfStale(ctx, lotteryCode)
	}
	if issue, ok := lottery.OpenIssueForGuajiBet(lotteryCode); ok {
		return issue, true
	}
	if sync != nil {
		_ = sync.ForceRefreshForMember(ctx, lotteryCode, account)
		if issue, ok := lottery.OpenIssueForGuajiBet(lotteryCode); ok {
			return issue, true
		}
	}
	return "", false
}

// waitForOpenIssue 等待可投期号；avoidIssue 非空时跳过该期（封盘后等下一期）。
func waitForOpenIssue(ctx context.Context, sync *periodsync.Syncer, account, lotteryCode, avoidIssue string, maxWait time.Duration) (issue string, polls int, err error) {
	if lottery.GuajiPeriodsNotProvided(lotteryCode) {
		return "", 0, nil
	}
	deadline := time.Now().Add(maxWait)
	for {
		polls++
		if issue, ok := refreshOpenIssue(ctx, sync, account, lotteryCode); ok {
			if avoidIssue == "" || issue != avoidIssue {
				return issue, polls, nil
			}
			waitForCloseAt(ctx, lotteryCode, avoidIssue)
		}
		if time.Now().After(deadline) {
			return "", polls, fmt.Errorf("等待开盘超时 (%v)", maxWait)
		}
		if err := sleepCtx(ctx, periodPollInterval); err != nil {
			return "", polls, err
		}
	}
}

func waitForCloseAt(ctx context.Context, lotteryCode, issue string) {
	ps, ok := lottery.PeriodsScheduleFor(lotteryCode)
	if !ok || ps.CloseAt.IsZero() {
		return
	}
	if strings.TrimSpace(ps.CurrentPeriod) != issue {
		return
	}
	rem := time.Until(ps.CloseAt.UTC()) + 800*time.Millisecond
	if rem <= 0 {
		return
	}
	if rem > 15*time.Second {
		rem = 15 * time.Second
	}
	_ = sleepCtx(ctx, rem)
}

func isTransientPeriodErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, guajibet.ErrPeriodClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "封盘") || strings.Contains(msg, "无开盘期号")
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
