package guaji

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the Guaji third-party HTTP adapter (T0 skeleton).
type Client struct {
	cfg Config
	http *http.Client
}

func NewClient(cfg Config) *Client {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	transport := &http.Transport{
		Proxy:                 httpProxyFunc(),
		DialContext:           dialContextPreferHealthy,
		ForceAttemptHTTP2:     false, // CDN 边缘对 HTTP/2 偶发异常；HTTP/1.1 更稳
		MaxIdleConns:          64,
		MaxConnsPerHost:       32,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (c *Client) Config() Config { return c.cfg }

func (c *Client) Enabled() bool { return c.cfg.Enabled }

func (c *Client) doJSON(ctx context.Context, method, baseURL, path, bearer string, body any, out *envelope) error {
	env, _, err := c.doJSONRaw(ctx, method, baseURL, path, bearer, body)
	if err != nil {
		return err
	}
	if out != nil {
		*out = env
	}
	return nil
}

// doJSONRaw 与 doJSON 相同，但额外返回原始响应体（用于裸对象接口如 users/i/info）。
func (c *Client) doJSONRaw(ctx context.Context, method, baseURL, path, bearer string, body any) (envelope, []byte, error) {
	var out envelope
	if !c.cfg.Enabled {
		return out, nil, ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if err := c.cfg.Valid(); err != nil {
		return out, nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return out, nil, fmt.Errorf("guaji encode: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	u := baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return out, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.Origin != "" {
		req.Header.Set("Origin", c.cfg.Origin)
	}
	if c.cfg.Referer != "" {
		req.Header.Set("Referer", c.cfg.Referer)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return out, nil, fmt.Errorf("guaji http %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return out, nil, fmt.Errorf("guaji read body: %w", err)
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		if resp.StatusCode >= 400 {
			return out, raw, fmt.Errorf("guaji http %s %s: status %d body=%s", method, path, resp.StatusCode, truncate(string(raw), 512))
		}
		return out, raw, fmt.Errorf("guaji decode envelope: %w", err)
	}
	// Third-party may use HTTP 4xx with JSON {code,message} business errors.
	if resp.StatusCode >= 500 {
		return out, raw, fmt.Errorf("guaji http %s %s: status %d body=%s", method, path, resp.StatusCode, truncate(string(raw), 512))
	}
	return out, raw, nil
}

// isSuccessCode：成功码兼容 0（如 agents/rate）、200、201。
func isSuccessCode(code int) bool {
	return code == 0 || code == 200 || code == 201
}

func (c *Client) parseEnvelope(env envelope) error {
	code := env.Code.Int()
	// 优先 success 字段（如 /auth/login {"success":true}）。
	if env.Success != nil {
		if *env.Success {
			return nil
		}
		extra := env.extraMap()
		if code == CodeMFARequired {
			return &MFARequiredError{Code: code, LoginKey: extractStringExtra(extra, "login_key", "key"), Extra: extra}
		}
		return &APIError{Code: code, Message: env.Message, Extra: extra}
	}
	// 无 success 字段：按 code（0/200/201 成功；裸对象 code 缺省为 0 视为成功）。
	if isSuccessCode(code) {
		return nil
	}
	extra := env.extraMap()
	switch code {
	case CodeMFARequired:
		loginKey := extractStringExtra(extra, "login_key", "key")
		return &MFARequiredError{Code: code, LoginKey: loginKey, Extra: extra}
	default:
		return &APIError{Code: code, Message: env.Message, Extra: extra}
	}
}

func extractStringExtra(extra map[string]json.RawMessage, keys ...string) string {
	if extra == nil {
		return ""
	}
	for _, k := range keys {
		raw, ok := extra[k]
		if !ok || len(raw) == 0 {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
