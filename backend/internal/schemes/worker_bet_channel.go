package schemes

import "caipiao/backend/internal/db/sqlcdb"

// usesGuajiThirdParty 未开启模拟投注时走第三方真实下单（仅看 simBet 开关）。
func (w *Worker) usesGuajiThirdParty(inst sqlcdb.SchemeInstance) bool {
	if inst.SimBet {
		return false
	}
	return w != nil && w.guajiRealEnabled()
}

// requiresGuajiRealBet 未开启模拟投注时必须走第三方。
func requiresGuajiRealBet(inst sqlcdb.SchemeInstance) bool {
	return !inst.SimBet
}
