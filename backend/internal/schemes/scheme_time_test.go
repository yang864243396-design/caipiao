package schemes

import (
	"testing"
	"time"
)

func TestValidateSchemeStartTimeAfterNowEmptyAllowsOpen(t *testing.T) {
	cfg := []byte(`{"schemeFunds":"1000"}`)
	if err := validateSchemeStartTimeAfterNow(cfg, time.Now()); err != nil {
		t.Fatalf("empty start should allow open: %v", err)
	}
}

func TestSchemeConfigStartTimePlaceholderIgnored(t *testing.T) {
	cfg := []byte(`{"startTime":"0000-00-00 00:00:00","endTime":"0000-00-00 00:00:00"}`)
	if _, ok := schemeConfigStartTime(cfg); ok {
		t.Fatal("0000-00-00 placeholder should be unset")
	}
}

func TestSchemeConfigStartTimeLegacyDefaultPairIgnored(t *testing.T) {
	cfg := []byte(`{"startTime":"00:00","endTime":"23:59"}`)
	if _, ok := schemeConfigStartTime(cfg); ok {
		t.Fatal("legacy 00:00/23:59 pair should be treated as unset")
	}
	if schemeConfigEndTimeReached(cfg, time.Now()) {
		t.Fatal("legacy pair should not trigger end time")
	}
}

func TestValidateSchemeStartTimeAfterNowFutureRequired(t *testing.T) {
	future := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04:05")
	cfg := []byte(`{"startTime":"` + future + `"}`)
	if err := validateSchemeStartTimeAfterNow(cfg, time.Now()); err != nil {
		t.Fatalf("future start should pass: %v", err)
	}
	past := time.Now().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	cfgPast := []byte(`{"startTime":"` + past + `"}`)
	if err := validateSchemeStartTimeAfterNow(cfgPast, time.Now()); err == nil {
		t.Fatal("past start should fail")
	}
}

func TestEvaluateSchemeScheduleGate(t *testing.T) {
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.Local)
	futureStart := now.Add(time.Hour).Format("2006-01-02 15:04:05")
	pastStart := now.Add(-time.Minute).Format("2006-01-02 15:04:05")
	futureEnd := now.Add(time.Hour).Format("2006-01-02 15:04:05")
	pastEnd := now.Add(-time.Minute).Format("2006-01-02 15:04:05")

	cfgOpen := []byte(`{"startTime":"` + pastStart + `","endTime":"` + futureEnd + `"}`)
	if g := evaluateSchemeScheduleGate(cfgOpen, now); g != schemeScheduleOK {
		t.Fatalf("within window want ok got %v", g)
	}

	cfgBeforeStart := []byte(`{"startTime":"` + futureStart + `","endTime":"` + futureEnd + `"}`)
	if g := evaluateSchemeScheduleGate(cfgBeforeStart, now); g != schemeScheduleBeforeStart {
		t.Fatalf("before start got %v", g)
	}

	cfgPastEnd := []byte(`{"startTime":"` + pastStart + `","endTime":"` + pastEnd + `"}`)
	if g := evaluateSchemeScheduleGate(cfgPastEnd, now); g != schemeSchedulePastEnd {
		t.Fatalf("past end got %v", g)
	}
}

func TestValidateSchemeEndTimeNotReached(t *testing.T) {
	now := time.Now()
	future := now.Add(2 * time.Hour).Format("2006-01-02 15:04:05")
	cfg := []byte(`{"endTime":"` + future + `"}`)
	if err := validateSchemeEndTimeNotReached(cfg, now); err != nil {
		t.Fatalf("future end should allow start: %v", err)
	}
	past := now.Add(-time.Minute).Format("2006-01-02 15:04:05")
	cfgPast := []byte(`{"endTime":"` + past + `"}`)
	if err := validateSchemeEndTimeNotReached(cfgPast, now); err != ErrEndTimeReached {
		t.Fatalf("past end want ErrEndTimeReached got %v", err)
	}
}
