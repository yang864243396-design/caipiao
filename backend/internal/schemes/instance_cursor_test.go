package schemes

import (
	"testing"
	"time"
)

func TestInstanceCursor_roundTripNano(t *testing.T) {
	ts := time.Date(2026, 7, 22, 7, 20, 30, 123456789, time.UTC)
	cur := encodeInstanceCursor(ts, "inst-abc")
	gotAt, gotID, err := decodeInstanceCursor(cur)
	if err != nil {
		t.Fatal(err)
	}
	if gotID != "inst-abc" {
		t.Fatalf("id=%q", gotID)
	}
	if !gotAt.Valid || !gotAt.Time.Equal(ts) {
		t.Fatalf("time=%v want %v", gotAt.Time, ts)
	}
}

func TestInstanceCursor_legacySecondPrecision(t *testing.T) {
	// 兼容旧游标（秒级）
	_, id, err := decodeInstanceCursor("2026-07-22T07:20:30Z|legacy-id")
	if err != nil {
		t.Fatal(err)
	}
	if id != "legacy-id" {
		t.Fatalf("id=%q", id)
	}
}
