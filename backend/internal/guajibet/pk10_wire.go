package guajibet

import "strings"

// isPK10DxdsComboMeta 冠亚/前三/后三「和值大小单双」（rule 221–223）。
func isPK10DxdsComboMeta(meta RuleMeta) bool {
	if meta.PlayTemplate != "pk10_std" || meta.TypeID != "g010" {
		return false
	}
	switch strings.TrimSpace(meta.RuleID) {
	case "221", "222", "223":
		return true
	}
	switch strings.TrimSpace(meta.SubID) {
	case "221", "222", "223", "dxds_guanya", "dxds_qian3", "dxds_hou3":
		return true
	}
	return false
}

// formatPK10DxdsComboWire 第三方 wire：「和」+ 大/小/单/双（抓包 rule 221–223）。
func formatPK10DxdsComboWire(groupContent string) string {
	tokens := splitPickTokens(groupContent)
	tok := "大"
	if len(tokens) > 0 && strings.TrimSpace(tokens[0]) != "" {
		tok = strings.TrimSpace(tokens[0])
	}
	return "和" + tok
}
