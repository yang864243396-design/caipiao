package lottery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/ws"
)

type DrawFactMeta struct {
	Source           string
	ProviderEventID  string
	RawPayloadDigest string
	ReceivedAt       time.Time
	ConfirmedAt      time.Time
}

func CanonicalDrawHash(lotteryCode, issueNo string, balls []string) string {
	payload, _ := json.Marshal(struct {
		LotteryCode string   `json:"lotteryCode"`
		IssueNo     string   `json:"issueNo"`
		Balls       []string `json:"balls"`
	}{strings.TrimSpace(lotteryCode), strings.TrimSpace(issueNo), balls})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func PersistDrawFactFromBalls(ctx context.Context, q *sqlcdb.Queries, hub *ws.Hub, lotteryCode, issueNo string, balls []string, drawnAt time.Time, meta DrawFactMeta) (sqlcdb.LotteryDraw, bool, error) {
	draw, inserted, err := PersistDrawFromBalls(ctx, q, hub, lotteryCode, issueNo, balls, drawnAt)
	if err != nil || q == nil || strings.TrimSpace(issueNo) == "" || len(balls) == 0 {
		return draw, inserted, err
	}
	receivedAt := meta.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	confirmedAt := meta.ConfirmedAt
	if confirmedAt.IsZero() {
		confirmedAt = receivedAt
	}
	source := strings.TrimSpace(meta.Source)
	if source == "" {
		source = "unknown"
	}
	ballsJSON, err := json.Marshal(balls)
	if err != nil {
		return draw, inserted, err
	}
	params := sqlcdb.LotteryDrawFactParams{
		LotteryCode: strings.TrimSpace(lotteryCode), IssueNo: strings.TrimSpace(issueNo), Balls: ballsJSON,
		Source: source, ProviderEventID: strings.TrimSpace(meta.ProviderEventID),
		DrawHash: CanonicalDrawHash(lotteryCode, issueNo, balls), RawPayloadDigest: strings.TrimSpace(meta.RawPayloadDigest),
		ReceivedAt: receivedAt.UTC(), ConfirmedAt: confirmedAt.UTC(),
	}
	attached, err := q.AttachLotteryDrawFact(ctx, params)
	if err != nil || attached {
		return draw, inserted, err
	}
	existingBallsJSON, existingHash, err := q.GetLotteryDrawFactBalls(ctx, params.LotteryCode, params.IssueNo)
	if err != nil {
		return draw, inserted, err
	}
	if existingHash == "" {
		existingHash = hashStoredDraw(params.LotteryCode, params.IssueNo, existingBallsJSON)
	}
	if err := q.InsertLotteryDrawCorrection(ctx, params, existingHash); err != nil {
		return draw, inserted, err
	}
	return draw, false, nil
}

func hashStoredDraw(lotteryCode, issueNo string, raw []byte) string {
	var balls []string
	if json.Unmarshal(raw, &balls) != nil {
		sum := sha256.Sum256(raw)
		return hex.EncodeToString(sum[:])
	}
	return CanonicalDrawHash(lotteryCode, issueNo, balls)
}
