package schemes

import "testing"

func TestValidateSchemeBetContent_ren4Zu24TriggerCell(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"143","catalogSubId":"143",
		"betMode":"zu24","playMethodLabel":"任四组选24","renPositionCount":4,"segmentLen":1,
		"runTypeId":"adv_trigger_bet",
		"triggerBet":{
			"mode":"always_pos",
			"positionIdxs":[0,1,2,3],
			"openPositionIdx":0,
			"rows":[{"enabled":true,"open":"0","pos":"1,2,3,4","neg":"5,6,7,8"}]
		}
	}`)
	// 开某投某格子无位名前缀：勿把 1,2,3,4 误展成整注 1234 再误报「至少 4 码」
	if vs := ValidateSchemeBetContent("custom", raw, "1,2,3,4", 0); len(vs) > 0 {
		t.Fatalf("trigger cell 1,2,3,4 should pass: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "1,2,3", 0); !hasDetail(vs, "组选24至少选择 4 个号码") {
		t.Fatalf("3 digits: %+v", vs)
	}
	// 单个四位 token 不是逗号分隔的号池，按 0 注拒绝。
	if vs := ValidateSchemeBetContent("custom", raw, "1234", 0); !hasViolation(vs, ViolationZeroUnits) {
		t.Fatalf("1234 as one token should fail zero_units: %+v", vs)
	}

	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if cfg.Trigger == nil {
		t.Fatal("trigger config missing")
	}
	vs := validateAdvTriggerBetConfig("custom", raw, cfg)
	if len(vs) > 0 {
		t.Fatalf("adv trigger validate: %+v", vs)
	}
}
