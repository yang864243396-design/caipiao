package schemes

import "caipiao/backend/internal/cloud/schemestate"

// 注入正式盘派奖后的出号游标推进逻辑，打破
// schemes -> periodsync -> accountsvc -> schemestate 依赖链导致的 import cycle。
func init() {
	schemestate.FormalPickAdvancer = AdvancePickAfterFormalSettlement
}
