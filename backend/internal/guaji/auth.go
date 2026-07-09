package guaji

import (
	"context"
	"fmt"
)

// Login performs POST /auth/login with username + password.
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	req := LoginRequest{
		Username: username,
		Password: password,
		IsAI:     c.cfg.IsAI,
	}
	var env envelope
	if err := c.doJSON(ctx, "POST", c.cfg.AuthBase, "/auth/login", "", req, &env); err != nil {
		return nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, err
	}
	var res LoginResult
	if err := env.dataInto(&res); err != nil {
		return nil, fmt.Errorf("guaji login decode: %w", err)
	}
	if res.Token == "" {
		return nil, fmt.Errorf("guaji login: empty token")
	}
	return &res, nil
}

// LoginWithMFA completes POST /auth/login after MFA challenge (T1).
func (c *Client) LoginWithMFA(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	req.IsAI = c.cfg.IsAI
	var env envelope
	if err := c.doJSON(ctx, "POST", c.cfg.AuthBase, "/auth/login", "", req, &env); err != nil {
		return nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, err
	}
	var res LoginResult
	if err := env.dataInto(&res); err != nil {
		return nil, fmt.Errorf("guaji login mfa decode: %w", err)
	}
	if res.Token == "" {
		return nil, fmt.Errorf("guaji login mfa: empty token")
	}
	return &res, nil
}

// RefreshToken calls POST /auth/refresh/token (T1+).
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
	body := map[string]any{
		"refresh_token": refreshToken,
		"is_ai":         c.cfg.IsAI,
	}
	var env envelope
	if err := c.doJSON(ctx, "POST", c.cfg.AuthBase, "/auth/refresh/token", "", body, &env); err != nil {
		return nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, err
	}
	var res LoginResult
	if err := env.dataInto(&res); err != nil {
		return nil, fmt.Errorf("guaji refresh decode: %w", err)
	}
	if res.Token == "" {
		return nil, fmt.Errorf("guaji refresh: empty token")
	}
	return &res, nil
}
