package schemebetting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PeriodSnapshot struct {
	PeriodNo   string
	OpenAt     time.Time
	CloseAt    time.Time
	ObservedAt time.Time
}

func SelectTargetPeriod(snapshots []PeriodSnapshot, sourcePeriod string, now time.Time, maxAge time.Duration) (PeriodSnapshot, bool) {
	now = now.UTC()
	sourcePeriod = strings.TrimSpace(sourcePeriod)
	candidates := make([]PeriodSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		snapshot.PeriodNo = strings.TrimSpace(snapshot.PeriodNo)
		if snapshot.PeriodNo == "" || snapshot.PeriodNo == sourcePeriod || snapshot.CloseAt.IsZero() || !now.Before(snapshot.CloseAt.UTC()) {
			continue
		}
		if !snapshot.OpenAt.IsZero() && now.Before(snapshot.OpenAt.UTC()) {
			continue
		}
		observedAt := snapshot.ObservedAt.UTC()
		preloadedCurrent := !snapshot.OpenAt.IsZero() && !observedAt.After(snapshot.OpenAt.UTC()) &&
			!now.Before(snapshot.OpenAt.UTC()) && now.Before(snapshot.CloseAt.UTC())
		if snapshot.ObservedAt.IsZero() || (maxAge > 0 && now.Sub(observedAt) > maxAge && !preloadedCurrent) {
			continue
		}
		candidates = append(candidates, snapshot)
	}
	if len(candidates) == 0 {
		return PeriodSnapshot{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].CloseAt.Before(candidates[j].CloseAt)
	})
	return candidates[0], true
}

type DeadlineBudget struct {
	ClockSkew time.Duration
	Queue     time.Duration
	Dispatch  time.Duration
	Network   time.Duration
}

func (b DeadlineBudget) Total() time.Duration {
	parts := []time.Duration{b.ClockSkew, b.Queue, b.Dispatch, b.Network}
	var total time.Duration
	for _, part := range parts {
		if part > 0 {
			total += part
		}
	}
	return total
}

func SafeDeadline(closeAt time.Time, budget DeadlineBudget) time.Time {
	if closeAt.IsZero() {
		return time.Time{}
	}
	return closeAt.UTC().Add(-budget.Total())
}

func IsSafeToCreate(now, safeDeadline time.Time) bool {
	return !safeDeadline.IsZero() && now.UTC().Before(safeDeadline.UTC())
}

type OutboxState string

type UnknownResolution struct {
	Outcome           string  `json:"outcome"`
	Evidence          string  `json:"evidence"`
	ProviderOrderID   string  `json:"providerOrderId"`
	AcceptedPeriod    string  `json:"acceptedPeriod"`
	ProviderAmount    float64 `json:"providerAmount"`
	ProviderAccountID int64   `json:"providerAccountId"`
	ProviderCurrency  string  `json:"providerCurrency"`
}

const (
	OutboxPending     OutboxState = "pending"
	OutboxLeased      OutboxState = "leased"
	OutboxSentUnknown OutboxState = "sent_unknown"
	OutboxAccepted    OutboxState = "accepted"
	OutboxRejected    OutboxState = "rejected"
	OutboxExpired     OutboxState = "expired"
	OutboxCancelled   OutboxState = "cancelled"
)

func CanTransition(from, to OutboxState) bool {
	switch from {
	case OutboxPending:
		return to == OutboxLeased || to == OutboxExpired || to == OutboxCancelled
	case OutboxLeased:
		return to == OutboxSentUnknown || to == OutboxAccepted || to == OutboxRejected || to == OutboxExpired || to == OutboxCancelled || to == OutboxAcceptedWrongPeriod || to == OutboxExternalAcceptanceUnknown
	case OutboxSentUnknown:
		return to == OutboxAccepted || to == OutboxRejected || to == OutboxExpired || to == OutboxCancelled || to == OutboxAcceptedWrongPeriod || to == OutboxExternalAcceptanceUnknown
	default:
		return false
	}
}

func CommandIdentity(schemeID, sourcePeriod, targetPeriod string, stateVersion int64) string {
	raw := strings.Join([]string{
		strings.TrimSpace(schemeID),
		strings.TrimSpace(sourcePeriod),
		strings.TrimSpace(targetPeriod),
		strconv.FormatInt(stateVersion, 10),
	}, "\x00")
	sum := sha256.Sum256([]byte(raw))
	return "sb_" + hex.EncodeToString(sum[:16])
}

func PayloadHash(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// CanonicalJSONPayloadHash keeps a JSON request hash stable after PostgreSQL
// JSONB parses and reorders object keys during a store/load round trip.
// Invalid JSON falls back to byte hashing so callers can retain the existing
// integrity behavior for non-JSON payloads.
func CanonicalJSONPayloadHash(payload []byte) string {
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return PayloadHash(payload)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return PayloadHash(payload)
	}
	return PayloadHash(canonical)
}

func ShardForScheme(schemeID string, shardCount uint32) uint32 {
	if shardCount == 0 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(schemeID)))
	return h.Sum32() % shardCount
}
