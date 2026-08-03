package guaji

import (
	"errors"
	"testing"
)

func TestClassifyUpstreamError_tokenHTTP(t *testing.T) {
	err := errors.New(`guaji http GET /api/users/i/info: status 401 body={"message":"unauthorized"}`)
	f := ClassifyUpstreamError(err)
	if !f.IsTokenInvalid || f.UserMessage != "授权已失效，请重新授权" {
		t.Fatalf("got %+v", f)
	}
}

func TestClassifyUpstreamError_transient502(t *testing.T) {
	err := errors.New(`guaji http GET /api/users/i/info: status 502 body={"title":"Error 502"}`)
	f := ClassifyUpstreamError(err)
	if f.IsTokenInvalid {
		t.Fatalf("502 should be transient: %+v", f)
	}
	if f.UserMessage != "第三方服务暂时不可用，请稍后重试" {
		t.Fatalf("message=%q", f.UserMessage)
	}
}

func TestClassifyUpstreamError_apiCode(t *testing.T) {
	err := &APIError{Code: CodeTokenInvalidAlt, Message: "token expired"}
	f := ClassifyUpstreamError(err)
	if !f.IsTokenInvalid {
		t.Fatalf("api token code should be invalid: %+v", f)
	}
}

func TestClassifyUpstreamError_apiCode40010(t *testing.T) {
	err := &APIError{Code: CodeTokenInvalidBiz, Message: "无效的令牌, 请重新登录."}
	f := ClassifyUpstreamError(err)
	if !f.IsTokenInvalid || f.UserMessage != "授权已失效，请重新授权" {
		t.Fatalf("40010 should be token invalid: %+v", f)
	}
}

func TestClassifyUpstreamError_friendlyPassthrough(t *testing.T) {
	err := errors.New("重新授权失败")
	f := ClassifyUpstreamError(err)
	if !f.IsTokenInvalid || f.UserMessage != "重新授权失败" {
		t.Fatalf("got %+v", f)
	}
}

func TestClassifyUpstreamError_businessChineseNotTokenInvalid(t *testing.T) {
	cases := []string{
		"第三方服务暂时不可用，请稍后重试",
		"第三方接单失败: 投注注数为0（选号无效或不足）",
		"投注注数为0（选号无效或不足）",
	}
	for _, msg := range cases {
		f := ClassifyUpstreamError(errors.New(msg))
		if f.IsTokenInvalid {
			t.Fatalf("%q should not be token invalid: %+v", msg, f)
		}
		if f.UserMessage != msg {
			t.Fatalf("UserMessage=%q want %q", f.UserMessage, msg)
		}
	}
}

func TestIsRetryableTransportError_errUpstreamMessage(t *testing.T) {
	err := errors.New("第三方服务暂时不可用，请稍后重试")
	if !IsRetryableTransportError(err) {
		t.Fatal("ErrUpstream message must be retryable (was misclassified as token invalid)")
	}
}

func TestIsRetryableTransportError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("guaji http POST /api/web_bets/lott: context deadline exceeded"), true},
		{errors.New("guaji http POST /x: connection refused"), true},
		{errors.New(`guaji http POST /x: status 502 body={}`), true},
		{errors.New(`guaji http POST /x: status 429 body={"detail":"Too Many Requests"}`), true},
		{&APIError{Code: 40055, Message: "封盘时间,下注失败"}, false},
		{&APIError{Code: CodeTokenInvalid, Message: "无效的令牌"}, false},
		{&APIError{Code: 40000, Message: "余额不足"}, false},
		{&APIError{Code: CodeBlockFetch, Message: "区块获取异常,下注失败"}, true},
		{errors.New("guaji api code=40050: 区块获取异常,下注失败"), true},
	}
	for _, c := range cases {
		if got := IsRetryableTransportError(c.err); got != c.want {
			t.Fatalf("%v => %v want %v", c.err, got, c.want)
		}
	}
}

func TestIsBlockFetchError(t *testing.T) {
	if !IsBlockFetchError(&APIError{Code: CodeBlockFetch, Message: "区块获取异常,下注失败"}) {
		t.Fatal("40050 should be block fetch")
	}
	if IsBlockFetchError(&APIError{Code: 40000, Message: "余额不足"}) {
		t.Fatal("balance error must not be block fetch")
	}
}

func TestIsSafeImmediateRetryError(t *testing.T) {
	if !IsSafeImmediateRetryError(errors.New("dial tcp: connection refused")) {
		t.Fatal("refused should be safe retry")
	}
	if IsSafeImmediateRetryError(errors.New("context deadline exceeded")) {
		t.Fatal("timeout must not immediate-retry place bet")
	}
}

func TestIsPeriodClosedError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&APIError{Code: 400, Message: "当前期已封盘"}, true},
		{&APIError{Code: 400, Message: "betting closed"}, true},
		{&APIError{Code: 400, Message: "余额不足"}, false},
		{errors.New("guaji api code=400: 已过投注截止时间"), true},
	}
	for _, c := range cases {
		if got := IsPeriodClosedError(c.err); got != c.want {
			t.Fatalf("%v => %v want %v", c.err, got, c.want)
		}
	}
}
