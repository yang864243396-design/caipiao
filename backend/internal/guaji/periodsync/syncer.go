package periodsync

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/lottery"
)

// Syncer 读取平台 periods 本地缓存；仅在缓存过期时按需兜底拉取第三方。
type Syncer struct {
	pool     *db.Pool
	client   *guaji.Client
	accounts *accountsvc.Service

	fallbackLocks         sync.Map // lotteryCode -> *sync.Mutex
	prePlaceVerifications *prePlaceVerifyCache
}

func NewSyncer(pool *db.Pool, client *guaji.Client, accounts *accountsvc.Service) *Syncer {
	if pool == nil || client == nil || !client.Enabled() || accounts == nil {
		return nil
	}
	return &Syncer{
		pool:                  pool,
		client:                client,
		accounts:              accounts,
		prePlaceVerifications: newPrePlaceVerifyCache(500 * time.Millisecond),
	}
}

func (s *Syncer) gameIDForLottery(ctx context.Context, lotteryCode string) (int, error) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return 0, fmt.Errorf("empty lottery code")
	}
	var gameKey string
	err := s.pool.QueryRow(ctx, `
SELECT COALESCE(NULLIF(TRIM(outbound_lottery_code), ''), code)
FROM lottery_catalog
WHERE code = $1`, lotteryCode).Scan(&gameKey)
	if err != nil {
		return 0, err
	}
	gameID, err := strconv.Atoi(strings.TrimSpace(gameKey))
	if err != nil || gameID <= 0 {
		return 0, fmt.Errorf("invalid outbound game_id for %s: %q", lotteryCode, gameKey)
	}
	return gameID, nil
}

func (s *Syncer) fetchAndApply(ctx context.Context, lotteryCode string, numPeriods int) error {
	token, err := s.accounts.SyncAccessToken(ctx)
	if err != nil {
		return err
	}
	return s.fetchAndApplyWithToken(ctx, lotteryCode, numPeriods, token)
}

func (s *Syncer) fetchAndApplyWithToken(ctx context.Context, lotteryCode string, numPeriods int, token string) error {
	_, err := s.fetchPeriodsAndApplyWithToken(ctx, lotteryCode, numPeriods, token)
	return err
}

func (s *Syncer) fetchPeriodsAndApplyWithToken(ctx context.Context, lotteryCode string, numPeriods int, token string) ([]guaji.LottPeriod, error) {
	gameID, err := s.gameIDForLottery(ctx, lotteryCode)
	if err != nil {
		return nil, err
	}
	periods, _, err := s.client.FetchLottPeriods(ctx, token, gameID, numPeriods)
	if err != nil {
		if guaji.IsPeriodClosedError(err) {
			lottery.ClearPeriodsSchedule(lotteryCode)
			return nil, nil
		}
		return nil, err
	}
	observedAt := time.Now()
	if err := persistProviderPeriodSnapshots(ctx, s.pool, lotteryCode, periods, observedAt); err != nil {
		return nil, err
	}
	applyPeriodsListToCache(lotteryCode, periods, observedAt)
	return periods, nil
}

// VerifyOpenPeriodForMember synchronously confirms the open period with the
// member's third-party account before an at-risk PlaceBet. Calls for the same
// lottery/account are coalesced and briefly reused, so many schemes do not
// amplify one close-window refresh into many upstream requests.
func (s *Syncer) VerifyOpenPeriodForMember(ctx context.Context, lotteryCode, memberAccount string) (string, time.Time, error) {
	if s == nil {
		return "", time.Time{}, nil
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	memberAccount = strings.TrimSpace(memberAccount)
	if lotteryCode == "" || memberAccount == "" {
		return "", time.Time{}, fmt.Errorf("empty lottery code or member account")
	}
	key := lotteryCode + "\x00" + memberAccount
	cache := s.prePlaceVerifications
	if cache == nil {
		cache = newPrePlaceVerifyCache(500 * time.Millisecond)
		s.prePlaceVerifications = cache
	}
	result, err := cache.getOrRefresh(ctx, key, time.Now(), func(ctx context.Context) (prePlaceVerifyResult, error) {
		token, err := s.accounts.MemberAccessToken(ctx, memberAccount)
		if err != nil {
			return prePlaceVerifyResult{}, err
		}
		periods, err := s.fetchPeriodsAndApplyWithToken(ctx, lotteryCode, workerNumPeriods, token)
		if err != nil {
			return prePlaceVerifyResult{}, err
		}
		open, closeAt, ok := guaji.PickOpenLottPeriod(periods, lotteryCode, time.Now())
		if !ok {
			return prePlaceVerifyResult{}, fmt.Errorf("no currently open period for %s", lotteryCode)
		}
		return prePlaceVerifyResult{Period: strings.TrimSpace(open.Period), CloseAt: closeAt.UTC()}, nil
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return result.Period, result.CloseAt, nil
}

// ForceRefreshForMember 用指定会员 token 拉取 periods（矩阵测试等）。
func (s *Syncer) ForceRefreshForMember(ctx context.Context, lotteryCode, memberAccount string) error {
	if s == nil {
		return nil
	}
	token, err := s.accounts.MemberAccessToken(ctx, memberAccount)
	if err != nil {
		return err
	}
	mu := s.fallbackMuFor(lotteryCode)
	// TryLock：避免与 worker/列表互相堵死（持锁期间 HTTP 可达数十秒）。
	if !mu.TryLock() {
		return nil
	}
	defer mu.Unlock()
	return s.fetchAndApplyWithToken(ctx, lotteryCode, workerNumPeriods, token)
}

// ForceRefresh 拉取第三方 periods 并写入缓存。锁被占用时立即返回，绝不阻塞 API。
func (s *Syncer) ForceRefresh(ctx context.Context, lotteryCode string) error {
	if s == nil {
		return nil
	}
	mu := s.fallbackMuFor(lotteryCode)
	if !mu.TryLock() {
		return nil
	}
	defer mu.Unlock()
	return s.fetchAndApply(ctx, lotteryCode, workerNumPeriods)
}

// EnsureFreshIfStale 缓存过期或当前期 end_time 已过时兜底拉取（Worker 故障场景）；新鲜则 no-op。
func (s *Syncer) EnsureFreshIfStale(ctx context.Context, lotteryCode string) error {
	if s == nil {
		return nil
	}
	now := time.Now()
	if !lottery.PeriodsScheduleNeedsRefresh(lotteryCode, now) {
		return nil
	}
	return s.fallbackFetch(ctx, lotteryCode)
}

// StartSkipPeriod 仅读本地缓存（开启路径绝不同步拉第三方）。
func (s *Syncer) StartSkipPeriod(_ context.Context, lotteryCode string) (string, bool, error) {
	p, ok := lottery.StartSkipPeriodFromCache(lotteryCode)
	return p, ok, nil
}

// StartSkipSnapshot 仅读本地缓存（开启路径绝不同步拉第三方，避免被 ForceRefresh 锁拖死）。
func (s *Syncer) StartSkipSnapshot(_ context.Context, lotteryCode string) (period string, closeAt time.Time, ok bool, err error) {
	p, ca, ok := lottery.StartSkipSnapshotFromCache(lotteryCode)
	return p, ca, ok, nil
}

func (s *Syncer) fallbackFetch(ctx context.Context, lotteryCode string) error {
	mu := s.fallbackMuFor(lotteryCode)
	if !mu.TryLock() {
		return nil
	}
	defer mu.Unlock()
	if !lottery.PeriodsScheduleNeedsRefresh(lotteryCode, time.Now()) {
		return nil
	}
	return s.fetchAndApply(ctx, lotteryCode, workerNumPeriods)
}

func (s *Syncer) fallbackMuFor(lotteryCode string) *sync.Mutex {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if v, ok := s.fallbackLocks.Load(lotteryCode); ok {
		return v.(*sync.Mutex)
	}
	mu := &sync.Mutex{}
	actual, _ := s.fallbackLocks.LoadOrStore(lotteryCode, mu)
	return actual.(*sync.Mutex)
}
