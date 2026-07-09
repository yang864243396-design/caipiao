package schemes

import (
	"fmt"
	"testing"
)

func TestBuildPlanTrendPeriodHistory_mergesConsecutive(t *testing.T) {
	rows := []detailPeriodRow{
		{periodShort: "461", hit: false},
		{periodShort: "462", hit: false},
		{periodShort: "463", hit: true},
		{periodShort: "464", hit: false},
		{periodShort: "465", hit: false},
	}
	history := buildPlanTrendPeriodHistory(rows, detailPreviewHistoryLimit)
	if len(history) != 3 {
		t.Fatalf("history=%d want 3", len(history))
	}
	if history[0].Period != "464 - 465" || history[0].Win {
		t.Fatalf("first=%+v", history[0])
	}
	if history[1].Period != "463" || !history[1].Win {
		t.Fatalf("second=%+v", history[1])
	}
	if history[2].Period != "461 - 462" || history[2].Win {
		t.Fatalf("third=%+v", history[2])
	}
}

func TestBuildPlanTrendPeriodHistory_last100(t *testing.T) {
	rows := make([]detailPeriodRow, 0, 105)
	for i := 0; i < 105; i++ {
		rows = append(rows, detailPeriodRow{
			periodShort: fmt.Sprintf("%03d", i+1),
			hit:         i%2 == 0,
		})
	}
	history := buildPlanTrendPeriodHistory(rows, detailPreviewHistoryLimit)
	if len(history) != detailPreviewHistoryLimit {
		t.Fatalf("history=%d want %d", len(history), detailPreviewHistoryLimit)
	}
	if history[0].Period != "105" || history[len(history)-1].Period != "006" {
		t.Fatalf("range got first=%s last=%s", history[0].Period, history[len(history)-1].Period)
	}
}

func TestBuildPlanTrendPeriodHistory_allMissMerged(t *testing.T) {
	rows := make([]detailPeriodRow, 0, 7)
	for _, p := range []string{"461", "462", "463", "464", "465", "466", "467"} {
		rows = append(rows, detailPeriodRow{periodShort: p, hit: false})
	}
	history := buildPlanTrendPeriodHistory(rows, detailPreviewHistoryLimit)
	if len(history) != 1 {
		t.Fatalf("history=%d want 1", len(history))
	}
	if history[0].Period != "461 - 467" || history[0].Win {
		t.Fatalf("got %+v", history[0])
	}
}

func TestBuildPlanTrendHistory_allMiss(t *testing.T) {
	rows := make([]detailPeriodRow, 0, 7)
	for _, p := range []string{"461", "462", "463", "464", "465", "466", "467"} {
		rows = append(rows, detailPeriodRow{periodShort: p, hit: false})
	}
	history := buildPlanTrendHistory(rows)
	if len(history) != 1 {
		t.Fatalf("history=%d want 1", len(history))
	}
	if history[0].Period != "461 - 467" || history[0].Win {
		t.Fatalf("got %+v", history[0])
	}
}

func TestBuildPlanTrendHistory_hitThenMissStreak(t *testing.T) {
	rows := []detailPeriodRow{
		{periodShort: "461", hit: false},
		{periodShort: "462", hit: false},
		{periodShort: "463", hit: true},
		{periodShort: "464", hit: false},
		{periodShort: "465", hit: false},
	}
	history := buildPlanTrendHistory(rows)
	if len(history) != 2 {
		t.Fatalf("history=%d want 2", len(history))
	}
	if history[0].Period != "464 - 465" || history[0].Win {
		t.Fatalf("first=%+v", history[0])
	}
	if history[1].Period != "461 - 463" || !history[1].Win {
		t.Fatalf("second=%+v", history[1])
	}
}

func TestBuildPlanTrendChart_cumulativeScore(t *testing.T) {
	rows := []detailPeriodRow{
		{periodShort: "461", hit: false},
		{periodShort: "462", hit: false},
		{periodShort: "463", hit: true},
		{periodShort: "464", hit: false},
	}
	chart := buildPlanTrendChart(rows)
	want := []int{-1, -2, -1, -2}
	if len(chart) != len(want) {
		t.Fatalf("chart=%d", len(chart))
	}
	for i, score := range want {
		if chart[i].Round != score {
			t.Fatalf("chart[%d].Round=%d want %d", i, chart[i].Round, score)
		}
	}
}
