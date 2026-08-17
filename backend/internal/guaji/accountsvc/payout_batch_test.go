package accountsvc

import (
	"context"
	"testing"

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
