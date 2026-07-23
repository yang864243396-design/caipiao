package accountsvc

import "time"

type Account struct {
	ID              int64      `json:"id"`
	GuajiUsername   string     `json:"guajiUsername"`
	IsActive        bool       `json:"isActive"`
	BoundAt         time.Time  `json:"boundAt"`
	LastSyncAt      *time.Time `json:"lastSyncAt,omitempty"`
	LastTokenError  *string    `json:"lastTokenError,omitempty"`
	LastBetAt       *time.Time `json:"lastBetAt,omitempty"`
	ReauthFailCount int        `json:"reauthFailCount,omitempty"`
	// 当前启用且第三方 Token 不可用（过期或失效）
	AuthExpired bool `json:"authExpired,omitempty"`
}

type AuthStatus struct {
	HasActiveGuajiAuth bool `json:"hasActiveGuajiAuth"`
	BindingCount       int  `json:"bindingCount"`
	ActiveUsername     string `json:"activeUsername,omitempty"`
	// 有启用账号但其 Token 已过期或失效
	ActiveAuthExpired bool `json:"activeAuthExpired,omitempty"`
}

type BalanceResult struct {
	// Currency / Amount：当前主币种及其可用余额（兼容旧调用方）
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
	Username string  `json:"username,omitempty"`
	// 三币种可用余额（与第三方 users/i/info 同步后的快照）
	USDT float64 `json:"usdt"`
	TRX  float64 `json:"trx"`
	CNY  float64 `json:"cny"`
}

type BindInput struct {
	Username   string
	Password   string
	LoginKey   string
	GoogleCode string
	EmailCode  string
	PhoneCode  string
}

type BindResult struct {
	Account     *Account `json:"account,omitempty"`
	MFARequired bool     `json:"mfaRequired,omitempty"`
	LoginKey    string   `json:"loginKey,omitempty"`
}

type AdminAccountRow struct {
	ID             int64      `json:"id"`
	GuajiUsername  string     `json:"guajiUsername"`
	IsActive       bool       `json:"isActive"`
	BoundAt        time.Time  `json:"boundAt"`
	LastSyncAt     *time.Time `json:"lastSyncAt,omitempty"`
	LastTokenError *string    `json:"lastTokenError,omitempty"`
	LastBetAt      *time.Time `json:"lastBetAt,omitempty"`
}
