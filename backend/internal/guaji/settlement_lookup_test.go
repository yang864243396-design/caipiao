package guaji

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// This catches the historical-order failure where a payout lookup stops after
// the first three list pages and can never reach an older accepted bet.
func TestFindWebBetFromPageRangeFindsHistoricalBet(t *testing.T) {
	const targetID = "126458342"
	var requested []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/web_bets/" {
			http.NotFound(w, r)
			return
		}
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			t.Fatalf("page: %v", err)
		}
		requested = append(requested, page)
		items := []map[string]any{}
		if page == 4 {
			items = append(items, map[string]any{
				"id":         126458342,
				"bet_amount": 8,
				"settled":    true,
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 201, "data": items})
	}))
	defer srv.Close()

	c := NewClient(Config{Enabled: true, HTTPBase: srv.URL, AuthBase: srv.URL, WSBase: "wss://example.test", HTTPTimeout: time.Second})
	got, nextPage, exhausted, err := c.FindWebBetFromPageRange(context.Background(), "token", targetID, 4, 3)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got == nil || got.ID != 126458342 {
		t.Fatalf("found=%+v", got)
	}
	if nextPage != 4 || exhausted {
		t.Fatalf("nextPage=%d exhausted=%v", nextPage, exhausted)
	}
	if len(requested) != 1 || requested[0] != 4 {
		t.Fatalf("requested pages=%v, want [4]", requested)
	}
}
