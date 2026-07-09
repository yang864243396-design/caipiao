package lottery

import (
	"strings"
	"sync"
	"time"
)

// CachedDrawResult 第三方开奖 WS / 历史 REST 入库前的内存快照，供玩法详情展示加速。
type CachedDrawResult struct {
	IssueNo string
	Balls   []string
	DrawnAt time.Time
}

var drawResultCache sync.Map // lotteryCode -> CachedDrawResult

// StoreDrawResult 写入最近一期开奖缓存（按 issue 单调更新）。
func StoreDrawResult(lotteryCode, issueNo string, balls []string, drawnAt time.Time) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	issueNo = strings.TrimSpace(issueNo)
	if lotteryCode == "" || issueNo == "" || len(balls) == 0 {
		return
	}
	if drawnAt.IsZero() {
		drawnAt = time.Now().UTC()
	} else {
		drawnAt = drawnAt.UTC()
	}
	cp := append([]string(nil), balls...)
	if v, ok := drawResultCache.Load(lotteryCode); ok {
		if prev, ok := v.(CachedDrawResult); ok && compareIssueNo(prev.IssueNo, issueNo) > 0 {
			return
		}
	}
	drawResultCache.Store(lotteryCode, CachedDrawResult{
		IssueNo: issueNo,
		Balls:   cp,
		DrawnAt: drawnAt,
	})
}

// DrawResultForIssue 读取指定期号的缓存开奖（issue 须完全匹配）。
func DrawResultForIssue(lotteryCode, issueNo string) (CachedDrawResult, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	issueNo = strings.TrimSpace(issueNo)
	if lotteryCode == "" || issueNo == "" {
		return CachedDrawResult{}, false
	}
	v, ok := drawResultCache.Load(lotteryCode)
	if !ok {
		return CachedDrawResult{}, false
	}
	cached, ok := v.(CachedDrawResult)
	if !ok || cached.IssueNo != issueNo || len(cached.Balls) == 0 {
		return CachedDrawResult{}, false
	}
	return cached, true
}
