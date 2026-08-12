package schemes

import "strings"

// resolveLHCPlayRule maps lhc_std catalog typeId/subId to settlement playRule.
// PlayTypeID 保留目录 type（如 g003），供查 sub_plays / rule_id；结算语义见 lhcSettlePlayType。
func resolveLHCPlayRule(typeID, subID, betMode string) playRule {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	if betMode == "" {
		betMode = inferLHCBetMode(typeID, subID)
	}
	// 仅规范化 betMode；勿改写 PlayTypeID（否则 resolve rule_id 会查 lhc_std/erquanzhong/280 而库内是 g003/280）
	if _, semMode := mapLHCCatalogPlay(typeID, subID, betMode); semMode != "" {
		betMode = semMode
	}
	return playRule{
		PlayTemplate:  "lhc_std",
		PlayTypeID:    typeID,
		SubPlayID:     betMode,
		BetMode:       betMode,
		CatalogSubID:  subID,
		// 方案内容均为单区号池/属性（如 01,13,25），不是时时彩按位；
		// SegmentLen=7 会让随机出号按 7 行生成，并误入直选复式「每位必填」校验。
		SegmentLen:    1,
		NumberPoolMin: 1,
		NumberPoolMax: 49,
	}
}

// mapLHCCatalogPlay 将目录 sub 映射为结算语义 type / betMode（不替代目录 typeId）。
func mapLHCCatalogPlay(typeID, subID, betMode string) (semanticType, semanticMode string) {
	switch strings.TrimSpace(subID) {
	case "279": // 二全中复式
		return "erquanzhong", "fushi"
	case "280": // 二全中拖头
		return "erquanzhong", "tuotou"
	case "281": // 二全中生肖对碰
		return "erquanzhong", "sx_dp"
	case "282": // 二全中尾数对碰
		return "erquanzhong", "ws_dp"
	case "283": // 二全中生尾对碰
		return "erquanzhong", "sw_dp"
	case "284": // 二全中任意对碰
		return "erquanzhong", "renyi_dp"
	case "285": // 二中特复式
		return "erzhongte", "fushi"
	case "287": // 二中特生肖对碰
		return "erzhongte", "sx_dp"
	case "288": // 二中特尾数对碰
		return "erzhongte", "ws_dp"
	case "289": // 二中特生尾对碰
		return "erzhongte", "sw_dp"
	case "290": // 二中特任意对碰
		return "erzhongte", "renyi_dp"
	case "293": // 特串生肖对碰
		return "techuan", "sx_dp"
	case "294": // 特串尾数对碰
		return "techuan", "ws_dp"
	case "295": // 特串生尾对碰
		return "techuan", "sw_dp"
	case "296": // 特串任意对碰
		return "techuan", "renyi_dp"
	case "286": // 二中特拖头
		return "erzhongte", "tuotou"
	case "291": // 特串复式
		return "techuan", "fushi"
	case "292": // 特串拖头
		return "techuan", "tuotou"
	case "297": // 三中二复式
		return "sanzhonger", "fushi"
	case "298": // 三中二拖头
		return "sanzhonger", "tuotou"
	case "299": // 三全中复式
		return "sanquanzhong", "fushi"
	case "300": // 三全中拖头
		return "sanquanzhong", "tuotou"
	case "272": // 特码A
		return "tema", "tema"
	}
	_ = typeID
	_ = betMode
	return "", ""
}

// lhcSettlePlayType 结算用玩法语义（二全中/二中特…）；目录 g003 等通过 CatalogSubID 映射。
func lhcSettlePlayType(rule playRule) string {
	if sem, _ := mapLHCCatalogPlay(rule.PlayTypeID, rule.CatalogSubID, rule.BetMode); sem != "" {
		return sem
	}
	tid := strings.TrimSpace(rule.PlayTypeID)
	switch tid {
	case "g003":
		// 连码缺省按二全中（最少选 2）
		return "erquanzhong"
	case "g001":
		return "tema"
	case "g002":
		return "zhengte"
	}
	return tid
}

func inferLHCBetMode(typeID, subID string) string {
	s := strings.ToLower(strings.TrimSpace(subID))
	typeID = strings.TrimSpace(typeID)
	if strings.HasPrefix(s, "zheng") && strings.HasSuffix(s, "_te") {
		return "zhengte"
	}
	// 目录数字 id：特码A=272；玩法树 g001=特码（仅在 resolveLHCPlayRule/lhc_std 路径调用）
	if s == "tema_a" || s == "272" || typeID == "tema" || typeID == "g001" {
		if !strings.HasPrefix(s, "zheng") {
			return "tema"
		}
	}
	if typeID == "buzhong_xuanyi" && (strings.HasSuffix(s, "bz") || strings.HasSuffix(s, "x1")) {
		if strings.HasSuffix(s, "x1") {
			return "xuanyi"
		}
		return "buzhong"
	}
	if strings.HasSuffix(s, "_bz") && strings.Contains(s, "xiao") {
		return "xiao_bz"
	}
	if strings.HasSuffix(s, "_z") && strings.Contains(s, "xiao") {
		return "xiao_z"
	}
	if strings.Contains(s, "xiao") {
		return "xiao"
	}
	if strings.HasSuffix(s, "_bz") && strings.Contains(s, "wei") {
		return "wei_bz"
	}
	if strings.HasSuffix(s, "_z") && strings.Contains(s, "wei") {
		return "wei_z"
	}
	if strings.HasSuffix(s, "_rz") {
		return "renzhong"
	}
	return s
}

func isLHCLotteryCode(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	return strings.HasPrefix(code, "tron_lhc")
}
