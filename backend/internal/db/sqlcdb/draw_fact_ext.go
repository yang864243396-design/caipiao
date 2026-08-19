package sqlcdb

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type LotteryDrawFactParams struct {
	LotteryCode      string
	IssueNo          string
	Balls            []byte
	Source           string
	ProviderEventID  string
	DrawHash         string
	RawPayloadDigest string
	ReceivedAt       time.Time
	ConfirmedAt      time.Time
}

func (q *Queries) AttachLotteryDrawFact(ctx context.Context, arg LotteryDrawFactParams) (bool, error) {
	var attached bool
	err := q.db.QueryRow(ctx, `
UPDATE lottery_draws
SET source = CASE WHEN source = 'legacy' THEN $4 ELSE source END,
    provider_event_id = COALESCE(provider_event_id, NULLIF($5, '')),
    draw_hash = COALESCE(draw_hash, NULLIF($6, '')),
    raw_payload_digest = COALESCE(raw_payload_digest, NULLIF($7, '')),
    received_at = LEAST(received_at, $8),
    confirmed_at = COALESCE(confirmed_at, $9)
WHERE lottery_code = $1
  AND issue_no = $2
  AND balls = $3::jsonb
RETURNING TRUE`, arg.LotteryCode, arg.IssueNo, arg.Balls, arg.Source, arg.ProviderEventID,
		arg.DrawHash, arg.RawPayloadDigest, arg.ReceivedAt, arg.ConfirmedAt).Scan(&attached)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return attached, err
}

func (q *Queries) GetLotteryDrawFactBalls(ctx context.Context, lotteryCode, issueNo string) ([]byte, string, error) {
	var balls []byte
	var drawHash string
	err := q.db.QueryRow(ctx, `
SELECT balls, COALESCE(draw_hash, '')
FROM lottery_draws
WHERE lottery_code = $1 AND issue_no = $2`, lotteryCode, issueNo).Scan(&balls, &drawHash)
	return balls, drawHash, err
}

func (q *Queries) InsertLotteryDrawCorrection(ctx context.Context, arg LotteryDrawFactParams, existingHash string) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO lottery_draw_corrections
    (lottery_code, issue_no, existing_draw_hash, corrected_draw_hash, source, provider_event_id, balls, observed_at)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7::jsonb, $8)
ON CONFLICT (lottery_code, issue_no, corrected_draw_hash) DO NOTHING`,
		arg.LotteryCode, arg.IssueNo, existingHash, arg.DrawHash, arg.Source, arg.ProviderEventID, arg.Balls, arg.ReceivedAt)
	return err
}
