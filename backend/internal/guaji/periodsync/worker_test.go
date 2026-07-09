package periodsync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/lottery"
)

func TestMergeSyncTarget_dedupAndParse(t *testing.T) {
	seen := map[string]bool{}
	tgt, ok := mergeSyncTarget(seen, " tron_ffc_1m ", " 29 ")
	if !ok || tgt.lotteryCode != "tron_ffc_1m" || tgt.gameID != 29 {
		t.Fatalf("first=%+v ok=%v", tgt, ok)
	}
	if _, ok := mergeSyncTarget(seen, "tron_ffc_1m", "29"); ok {
		t.Fatal("expected duplicate lottery to be skipped")
	}
}

func TestMergeSyncTarget_skipsInvalidGameID(t *testing.T) {
	seen := map[string]bool{}
	cases := []struct {
		lottery, gameKey string
	}{
		{"", "29"},
		{"tron_ffc_1m", ""},
		{"tron_ffc_1m", "abc"},
		{"tron_ffc_1m", "0"},
		{"tron_ffc_1m", "-1"},
	}
	for _, c := range cases {
		if _, ok := mergeSyncTarget(seen, c.lottery, c.gameKey); ok {
			t.Fatalf("expected skip for %+v", c)
		}
	}
}

func TestWorker_syncToken_cacheHit(t *testing.T) {
	w := &Worker{interval: defaultSyncInterval}
	w.cachedToken = "cached-token"
	w.tokenUntil = time.Now().Add(tokenCacheTTL)

	got, err := w.syncToken(context.Background())
	if err != nil || got != "cached-token" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestWorker_invalidateToken(t *testing.T) {
	w := &Worker{}
	w.cachedToken = "tok"
	w.tokenUntil = time.Now().Add(tokenCacheTTL)
	w.invalidateToken()
	if w.cachedToken != "" || !w.tokenUntil.IsZero() {
		t.Fatalf("token cache not cleared: token=%q until=%v", w.cachedToken, w.tokenUntil)
	}
}

func TestWorker_syncTargets_cacheHit(t *testing.T) {
	w := &Worker{}
	want := []syncTarget{{lotteryCode: "tron_ffc_1m", gameID: 29}}
	w.targetsCache = append([]syncTarget(nil), want...)
	w.targetsUntil = time.Now().Add(targetsCacheTTL)

	got, err := w.syncTargets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}

func TestWorker_syncOne_updatesSchedule(t *testing.T) {
	endLocal := time.Now().UTC().Add(40 * time.Second)
	endTime := endLocal.Format("2006-01-02 15:04:05")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "/api/web_bets/lott/periods") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Authorization"); !strings.Contains(got, "Bearer test-token") {
			t.Fatalf("auth=%q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 201,
			"data": []map[string]string{
				{
					"period":     "115001",
					"start_time": endLocal.Add(-time.Minute).Format("2006-01-02 15:04:05"),
					"end_time":   endTime,
				},
			},
		})
	}))
	defer srv.Close()

	client := guaji.NewClient(guaji.Config{
		Enabled:     true,
		HTTPBase:    srv.URL,
		AuthBase:    srv.URL,
		WSBase:      "wss://example.test/ws",
		HTTPTimeout: 5 * time.Second,
	})
	worker := &Worker{client: client}

	code := "periodsync_sync_one_test"
	now := time.Now()
	if err := worker.syncOne(context.Background(), "test-token", syncTarget{lotteryCode: code, gameID: 29}, now); err != nil {
		t.Fatal(err)
	}

	ps, ok := lottery.PeriodsScheduleFor(code)
	if !ok || ps.CurrentPeriod != "115001" || ps.StartSkipPeriod != "115001" {
		t.Fatalf("schedule=%+v ok=%v", ps, ok)
	}
	rem, ok := lottery.PeriodsCountdownSec(code, now)
	if !ok || rem < 35 || rem > 40 {
		t.Fatalf("countdown=%d ok=%v want ~40", rem, ok)
	}
}

func TestWorker_tick_skipsFreshSchedule(t *testing.T) {
	code := "periodsync_fresh_skip_test"
	closeAt := time.Now().UTC().Add(time.Minute)
	lottery.UpdatePeriodsSchedule(code, "115002", closeAt)

	w := &Worker{
		interval: defaultSyncInterval,
		client: guaji.NewClient(guaji.Config{
			Enabled:     true,
			HTTPBase:    "http://127.0.0.1:1",
			AuthBase:    "http://127.0.0.1:1",
			WSBase:      "wss://example.test/ws",
			HTTPTimeout: time.Millisecond,
		}),
	}
	w.targetsCache = []syncTarget{{lotteryCode: code, gameID: 29}}
	w.targetsUntil = time.Now().Add(targetsCacheTTL)
	w.cachedToken = "tok"
	w.tokenUntil = time.Now().Add(tokenCacheTTL)

	// fresh schedule should skip syncOne; unreachable HTTP base must not be called.
	w.tick(context.Background())

	ps, ok := lottery.PeriodsScheduleFor(code)
	if !ok || ps.CurrentPeriod != "115002" {
		t.Fatalf("schedule changed unexpectedly: %+v ok=%v", ps, ok)
	}
}
