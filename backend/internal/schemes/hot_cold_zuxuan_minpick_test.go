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

func TestValidateSchemeBetContent_baodanSingleDan(t *testing.T) {
	t.Parallel()
	// 中三组选包胆：单胆合法；勿误报「号码池至少选择 3 个」；多胆拒。
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g002","subPlayId":"263",
		"catalogSubId":"zhong3_zuxuan_bd","betMode":"baodan",
		"playMethodLabel":"中三组选包胆","segmentLen":3
	}`)
	if vs := ValidateSchemeBetContent("custom", raw, "5", 0); len(vs) > 0 {
		t.Fatalf("单胆应通过: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "5,6", 0); !hasDetail(vs, "包胆：只能选择一个 0–9 的号码") {
		t.Fatalf("多胆应拒: %+v", vs)
	}
	cfg := []byte(`{
		"runTypeId":"fixed_rotate","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"263","catalogSubId":"zhong3_zuxuan_bd",
		"betMode":"baodan","playMethodLabel":"中三组选包胆",
		"schemeGroups":["5","7"]
	}`)
	for _, v := range ValidateSchemeConfig("custom", cfg) {
		if strings.Contains(v.Detail, "至少选择") {
			t.Fatalf("包胆方案不应报号池下限: %+v", v)
		}
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
