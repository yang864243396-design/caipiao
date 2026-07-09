package schemes

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
)

func bumpIssueNo(issue string) string {
	n, err := strconv.ParseInt(strings.TrimSpace(issue), 10, 64)
	if err != nil {
		return issue + "1"
	}
	return strconv.FormatInt(n+1, 10)
}

func issuePeriodShort(issueNo string) string {
	n, err := strconv.ParseInt(strings.TrimSpace(issueNo), 10, 64)
	if err != nil || n <= 0 {
		return issueNo
	}
	return fmt.Sprintf("%03d", n%1000)
}

// issueAfter 期号数值比较：candidate 是否 strictly 在 after 之后。
func issueAfter(candidate, after string) bool {
	candidate = strings.TrimSpace(candidate)
	after = strings.TrimSpace(after)
	if candidate == "" {
		return false
	}
	if after == "" || after == "0" {
		return true
	}
	c, err1 := strconv.ParseInt(candidate, 10, 64)
	a, err2 := strconv.ParseInt(after, 10, 64)
	if err1 == nil && err2 == nil {
		return c > a
	}
	return candidate > after
}

func drawForOpenIssue(ctx context.Context, q *sqlcdb.Queries, lotteryCode, issueNo string) (sqlcdb.LotteryDraw, bool, error) {
	draw, err := q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
		LotteryCode: lotteryCode,
		IssueNo:     issueNo,
	})
	if err == nil {
		return sqlcdb.LotteryDrawFromIssueRow(draw), true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlcdb.LotteryDraw{}, false, err
	}
	return sqlcdb.LotteryDraw{
		LotteryCode: lotteryCode,
		IssueNo:     issueNo,
		PeriodShort: issuePeriodShort(issueNo),
	}, true, nil
}

func synthDrawBalls(lotteryCode, issueNo string) []string {
	switch {
	case isLHCLotteryCode(lotteryCode):
		return synthLHCDrawBalls(lotteryCode, issueNo)
	case isSYXWLotteryCode(lotteryCode):
		return synthSYXWDrawBalls(lotteryCode, issueNo)
	case isPK10LotteryCode(lotteryCode):
		return synthPK10DrawBalls(lotteryCode, issueNo)
	case isK3LotteryCode(lotteryCode):
		return synthK3DrawBalls(lotteryCode, issueNo)
	case isPC28LotteryCode(lotteryCode):
		return synthPC28DrawBalls(lotteryCode, issueNo)
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(lotteryCode + ":" + issueNo))
	seed := h.Sum32()
	balls := make([]string, 5)
	for i := range balls {
		seed = seed*1664525 + 1013904223
		balls[i] = strconv.Itoa(int(seed % 10))
	}
	return balls
}

func synthSYXWDrawBalls(lotteryCode, issueNo string) []string {
	h := fnv.New32a()
	_, _ = h.Write([]byte("syxw:" + lotteryCode + ":" + issueNo))
	seed := h.Sum32()
	used := map[int]bool{}
	balls := make([]string, 5)
	for i := range balls {
		for tries := 0; tries < 64; tries++ {
			seed = seed*1664525 + 1013904223
			n := int(seed%11) + 1
			if !used[n] {
				used[n] = true
				balls[i] = fmt.Sprintf("%02d", n)
				break
			}
		}
		if balls[i] == "" {
			balls[i] = fmt.Sprintf("%02d", i+1)
		}
	}
	return balls
}

func synthPK10DrawBalls(lotteryCode, issueNo string) []string {
	h := fnv.New32a()
	_, _ = h.Write([]byte("pk10:" + lotteryCode + ":" + issueNo))
	seed := h.Sum32()
	pool := make([]int, 10)
	for i := range pool {
		pool[i] = i + 1
	}
	for i := len(pool) - 1; i > 0; i-- {
		seed = seed*1664525 + 1013904223
		j := int(seed) % (i + 1)
		pool[i], pool[j] = pool[j], pool[i]
	}
	balls := make([]string, 10)
	for i, n := range pool {
		balls[i] = strconv.Itoa(n)
	}
	return balls
}

func synthK3DrawBalls(lotteryCode, issueNo string) []string {
	h := fnv.New32a()
	_, _ = h.Write([]byte("k3:" + lotteryCode + ":" + issueNo))
	seed := h.Sum32()
	balls := make([]string, 3)
	for i := range balls {
		seed = seed*1664525 + 1013904223
		balls[i] = strconv.Itoa(int(seed%6) + 1)
	}
	return balls
}

func synthPC28DrawBalls(lotteryCode, issueNo string) []string {
	h := fnv.New32a()
	_, _ = h.Write([]byte("pc28:" + lotteryCode + ":" + issueNo))
	seed := h.Sum32()
	balls := make([]string, 3)
	for i := range balls {
		seed = seed*1664525 + 1013904223
		balls[i] = strconv.Itoa(int(seed % 10))
	}
	return balls
}

func synthLHCDrawBalls(lotteryCode, issueNo string) []string {
	h := fnv.New32a()
	_, _ = h.Write([]byte("lhc:" + lotteryCode + ":" + issueNo))
	seed := h.Sum32()
	used := map[int]bool{}
	balls := make([]string, 7)
	for i := range balls {
		for tries := 0; tries < 64; tries++ {
			seed = seed*1664525 + 1013904223
			n := int(seed%49) + 1
			if !used[n] {
				used[n] = true
				balls[i] = strconv.Itoa(n)
				break
			}
		}
		if balls[i] == "" {
			balls[i] = strconv.Itoa(i + 1)
		}
	}
	return balls
}
