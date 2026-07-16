package guaji

import (
	"encoding/json"
	"fmt"
)

const (
	CodeMFARequired     = 40045
	CodeSecuritySetup   = 40061
	CodeSecurityRequired = 40060 // 下单等业务要求密保验证（实测：账号已设密保仍报，需第三方确认解锁机制）
	CodeTokenInvalid    = 401
	CodeTokenInvalidAlt = 42001
	// CodeTokenInvalidBiz 第三方业务码：无效令牌（users/i/info 等实测返回）。
	CodeTokenInvalidBiz = 40010
)

// MisconfiguredError indicates missing or invalid Guaji env configuration.
type MisconfiguredError struct {
	Msg string
}

func (e MisconfiguredError) Error() string { return e.Msg }

func ErrMisconfigured(msg string) error { return MisconfiguredError{Msg: msg} }

// APIError is a structured error from the third-party JSON envelope.
type APIError struct {
	Code    int
	Message string
	Extra   map[string]json.RawMessage
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("guaji api code=%d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("guaji api code=%d", e.Code)
}

// MFARequiredError is returned when login needs a second factor (T1 will handle).
type MFARequiredError struct {
	Code     int
	LoginKey string
	Extra    map[string]json.RawMessage
}

func (e *MFARequiredError) Error() string {
	return fmt.Sprintf("guaji login requires mfa (code=%d)", e.Code)
}
