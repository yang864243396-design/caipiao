package schemes

import (
	"encoding/json"
	"testing"
)

func TestMergeDefinitionConfig_BuiltinPlanPreservesMaterializedPlay(t *testing.T) {
	existing := []byte(`{
		"runTypeId":"builtin_plan",
		"playTemplate":"ssc_std",
		"typeId":"g011",
		"subId":"74",
		"playTypeId":"g011",
		"subPlayId":"74",
		"betMode":"fushi",
		"betUnit":"1",
		"schemeGroups":["0\n1\n\n\n"],
		"builtinPlan":{"snapshotId":"SD1783802730483"}
	}`)

	got, err := mergeDefinitionConfig(existing, AddToCloudConfigPatch{
		SchemeGroups: []string{""},
		BetUnit:      "2",
		BetMode:      "dingwei",
		PlayTemplate: "ssc_std",
		TypeID:       "g006",
		SubID:        "13",
	})
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"typeId": "g011", "subId": "74", "playTypeId": "g011", "subPlayId": "74", "betMode": "fushi", "betUnit": "1",
	} {
		if got := cfg[key]; got != want {
			t.Fatalf("%s=%v want %q; config=%s", key, got, want, got)
		}
	}
	groups, ok := cfg["schemeGroups"].([]any)
	if !ok || len(groups) != 1 || groups[0] != "0\n1\n\n\n" {
		t.Fatalf("schemeGroups=%#v; config=%s", cfg["schemeGroups"], got)
	}
}

func TestMergeUpdateDefinitionConfig_BuiltinPlanIgnoresStaleBetFields(t *testing.T) {
	existing := []byte(`{"runTypeId":"builtin_plan","typeId":"g011","subId":"74","betMode":"fushi","betUnit":"1"}`)
	patch := UpdateDefinitionPatch{
		BetUnit:    "2",
		HasBetUnit: true,
		BetMode:    "dingwei",
		HasBetMode: true,
		TypeID:     "g006",
		SubID:      "13",
	}
	got, err := mergeUpdateDefinitionConfig(existing, patch, nil)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"typeId": "g011", "subId": "74", "betMode": "fushi", "betUnit": "1"} {
		if got := cfg[key]; got != want {
			t.Fatalf("%s=%v want %q; config=%s", key, got, want, got)
		}
	}
}
