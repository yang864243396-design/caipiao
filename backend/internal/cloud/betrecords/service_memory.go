package betrecords

import "github.com/jackc/pgx/v5/pgtype"

// 内存演示数据：仅 DB 不可用或未注入 memberID 时回退

var realRows = []Row{
	rowSeed{scheme: "sch-wan", name: "禄螭万位计划", period: "20240310031", play: "万位定位", thirdParty: "398701", mult: "2", round: "1", amount: 10, pnl: 5, status: StatusHit}.row(),
	rowSeed{scheme: "sch-qian", name: "千位稳赢方案", period: "20240310030", play: "千位定位", thirdParty: "398702", mult: "3", round: "2", amount: 20, pnl: -20, status: StatusMiss}.row(),
	rowSeed{scheme: "sch-ge", name: "个位轻量追号", period: "20220523029", play: "个位定位", thirdParty: "398703", mult: "2", round: "1", amount: 5, pnl: 2.5, status: StatusHit}.row(),
	rowSeed{scheme: "sch-bai", name: "百位进阶倍投", period: "20240310028", play: "百位定位", thirdParty: "398704", mult: "4", round: "3", amount: 50, pnl: 60, status: StatusHit}.row(),
	rowSeed{scheme: "sch-wan", name: "禄螭万位计划", period: "20240310027", play: "万位定位", thirdParty: "398705", mult: "1", round: "1", amount: 100, pnl: -100, status: StatusMiss}.row(),
	rowSeed{scheme: "sch-shi", name: "十位云策略", period: "20240310026", play: "十位定位", thirdParty: "398706", mult: "2", round: "1", amount: 10, pnl: 5, status: StatusHit}.row(),
	rowSeed{scheme: "sch-wan", name: "禄螭万位计划", period: "20240310025", play: "万位定位", thirdParty: "398707", mult: "2", round: "2", amount: 30, pnl: 15, status: StatusHit}.row(),
	rowSeed{scheme: "sch-ge", name: "个位轻量追号", period: "20240310024", play: "个位定位", thirdParty: "398708", mult: "2", round: "1", amount: 10, pnl: 5, status: StatusHit}.row(),
}

type rowSeed struct {
	scheme, name, period, play, thirdParty, mult, round string
	amount, pnl                                         float64
	status                                              Status
}

func (s rowSeed) row() Row {
	tp := pgtype.Text{}
	if id := s.thirdParty; id != "" {
		tp = pgtype.Text{String: id, Valid: true}
	}
	return Row{
		ID:              s.thirdParty,
		ThirdPartyBetID: tp,
		SchemeID:        s.scheme,
		SchemeName:      s.name,
		Period:          s.period,
		PlayType:        s.play,
		Multiplier:      s.mult,
		Round:           s.round,
		Amount:          s.amount,
		PnL:             s.pnl,
		Status:          s.status,
	}
}

func (s *Service) rows(mode Mode) []Row {
	if mode == ModeSim {
		return nil
	}
	out := make([]Row, len(realRows))
	copy(out, realRows)
	return out
}
