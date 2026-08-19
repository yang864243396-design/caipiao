package accountsvc

import (
	"sync"
	"time"
)

const payoutDiagnosticCapacity = 1024

// PayoutSyncDiagnostics is the latest in-memory payout synchronization state
// for one third-party account.
type PayoutSyncDiagnostics struct {
	AccountID              int64      `json:"accountId"`
	LastAttemptAt          *time.Time `json:"lastAttemptAt,omitempty"`
	LastSuccessAt          *time.Time `json:"lastSuccessAt,omitempty"`
	LastError              string     `json:"lastError,omitempty"`
	LastErrorAt            *time.Time `json:"lastErrorAt,omitempty"`
	PendingCount           int        `json:"pendingCount"`
	ProviderListCount      int        `json:"providerListCount"`
	SettledCount           int        `json:"settledCount"`
	ProviderUnsettledCount int        `json:"providerUnsettledCount"`
}

type payoutDiagnosticStore struct {
	mu        sync.RWMutex
	byAccount map[int64]PayoutSyncDiagnostics
}

func newPayoutDiagnosticStore() *payoutDiagnosticStore {
	return &payoutDiagnosticStore{byAccount: make(map[int64]PayoutSyncDiagnostics)}
}

func (s *payoutDiagnosticStore) begin(accountID int64, pending int, at time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byAccount == nil {
		s.byAccount = make(map[int64]PayoutSyncDiagnostics)
	}
	diagnostic := s.diagnosticForUpdateLocked(accountID)
	diagnostic.AccountID = accountID
	diagnostic.LastAttemptAt = payoutDiagnosticTimePtr(at)
	diagnostic.PendingCount = pending
	diagnostic.ProviderListCount = 0
	diagnostic.SettledCount = 0
	diagnostic.ProviderUnsettledCount = 0
	s.byAccount[accountID] = diagnostic
}

func (s *payoutDiagnosticStore) succeed(accountID int64, providerList, settled, unresolved int, at time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byAccount == nil {
		s.byAccount = make(map[int64]PayoutSyncDiagnostics)
	}
	diagnostic := s.diagnosticForUpdateLocked(accountID)
	diagnostic.AccountID = accountID
	diagnostic.LastSuccessAt = payoutDiagnosticTimePtr(at)
	diagnostic.LastError = ""
	diagnostic.LastErrorAt = nil
	diagnostic.ProviderListCount = providerList
	diagnostic.SettledCount = settled
	diagnostic.ProviderUnsettledCount = unresolved
	s.byAccount[accountID] = diagnostic
}

func (s *payoutDiagnosticStore) fail(accountID int64, err error, at time.Time) {
	if s == nil || err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byAccount == nil {
		s.byAccount = make(map[int64]PayoutSyncDiagnostics)
	}
	diagnostic := s.diagnosticForUpdateLocked(accountID)
	diagnostic.AccountID = accountID
	diagnostic.LastError = err.Error()
	diagnostic.LastErrorAt = payoutDiagnosticTimePtr(at)
	s.byAccount[accountID] = diagnostic
}

func (s *payoutDiagnosticStore) partial(accountID int64, providerList, settled, unresolved int, err error, at time.Time) {
	if s == nil || err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byAccount == nil {
		s.byAccount = make(map[int64]PayoutSyncDiagnostics)
	}
	diagnostic := s.diagnosticForUpdateLocked(accountID)
	diagnostic.AccountID = accountID
	diagnostic.LastError = err.Error()
	diagnostic.LastErrorAt = payoutDiagnosticTimePtr(at)
	diagnostic.ProviderListCount = providerList
	diagnostic.SettledCount = settled
	diagnostic.ProviderUnsettledCount = unresolved
	s.byAccount[accountID] = diagnostic
}

func (s *payoutDiagnosticStore) diagnosticForUpdateLocked(accountID int64) PayoutSyncDiagnostics {
	if diagnostic, ok := s.byAccount[accountID]; ok {
		return diagnostic
	}
	if len(s.byAccount) >= payoutDiagnosticCapacity {
		s.evictOldestLocked()
	}
	return PayoutSyncDiagnostics{AccountID: accountID}
}

func (s *payoutDiagnosticStore) evictOldestLocked() {
	var oldestID int64
	var oldest PayoutSyncDiagnostics
	found := false
	for accountID, diagnostic := range s.byAccount {
		if !found || payoutDiagnosticEvictsBefore(accountID, diagnostic, oldestID, oldest) {
			oldestID = accountID
			oldest = diagnostic
			found = true
		}
	}
	if found {
		delete(s.byAccount, oldestID)
	}
}

func payoutDiagnosticEvictsBefore(accountID int64, diagnostic PayoutSyncDiagnostics, otherID int64, other PayoutSyncDiagnostics) bool {
	switch {
	case diagnostic.LastAttemptAt == nil && other.LastAttemptAt != nil:
		return true
	case diagnostic.LastAttemptAt != nil && other.LastAttemptAt == nil:
		return false
	case diagnostic.LastAttemptAt != nil && other.LastAttemptAt != nil:
		if diagnostic.LastAttemptAt.Before(*other.LastAttemptAt) {
			return true
		}
		if other.LastAttemptAt.Before(*diagnostic.LastAttemptAt) {
			return false
		}
	}
	return accountID < otherID
}

func (s *payoutDiagnosticStore) snapshot(accountID int64) (PayoutSyncDiagnostics, bool) {
	if s == nil {
		return PayoutSyncDiagnostics{}, false
	}
	s.mu.RLock()
	diagnostic, ok := s.byAccount[accountID]
	s.mu.RUnlock()
	if !ok {
		return PayoutSyncDiagnostics{}, false
	}
	return clonePayoutSyncDiagnostics(diagnostic), true
}

func clonePayoutSyncDiagnostics(diagnostic PayoutSyncDiagnostics) PayoutSyncDiagnostics {
	diagnostic.LastAttemptAt = clonePayoutDiagnosticTime(diagnostic.LastAttemptAt)
	diagnostic.LastSuccessAt = clonePayoutDiagnosticTime(diagnostic.LastSuccessAt)
	diagnostic.LastErrorAt = clonePayoutDiagnosticTime(diagnostic.LastErrorAt)
	return diagnostic
}

func payoutDiagnosticTimePtr(at time.Time) *time.Time {
	copy := at
	return &copy
}

func clonePayoutDiagnosticTime(at *time.Time) *time.Time {
	if at == nil {
		return nil
	}
	return payoutDiagnosticTimePtr(*at)
}

// PayoutSyncDiagnostics returns a copy of the latest in-memory account
// synchronization snapshot without issuing a provider request.
func (s *Service) PayoutSyncDiagnostics(accountID int64) (PayoutSyncDiagnostics, bool) {
	if s == nil {
		return PayoutSyncDiagnostics{}, false
	}
	return s.payoutDiagnostics.snapshot(accountID)
}
