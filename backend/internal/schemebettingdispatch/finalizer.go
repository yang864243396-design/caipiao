package schemebettingdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebetting"
)

type AcceptanceFinalizer struct {
	pool   *db.Pool
	placer guajibet.Placer
}

func NewAcceptanceFinalizer(pool *db.Pool, placer guajibet.Placer) *AcceptanceFinalizer {
	if pool == nil || placer == nil {
		return nil
	}
	return &AcceptanceFinalizer{pool: pool, placer: placer}
}

func validateUnknownResolution(resolution schemebetting.UnknownResolution, frozen FrozenGuajiRequest, targetPeriod string) (schemebetting.OutboxState, error) {
	resolution.Outcome = strings.ToLower(strings.TrimSpace(resolution.Outcome))
	resolution.Evidence = strings.TrimSpace(resolution.Evidence)
	if len([]rune(resolution.Evidence)) < 8 {
		return "", errors.New("provider reconciliation evidence must be at least 8 characters")
	}
	if resolution.Outcome == "rejected" {
		return schemebetting.OutboxRejected, nil
	}
	if resolution.Outcome != "accepted" {
		return "", errors.New("unknown resolution outcome must be accepted or rejected")
	}
	resolution.ProviderOrderID = strings.TrimSpace(resolution.ProviderOrderID)
	resolution.AcceptedPeriod = strings.TrimSpace(resolution.AcceptedPeriod)
	resolution.ProviderCurrency = strings.TrimSpace(resolution.ProviderCurrency)
	if resolution.ProviderOrderID == "" || resolution.AcceptedPeriod == "" ||
		resolution.ProviderAmount <= 0 || resolution.ProviderAccountID <= 0 || resolution.ProviderCurrency == "" {
		return "", errors.New("accepted resolution requires complete provider identity and financial evidence")
	}
	if math.Abs(resolution.ProviderAmount-frozen.Request.Amount) > 0.001 {
		return "", errors.New("provider accepted amount differs from frozen request")
	}
	if !strings.EqualFold(resolution.ProviderCurrency, strings.TrimSpace(frozen.Request.Currency)) {
		return "", errors.New("provider accepted currency differs from frozen request")
	}
	if resolution.AcceptedPeriod != strings.TrimSpace(targetPeriod) {
		return schemebetting.OutboxAcceptedWrongPeriod, nil
	}
	return schemebetting.OutboxAccepted, nil
}

func (finalizer *AcceptanceFinalizer) ResolveUnknown(ctx context.Context, outboxID int64, actor, reason string, resolution schemebetting.UnknownResolution) error {
	if finalizer == nil || outboxID <= 0 {
		return errors.New("accepted bet finalizer is not configured")
	}
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if actor == "" || len([]rune(reason)) < 4 {
		return errors.New("actor and a reason of at least 4 characters are required")
	}
	tx, err := finalizer.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := sqlcdb.New(tx)
	var schemeID, state, targetPeriod string
	var frozenRaw []byte
	if err := tx.QueryRow(ctx, `
SELECT COALESCE(scheme_id, ''), state, target_period_no, frozen_request
FROM scheme_bet_outbox
WHERE id = $1
FOR UPDATE`, outboxID).Scan(&schemeID, &state, &targetPeriod, &frozenRaw); err != nil {
		return err
	}
	if state != string(schemebetting.OutboxSentUnknown) && state != string(schemebetting.OutboxExternalAcceptanceUnknown) {
		return errors.New("only unresolved external acceptance can be reconciled")
	}
	var frozen FrozenGuajiRequest
	if err := json.Unmarshal(frozenRaw, &frozen); err != nil {
		return fmt.Errorf("decode frozen request: %w", err)
	}
	resolvedState, err := validateUnknownResolution(resolution, frozen, targetPeriod)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	resolution.Outcome = strings.ToLower(strings.TrimSpace(resolution.Outcome))
	resolution.Evidence = strings.TrimSpace(resolution.Evidence)
	resolution.ProviderOrderID = strings.TrimSpace(resolution.ProviderOrderID)
	resolution.AcceptedPeriod = strings.TrimSpace(resolution.AcceptedPeriod)
	resolution.ProviderCurrency = strings.TrimSpace(resolution.ProviderCurrency)
	evidence, _ := json.Marshal(map[string]any{
		"summary": resolution.Evidence, "actor": actor, "resolvedAt": now,
	})
	beforeState, _ := json.Marshal(map[string]any{"state": state})
	outcomeReason := "manual_reconciliation_" + resolution.Outcome
	if resolvedState == schemebetting.OutboxAcceptedWrongPeriod {
		outcomeReason = "manual_reconciliation_accepted_wrong_period"
	}
	_, err = tx.Exec(ctx, `
UPDATE scheme_bet_outbox
SET state = $2,
    outcome_reason = $3,
    provider_order_no = NULLIF($4, ''),
    accepted_period_no = NULLIF($5, ''),
    provider_account_id = NULLIF($6, 0),
    provider_currency = NULLIF($7, ''),
    provider_amount = NULLIF($8::double precision, 0)::numeric,
    reconciliation_evidence = $9,
    terminal_at = $10,
    updated_at = $10
WHERE id = $1`, outboxID, string(resolvedState), outcomeReason,
		resolution.ProviderOrderID, resolution.AcceptedPeriod, resolution.ProviderAccountID,
		resolution.ProviderCurrency, resolution.ProviderAmount, evidence, now)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
UPDATE scheme_bet_attempts
SET outcome = $2,
    finished_at = COALESCE(finished_at, $3),
    provider_order_no = NULLIF($4, ''),
    accepted_period_no = NULLIF($5, ''),
    error_message = $6
WHERE outbox_id = $1
  AND attempt_no = (SELECT max(attempt_no) FROM scheme_bet_attempts WHERE outbox_id = $1)`,
		outboxID, string(resolvedState), now, resolution.ProviderOrderID, resolution.AcceptedPeriod, outcomeReason)
	if err != nil {
		return err
	}
	afterState, _ := json.Marshal(map[string]any{
		"state": resolvedState, "evidence": resolution.Evidence,
		"providerOrderId": resolution.ProviderOrderID, "acceptedPeriod": resolution.AcceptedPeriod,
		"providerAmount": resolution.ProviderAmount, "providerCurrency": resolution.ProviderCurrency,
		"providerAccountId": resolution.ProviderAccountID,
	})
	if err := q.InsertSchemeBettingAdminAction(ctx, sqlcdb.InsertSchemeBettingAdminActionParams{
		SchemeID: schemeID, OutboxID: outboxID, Action: "resolve_unknown", Actor: actor,
		Reason: reason, BeforeState: beforeState, AfterState: afterState,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if resolvedState == schemebetting.OutboxAccepted || resolvedState == schemebetting.OutboxAcceptedWrongPeriod {
		return finalizer.FinalizeAccepted(ctx, outboxID)
	}
	return nil
}

func (finalizer *AcceptanceFinalizer) RecoverAccepted(ctx context.Context, limit int32) error {
	if finalizer == nil {
		return nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := finalizer.pool.Query(ctx, `
SELECT id
FROM scheme_bet_outbox
WHERE state IN ('accepted', 'accepted_wrong_period') AND financial_finalized_at IS NULL
ORDER BY terminal_at, id
LIMIT $1`, limit)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := finalizer.FinalizeAccepted(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

type unknownRecoveryCandidate struct {
	id     int64
	frozen FrozenGuajiRequest
}

// RecoverUnknown claims a bounded batch of ambiguous placements and performs
// read-only exact provider lookups. A missing or ambiguous match is left
// frozen for a later backoff attempt; it is never converted to rejected.
func (finalizer *AcceptanceFinalizer) RecoverUnknown(ctx context.Context, limit int32) error {
	if finalizer == nil {
		return nil
	}
	resolver, hasSingleResolver := finalizer.placer.(guajibet.AcceptanceResolver)
	batchResolver, hasBatchResolver := finalizer.placer.(guajibet.AcceptanceBatchResolver)
	if !hasSingleResolver && !hasBatchResolver {
		return nil
	}
	if limit <= 0 {
		limit = 32
	}
	rows, err := finalizer.pool.Query(ctx, `
WITH candidates AS (
    SELECT id
    FROM scheme_bet_outbox
    WHERE state IN ('sent_unknown', 'external_acceptance_unknown')
      AND provider_reconcile_next_at <= clock_timestamp()
    ORDER BY provider_reconcile_next_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT $1
), claimed AS (
    UPDATE scheme_bet_outbox o
    SET provider_reconcile_attempts = provider_reconcile_attempts + 1,
        provider_reconcile_next_at = clock_timestamp() + LEAST(
            interval '30 seconds',
            interval '250 milliseconds' * power(2::double precision, LEAST(provider_reconcile_attempts, 7))
        ),
        updated_at = clock_timestamp()
    FROM candidates c
    WHERE o.id = c.id
    RETURNING o.id, o.frozen_request
)
SELECT id, frozen_request FROM claimed`, limit)
	if err != nil {
		return err
	}
	candidates := make([]unknownRecoveryCandidate, 0, limit)
	for rows.Next() {
		var candidate unknownRecoveryCandidate
		var frozenRaw []byte
		if err := rows.Scan(&candidate.id, &frozenRaw); err != nil {
			rows.Close()
			return err
		}
		if err := json.Unmarshal(frozenRaw, &candidate.frozen); err != nil {
			continue
		}
		candidates = append(candidates, candidate)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return rowsErr
	}
	if hasBatchResolver {
		grouped := make(map[string][]unknownRecoveryCandidate)
		for _, candidate := range candidates {
			account := strings.TrimSpace(candidate.frozen.MemberAccount)
			grouped[account] = append(grouped[account], candidate)
		}
		for account, accountCandidates := range grouped {
			requests := make([]guajibet.Request, len(accountCandidates))
			for i := range accountCandidates {
				requests[i] = accountCandidates[i].frozen.Request
			}
			lookups := batchResolver.ResolveAcceptedBets(ctx, account, requests)
			for i, lookup := range lookups {
				if i >= len(accountCandidates) || lookup.Err != nil {
					continue
				}
				if err := finalizer.resolveRecoveredUnknown(ctx, accountCandidates[i].id, lookup.Result); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, candidate := range candidates {
		accepted, err := resolver.ResolveAcceptedBet(ctx, candidate.frozen.MemberAccount, candidate.frozen.Request)
		if err == nil {
			if err := finalizer.resolveRecoveredUnknown(ctx, candidate.id, accepted); err != nil {
				return err
			}
		}
	}
	return nil
}

func (finalizer *AcceptanceFinalizer) resolveRecoveredUnknown(ctx context.Context, outboxID int64, accepted guajibet.Result) error {
	resolution := schemebetting.UnknownResolution{
		Outcome:           "accepted",
		Evidence:          "automatic exact provider order reconciliation",
		ProviderOrderID:   accepted.ThirdPartyBetID,
		AcceptedPeriod:    accepted.Periods,
		ProviderAmount:    accepted.Amount,
		ProviderAccountID: accepted.GuajiAccountID,
		ProviderCurrency:  accepted.Currency,
	}
	return finalizer.ResolveUnknown(ctx, outboxID, "scheme-betting-reconciler", "exact provider order fingerprint matched", resolution)
}

func (finalizer *AcceptanceFinalizer) FinalizeAccepted(ctx context.Context, outboxID int64) error {
	if finalizer == nil || outboxID <= 0 {
		return errors.New("accepted bet finalizer is not configured")
	}
	tx, err := finalizer.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := sqlcdb.New(tx)
	var frozenRaw []byte
	var providerOrder, acceptedPeriod, providerCurrency, origin, localOrderNo string
	var providerAccountID, memberID, chainSeq int64
	var providerAmount pgtype.Numeric
	var schemeID string
	err = tx.QueryRow(ctx, `
SELECT frozen_request, COALESCE(provider_order_no, ''), COALESCE(accepted_period_no, ''),
       COALESCE(provider_account_id, 0), COALESCE(provider_currency, ''), provider_amount,
       member_id, COALESCE(scheme_id, ''), chain_seq, origin, COALESCE(local_order_no, '')
FROM scheme_bet_outbox
WHERE id = $1 AND state IN ('accepted', 'accepted_wrong_period') AND financial_finalized_at IS NULL
FOR UPDATE`, outboxID).Scan(&frozenRaw, &providerOrder, &acceptedPeriod, &providerAccountID,
		&providerCurrency, &providerAmount, &memberID, &schemeID, &chainSeq, &origin, &localOrderNo)
	if err != nil {
		return err
	}
	var frozen FrozenGuajiRequest
	if err := json.Unmarshal(frozenRaw, &frozen); err != nil {
		return err
	}
	requestedAmount := frozen.Request.Amount
	actualAmount := numericFloat(providerAmount)
	if actualAmount <= 0 || math.Abs(actualAmount-requestedAmount) > 0.001 {
		if _, err := tx.Exec(ctx, `UPDATE scheme_bet_outbox SET outcome_reason = 'strategy_finance_mismatch', updated_at = now() WHERE id = $1`, outboxID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE scheme_instances SET strict_chain_state = 'blocked_requires_rearm', bet_failed_detail = 'strategy_finance_mismatch', updated_at = now() WHERE id = $1`, schemeID); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return errors.New("provider accepted amount is missing or differs from frozen request")
	}
	if strings.TrimSpace(providerCurrency) == "" || providerAccountID <= 0 || strings.TrimSpace(providerOrder) == "" || strings.TrimSpace(acceptedPeriod) == "" {
		if _, err := tx.Exec(ctx, `UPDATE scheme_bet_outbox SET outcome_reason = 'provider_financial_identity_missing', updated_at = now() WHERE id = $1`, outboxID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE scheme_instances SET strict_chain_state = 'blocked_requires_rearm', bet_failed_detail = 'provider_financial_identity_missing', updated_at = now() WHERE id = $1`, schemeID); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return errors.New("provider acceptance financial identity is incomplete")
	}
	amountNumeric := numericFromFloat(actualAmount)
	zeroNumeric := numericFromFloat(0)
	orderNo := fmt.Sprintf("EBO%020d", outboxID)
	if origin == "api" {
		orderNo = strings.TrimSpace(localOrderNo)
		if orderNo == "" {
			return errors.New("formal API bet local order identity is missing")
		}
	}
	recordNo := fmt.Sprintf("ECB%020d", outboxID)
	betPayload, _ := json.Marshal(map[string]any{"groupContent": frozen.BetContent, "requestId": frozen.RequestID})
	if origin == "api" {
		if len(frozen.LocalBetPayload) == 0 || !json.Valid(frozen.LocalBetPayload) {
			return errors.New("formal API bet settlement payload is invalid")
		}
		betPayload = frozen.LocalBetPayload
	}
	var betOrderID int64
	err = tx.QueryRow(ctx, `
WITH lock_row AS MATERIALIZED (
    SELECT pg_advisory_xact_lock(hashtextextended($1, 0))
),
inserted AS (
    INSERT INTO bet_orders (
        order_no, member_id, lottery_code, lottery_name, lottery_category, issue_no, amount,
        play_method, bet_payload, outbound_lottery_code, outbound_play_code,
        guaji_account_id, third_party_bet_id, currency
    )
    SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),NULLIF($11,''),NULLIF($12,0),$13,$14
    FROM lock_row
    WHERE NOT EXISTS (
        SELECT 1 FROM bet_order_identity WHERE order_no = $1
    )
    RETURNING id
)
SELECT id FROM inserted
UNION ALL
SELECT id FROM bet_order_identity WHERE order_no = $1
LIMIT 1`, orderNo, memberID, frozen.Request.LotteryCode, frozen.LotteryLabel,
		frozen.LotteryCategory, acceptedPeriod, amountNumeric, frozen.Request.PlayMethod, betPayload,
		frozen.Request.GameID, frozen.Request.RuleID, providerAccountID, providerOrder, providerCurrency).Scan(&betOrderID)
	if err != nil {
		return err
	}
	if err := finalizer.placer.MirrorBetDebitLedger(ctx, q, memberID, orderNo, actualAmount, providerAccountID, providerCurrency); err != nil {
		return err
	}
	if origin == "api" {
		if _, err := tx.Exec(ctx, `
UPDATE scheme_bet_outbox
SET financial_finalized_at = now(), updated_at = now()
WHERE id = $1 AND state IN ('accepted', 'accepted_wrong_period')`, outboxID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	_, err = q.ReserveCloudBetPeriod(ctx, sqlcdb.ReserveCloudBetPeriodParams{
		RecordNo: recordNo, MemberID: memberID, SimBet: false, SchemeID: schemeID, SchemeName: frozen.SchemeName,
		PeriodNo: acceptedPeriod, PlayType: frozen.PlayType, Multiplier: strconv.Itoa(frozen.Request.Multiplier),
		RoundLabel: frozen.RoundLabel, Amount: amountNumeric, Pnl: zeroNumeric, Status: "pending",
		BetContent: frozen.BetContent, GuajiAccountID: pgtype.Int8{Int64: providerAccountID, Valid: providerAccountID > 0},
		Currency: providerCurrency, LotteryCode: frozen.Request.LotteryCode, LotteryLabel: frozen.LotteryLabel,
		DefinitionID: frozen.DefinitionID, BetUnits: frozen.BetUnits,
	})
	if err != nil {
		return err
	}
	if err := q.UpdateCloudBetRecordGuajiMeta(ctx, schemeID, acceptedPeriod,
		pgtype.Text{String: providerOrder, Valid: providerOrder != ""}, pgtype.Text{String: orderNo, Valid: true},
		pgtype.Text{String: acceptedPeriod, Valid: true}, zeroNumeric, "pending", amountNumeric,
		frozen.BetUnits, frozen.PlayType, frozen.BetContent); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
UPDATE cloud_bet_records
SET rule_snapshot = $3, rule_version = $4, rule_snapshot_hash = NULLIF($5, '')
WHERE scheme_id = $1 AND period_no = $2`, schemeID, acceptedPeriod, frozen.RuleSnapshot, frozen.RuleVersion, frozen.RuleSnapshotHash); err != nil {
		return err
	}
	var cloudRecordID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM cloud_bet_records WHERE scheme_id = $1 AND period_no = $2`, schemeID, acceptedPeriod).Scan(&cloudRecordID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE scheme_bet_outbox
SET local_order_no = $2, local_cloud_record_id = $3, financial_finalized_at = now(), updated_at = now()
WHERE id = $1 AND state IN ('accepted', 'accepted_wrong_period')`, outboxID, orderNo, cloudRecordID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE scheme_instances
SET chain_seq = GREATEST(chain_seq, $2),
    strict_chain_state = CASE
        WHEN strict_chain_state = 'blocked_requires_rearm' THEN strict_chain_state
        ELSE 'active'
    END,
    turnover = turnover + $3, last_settled_issue = $4, status_reason = 'cloud_active', updated_at = now()
WHERE id = $1`, schemeID, chainSeq, amountNumeric, acceptedPeriod); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func numericFromFloat(value float64) pgtype.Numeric {
	var numeric pgtype.Numeric
	_ = numeric.Scan(strconv.FormatFloat(value, 'f', 3, 64))
	return numeric
}

func numericFloat(value pgtype.Numeric) float64 {
	if !value.Valid {
		return 0
	}
	floatValue, err := value.Float64Value()
	if err != nil || !floatValue.Valid {
		return 0
	}
	return floatValue.Float64
}
