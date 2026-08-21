package schemebettingdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/providerperiodtarget"
	"caipiao/backend/internal/schemebetting"
)

var (
	ErrAPIBetRejected          = errors.New("formal API bet was not accepted")
	ErrAPIBetAcceptanceUnknown = errors.New("formal API bet acceptance is unresolved")
)

type APIBetCommand struct {
	RequestID       string
	LocalOrderNo    string
	MemberID        int64
	MemberAccount   string
	LotteryLabel    string
	LotteryCategory string
	BetPayload      json.RawMessage
	Request         guajibet.Request
}

type APIBetResult struct {
	OrderNo         string
	IssueNo         string
	Amount          float64
	Status          string
	PlacedAt        time.Time
	ThirdPartyBetID string
}

func validateAPIBetCommand(command APIBetCommand) error {
	command.RequestID = strings.TrimSpace(command.RequestID)
	command.LocalOrderNo = strings.TrimSpace(command.LocalOrderNo)
	command.MemberAccount = strings.TrimSpace(command.MemberAccount)
	command.Request.LotteryCode = strings.TrimSpace(command.Request.LotteryCode)
	command.Request.IssueNo = strings.TrimSpace(command.Request.IssueNo)
	command.Request.Currency = strings.TrimSpace(command.Request.Currency)
	if command.RequestID == "" || len(command.RequestID) > 76 || command.LocalOrderNo == "" || len(command.LocalOrderNo) > 64 {
		return errors.New("formal API bet requires a bounded request and order identity")
	}
	if command.MemberID <= 0 || command.MemberAccount == "" || command.Request.LotteryCode == "" ||
		command.Request.IssueNo == "" || command.Request.Amount <= 0 || command.Request.Currency == "" {
		return errors.New("formal API bet identity and financial fields are incomplete")
	}
	if len(command.BetPayload) == 0 || !json.Valid(command.BetPayload) {
		return errors.New("formal API bet settlement payload is invalid")
	}
	return nil
}

func apiDeadlineBudget(target schemebetting.PeriodSnapshot) schemebetting.DeadlineBudget {
	duration := target.CloseAt.Sub(target.OpenAt)
	switch {
	case duration > 0 && duration <= 6*time.Second:
		return schemebetting.DeadlineBudget{ClockSkew: 150 * time.Millisecond, Queue: 100 * time.Millisecond, Dispatch: 150 * time.Millisecond, Network: 900 * time.Millisecond}
	case duration > 0 && duration <= 15*time.Second:
		return schemebetting.DeadlineBudget{ClockSkew: 200 * time.Millisecond, Queue: 200 * time.Millisecond, Dispatch: 200 * time.Millisecond, Network: 1100 * time.Millisecond}
	default:
		return schemebetting.DeadlineBudget{ClockSkew: 250 * time.Millisecond, Queue: 500 * time.Millisecond, Dispatch: 300 * time.Millisecond, Network: 1500 * time.Millisecond}
	}
}

func (runtime *Runtime) SubmitAPIBet(ctx context.Context, command APIBetCommand) (APIBetResult, error) {
	if runtime == nil || runtime.q == nil {
		return APIBetResult{}, errors.New("formal API bet dispatcher is unavailable")
	}
	if err := validateAPIBetCommand(command); err != nil {
		return APIBetResult{}, err
	}
	allowed := false
	for _, code := range runtime.cfg.LotteryCodes {
		if code == command.Request.LotteryCode {
			allowed = true
			break
		}
	}
	if !allowed {
		return APIBetResult{}, errors.New("formal API bet lottery is not allowlisted")
	}
	now := time.Now().UTC()
	target, snapshotID, ok, err := providerperiodtarget.CurrentForInitialDispatch(ctx, runtime.q, command.Request.LotteryCode, now)
	if err != nil {
		return APIBetResult{}, err
	}
	if !ok || snapshotID <= 0 {
		return APIBetResult{}, errors.New("no current provider period for formal API bet target")
	}
	if strings.TrimSpace(target.PeriodNo) != strings.TrimSpace(command.Request.IssueNo) {
		return APIBetResult{}, errors.New("formal API bet target is not the current provider period")
	}
	safeDeadline := schemebetting.SafeDeadline(target.CloseAt, apiDeadlineBudget(target))
	if !schemebetting.IsSafeToCreate(now, safeDeadline) {
		return APIBetResult{}, errors.New("formal API bet target is inside the safety window")
	}
	command.RequestID = strings.TrimSpace(command.RequestID)
	command.LocalOrderNo = strings.TrimSpace(command.LocalOrderNo)
	command.MemberAccount = strings.TrimSpace(command.MemberAccount)
	storageRequestID := "api:" + command.RequestID
	frozen, err := json.Marshal(FrozenGuajiRequest{
		Origin: "api", LocalOrderNo: command.LocalOrderNo, LocalBetPayload: command.BetPayload,
		RequestID: storageRequestID, MemberAccount: command.MemberAccount, Request: command.Request,
		LotteryLabel: command.LotteryLabel, LotteryCategory: command.LotteryCategory,
		PlayType: command.Request.PlayMethod, BetContent: command.Request.Content, BetUnits: command.Request.BetsNums,
	})
	if err != nil {
		return APIBetResult{}, err
	}
	frozenHash := schemebetting.CanonicalJSONPayloadHash(frozen)
	shardIndex := int(schemebetting.ShardForScheme(command.MemberAccount, uint32(len(runtime.cfg.Shards))))
	shardNo := runtime.cfg.Shards[shardIndex]
	_, shardAcquired, err := runtime.q.AcquireSchemeBettingShardLease(
		ctx, "dispatcher", shardNo, runtime.cfg.Owner, 2*runtime.cfg.LeaseDuration,
	)
	if err != nil {
		return APIBetResult{}, err
	}
	if !shardAcquired {
		return APIBetResult{}, errors.New("formal API bet dispatcher shard is owned by another worker")
	}
	outboxID, existingHash, err := runtime.q.InsertAPIFormalBetOutbox(ctx, sqlcdb.InsertAPIFormalBetOutboxParams{
		MemberID: command.MemberID, LotteryCode: command.Request.LotteryCode, TargetPeriodNo: command.Request.IssueNo,
		Mode: runtime.cfg.Mode, RequestID: storageRequestID, PayloadHash: frozenHash, Payload: frozen,
		FrozenRequest: frozen, FrozenRequestHash: frozenHash, ProviderSnapshotID: snapshotID,
		CloseAt: target.CloseAt, SafeDeadlineAt: safeDeadline, ShardNo: shardNo, LocalOrderNo: command.LocalOrderNo,
	})
	if err != nil {
		return APIBetResult{}, err
	}
	if existingHash != frozenHash {
		return APIBetResult{}, errors.New("formal API bet requestId was reused with a different payload")
	}
	if runtime.events != nil {
		if err := runtime.events.PublishBetReady(ctx, outboxID, storageRequestID, shardNo, safeDeadline); err == nil {
			if err := runtime.q.MarkBetReadyPublished(ctx, outboxID, time.Now().UTC()); err != nil {
				return APIBetResult{}, err
			}
		} else {
			_ = runtime.q.MarkBetReadyPublishFailed(ctx, outboxID)
		}
	}
	leased, acquired, err := runtime.q.LeaseFormalOutboxByID(ctx, outboxID, runtime.cfg.Owner, runtime.cfg.LeaseDuration)
	if err != nil {
		return APIBetResult{}, err
	}
	if acquired {
		if err := runtime.dispatcher.Dispatch(ctx, leased); err != nil && !errors.Is(err, schemebetting.ErrStaleLease) {
			return APIBetResult{}, err
		}
	}
	return runtime.waitForAPIBet(ctx, outboxID, safeDeadline)
}

func (runtime *Runtime) waitForAPIBet(ctx context.Context, outboxID int64, safeDeadline time.Time) (APIBetResult, error) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		row, err := runtime.q.GetAPIFormalBetOutboxResult(ctx, outboxID)
		if err != nil {
			return APIBetResult{}, err
		}
		switch schemebetting.OutboxState(row.State) {
		case schemebetting.OutboxAccepted:
			if row.FinancialFinalized {
				return APIBetResult{
					OrderNo: row.LocalOrderNo, IssueNo: row.TargetPeriodNo, Amount: row.ProviderAmount,
					Status: "pending", PlacedAt: row.CreatedAt, ThirdPartyBetID: row.ProviderOrderNo,
				}, nil
			}
		case schemebetting.OutboxSentUnknown, schemebetting.OutboxExternalAcceptanceUnknown:
			return APIBetResult{}, fmt.Errorf("%w: %s", ErrAPIBetAcceptanceUnknown, row.OutcomeReason)
		case schemebetting.OutboxRejected, schemebetting.OutboxExpired, schemebetting.OutboxCancelled, schemebetting.OutboxAcceptedWrongPeriod:
			return APIBetResult{}, fmt.Errorf("%w: %s", ErrAPIBetRejected, row.OutcomeReason)
		}
		if !time.Now().UTC().Before(safeDeadline) {
			return APIBetResult{}, ErrAPIBetAcceptanceUnknown
		}
		select {
		case <-ctx.Done():
			return APIBetResult{}, ctx.Err()
		case <-ticker.C:
		}
	}
}
