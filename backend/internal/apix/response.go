package apix

import (
	"encoding/json"
	"net/http"
)

const (
	CodeOK           = 0
	CodeUnauthorized = 40100
	CodeForbidden    = 40300
	CodeNotFound     = 40400
	CodeValidation   = 42200
	CodeInternal     = 50000
)

type Envelope struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

func OK(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, Envelope{Code: CodeOK, Message: "ok", Data: data})
}

func Fail(w http.ResponseWriter, httpStatus, code int, message string) {
	WriteJSON(w, httpStatus, Envelope{Code: code, Message: message, Data: nil})
}

func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "未登录或 Token 已失效"
	}
	Fail(w, http.StatusUnauthorized, CodeUnauthorized, message)
}

func Validation(w http.ResponseWriter, message string) {
	if message == "" {
		message = "参数校验失败"
	}
	Fail(w, http.StatusOK, CodeValidation, message)
}

func Internal(w http.ResponseWriter) {
	Fail(w, http.StatusInternalServerError, CodeInternal, "服务器内部错误")
}
