package games

import "strings"

// categoryCodeToBetCategory maps lottery_catalog.category_code to bet_orders.lottery_category.
func categoryCodeToBetCategory(categoryCode string) string {
	switch strings.TrimSpace(categoryCode) {
	case "pk10", "feiting":
		return "pk10"
	case "k3":
		return "k3"
	case "syxw", "x5":
		return "x5"
	case "ffc", "jisu", "ssc", "pc28", "lhc":
		return "ssc"
	default:
		return "other"
	}
}
