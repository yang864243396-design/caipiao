package schemes

import (
	"errors"
	"strings"
	"unicode/utf8"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
)

func normalizeBetFailedDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	detail = strings.ReplaceAll(detail, "\n", " ")
	if detail == "" {
		return ""
	}
	const maxRunes = 120
	if utf8.RuneCountInString(detail) > maxRunes {
		return string([]rune(detail)[:maxRunes]) + "…"
	}
	return detail
}

func guajiBetFailedDetail(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, guajibet.ErrNoActiveAuth):
		return "无启用中的授权账号"
	case errors.Is(err, guajibet.ErrTokenInvalid):
		return "授权已失效，请重新授权"
	case errors.Is(err, guajibet.ErrUpstream):
		return guajibet.ErrUpstream.Error()
	case errors.Is(err, guajibet.ErrInsufficient):
		return guajibet.ErrInsufficient.Error()
	case errors.Is(err, guajibet.ErrPeriodClosed):
		return guajibet.ErrPeriodClosed.Error()
	case errors.Is(err, guajibet.ErrZeroBets):
		return guajibet.ErrZeroBets.Error()
	}
	var api *guaji.APIError
	if errors.As(err, &api) && strings.TrimSpace(api.Message) != "" {
		return normalizeBetFailedDetail(api.Message)
	}
	// 接单失败包装：剥前缀，避免 Classify 把整串「第三方接单失败: …」原样吐出。
	if errors.Is(err, guajibet.ErrPlaceRejected) {
		base := guajibet.ErrPlaceRejected.Error()
		msg := strings.TrimSpace(err.Error())
		msg = strings.TrimPrefix(msg, base)
		msg = strings.TrimPrefix(msg, ":")
		msg = strings.TrimSpace(msg)
		if msg != "" && msg != base {
			return normalizeBetFailedDetail(msg)
		}
		return base
	}
	fault := guaji.ClassifyUpstreamError(err)
	if msg := strings.TrimSpace(fault.UserMessage); msg != "" {
		return normalizeBetFailedDetail(msg)
	}
	return normalizeBetFailedDetail(err.Error())
}
