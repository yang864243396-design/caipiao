package betrecords

import "github.com/jackc/pgx/v5/pgtype"

type Mode string

const (
	ModeReal Mode = "real"
	ModeSim  Mode = "sim"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusHit     Status = "hit"
	StatusMiss    Status = "miss"
)

type Row struct {
	ID               string
	ThirdPartyBetID  pgtype.Text
	SchemeID         string
	SchemeName       string
	LotteryCode      string
	Period           string
	ThirdPartyPeriod string
	PlayType         string
	Multiplier       string
	Round            string
	Amount           float64
	PnL              float64
	Status           Status
	BetContent       string
}

type Group struct {
	SchemeID   string  `json:"schemeId"`
	SchemeName string  `json:"schemeName"`
	TotalBet   float64 `json:"totalBet"`
	TotalPrize float64 `json:"totalPrize"`
	DayPnL     float64 `json:"dayPnl"`
	WinRate    float64 `json:"winRate"`
}

type Summary struct {
	TotalBet   float64 `json:"totalBet"`
	TotalPrize float64 `json:"totalPrize"`
	DayPnL     float64 `json:"dayPnl"`
	WinRate    float64 `json:"winRate"`
}

type GroupsResult struct {
	Mode     Mode       `json:"mode"`
	Days     int        `json:"days"`
	DateFrom string     `json:"dateFrom"`
	DateTo   string     `json:"dateTo"`
	Summary  Summary    `json:"summary"`
	Groups   GroupsPage `json:"groups"`
}

type GroupsPage struct {
	Items []Group  `json:"items"`
	Page  PageMeta `json:"page"`
}

type Item struct {
	ID         string  `json:"id"`
	RecordNo   string  `json:"recordNo"`
	Period     string  `json:"period"`
	Periods    string  `json:"periods,omitempty"`
	PlayType   string  `json:"playType"`
	Multiplier string  `json:"multiplier"`
	Round      string  `json:"round"`
	Amount     float64 `json:"amount"`
	PnL        float64 `json:"pnl"`
	Status     Status  `json:"status"`
	BetContent string  `json:"betContent,omitempty"`
}

type PageMeta struct {
	NextCursor *string `json:"nextCursor,omitempty"`
	HasMore    bool    `json:"hasMore"`
}

type DetailResult struct {
	SchemeID   string  `json:"schemeId"`
	SchemeName string  `json:"schemeName"`
	Mode       Mode    `json:"mode"`
	Days       int     `json:"days"`
	DateFrom   string  `json:"dateFrom"`
	DateTo     string  `json:"dateTo"`
	Summary    Summary `json:"summary"`
	Records    Page    `json:"records"`
}

type Page struct {
	Items []Item   `json:"items"`
	Page  PageMeta `json:"page"`
}

// ItemDetail 单笔投注详情（GET /client/cloud/bet-records/item/{recordNo}）。
type ItemDetail struct {
	RecordNo     string   `json:"recordNo"`
	ThirdPartyID string   `json:"thirdPartyId"`
	Period       string   `json:"period"`
	LotteryLabel string   `json:"lotteryLabel"`
	PlayType     string   `json:"playType"`
	Status       string   `json:"status"`
	StatusLabel  string   `json:"statusLabel"`
	DrawNumbers  string   `json:"drawNumbers"`
	BetUnits     *int     `json:"betUnits"`
	Multiplier   string   `json:"multiplier"`
	Round        string   `json:"round"`
	Amount       float64  `json:"amount"`
	Currency     string   `json:"currency"`
	PayoutAmount *float64 `json:"payoutAmount"`
	PlacedAt     string   `json:"placedAt"`
	BetContent   string   `json:"betContent"`
	// BetContentLines 已按玩法位段标好位名的展示行（如「千位 1 3 5」）。
	// 玩法无按位语义或位段解析不出时为空，前端回退到 BetContent 原样展示。
	BetContentLines []string `json:"betContentLines,omitempty"`
	SimBet          bool     `json:"simBet"`
}
