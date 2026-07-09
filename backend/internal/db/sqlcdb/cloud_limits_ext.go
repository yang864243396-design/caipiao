package sqlcdb

// PausedRunningInstance 云端总限停投时返回的 running 实例快照。
type PausedRunningInstance struct {
	ID     string `json:"id"`
	SimBet bool   `json:"sim_bet"`
}
