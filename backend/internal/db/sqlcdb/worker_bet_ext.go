package sqlcdb

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// CloudBetPeriodHandled 方案在指定期号（含第三方 periods）是否已有投注记录（含 pending）。
func (q *Queries) CloudBetPeriodHandled(ctx context.Context, schemeID, periodNo string) (bool, error) {
	periodNo = strings.TrimSpace(periodNo)
	if schemeID == "" || periodNo == "" {
		return false, nil
	}
	var exists bool
	err := q.db.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1 FROM cloud_bet_records
    WHERE scheme_id = $1
      AND (
        period_no = $2
        OR NULLIF(TRIM(third_party_period), '') = $2
      )
)`, schemeID, periodNo).Scan(&exists)
	return exists, err
}

// GuajiPeriodAlreadyTaken cloud 记录或 bet_orders 待开奖占用该期号。
func (q *Queries) GuajiPeriodAlreadyTaken(ctx context.Context, schemeID string, memberID int64, periodNo string) (bool, error) {
	handled, err := q.CloudBetPeriodHandled(ctx, schemeID, periodNo)
	if err != nil || handled {
		return handled, err
	}
	var exists bool
	err = q.db.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1 FROM bet_orders
    WHERE member_id = $1
      AND issue_no = $2
      AND status = 'pending'
      AND guaji_account_id IS NOT NULL
      AND NULLIF(TRIM(third_party_bet_id), '') IS NOT NULL
)`, memberID, periodNo).Scan(&exists)
	return exists, err
}

// UpdateSchemeInstanceLastSettledIssue 仅云端挂机阶段推进第三方期号游标；await_next_bet 跳过期标记不可被覆盖。
func (q *Queries) UpdateSchemeInstanceLastSettledIssue(ctx context.Context, instanceID, periodNo string) error {
	periodNo = strings.TrimSpace(periodNo)
	if instanceID == "" || periodNo == "" {
		return nil
	}
	_, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET last_settled_issue = $2, updated_at = now()
WHERE id = $1
  AND status = 'running'
  AND status_reason = 'cloud_active'`, instanceID, periodNo)
	return err
}

// ClearStartSkipLastSettledCursor 云端挂机已激活后，清除开启跳过期误写在 last_settled_issue 的游标。
func (q *Queries) ClearStartSkipLastSettledCursor(ctx context.Context, instanceID string) error {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return nil
	}
	_, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET last_settled_issue = NULL, updated_at = now()
WHERE id = $1
  AND status = 'running'
  AND status_reason = 'cloud_active'
  AND start_skip_period IS NOT NULL
  AND last_settled_issue = start_skip_period`, instanceID)
	return err
}

// ActivateSchemeInstanceCloud 跳过当期结束后：await_next_bet → cloud_active。
// last_settled_issue 清零：开启跳过期号保留在 start_skip_period，避免误触 period_cursor_taken 防重。
func (q *Queries) ActivateSchemeInstanceCloud(ctx context.Context, instanceID string) (int64, error) {
	tag, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET status_reason = 'cloud_active',
    last_settled_issue = NULL,
    updated_at = now()
WHERE id = $1
  AND status = 'running'
  AND status_reason = 'await_next_bet'`, instanceID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// SchemeLastThirdPartyBetPeriod 本方案在指定通道（模拟/真实）最近一次下注的第三方 periods。
// 按 sim_bet 隔离，避免正式盘历史期号挡住模拟盘首投。
func (q *Queries) SchemeLastThirdPartyBetPeriod(ctx context.Context, schemeID string, simBet bool) (string, error) {
	var period string
	err := q.db.QueryRow(ctx, `
SELECT COALESCE(NULLIF(TRIM(c.third_party_period), ''), c.period_no)
FROM cloud_bet_records c
WHERE c.scheme_id = $1
  AND c.sim_bet = $2
ORDER BY c.placed_at DESC, c.id DESC
LIMIT 1`, schemeID, simBet).Scan(&period)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(period), nil
}

// SchemeUnsettledGuajiPeriod 方案是否有待开奖第三方注单（已接单未派奖）。
func (q *Queries) SchemeUnsettledGuajiPeriod(ctx context.Context, schemeID string) (string, bool, error) {
	var period string
	err := q.db.QueryRow(ctx, `
SELECT COALESCE(NULLIF(TRIM(c.third_party_period), ''), c.period_no)
FROM cloud_bet_records c
WHERE c.scheme_id = $1
  AND c.status = 'pending'
  AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
ORDER BY c.placed_at DESC, c.id DESC
LIMIT 1`, schemeID).Scan(&period)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	period = strings.TrimSpace(period)
	if period == "" {
		return "", false, nil
	}
	return period, true, nil
}

// TryClaimCloudBetPeriod 事务内占位 (scheme_id, period_no)；冲突返回 false。
func (q *Queries) TryClaimCloudBetPeriod(ctx context.Context, arg ReserveCloudBetPeriodParams) (bool, error) {
	var id int64
	err := q.db.QueryRow(ctx, `
INSERT INTO cloud_bet_records (
    record_no, member_id, sim_bet, scheme_id, scheme_name,
    period_no, play_type, multiplier, round_label, amount, pnl, status, bet_content,
    guaji_account_id, currency, lottery_code, lottery_label, definition_id, placed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, now()
)
ON CONFLICT (scheme_id, period_no) DO NOTHING
RETURNING id`,
		arg.RecordNo, arg.MemberID, arg.SimBet, arg.SchemeID, arg.SchemeName,
		arg.PeriodNo, arg.PlayType, arg.Multiplier, arg.RoundLabel, arg.Amount, arg.Pnl,
		arg.Status, arg.BetContent, arg.GuajiAccountID,
		arg.Currency, arg.LotteryCode, arg.LotteryLabel, arg.DefinitionID,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return id > 0, nil
}

type CloudBetPeriodSnapshot struct {
	Found           bool
	ThirdPartyBetID string
	BetOrderNo      string
	BetContent      string
}

func (q *Queries) GetCloudBetPeriodSnapshot(ctx context.Context, schemeID, periodNo string) (CloudBetPeriodSnapshot, error) {
	var tpID, orderNo pgtype.Text
	var betContent string
	err := q.db.QueryRow(ctx, `
SELECT third_party_bet_id, bet_order_no, COALESCE(bet_content, '')
FROM cloud_bet_records
WHERE scheme_id = $1 AND period_no = $2`, schemeID, periodNo).Scan(&tpID, &orderNo, &betContent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CloudBetPeriodSnapshot{}, nil
		}
		return CloudBetPeriodSnapshot{}, err
	}
	out := CloudBetPeriodSnapshot{Found: true, BetContent: betContent}
	if tpID.Valid {
		out.ThirdPartyBetID = strings.TrimSpace(tpID.String)
	}
	if orderNo.Valid {
		out.BetOrderNo = strings.TrimSpace(orderNo.String)
	}
	return out, nil
}

// ResetSchemeInstanceLookbackRoundEx 回头重置时仅归零倍投轮次与回头盈亏，保留基础倍数系数。
// 出号游标（pick_index / current_pick / last_direction）与倍投独立，不得清空，
// 否则高级定码轮换「中后跳局」会被回头复位打回第 1 局。
func (q *Queries) ResetSchemeInstanceLookbackRoundEx(ctx context.Context, id string) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET round_index = 0,
    lookback_pnl = 0,
    updated_at = now()
WHERE id = $1 AND status = 'running'`, id)
	return err
}

// GetSchemeInstanceStatus 读取方案实例当前状态（worker 下注前复核）。
func (q *Queries) GetSchemeInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	var status string
	err := q.db.QueryRow(ctx, `SELECT status FROM scheme_instances WHERE id = $1`, instanceID).Scan(&status)
	return status, err
}

// ReserveCloudBetPeriodParams 占位同一方案同一期，防止并发重复向第三方下单。
type ReserveCloudBetPeriodParams struct {
	RecordNo       string
	MemberID       int64
	SimBet         bool
	SchemeID       string
	SchemeName     string
	PeriodNo       string
	PlayType       string
	Multiplier     string
	RoundLabel     string
	Amount         pgtype.Numeric
	Pnl            pgtype.Numeric
	Status         string
	BetContent     string
	GuajiAccountID pgtype.Int8
	Currency       string
	LotteryCode    string
	LotteryLabel   string
	DefinitionID   string
}

// ReserveCloudBetPeriod 独立提交占位记录；冲突或已存在返回 false。
func (q *Queries) ReserveCloudBetPeriod(ctx context.Context, arg ReserveCloudBetPeriodParams) (bool, error) {
	var id int64
	err := q.db.QueryRow(ctx, `
INSERT INTO cloud_bet_records (
    record_no, member_id, sim_bet, scheme_id, scheme_name,
    period_no, play_type, multiplier, round_label, amount, pnl, status, bet_content,
    guaji_account_id, currency, lottery_code, lottery_label, definition_id, placed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, now()
)
ON CONFLICT (scheme_id, period_no) DO NOTHING
RETURNING id`,
		arg.RecordNo, arg.MemberID, arg.SimBet, arg.SchemeID, arg.SchemeName,
		arg.PeriodNo, arg.PlayType, arg.Multiplier, arg.RoundLabel, arg.Amount, arg.Pnl,
		arg.Status, arg.BetContent, arg.GuajiAccountID,
		arg.Currency, arg.LotteryCode, arg.LotteryLabel, arg.DefinitionID,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return id > 0, nil
}

// UpdateCloudBetRecordGuajiMeta 第三方接单后回写注单号、接单期号与对齐后的金额。
func (q *Queries) UpdateCloudBetRecordGuajiMeta(ctx context.Context, schemeID, periodNo string, thirdPartyBetID, betOrderNo, thirdPartyPeriod pgtype.Text, pnl pgtype.Numeric, status string, amount pgtype.Numeric) error {
	_, err := q.db.Exec(ctx, `
UPDATE cloud_bet_records
SET third_party_bet_id = $3,
    bet_order_no = $4,
    third_party_period = $5,
    pnl = $6,
    status = $7,
    amount = $8
WHERE scheme_id = $1 AND period_no = $2`, schemeID, periodNo, thirdPartyBetID, betOrderNo, thirdPartyPeriod, pnl, status, amount)
	return err
}

// MoveCloudBetRecordPeriod 将占位记录从预期期号改到第三方实际接单期号。
func (q *Queries) MoveCloudBetRecordPeriod(ctx context.Context, schemeID, fromPeriod, toPeriod string) error {
	fromPeriod = strings.TrimSpace(fromPeriod)
	toPeriod = strings.TrimSpace(toPeriod)
	if schemeID == "" || fromPeriod == "" || toPeriod == "" || fromPeriod == toPeriod {
		return nil
	}
	_, err := q.db.Exec(ctx, `
UPDATE cloud_bet_records
SET period_no = $3
WHERE scheme_id = $1 AND period_no = $2
  AND NOT EXISTS (
    SELECT 1 FROM cloud_bet_records WHERE scheme_id = $1 AND period_no = $3
  )`, schemeID, fromPeriod, toPeriod)
	return err
}

func (q *Queries) DeleteCloudBetRecordForInstancePeriod(ctx context.Context, schemeID, periodNo string) error {
	_, err := q.db.Exec(ctx, `DELETE FROM cloud_bet_records WHERE scheme_id = $1 AND period_no = $2`, schemeID, periodNo)
	return err
}

// InsertCloudBetRecordEx 写入 cloud 投注明细（含第三方注单号与接单期号）。
type InsertCloudBetRecordExParams struct {
	RecordNo         string
	MemberID         int64
	SimBet           bool
	SchemeID         string
	SchemeName       string
	PeriodNo         string
	PlayType         string
	Multiplier       string
	RoundLabel       string
	Amount           pgtype.Numeric
	Pnl              pgtype.Numeric
	Status           string
	BetContent       string
	GuajiAccountID   pgtype.Int8
	ThirdPartyBetID  pgtype.Text
	ThirdPartyPeriod pgtype.Text
	BetOrderNo       pgtype.Text
	Currency         string
	LotteryCode      string
	LotteryLabel     string
	DefinitionID     string
}

func (q *Queries) InsertCloudBetRecordEx(ctx context.Context, arg InsertCloudBetRecordExParams) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO cloud_bet_records (
    record_no, member_id, sim_bet, scheme_id, scheme_name,
    period_no, play_type, multiplier, round_label, amount, pnl, status, bet_content,
    guaji_account_id, third_party_bet_id, third_party_period, bet_order_no,
    currency, lottery_code, lottery_label, definition_id, placed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, now()
)
ON CONFLICT (scheme_id, period_no) DO UPDATE SET
    third_party_bet_id = COALESCE(EXCLUDED.third_party_bet_id, cloud_bet_records.third_party_bet_id),
    bet_order_no = COALESCE(EXCLUDED.bet_order_no, cloud_bet_records.bet_order_no),
    third_party_period = COALESCE(EXCLUDED.third_party_period, cloud_bet_records.third_party_period),
    currency = CASE WHEN EXCLUDED.currency <> '' THEN EXCLUDED.currency ELSE cloud_bet_records.currency END,
    lottery_code = CASE WHEN EXCLUDED.lottery_code <> '' THEN EXCLUDED.lottery_code ELSE cloud_bet_records.lottery_code END,
    lottery_label = CASE WHEN EXCLUDED.lottery_label <> '' THEN EXCLUDED.lottery_label ELSE cloud_bet_records.lottery_label END,
    definition_id = CASE WHEN EXCLUDED.definition_id <> '' THEN EXCLUDED.definition_id ELSE cloud_bet_records.definition_id END,
    amount = EXCLUDED.amount`,
		arg.RecordNo, arg.MemberID, arg.SimBet, arg.SchemeID, arg.SchemeName,
		arg.PeriodNo, arg.PlayType, arg.Multiplier, arg.RoundLabel, arg.Amount, arg.Pnl, arg.Status,
		arg.BetContent, arg.GuajiAccountID, arg.ThirdPartyBetID, arg.ThirdPartyPeriod, arg.BetOrderNo,
		arg.Currency, arg.LotteryCode, arg.LotteryLabel, arg.DefinitionID,
	)
	return err
}

// GetSchemeInstanceFull 读取 worker/回头所需的完整实例字段。
func (q *Queries) GetSchemeInstanceFull(ctx context.Context, id string) (SchemeInstance, error) {
	id = strings.TrimSpace(id)
	var i SchemeInstance
	err := q.db.QueryRow(ctx, `
SELECT id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    round_index, last_settled_issue, pick_index, current_pick, last_direction,
    start_skip_period, start_skip_close_at,
    created_at, updated_at
FROM scheme_instances WHERE id = $1`, id).Scan(
		&i.ID, &i.DefinitionID, &i.MemberID, &i.Kind, &i.SchemeName, &i.LotteryCode, &i.LotteryLabel,
		&i.Status, &i.StatusReason, &i.Turnover, &i.Pnl, &i.RunTimeSec, &i.LookbackPnl, &i.SessionPnl,
		&i.Multiplier, &i.CountdownSec, &i.SimBet,
		&i.RoundIndex, &i.LastSettledIssue, &i.PickIndex, &i.CurrentPick, &i.LastDirection,
		&i.StartSkipPeriod, &i.StartSkipCloseAt,
		&i.CreatedAt, &i.UpdatedAt,
	)
	return i, err
}

// GetCloudBetPeriodByOrderNo 按平台注单号查 cloud 记录期号。
func (q *Queries) GetCloudBetPeriodByOrderNo(ctx context.Context, orderNo string) (string, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return "", nil
	}
	var period string
	err := q.db.QueryRow(ctx, `SELECT period_no FROM cloud_bet_records WHERE bet_order_no = $1 LIMIT 1`, orderNo).Scan(&period)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(period), nil
}

// SchemeInstanceFromAdminRow 将 admin 查询行转为 SchemeInstance（止盈/止损检查用）。
func SchemeInstanceFromAdminRow(r GetSchemeInstanceByIDRow) SchemeInstance {
	return SchemeInstance{
		ID:           r.ID,
		DefinitionID: r.DefinitionID,
		MemberID:     r.MemberID,
		Kind:         r.Kind,
		SchemeName:   r.SchemeName,
		LotteryCode:  r.LotteryCode,
		LotteryLabel: r.LotteryLabel,
		Status:       r.Status,
		StatusReason: r.StatusReason,
		Turnover:     r.Turnover,
		Pnl:          r.Pnl,
		RunTimeSec:   r.RunTimeSec,
		LookbackPnl:  r.LookbackPnl,
		SessionPnl:   r.SessionPnl,
		Multiplier:   r.Multiplier,
		CountdownSec: r.CountdownSec,
		SimBet:       r.SimBet,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// ApplySchemeStatsFromCloudBetSettlement 派奖后回写方案累计盈亏（与方案是否仍在运行无关）。
func (q *Queries) ApplySchemeStatsFromCloudBetSettlement(ctx context.Context, betOrderNo string, pnl pgtype.Numeric) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_instances si
SET pnl = COALESCE(si.pnl, 0) + $2,
    session_pnl = COALESCE(si.session_pnl, 0) + $2,
    updated_at = now()
FROM cloud_bet_records c
WHERE c.bet_order_no = $1
  AND c.scheme_id = si.id
  AND c.member_id = si.member_id`, betOrderNo, pnl)
	return err
}

// PendingSimCloudBetRow 已开奖、待本地验奖的模拟盘 cloud 注单。
type PendingSimCloudBetRow struct {
	ID          int64
	SchemeID    string
	MemberID    int64
	PeriodNo    string
	LotteryCode string
	BetContent  string
	Amount      float64
	Balls       []byte
}

// ListPendingSimCloudBetsReady 模拟盘 pending 且 lottery_draws 已有真实开奖球号。
func (q *Queries) ListPendingSimCloudBetsReady(ctx context.Context, rowLimit int32) ([]PendingSimCloudBetRow, error) {
	if rowLimit <= 0 {
		rowLimit = 50
	}
	rows, err := q.db.Query(ctx, `
SELECT c.id, c.scheme_id, c.member_id, c.period_no, c.lottery_code,
       COALESCE(c.bet_content, ''), c.amount::float8, d.balls
FROM cloud_bet_records c
JOIN lottery_draws d
  ON d.lottery_code = c.lottery_code
 AND d.issue_no = c.period_no
WHERE c.sim_bet = true
  AND c.status = 'pending'
  AND d.balls IS NOT NULL
  AND jsonb_typeof(d.balls) = 'array'
  AND jsonb_array_length(d.balls) > 0
ORDER BY c.placed_at ASC, c.id ASC
LIMIT $1`, rowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PendingSimCloudBetRow, 0, rowLimit)
	for rows.Next() {
		var r PendingSimCloudBetRow
		if err := rows.Scan(&r.ID, &r.SchemeID, &r.MemberID, &r.PeriodNo, &r.LotteryCode,
			&r.BetContent, &r.Amount, &r.Balls); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateCloudBetRecordFromSettlementByID 按记录 id 结算 pending 模拟注（无第三方 order_no）。
func (q *Queries) UpdateCloudBetRecordFromSettlementByID(ctx context.Context, id int64, status string, pnl pgtype.Numeric) (int64, error) {
	tag, err := q.db.Exec(ctx, `
UPDATE cloud_bet_records
SET status = $2,
    pnl = $3
WHERE id = $1
  AND status = 'pending'
  AND sim_bet = true`, id, status, pnl)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ApplySchemeStatsFromCloudBetSettlementByID 按 cloud_bet_records.id 回写方案盈亏。
func (q *Queries) ApplySchemeStatsFromCloudBetSettlementByID(ctx context.Context, recordID int64, pnl pgtype.Numeric) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_instances si
SET pnl = COALESCE(si.pnl, 0) + $2,
    session_pnl = COALESCE(si.session_pnl, 0) + $2,
    updated_at = now()
FROM cloud_bet_records c
WHERE c.id = $1
  AND c.scheme_id = si.id
  AND c.member_id = si.member_id`, recordID, pnl)
	return err
}

// SetSchemeInstanceCountdownSec 仅更新展示倒计时（Worker 周期性对齐第三方 periods）。
func (q *Queries) SetSchemeInstanceCountdownSec(ctx context.Context, id string, sec int32) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET countdown_sec = $2,
    updated_at = now()
WHERE id = $1`, id, sec)
	return err
}

// GetCloudBetSchemeIDByOrderNo 按平台注单号查关联方案实例 id。
func (q *Queries) GetCloudBetSchemeIDByOrderNo(ctx context.Context, betOrderNo string) (string, error) {
	betOrderNo = strings.TrimSpace(betOrderNo)
	if betOrderNo == "" {
		return "", nil
	}
	var schemeID string
	err := q.db.QueryRow(ctx, `
SELECT scheme_id FROM cloud_bet_records WHERE bet_order_no = $1 LIMIT 1`, betOrderNo).Scan(&schemeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(schemeID), nil
}

// CloudBetListRow 投注记录列表行（含第三方注单号）。
type CloudBetListRow struct {
	ID                int64
	RecordNo          string
	ThirdPartyBetID   pgtype.Text
	SchemeName        string
	LotteryLabel      string
	PeriodNo          string
	Amount            float64
	Pnl               float64
	Status            string
	PlacedAt          pgtype.Timestamptz
}

// cloudBetLotteryLabelSQL：优先本表冗余彩种名（方案删除后仍可用）。
const cloudBetLotteryLabelSQL = `
COALESCE(NULLIF(TRIM(c.lottery_label), ''), c.scheme_name)`

// cloudBetOrderNoFilterSQL：注单号筛选——等值/前缀优先（可走索引），避免默认 leading ILIKE 全扫。
// 参数占位：$ORDER_NO（由各查询替换为实际 $n）。
const cloudBetOrderNoFilterSQL = `
($ORDER_NO::text IS NULL OR $ORDER_NO::text = ''
  OR c.record_no = $ORDER_NO::text
  OR c.bet_order_no = $ORDER_NO::text
  OR c.third_party_bet_id = $ORDER_NO::text
  OR c.record_no LIKE ($ORDER_NO::text || '%')
  OR c.bet_order_no LIKE ($ORDER_NO::text || '%')
  OR c.third_party_bet_id LIKE ($ORDER_NO::text || '%'))`

// cloudBetRealSQL：会员投注记录只排除模拟单。
// 不再按「当前启用挂机账号」过滤——方案删除 / 挂机解绑后 third_party_bet_id 仍须可检索。
const cloudBetRealSQL = `c.sim_bet = false`

func cloudBetOrderNoFilter(param string) string {
	return strings.ReplaceAll(cloudBetOrderNoFilterSQL, "$ORDER_NO", param)
}

func scanCloudBetListRow(rows interface {
	Next() bool
	Scan(dest ...any) error
}) (CloudBetListRow, error) {
	var i CloudBetListRow
	err := rows.Scan(
		&i.ID,
		&i.RecordNo,
		&i.ThirdPartyBetID,
		&i.SchemeName,
		&i.LotteryLabel,
		&i.PeriodNo,
		&i.Amount,
		&i.Pnl,
		&i.Status,
		&i.PlacedAt,
	)
	return i, err
}

func (q *Queries) ListCloudBetRecordsByDefinitionEx(ctx context.Context, arg ListCloudBetRecordsByDefinitionParams, currency pgtype.Text) ([]CloudBetListRow, error) {
	// 按 definition_id 冗余字段筛选；已删除方案走「全部方案」路径（ByLotteryEx）。
	_ = arg.GuajiAccountID
	rows, err := q.db.Query(ctx, `
SELECT c.id, c.record_no, c.third_party_bet_id, c.scheme_name,
       `+cloudBetLotteryLabelSQL+` AS lottery_label,
       c.period_no, c.amount::float8, c.pnl::float8, c.status, c.placed_at
FROM cloud_bet_records c
WHERE c.member_id = $1 AND c.definition_id = $2
  AND c.placed_at >= $3 AND c.placed_at < $4
  AND `+cloudBetOrderNoFilter("$5")+`
  AND `+cloudBetRealSQL+`
  AND ($6::text IS NULL OR $6::text = '' OR UPPER(c.currency) = UPPER($6::text))
ORDER BY c.placed_at DESC, c.id DESC
LIMIT $7`,
		arg.MemberID, arg.DefinitionID, arg.SinceAt, arg.UntilAt, arg.OrderNo, currency, arg.RowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CloudBetListRow
	for rows.Next() {
		i, err := scanCloudBetListRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) ListCloudBetRecordsByDefinitionAfterCursorEx(ctx context.Context, arg ListCloudBetRecordsByDefinitionAfterCursorParams, currency pgtype.Text) ([]CloudBetListRow, error) {
	_ = arg.GuajiAccountID
	rows, err := q.db.Query(ctx, `
SELECT c.id, c.record_no, c.third_party_bet_id, c.scheme_name,
       `+cloudBetLotteryLabelSQL+` AS lottery_label,
       c.period_no, c.amount::float8, c.pnl::float8, c.status, c.placed_at
FROM cloud_bet_records c
WHERE c.member_id = $1 AND c.definition_id = $2
  AND c.placed_at >= $3 AND c.placed_at < $4
  AND `+cloudBetOrderNoFilter("$5")+`
  AND `+cloudBetRealSQL+`
  AND ($6::text IS NULL OR $6::text = '' OR UPPER(c.currency) = UPPER($6::text))
  AND (c.placed_at < $7 OR (c.placed_at = $7 AND c.id < $8))
ORDER BY c.placed_at DESC, c.id DESC
LIMIT $9`,
		arg.MemberID, arg.DefinitionID, arg.SinceAt, arg.UntilAt, arg.OrderNo, currency,
		arg.CursorTime, arg.CursorID, arg.RowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CloudBetListRow
	for rows.Next() {
		i, err := scanCloudBetListRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) ListCloudBetRecordsByLotteryEx(ctx context.Context, arg ListCloudBetRecordsByLotteryParams, currency pgtype.Text) ([]CloudBetListRow, error) {
	// 仅扫 cloud_bet_records：方案删除后仍可按冗余 lottery/currency 检索。
	_ = arg.GuajiAccountID
	rows, err := q.db.Query(ctx, `
SELECT c.id, c.record_no, c.third_party_bet_id, c.scheme_name, `+cloudBetLotteryLabelSQL+` AS lottery_label, c.period_no,
       c.amount::float8, c.pnl::float8, c.status, c.placed_at
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND ($2::text = '' OR c.lottery_code = $2)
  AND c.placed_at >= $3 AND c.placed_at < $4
  AND `+cloudBetOrderNoFilter("$5")+`
  AND `+cloudBetRealSQL+`
  AND ($6::text IS NULL OR $6::text = '' OR UPPER(c.currency) = UPPER($6::text))
ORDER BY c.placed_at DESC, c.id DESC
LIMIT $7`,
		arg.MemberID, arg.LotteryCode, arg.SinceAt, arg.UntilAt, arg.OrderNo, currency, arg.RowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CloudBetListRow
	for rows.Next() {
		i, err := scanCloudBetListRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) ListCloudBetRecordsByLotteryAfterCursorEx(ctx context.Context, arg ListCloudBetRecordsByLotteryAfterCursorParams, currency pgtype.Text) ([]CloudBetListRow, error) {
	_ = arg.GuajiAccountID
	rows, err := q.db.Query(ctx, `
SELECT c.id, c.record_no, c.third_party_bet_id, c.scheme_name, `+cloudBetLotteryLabelSQL+` AS lottery_label, c.period_no,
       c.amount::float8, c.pnl::float8, c.status, c.placed_at
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND ($2::text = '' OR c.lottery_code = $2)
  AND c.placed_at >= $3 AND c.placed_at < $4
  AND `+cloudBetOrderNoFilter("$5")+`
  AND `+cloudBetRealSQL+`
  AND ($6::text IS NULL OR $6::text = '' OR UPPER(c.currency) = UPPER($6::text))
  AND (c.placed_at < $7 OR (c.placed_at = $7 AND c.id < $8))
ORDER BY c.placed_at DESC, c.id DESC
LIMIT $9`,
		arg.MemberID, arg.LotteryCode, arg.SinceAt, arg.UntilAt, arg.OrderNo, currency,
		arg.CursorTime, arg.CursorID, arg.RowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CloudBetListRow
	for rows.Next() {
		i, err := scanCloudBetListRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// BetOrderListRow 含第三方注单号的 bet_orders 列表行。
type BetOrderListRow struct {
	ID                int64
	OrderNo           string
	ThirdPartyBetID   pgtype.Text
	LotteryName       string
	IssueNo           string
	Amount            float64
	Pnl               float64
	Status            string
	PlacedAt          pgtype.Timestamptz
}

// CloudBetCurrencySummaryRow 云端投注按币种汇总行。
type CloudBetCurrencySummaryRow struct {
	Currency    string
	OrderCount  int64
	ValidAmount float64
	Pnl         float64
}

// SummarizeCloudBetRecordsByCurrencyEx 按币种汇总 real 云端投注（有效投注=amount 合计，输赢=pnl 合计）。
// definitionID / lotteryCode 为空表示不限；币种/彩种取自本表冗余字段，无 JOIN。
func (q *Queries) SummarizeCloudBetRecordsByCurrencyEx(
	ctx context.Context,
	memberID int64,
	definitionID string,
	lotteryCode string,
	sinceAt, untilAt pgtype.Timestamptz,
	orderNo pgtype.Text,
	guajiAccountID pgtype.Int8,
) ([]CloudBetCurrencySummaryRow, error) {
	_ = guajiAccountID
	rows, err := q.db.Query(ctx, `
SELECT UPPER(COALESCE(NULLIF(TRIM(c.currency), ''), '')) AS currency,
       COUNT(*)::bigint AS order_count,
       COALESCE(SUM(c.amount), 0)::float8 AS valid_amount,
       COALESCE(SUM(c.pnl), 0)::float8 AS pnl
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND `+cloudBetRealSQL+`
  AND c.placed_at >= $2 AND c.placed_at < $3
  AND ($4::text = '' OR c.definition_id = $4::text)
  AND ($5::text = '' OR c.lottery_code = $5::text)
  AND `+cloudBetOrderNoFilter("$6")+`
GROUP BY 1`,
		memberID, sinceAt, untilAt, definitionID, lotteryCode, orderNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CloudBetCurrencySummaryRow
	for rows.Next() {
		var i CloudBetCurrencySummaryRow
		if err := rows.Scan(&i.Currency, &i.OrderCount, &i.ValidAmount, &i.Pnl); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// SummarizeBetOrdersByCurrencyEx 按币种汇总 bet_orders。
func (q *Queries) SummarizeBetOrdersByCurrencyEx(
	ctx context.Context,
	memberID int64,
	timeFrom, timeTo pgtype.Timestamptz,
	lotteryCode pgtype.Text,
	orderNo pgtype.Text,
) ([]CloudBetCurrencySummaryRow, error) {
	rows, err := q.db.Query(ctx, `
SELECT UPPER(COALESCE(NULLIF(TRIM(b.currency), ''), '')) AS currency,
       COUNT(*)::bigint AS order_count,
       COALESCE(SUM(b.amount), 0)::float8 AS valid_amount,
       COALESCE(SUM(b.pnl), 0)::float8 AS pnl
FROM bet_orders b
WHERE b.member_id = $1
  AND b.placed_at >= $2 AND b.placed_at < $3
  AND ($4::text IS NULL OR b.lottery_code = $4::text)
  AND ($5::text IS NULL OR $5::text = ''
       OR b.order_no = $5::text OR b.third_party_bet_id = $5::text
       OR b.order_no LIKE ($5::text || '%') OR b.third_party_bet_id LIKE ($5::text || '%'))
GROUP BY 1`,
		memberID, timeFrom, timeTo, lotteryCode, orderNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CloudBetCurrencySummaryRow
	for rows.Next() {
		var i CloudBetCurrencySummaryRow
		if err := rows.Scan(&i.Currency, &i.OrderCount, &i.ValidAmount, &i.Pnl); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) ListBetOrdersEx(ctx context.Context, arg ListBetOrdersParams, currency pgtype.Text) ([]BetOrderListRow, error) {
	rows, err := q.db.Query(ctx, `
SELECT b.id, b.order_no, b.third_party_bet_id, b.lottery_name, b.issue_no,
       b.amount::float8, b.pnl::float8, b.status, b.placed_at
FROM bet_orders b
WHERE b.member_id = $1 AND b.placed_at >= $2 AND b.placed_at < $3
  AND ($4::text IS NULL OR b.status = $4::text)
  AND ($5::text IS NULL OR b.lottery_category = $5::text)
  AND ($6::text IS NULL OR b.lottery_code = $6::text)
  AND ($7::text IS NULL OR $7::text = ''
       OR b.order_no = $7::text OR b.third_party_bet_id = $7::text
       OR b.order_no LIKE ($7::text || '%') OR b.third_party_bet_id LIKE ($7::text || '%'))
  AND ($8::text IS NULL OR $8::text = '' OR UPPER(COALESCE(b.currency, '')) = UPPER($8::text))
ORDER BY b.placed_at DESC, b.id DESC
LIMIT $9`,
		arg.MemberID, arg.TimeFrom, arg.TimeTo, arg.Status, arg.LotteryCategory, arg.LotteryCode, arg.OrderNo, currency, arg.RowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BetOrderListRow
	for rows.Next() {
		var i BetOrderListRow
		if err := rows.Scan(&i.ID, &i.OrderNo, &i.ThirdPartyBetID, &i.LotteryName, &i.IssueNo, &i.Amount, &i.Pnl, &i.Status, &i.PlacedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) ListBetOrdersAfterCursorEx(ctx context.Context, arg ListBetOrdersAfterCursorParams, currency pgtype.Text) ([]BetOrderListRow, error) {
	rows, err := q.db.Query(ctx, `
SELECT b.id, b.order_no, b.third_party_bet_id, b.lottery_name, b.issue_no,
       b.amount::float8, b.pnl::float8, b.status, b.placed_at
FROM bet_orders b
WHERE b.member_id = $1 AND b.placed_at >= $2 AND b.placed_at < $3
  AND ($4::text IS NULL OR b.status = $4::text)
  AND ($5::text IS NULL OR b.lottery_category = $5::text)
  AND ($6::text IS NULL OR b.lottery_code = $6::text)
  AND ($7::text IS NULL OR $7::text = ''
       OR b.order_no = $7::text OR b.third_party_bet_id = $7::text
       OR b.order_no LIKE ($7::text || '%') OR b.third_party_bet_id LIKE ($7::text || '%'))
  AND ($8::text IS NULL OR $8::text = '' OR UPPER(COALESCE(b.currency, '')) = UPPER($8::text))
  AND (b.placed_at < $9 OR (b.placed_at = $9 AND b.id < $10))
ORDER BY b.placed_at DESC, b.id DESC
LIMIT $11`,
		arg.MemberID, arg.TimeFrom, arg.TimeTo, arg.Status, arg.LotteryCategory, arg.LotteryCode, arg.OrderNo, currency,
		arg.CursorTime, arg.CursorID, arg.RowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BetOrderListRow
	for rows.Next() {
		var i BetOrderListRow
		if err := rows.Scan(&i.ID, &i.OrderNo, &i.ThirdPartyBetID, &i.LotteryName, &i.IssueNo, &i.Amount, &i.Pnl, &i.Status, &i.PlacedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
