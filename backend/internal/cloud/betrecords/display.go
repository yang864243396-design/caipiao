package betrecords

import (
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func thirdPartyBetOrderNo(thirdParty pgtype.Text) string {
	if !thirdParty.Valid {
		return ""
	}
	id := strings.TrimSpace(thirdParty.String)
	if id == "" {
		return ""
	}
	// 第三方 web_bets 注单号为纯数字（常见 6 位，如 398698）。
	for _, ch := range id {
		if ch < '0' || ch > '9' {
			return ""
		}
	}
	return id
}

func formatMultiplierDisplay(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "1"
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		n := int(math.Round(f))
		if n <= 0 {
			return "1"
		}
		return strconv.Itoa(n)
	}
	if i := strings.IndexByte(raw, '.'); i > 0 {
		if head := strings.TrimSpace(raw[:i]); head != "" {
			return head
		}
	}
	return raw
}

func formatRoundDisplay(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "1"
	}
	if slash := strings.IndexByte(raw, '/'); slash > 0 {
		if head := strings.TrimSpace(raw[:slash]); head != "" {
			return head
		}
	}
	return raw
}
