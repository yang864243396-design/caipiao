package guaji

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// 开奖 WS 端点校验浏览器特征，缺 User-Agent 会直接 bad handshake。
// 探针与 SubscribeDraws 必须用同一套拨号参数，否则探针会误报 WS 不可达
// （曾出现 guaji-smoke 报 WS 挂了、而 drawsync 实际正常入库）。
const wsBrowserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

func newWSDialer() websocket.Dialer {
	return websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		NetDialContext:   dialContextPreferHealthy,
		Proxy:            httpProxyFunc(),
	}
}

func (c *Client) wsDialHeaders() http.Header {
	hdr := http.Header{}
	if c.cfg.Origin != "" {
		hdr.Set("Origin", c.cfg.Origin)
	}
	hdr.Set("User-Agent", wsBrowserUA)
	return hdr
}

// dialWS 拨开奖 WS，并在 3xx 时按 Location 重试一次。
//
// WebSocket 握手不跟随重定向：端点在 /ws 与 /ws/ 之间切换时（2026-07-28
// 上游把 /ws 改成 301 → /ws/），拨号只报 bad handshake，开奖入库全线中断。
// 跟随一次让两种写法都能连上。
func (c *Client) dialWS(ctx context.Context, rawURL string) (*websocket.Conn, error) {
	dialer := newWSDialer()
	conn, resp, err := dialer.DialContext(ctx, rawURL, c.wsDialHeaders())
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		return conn, nil
	}
	if resp == nil || resp.StatusCode < 300 || resp.StatusCode >= 400 {
		if resp != nil {
			return nil, fmt.Errorf("guaji ws dial: %w (http %d)", err, resp.StatusCode)
		}
		return nil, fmt.Errorf("guaji ws dial: %w", err)
	}

	next, nerr := wsRedirectTarget(rawURL, resp.Header.Get("Location"))
	if nerr != nil {
		return nil, fmt.Errorf("guaji ws dial: %w (http %d, location %q)",
			err, resp.StatusCode, resp.Header.Get("Location"))
	}
	conn2, resp2, err2 := dialer.DialContext(ctx, next, c.wsDialHeaders())
	if resp2 != nil && resp2.Body != nil {
		defer resp2.Body.Close()
	}
	if err2 != nil {
		if resp2 != nil {
			return nil, fmt.Errorf("guaji ws dial %s: %w (http %d)", next, err2, resp2.StatusCode)
		}
		return nil, fmt.Errorf("guaji ws dial %s: %w", next, err2)
	}
	return conn2, nil
}

// wsRedirectTarget 把重定向 Location 换回 ws/wss，并保留原查询串（token=Anonymous）。
func wsRedirectTarget(rawURL, location string) (string, error) {
	if strings.TrimSpace(location) == "" {
		return "", fmt.Errorf("无 Location")
	}
	base, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	loc, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	next := base.ResolveReference(loc)
	switch next.Scheme {
	case "http", "ws":
		next.Scheme = "ws"
	case "https", "wss":
		next.Scheme = "wss"
	default:
		return "", fmt.Errorf("非法 scheme %q", next.Scheme)
	}
	// 上游 301 常把 https 降级成 http；原本是 wss 就别退回明文。
	if base.Scheme == "wss" {
		next.Scheme = "wss"
	}
	if next.RawQuery == "" {
		next.RawQuery = base.RawQuery
	}
	if next.String() == rawURL {
		return "", fmt.Errorf("Location 与原地址相同")
	}
	return next.String(), nil
}

// PingAnonymousWS dials wss://…/?token=Anonymous to verify WS reachability (T0).
func (c *Client) PingAnonymousWS(ctx context.Context) error {
	if !c.cfg.Enabled {
		return ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if err := c.cfg.Valid(); err != nil {
		return err
	}
	u, err := url.Parse(c.cfg.WSBase + wsPathOrDefault(c.cfg.WSPath))
	if err != nil {
		return fmt.Errorf("guaji ws url: %w", err)
	}
	q := u.Query()
	q.Set("token", "Anonymous")
	u.RawQuery = q.Encode()

	conn, err := c.dialWS(ctx, u.String())
	if err != nil {
		return err
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	_ = conn.SetReadDeadline(deadline)
	_, _, err = conn.ReadMessage()
	if err != nil {
		// Anonymous subscription may stay quiet; dial success is enough for T0.
		return nil
	}
	return nil
}
