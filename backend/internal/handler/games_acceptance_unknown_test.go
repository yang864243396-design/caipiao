package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"caipiao/backend/internal/games"
)

func TestHandleGamesErrExplainsUnknownAcceptanceWithoutSuggestingRetry(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&Handler{}).handleGamesErr(recorder, games.ErrGuajiAcceptanceUnknown)
	body := recorder.Body.String()
	if !strings.Contains(body, "\u63a5\u5355\u72b6\u6001\u672a\u77e5") {
		t.Fatalf("response=%s", body)
	}
	if strings.Contains(body, "\u8bf7\u7a0d\u540e\u91cd\u8bd5") {
		t.Fatalf("unknown acceptance must not suggest retry: %s", body)
	}
}
