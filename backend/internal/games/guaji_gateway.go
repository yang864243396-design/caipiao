package games

import "caipiao/backend/internal/guajibet"

// 兼容既有 games 包引用；实现见 guajibet.Placer / accountsvc.PlaceRealBet。
var (
	ErrGuajiNoActiveAuth  = guajibet.ErrNoActiveAuth
	ErrGuajiTokenInvalid  = guajibet.ErrTokenInvalid
	ErrGuajiUpstream      = guajibet.ErrUpstream
	ErrGuajiInsufficient  = guajibet.ErrInsufficient
	ErrGuajiPlaceRejected = guajibet.ErrPlaceRejected
)

type GuajiBetRequest = guajibet.Request
type GuajiBetResult = guajibet.Result
type GuajiBetPlacer = guajibet.Placer

// SetGuajiBetPlacer 注入第三方下单网关（server 启动时调用；nil 时 real 走本地降级）。
func (s *Service) SetGuajiBetPlacer(p GuajiBetPlacer) {
	if s == nil {
		return
	}
	s.guajiBets = p
}

func (s *Service) guajiRealEnabled() bool {
	return s != nil && s.guajiBets != nil && s.guajiBets.Enabled()
}
