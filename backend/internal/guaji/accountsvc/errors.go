package accountsvc

import "errors"

var (
	ErrUnavailable       = errors.New("guaji account service unavailable")
	ErrGuajiDisabled     = errors.New("第三方对接未启用")
	ErrCredentialsKey    = errors.New("GUAJI_CREDENTIALS_KEY 未配置")
	ErrUsernameTaken     = errors.New("该第三方账号已被其他会员绑定")
	ErrAccountNotFound   = errors.New("授权账号不存在")
	ErrNoActiveAccount   = errors.New("无启用中的授权账号")
	ErrTokenInvalid      = errors.New("授权已失效，请重新授权")
	ErrReauthNeedsBind   = errors.New("自动重新授权失败，请前往绑定页重填密码")
	ErrInvalidCredentials = errors.New("第三方账号或密码错误")
	ErrInvalidCurrency    = errors.New("主币种仅支持 USDT / TRX / CNY")
	ErrGuajiUpstream      = errors.New("第三方服务暂时不可用，请稍后重试")
)
