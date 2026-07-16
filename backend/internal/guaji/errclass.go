package guaji

import (
	"errors"
	"regexp"
	"strings"
)

// UpstreamFault 第三方上游错误分类（是否应视为 Token 失效）。
type UpstreamFault struct {
	UserMessage    string
	IsTokenInvalid bool
}

var httpStatusRe = regexp.MustCompile(`status (\d{3})`)

// ClassifyUpstreamError 将 guaji 客户端/上游错误转为用户可读文案，并区分 Token 失效与临时故障。
func ClassifyUpstreamError(err error) UpstreamFault {
	if err == nil {
		return UpstreamFault{}
	}
	msg := err.Error()

	var api *APIError
	if errors.As(err, &api) {
		if isTokenInvalidCode(api.Code) || isTokenInvalidMessage(api.Message) {
			return UpstreamFault{UserMessage: "授权已失效，请重新授权", IsTokenInvalid: true}
		}
		if api.Message != "" {
			return UpstreamFault{UserMessage: api.Message, IsTokenInvalid: false}
		}
	}

	if isTokenInvalidMessage(msg) {
		return UpstreamFault{UserMessage: "授权已失效，请重新授权", IsTokenInvalid: true}
	}

	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "context deadline exceeded"),
		strings.Contains(lower, "timeout"),
		strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "tls:"),
		strings.Contains(lower, "eof"):
		return UpstreamFault{UserMessage: "第三方服务连接失败，请稍后重试", IsTokenInvalid: false}
	}

	if m := httpStatusRe.FindStringSubmatch(msg); len(m) == 2 {
		switch m[1] {
		case "401", "403":
			return UpstreamFault{UserMessage: "授权已失效，请重新授权", IsTokenInvalid: true}
		case "502", "503", "504", "429":
			return UpstreamFault{UserMessage: "第三方服务暂时不可用，请稍后重试", IsTokenInvalid: false}
		}
	}

	if strings.Contains(msg, "guaji login requires mfa") {
		return UpstreamFault{UserMessage: "需要二次验证，请重新绑定授权", IsTokenInvalid: true}
	}
	if strings.Contains(msg, "第三方账号或密码错误") || strings.Contains(lower, "invalid credentials") {
		return UpstreamFault{UserMessage: "第三方账号或密码错误", IsTokenInvalid: true}
	}

	// 已是友好中文文案（重新授权失败等）则原样保留。
	if !strings.Contains(msg, "guaji ") && !strings.Contains(msg, "body=") {
		return UpstreamFault{UserMessage: msg, IsTokenInvalid: true}
	}

	return UpstreamFault{UserMessage: "第三方服务异常，请稍后重试", IsTokenInvalid: false}
}

// IsPeriodClosedError 判断是否为封盘/截止类拒单（方案应继续运行等下期）。
func IsPeriodClosedError(err error) bool {
	if err == nil {
		return false
	}
	var api *APIError
	if errors.As(err, &api) {
		return isPeriodClosedMessage(api.Message)
	}
	return isPeriodClosedMessage(err.Error())
}

func isPeriodClosedMessage(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))
	if msg == "" {
		return false
	}
	keywords := []string{
		"封盘", "已封", "截止", "停售", "已开奖", "已过", "不能投注", "无法投注",
		"不可投注", "期已关", "投注时间", "投注截止", "不在销售", "未开盘",
		"closed", "not open", "period closed", "periods closed", "betting closed",
	}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

// IsTransientUpstreamError 临时上游故障（不应写入 last_token_error）。
func IsTransientUpstreamError(err error) bool {
	return !ClassifyUpstreamError(err).IsTokenInvalid
}

func isTokenInvalidCode(code int) bool {
	switch code {
	case CodeTokenInvalid, CodeTokenInvalidAlt, CodeTokenInvalidBiz:
		return true
	default:
		return false
	}
}

func isTokenInvalidMessage(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	lower := strings.ToLower(msg)
	needles := []string{
		"无效的令牌", "令牌无效", "请重新登录", "token invalid", "invalid token",
		"unauthorized", "jwt expired", "token expired",
	}
	for _, n := range needles {
		if strings.Contains(lower, strings.ToLower(n)) {
			return true
		}
	}
	return false
}
