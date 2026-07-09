package guaji

import "context"

// Probe runs WS + optional test-account login for health / smoke (T0).
func (c *Client) Probe(ctx context.Context) ProbeResult {
	res := ProbeResult{
		Enabled:  c.cfg.Enabled,
		HTTPBase: c.cfg.HTTPBase,
		AuthBase: c.cfg.AuthBase,
		WSBase:   c.cfg.WSBase,
	}
	if !c.cfg.Enabled {
		return res
	}
	if err := c.cfg.Valid(); err != nil {
		res.WSError = err.Error()
		res.HTTPError = err.Error()
		return res
	}

	if err := c.PingAuthEndpoint(ctx); err != nil {
		res.HTTPError = err.Error()
	} else {
		res.HTTPReachable = true
	}

	if err := c.PingAnonymousWS(ctx); err != nil {
		res.WSError = err.Error()
	} else {
		res.WSReachable = true
	}

	user := c.cfg.TestUsername
	pass := c.cfg.TestPassword
	if user == "" || pass == "" {
		return res
	}
	res.TestUsername = user

	login, err := c.Login(ctx, user, pass)
	if err != nil {
		var mfa *MFARequiredError
		if ok := asMFA(err, &mfa); ok {
			res.MFARequired = true
			res.LoginError = "需要二次认证（MFA）；T1 绑号流程将自动处理"
			return res
		}
		res.LoginError = err.Error()
		return res
	}

	res.LoginOK = true
	bal, err := c.BalanceCNY(ctx, login.Token)
	if err != nil {
		res.LoginError = "登录成功但拉取余额失败: " + err.Error()
		res.LoginOK = false
		return res
	}
	res.BalanceCNY = &bal
	return res
}

func asMFA(err error, target **MFARequiredError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*MFARequiredError); ok {
		*target = e
		return true
	}
	return false
}
