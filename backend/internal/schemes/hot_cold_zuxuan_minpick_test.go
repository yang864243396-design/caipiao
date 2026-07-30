package schemes

import (
	"strings"
	"testing"
)

func TestValidateHotColdWarm_zu3Zu6MinPick(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		betMode string
		subID   string
		pool    string
		ranks   string
		wantMsg string
	}{
		{
			name: "组三1码拒", betMode: "zu3", subID: "19",
			pool: `["1"]`, ranks: `[[0]]`, wantMsg: "组三至少选择 2 个号码",
		},
		{
			name: "组三2码过", betMode: "zu3", subID: "19",
			pool: `["1,2"]`, ranks: `[[0,1]]`, wantMsg: "",
		},
		{
			name: "组六2码拒", betMode: "zu6", subID: "261",
			pool: `["1,2"]`, ranks: `[[0,1]]`, wantMsg: "组六至少选择 3 个号码",
		},
		{
			name: "组六3码过", betMode: "zu6", subID: "261",
			pool: `["1,2,6"]`, ranks: `[[0,1,2]]`, wantMsg: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(`{
				"runTypeId":"hot_cold_warm",
				"playTemplate":"ssc_std",
				"playTypeId":"g002",
				"subPlayId":"` + tc.subID + `",
				"betMode":"` + tc.betMode + `",
				"schemeGroups":["0"],
				"hotColdWarm":{
					"totalPeriods":20,
					"strategy":"keep",
					"pickTypes":["hot"],
					"pool":` + tc.pool + `,
					"ranks":` + tc.ranks + `
				}
			}`)
			vs := ValidateSchemeConfig("custom", raw)
			if tc.wantMsg == "" {
				for _, v := range vs {
					if strings.Contains(v.Detail, "至少选择") {
						t.Fatalf("unexpected min-pick violation: %+v", vs)
					}
				}
				return
			}
			found := false
			for _, v := range vs {
				if v.Detail == tc.wantMsg {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("want detail %q, got %+v", tc.wantMsg, vs)
			}
		})
	}
}

func TestValidateSchemeBetContent_zu3Zu6MinPick(t *testing.T) {
	t.Parallel()
	zu3 := []byte(`{"playTemplate":"ssc_std","playTypeId":"g002","subPlayId":"19","betMode":"zu3"}`)
	zu6 := []byte(`{"playTemplate":"ssc_std","playTypeId":"g002","subPlayId":"261","betMode":"zu6"}`)
	if vs := ValidateSchemeBetContent("custom", zu3, "1", 0); !hasDetail(vs, "组三至少选择 2 个号码") {
		t.Fatalf("zu3 1 digit: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", zu3, "1,2", 0); hasDetail(vs, "至少选择") {
		t.Fatalf("zu3 2 digits should pass: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", zu6, "1,2", 0); !hasDetail(vs, "组六至少选择 3 个号码") {
		t.Fatalf("zu6 2 digits: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", zu6, "1,2,6", 0); hasDetail(vs, "至少选择") {
		t.Fatalf("zu6 3 digits should pass: %+v", vs)
	}
}

func hasDetail(vs []Violation, want string) bool {
	for _, v := range vs {
		if v.Detail == want {
			return true
		}
	}
	return false
}
