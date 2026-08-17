package accountsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
)

func TestPayoutSyncFetchesOneAccountListForMultiplePendingOrders(t *testing.T) {
	var calls []int
	items, err := fetchRecentAccountSettlements(context.Background(), func(_ context.Context, limit, page int) ([]guaji.WebBetRecord, error) {
		if limit != payoutSyncPageSize {
			t.Fatalf("limit=%d, want %d", limit, payoutSyncPageSize)
		}
		calls = append(calls, page)
		switch page {
		case 1:
			return []guaji.WebBetRecord{{ID: 101}, {ID: 102}}, nil
		default:
			return nil, nil
		}
	})
	if err != nil {
		t.Fatalf("fetch recent account settlements: %v", err)
	}
	if len(calls) != 1 || calls[0] != 1 {
		t.Fatalf("list calls=%v, want one recent account list request", calls)
	}
	if len(items) != 2 || items["101"].ID != 101 || items["102"].ID != 102 {
		t.Fatalf("items=%#v, want both pending orders from the same account list", items)
	}
}

func TestPayoutSyncFetchErrorIsReturned(t *testing.T) {
	want := errors.New("tls handshake timeout")
	_, err := fetchRecentAccountSettlements(context.Background(), func(context.Context, int, int) ([]guaji.WebBetRecord, error) {
		return nil, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
}

func TestPayoutBatchCountsUnresolvedRows(t *testing.T) {
	settled, unresolved := payoutBatchCounts(3, 1)
	if settled != 1 || unresolved != 2 {
		t.Fatalf("settled=%d unresolved=%d", settled, unresolved)
	}
}

func TestSettlePayoutBatchRowsReturnsHistoricalProviderErrorAfterCursorSave(t *testing.T) {
	providerErr := errors.New("provider list timeout")
	savedPage := 0
	savedError := ""

	settled, err := settlePayoutBatchRows(
		[]sqlcdb.ListPendingGuajiBetOrdersRow{payoutBatchTestRow("101")},
		map[string]guaji.WebBetRecord{},
		func(sqlcdb.ListPendingGuajiBetOrdersRow, *guaji.BetSettlement) error {
			t.Fatal("recent commit called for historical row")
			return nil
		},
		func(sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error) {
			return resolveHistoricalSettlementResult(
				nil,
				7,
				false,
				providerErr,
				func(page int, lastErr string) error {
					savedPage = page
					savedError = lastErr
					return nil
				},
				func(*guaji.BetSettlement) error {
					t.Fatal("commit called after provider error")
					return nil
				},
				func() error {
					t.Fatal("cleanup called after provider error")
					return nil
				},
			)
		},
	)
	if settled != 0 {
		t.Fatalf("settled=%d, want 0", settled)
	}
	if err != providerErr {
		t.Fatalf("err=%v, want original provider error", err)
	}
	if savedPage != 7 || savedError != providerErr.Error() {
		t.Fatalf("saved page=%d error=%q", savedPage, savedError)
	}
}

func TestSettlePayoutBatchRowsPreservesProviderAndCursorErrors(t *testing.T) {
	providerErr := errors.New("provider list timeout")
	cursorErr := errors.New("cursor write failed")

	settled, err := settlePayoutBatchRows(
		[]sqlcdb.ListPendingGuajiBetOrdersRow{payoutBatchTestRow("101")},
		map[string]guaji.WebBetRecord{},
		func(sqlcdb.ListPendingGuajiBetOrdersRow, *guaji.BetSettlement) error {
			t.Fatal("recent commit called for historical row")
			return nil
		},
		func(sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error) {
			return resolveHistoricalSettlementResult(
				nil,
				7,
				false,
				providerErr,
				func(int, string) error { return cursorErr },
				func(*guaji.BetSettlement) error {
					t.Fatal("commit called after provider error")
					return nil
				},
				func() error {
					t.Fatal("cleanup called after provider error")
					return nil
				},
			)
		},
	)
	if settled != 0 {
		t.Fatalf("settled=%d, want 0", settled)
	}
	if !errors.Is(err, providerErr) || !errors.Is(err, cursorErr) {
		t.Fatalf("err=%v, want provider and cursor causes", err)
	}
	if !strings.Contains(err.Error(), "persist historical settlement cursor") {
		t.Fatalf("err=%q, want cursor persistence context", err)
	}
}

func TestSettlePayoutBatchRowsCountsRecentOnlyAfterCommit(t *testing.T) {
	rows := []sqlcdb.ListPendingGuajiBetOrdersRow{payoutBatchTestRow("101")}
	items := map[string]guaji.WebBetRecord{"101": {ID: 101, Settled: true}}
	commitErr := errors.New("settlement commit failed")

	settled, err := settlePayoutBatchRows(
		rows,
		items,
		func(sqlcdb.ListPendingGuajiBetOrdersRow, *guaji.BetSettlement) error { return commitErr },
		func(sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error) {
			t.Fatal("historical lookup called for recent row")
			return false, nil
		},
	)
	if settled != 0 || !errors.Is(err, commitErr) {
		t.Fatalf("settled=%d err=%v, want 0 and commit error", settled, err)
	}

	settled, err = settlePayoutBatchRows(
		rows,
		items,
		func(sqlcdb.ListPendingGuajiBetOrdersRow, *guaji.BetSettlement) error { return nil },
		func(sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error) {
			t.Fatal("historical lookup called for recent row")
			return false, nil
		},
	)
	if settled != 1 || err != nil {
		t.Fatalf("settled=%d err=%v, want 1 and nil", settled, err)
	}
}

func TestSettlePayoutBatchRowsCountsHistoricalOnlyAfterCommitAndCleanup(t *testing.T) {
	commitErr := errors.New("settlement commit failed")
	cleanupErr := errors.New("cursor cleanup failed")
	tests := []struct {
		name        string
		commitErr   error
		cleanupErr  error
		wantSettled int
		wantErr     error
		wantCleanup bool
	}{
		{name: "commit failure", commitErr: commitErr, wantErr: commitErr},
		{name: "cleanup failure", cleanupErr: cleanupErr, wantErr: cleanupErr, wantCleanup: true},
		{name: "commit and cleanup succeed", wantSettled: 1, wantCleanup: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupCalled := false
			settled, err := settlePayoutBatchRows(
				[]sqlcdb.ListPendingGuajiBetOrdersRow{payoutBatchTestRow("101")},
				map[string]guaji.WebBetRecord{},
				func(sqlcdb.ListPendingGuajiBetOrdersRow, *guaji.BetSettlement) error {
					t.Fatal("recent commit called for historical row")
					return nil
				},
				func(sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error) {
					return resolveHistoricalSettlementResult(
						&guaji.BetSettlement{Settled: true},
						0,
						false,
						nil,
						func(int, string) error {
							t.Fatal("cursor save called for settled provider result")
							return nil
						},
						func(*guaji.BetSettlement) error { return tt.commitErr },
						func() error {
							cleanupCalled = true
							return tt.cleanupErr
						},
					)
				},
			)
			if settled != tt.wantSettled || !errors.Is(err, tt.wantErr) {
				t.Fatalf("settled=%v err=%v, want settled=%v err=%v", settled, err, tt.wantSettled, tt.wantErr)
			}
			if cleanupCalled != tt.wantCleanup {
				t.Fatalf("cleanupCalled=%v, want %v", cleanupCalled, tt.wantCleanup)
			}
		})
	}
}

func payoutBatchTestRow(betID string) sqlcdb.ListPendingGuajiBetOrdersRow {
	return sqlcdb.ListPendingGuajiBetOrdersRow{
		ThirdPartyBetID: pgtype.Text{String: betID, Valid: true},
	}
}
