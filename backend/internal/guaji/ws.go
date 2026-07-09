package guaji

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

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

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	hdr := http.Header{}
	if c.cfg.Origin != "" {
		hdr.Set("Origin", c.cfg.Origin)
	}
	conn, resp, err := dialer.DialContext(ctx, u.String(), hdr)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			return fmt.Errorf("guaji ws dial: %w (http %d)", err, resp.StatusCode)
		}
		return fmt.Errorf("guaji ws dial: %w", err)
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
