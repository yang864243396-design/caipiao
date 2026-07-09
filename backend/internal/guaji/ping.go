package guaji

import (
	"context"
	"fmt"
)

// PingAuthEndpoint verifies POST /auth/login is reachable (expects business error without creds).
func (c *Client) PingAuthEndpoint(ctx context.Context) error {
	if !c.cfg.Enabled {
		return ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if err := c.cfg.Valid(); err != nil {
		return err
	}
	_, err := c.Login(ctx, "probe0", "probe0")
	if err == nil {
		return nil
	}
	switch err.(type) {
	case *APIError, *MFARequiredError:
		return nil
	default:
		if _, ok := err.(*MisconfiguredError); ok {
			return err
		}
		// Network / decode errors bubble up.
		return fmt.Errorf("guaji auth ping: %w", err)
	}
}
