package guaji

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchHistoryDrawLogs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/lottery_logs" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("page") != "1" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":1,"created":"2026-07-01T08:53:36+00:00","block_time":"2026-07-01T08:53:36+00:00","periods":"1014012500271","last5_num":"37844","last11_5_num":"11,03,07,08,04","last_pk10_num":"03,06,09,10,02,04,08,01,05,07","last_k3_num":"2,4,4"}],"code":"0","page":1,"per_page":2,"count":3}`))
	}))
	defer srv.Close()

	c := NewClient(Config{
		Enabled:     true,
		HTTPBase:    srv.URL,
		AuthBase:    srv.URL,
		WSBase:      "wss://example.test",
		HTTPTimeout: 5 * time.Second,
	})
	logs, err := c.FetchHistoryDrawLogs(context.Background(), "lottery_logs", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("len=%d", len(logs))
	}
	if logs[0].Periods != "1014012500271" {
		t.Fatalf("periods=%q", logs[0].Periods)
	}
	balls := logs[0].Balls.BallsFor("ssc_std")
	if len(balls) != 5 || balls[0] != "3" {
		t.Fatalf("balls=%v", balls)
	}
}
