package guaji

import (
	"encoding/json"
	"strconv"
	"strings"
)

// flexCode 兼容第三方 code 为 int 或 string（如 web_bets 列表返回 "code":"0"）。
type flexCode int

func (c flexCode) Int() int { return int(c) }

func (c *flexCode) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*c = 0
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		str = strings.TrimSpace(str)
		if str == "" {
			*c = 0
			return nil
		}
		n, err := strconv.Atoi(str)
		if err != nil {
			return err
		}
		*c = flexCode(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*c = flexCode(n)
	return nil
}

// envelope is the common third-party JSON wrapper.
//
// 第三方响应格式不统一（实测）：
//   - /auth/login         {"success":true,"data":{...}}（无 code）
//   - /api/users/i/info   裸对象（无 code/data 包裹）
//   - /api/agents/i/...   {"code":0,"data":{...}}（code=0 成功）
//   - .../periods         {"code":201,"data":[...]}（201 成功）
type envelope struct {
	Success *bool           `json:"success"`
	Code    flexCode        `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Extra   json.RawMessage `json:"extra"`
}

func (e envelope) dataInto(v any) error {
	if len(e.Data) == 0 || string(e.Data) == "null" {
		return nil
	}
	return json.Unmarshal(e.Data, v)
}

func (e envelope) extraMap() map[string]json.RawMessage {
	if len(e.Extra) == 0 || string(e.Extra) == "null" {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(e.Extra, &m); err != nil {
		return nil
	}
	return m
}

// LoginRequest is POST /auth/login body.
type LoginRequest struct {
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	IsAI        bool   `json:"is_ai,omitempty"`
	LoginKey    string `json:"login_key,omitempty"`
	GoogleCode  string `json:"google_code,omitempty"`
	Email       string `json:"email,omitempty"`
	EmailCode   string `json:"email_code,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Phone       string `json:"phone,omitempty"`
	PhoneCode   string `json:"phone_code,omitempty"`
}

// LoginResult is a successful login token bundle.
type LoginResult struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	Username     string `json:"username"`
	TokenType    string `json:"token_type"`
}

// UserAccount holds wallet fields from GET /api/users/i/info.
type UserAccount struct {
	Balance              float64 `json:"balance"`
	AvailableBalance     float64 `json:"available_balance"`
	BalanceTRX           float64 `json:"balance_trx"`
	AvailableBalanceTRX  float64 `json:"available_balance_trx"`
	BalanceCNY           float64 `json:"balance_cny"`
	AvailableBalanceCNY  float64 `json:"available_balance_cny"`
	BalanceFixed         float64 `json:"balance_fixed"`
	BalanceFixedCNY      float64 `json:"balance_fixed_cny"`
}

// UserInfo is GET /api/users/i/info data.
type UserInfo struct {
	ID       int64       `json:"id"`
	Username string      `json:"username"`
	Account  UserAccount `json:"account"`
}

// CNYBalance returns the CNY balance per product spec (§21.1: missing → 0).
func (u UserInfo) CNYBalance() float64 {
	if u.Account.AvailableBalanceCNY != 0 {
		return u.Account.AvailableBalanceCNY
	}
	if u.Account.BalanceCNY != 0 {
		return u.Account.BalanceCNY
	}
	return u.Account.BalanceFixedCNY
}

// 主币种枚举（§4.4：USDT/TRX/CNY）。
const (
	CurrencyUSDT = "USDT"
	CurrencyTRX  = "TRX"
	CurrencyCNY  = "CNY"
)

// NormalizeCurrency 归一化主币种，非法值回退 CNY。
func NormalizeCurrency(c string) string {
	switch c {
	case CurrencyUSDT, CurrencyTRX, CurrencyCNY:
		return c
	default:
		return CurrencyCNY
	}
}

// CurrencyCode 主币种 → 第三方数字编码（接口文档 §11：0 usdt / 1 trx / 3 cny）。
func CurrencyCode(c string) int {
	switch NormalizeCurrency(c) {
	case CurrencyUSDT:
		return 0
	case CurrencyTRX:
		return 1
	default: // CNY
		return 3
	}
}

// BalanceByCurrency 按主币种返回 users/i/info 中对应可用余额（实测字段映射）。
//   USDT → available_balance / balance；TRX → available_balance_trx / balance_trx；CNY → balance_cny
func (u UserInfo) BalanceByCurrency(currency string) float64 {
	switch NormalizeCurrency(currency) {
	case CurrencyTRX:
		if u.Account.AvailableBalanceTRX != 0 {
			return u.Account.AvailableBalanceTRX
		}
		return u.Account.BalanceTRX
	case CurrencyUSDT:
		if u.Account.AvailableBalance != 0 {
			return u.Account.AvailableBalance
		}
		return u.Account.Balance
	default:
		return u.CNYBalance()
	}
}

// ProbeResult summarizes connectivity for health / smoke checks (T0).
type ProbeResult struct {
	Enabled       bool     `json:"enabled"`
	HTTPBase      string   `json:"httpBase,omitempty"`
	AuthBase      string   `json:"authBase,omitempty"`
	WSBase        string   `json:"wsBase,omitempty"`
	HTTPReachable bool     `json:"httpReachable"`
	HTTPError     string   `json:"httpError,omitempty"`
	WSReachable   bool     `json:"wsReachable"`
	WSError       string   `json:"wsError,omitempty"`
	LoginOK      bool     `json:"loginOk"`
	LoginError   string   `json:"loginError,omitempty"`
	MFARequired  bool     `json:"mfaRequired,omitempty"`
	TestUsername string   `json:"testUsername,omitempty"`
	BalanceCNY   *float64 `json:"balanceCny,omitempty"`
}
