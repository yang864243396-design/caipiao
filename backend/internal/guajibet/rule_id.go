package guajibet

import (
	"encoding/json"
	"regexp"
	"strings"
)

var numericGuajiRuleID = regexp.MustCompile(`^[0-9]+$`)

// ExtractGuajiRuleID 从 outbound_play_code / segment_rule / sub_id 提取第三方数字 rule_id。
func ExtractGuajiRuleID(outboundPlayCode string, segmentRule []byte, subID string) string {
	outboundPlayCode = strings.TrimSpace(outboundPlayCode)
	if numericGuajiRuleID.MatchString(outboundPlayCode) {
		return outboundPlayCode
	}
	if id := segmentRuleGuajiRuleID(segmentRule); id != "" {
		return id
	}
	subID = strings.TrimSpace(subID)
	if numericGuajiRuleID.MatchString(subID) {
		return subID
	}
	if i := strings.LastIndex(outboundPlayCode, ":"); i >= 0 {
		tail := strings.TrimSpace(outboundPlayCode[i+1:])
		if numericGuajiRuleID.MatchString(tail) {
			return tail
		}
	}
	return ""
}

func segmentRuleGuajiRuleID(raw []byte) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var seg struct {
		GuajiRuleID string `json:"guajiRuleId"`
	}
	if json.Unmarshal(raw, &seg) != nil {
		return ""
	}
	seg.GuajiRuleID = strings.TrimSpace(seg.GuajiRuleID)
	if numericGuajiRuleID.MatchString(seg.GuajiRuleID) {
		return seg.GuajiRuleID
	}
	return ""
}

// IsNumericGuajiRuleID 判断是否为第三方 web_bets/lott 可接受的 rule_id。
func IsNumericGuajiRuleID(ruleID string) bool {
	return numericGuajiRuleID.MatchString(strings.TrimSpace(ruleID))
}
