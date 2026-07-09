package betrecords

import "github.com/jackc/pgx/v5/pgtype"

type Mode string

const (
	ModeReal Mode = "real"
	ModeSim  Mode = "sim"
)

type Status string

const (
	StatusHit  Status = "hit"
	StatusMiss Status = "miss"
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
	Multiplier      string
	Round           string
	Amount          float64
	PnL             float64
	Status          Status
	BetContent      string
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
