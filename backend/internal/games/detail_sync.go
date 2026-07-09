package games

import (
	"context"
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji/historysync"
	"caipiao/backend/internal/guaji/periodsync"
	"caipiao/backend/internal/lottery"
)

const detailThirdPartySyncMinGap = 5 * time.Second

var detailSyncLastThirdParty sync.Map // lotteryCode -> time.Time

type detailDisplaySync struct {
	periodSync  *periodsync.Syncer
	historySync *historysync.Worker
}

func (s *Service) SetDetailDisplaySync(periodSync *periodsync.Syncer, historySync *historysync.Worker) {
	if s == nil {
		return
	}
	s.detailSync = &detailDisplaySync{
		periodSync:  periodSync,
		historySync: historySync,
	}
}

// ensureDetailDisplayFresh 玩法详情展示前刷新 periods / 历史开奖，避免倒计时已推进但开奖区与 betRecords 仍落后。
func (s *Service) ensureDetailDisplayFresh(ctx context.Context, lotteryCode string) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" || s == nil || s.detailSync == nil {
		return
	}
	now := time.Now().UTC()
	openPeriod := openBettingPeriod(lotteryCode, now)
	wantIssue := ""
	if openPeriod != "" {
		wantIssue = prevIssueNo(openPeriod)
	}

	if s.detailSync.periodSync != nil {
		_ = s.detailSync.periodSync.EnsureFreshIfStale(ctx, lotteryCode)
	}

	if s.detailSync.historySync == nil || s.q == nil || wantIssue == "" {
		s.persistCachedDrawIfMissing(ctx, lotteryCode, wantIssue)
		return
	}

	dbIssue, err := latestDrawIssueFromDB(ctx, s.q, lotteryCode)
	if err != nil {
		s.persistCachedDrawIfMissing(ctx, lotteryCode, wantIssue)
		return
	}
	if dbIssue != "" && compareIssueNo(dbIssue, wantIssue) >= 0 {
		return
	}

	needThirdParty := true
	if last, ok := detailSyncLastThirdParty.Load(lotteryCode); ok {
		if t, ok := last.(time.Time); ok && time.Since(t) < detailThirdPartySyncMinGap {
			needThirdParty = false
		}
	}
	if needThirdParty {
		detailSyncLastThirdParty.Store(lotteryCode, time.Now())
		if s.detailSync.periodSync != nil {
			_ = s.detailSync.periodSync.ForceRefresh(ctx, lotteryCode)
		}
		_ = s.detailSync.historySync.SyncLottery(ctx, lotteryCode)
	}
	s.persistCachedDrawIfMissing(ctx, lotteryCode, wantIssue)
}

func (s *Service) persistCachedDrawIfMissing(ctx context.Context, lotteryCode, issueNo string) {
	if s == nil || s.q == nil || issueNo == "" {
		return
	}
	if _, err := s.q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
		LotteryCode: lotteryCode,
		IssueNo:     issueNo,
	}); err == nil {
		return
	}
	cached, ok := lottery.DrawResultForIssue(lotteryCode, issueNo)
	if !ok {
		return
	}
	_, _, _ = lottery.PersistDrawFromBalls(ctx, s.q, nil, lotteryCode, issueNo, cached.Balls, cached.DrawnAt)
}
