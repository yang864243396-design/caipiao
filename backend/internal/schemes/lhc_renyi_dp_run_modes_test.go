package schemes

import (
	"strings"
	"testing"
)

func TestLHCRenyiDuipengHotColdPickBuildsDistinctZones(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm","playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp",
		"hotColdWarm":{"totalPeriods":20,"ranks":[[0,1],[2]]}
	}`)
	draws := [][]string{
		{"1", "2", "3", "4", "5", "6", "7"},
		{"1", "2", "3", "8", "9", "10", "11"},
	}

	got := buildHotColdPickContent(cfg, draws)
	if !strings.Contains(got, "|") {
		t.Fatalf("content=%q, want A|B content", got)
	}
	if vs := validateLHCRenyiDuipengBetContent(got); len(vs) != 0 {
		t.Fatalf("content=%q violations=%+v", got, vs)
	}
	parts := strings.Split(got, "|")
	if len(parts) != 2 || len(parseTextTokens(parts[0])) != 2 || len(parseTextTokens(parts[1])) != 1 {
		t.Fatalf("content=%q, want two A picks and one B pick", got)
	}
}

func TestLHCRenyiDuipengHotColdPickPadsSingleDigitNumbers(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm","playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp",
		"hotColdWarm":{"totalPeriods":20,"ranks":[[0,1],[2]]}
	}`)
	got := buildHotColdPickContent(cfg, [][]string{
		{"1", "2", "3", "4", "5", "6", "7"},
		{"1", "2", "3", "8", "9", "10", "11"},
	})

	for _, side := range strings.Split(got, "|") {
		for _, token := range parseTextTokens(side) {
			if len(token) != 2 {
				t.Fatalf("content=%q includes non-padded LHC number %q", got, token)
			}
		}
	}
}

func TestLHCRenyiDuipengHotColdPickSkipsIncompleteZones(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm","playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp",
		"hotColdWarm":{"totalPeriods":20,"ranks":[[0,1]]}
	}`)

	if got := buildHotColdPickContent(cfg, [][]string{{"1", "2", "3", "4", "5", "6", "7"}}); got != "" {
		t.Fatalf("content=%q, want incomplete A/B ranks to skip pick", got)
	}
}

func TestLHCRenyiDuipengHotColdPickRanksAllLhcBallsByFrequency(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"hot_cold_warm","playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp",
		"hotColdWarm":{"totalPeriods":20,"ranks":[[0],[1]]}
	}`)
	draws := [][]string{
		{"49", "49", "49", "1", "1", "2", "3"},
		{"49", "4", "5", "6", "7", "8", "9"},
	}

	if got := buildHotColdPickContent(cfg, draws); got != "49|01" {
		t.Fatalf("content=%q, want hottest 49 in A and next-hottest 01 in B", got)
	}
}

func TestValidateSchemeConfig_LHCRenyiDuipengHotColdRejectsOverlappingRanks(t *testing.T) {
	raw := []byte(`{
		"runTypeId":"hot_cold_warm","playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp",
		"hotColdWarm":{"totalPeriods":20,"ranks":[[0],[0]]},"schemeGroups":["01|02"]
	}`)
	for _, violation := range ValidateSchemeConfig("custom", raw) {
		if strings.Contains(violation.Detail, "冷热任意对碰") {
			return
		}
	}
	t.Fatal("overlapping A/B hot-cold ranks should be rejected on save")
}
