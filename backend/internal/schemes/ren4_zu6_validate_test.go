package schemes

import "testing"

func TestValidateSchemeBetContent_ren4Zu6MinPickAndUnits(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"145","catalogSubId":"145",
		"betMode":"zu6","playMethodLabel":"任选四组选6","renPositionCount":4,"segmentLen":1
	}`)

	if vs := ValidateSchemeBetContent("custom", raw, "万,千,百,十\n1", 0); !hasDetail(vs, "组选6至少选择 2 个号码") {
		t.Fatalf("1 digit should fail: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "万,千,百,十\n1,2", 0); len(vs) > 0 {
		t.Fatalf("2 digits should pass: %+v", vs)
	}

	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if !isSixingZu6PlayRule(cfg.Play) {
		t.Fatalf("want sixing zu6 rule, got %+v", cfg.Play)
	}
	if got := zuxuanPoolMinPick(cfg.Play); got != 2 {
		t.Fatalf("minPick=%d want 2", got)
	}
	if got := zuxuanPoolUnitsForRule(cfg.Play, []string{"1", "2"}); got != 1 {
		t.Fatalf("units(1,2)=%d want 1", got)
	}
	if got := zuxuanPoolUnitsForRule(cfg.Play, []string{"1", "2", "3"}); got != 3 {
		t.Fatalf("units(1,2,3)=%d want 3", got)
	}

	// C(5,4)×C(2,2) = 5×1
	if got := countRenxuanNeedsPositionBetUnits(cfg.Play, "万,千,百,十,个\n1,2"); got != 5 {
		t.Fatalf("betUnits=%d want 5", got)
	}
	// C(4,4)×C(3,2) = 3
	if got := countRenxuanNeedsPositionBetUnits(cfg.Play, "万,千,百,十\n1,2,3"); got != 3 {
		t.Fatalf("betUnits=%d want 3", got)
	}
}

func TestValidateSchemeConfig_ren4Zu6TypeIdSubIdPayload(t *testing.T) {
	t.Parallel()
	// 前端保存常只带 typeId/subId（无 playMethodLabel），须仍识别为组选6
	raw := []byte(`{
		"simBet":false,"schemeFunds":"10000","schemeCurrency":"USDT",
		"betMode":"zu6","betUnit":"2","playTemplate":"ssc_std",
		"schemeGroups":[
			"万,千,百,十\n1,2",
			"万,千,百,十\n0,1,2,3,4,5,6,7,8,9",
			"万,千,百,十,个\n3,4",
			"万,千,百,十,个\n0,1,2,3,4,5,6,7,8,9"
		],
		"subId":"145","typeId":"g011"
	}`)
	if vs := ValidateSchemeConfig("custom", raw); len(vs) > 0 {
		t.Fatalf("user-like payload should pass: %+v", vs)
	}
}

func TestValidateSchemeBetContent_sixingZu6FlatDigitRun(t *testing.T) {
	t.Parallel()
	// 四星组选6：粘连号池「1234567890,1234567890」须按位拆开，勿报「不在合法号池」
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g013","subPlayId":"132","catalogSubId":"132",
		"betMode":"zu6","playMethodLabel":"组选6","segmentLen":4,
		"playTypeLabel":"四星","guajiGroup":"四星"
	}`)
	if vs := ValidateSchemeBetContent("custom", raw, "1234567890,1234567890", 0); len(vs) > 0 {
		t.Fatalf("flat digit run should pass: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "12", 0); len(vs) > 0 {
		t.Fatalf("glued 12 should expand and pass: %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "1,2", 0); len(vs) > 0 {
		t.Fatalf("1,2 should pass: %+v", vs)
	}
}

func TestZuxuanPoolMinPick_zhong3Zu6StillThree(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"playTemplate":"ssc_std","playTypeId":"g002","subPlayId":"261","betMode":"zu6","playMethodLabel":"中三组六"}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if isSixingZu6PlayRule(cfg.Play) {
		t.Fatalf("zhong3 zu6 should not be sixing: %+v", cfg.Play)
	}
	if got := zuxuanPoolMinPick(cfg.Play); got != 3 {
		t.Fatalf("minPick=%d want 3", got)
	}
	if got := zuxuanPoolUnitsForRule(cfg.Play, []string{"1", "2"}); got != 0 {
		t.Fatalf("units(1,2)=%d want 0", got)
	}
	if got := zuxuanPoolUnitsForRule(cfg.Play, []string{"1", "2", "6"}); got != 1 {
		t.Fatalf("units(1,2,6)=%d want 1", got)
	}
}
