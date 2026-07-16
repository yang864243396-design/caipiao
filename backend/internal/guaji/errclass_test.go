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
