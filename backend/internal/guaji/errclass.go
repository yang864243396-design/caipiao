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

	// 已是友好中文文案：仅授权/登录类标 Token 失效。
	// 切勿把「接单失败 / 注数为0 / 暂时不可用」一并标成 Token 失效，
	// 否则 IsRetryableTransportError 会误停投（ErrUpstream 曾因此被 pause）。
	if !strings.Contains(msg, "guaji ") && !strings.Contains(msg, "body=") {
		if isTokenInvalidMessage(msg) || isAuthFailureChinese(msg) {
			return UpstreamFault{UserMessage: msg, IsTokenInvalid: true}
		}
		return UpstreamFault{UserMessage: msg, IsTokenInvalid: false}
	}

	return UpstreamFault{UserMessage: "第三方服务异常，请稍后重试", IsTokenInvalid: false}
}

// isAuthFailureChinese 本地/网关返回的授权失败中文（无 guaji body 包装）。
func isAuthFailureChinese(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	needles := []string{
		"重新授权", "需要二次验证", "重新绑定授权", "授权已失效",
		"账号或密码错误", "无启用中的授权",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
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

// IsRetryableTransportError 传输层/网关/上游临时故障：方案 Worker 应保留运行并下 tick 再试，勿立刻停投。
// 不含业务拒单、封盘、余额不足、Token 失效（那些仍应按原逻辑停投或跳过）。
func IsRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	if IsPeriodClosedError(err) {
		return false
	}
	if IsBlockFetchError(err) {
		return true
	}
	fault := ClassifyUpstreamError(err)
	if fault.IsTokenInvalid {
		return false
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"context deadline exceeded", "timeout", "connection refused", "connectex",
		"no such host", "tls:", "eof", "status 502", "status 503", "status 504", "status 429",
		"too many requests", "第三方服务连接失败", "第三方服务暂时不可用",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) || strings.Contains(fault.UserMessage, n) {
			return true
		}
	}
	// UserMessage 可能是中文完整句
	switch fault.UserMessage {
	case "第三方服务连接失败，请稍后重试", "第三方服务暂时不可用，请稍后重试", "第三方服务异常，请稍后重试":
		return true
	}
	return false
}

// CodeBlockFetch 第三方返回「区块获取异常」类临时故障（波场链路抖动，下期/下 tick 可再试）。
const CodeBlockFetch = 40050

// IsBlockFetchError 区块高度/链路暂不可用导致的拒单（非注单内容错误）。
func IsBlockFetchError(err error) bool {
	if err == nil {
		return false
	}
	var api *APIError
	if errors.As(err, &api) {
		if api.Code == CodeBlockFetch {
			return true
		}
		if isBlockFetchMessage(api.Message) {
			return true
		}
	}
	return isBlockFetchMessage(err.Error())
}

func isBlockFetchMessage(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "区块获取") || strings.Contains(strings.ToLower(msg), "block fetch")
}

// IsSafeImmediateRetryError 请求很可能未发出的错误，允许同 tick 内短暂重试 PlaceBet（避免超时类二次下单）。
func IsSafeImmediateRetryError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"connection refused", "connectex", "no such host", "network is unreachable",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
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
