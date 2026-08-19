package games

import (
	"context"

	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebettingdispatch"
)

// 兼容既有 games 包引用；实现见 guajibet.Placer / accountsvc.PlaceRealBet。
var (
	ErrGuajiNoActiveAuth      = guajibet.ErrNoActiveAuth
	ErrGuajiTokenInvalid      = guajibet.ErrTokenInvalid
	ErrGuajiUpstream          = guajibet.ErrUpstream
	ErrGuajiInsufficient      = guajibet.ErrInsufficient
	ErrGuajiPlaceRejected     = guajibet.ErrPlaceRejected
	ErrGuajiAcceptanceUnknown = schemebettingdispatch.ErrAPIBetAcceptanceUnknown
)

type GuajiBetRequest = guajibet.Request
type GuajiBetResult = guajibet.Result
type GuajiBetPlacer = guajibet.Placer

type FormalBetSubmitter interface {
	SubmitAPIBet(context.Context, schemebettingdispatch.APIBetCommand) (schemebettingdispatch.APIBetResult, error)
}

// SetGuajiBetPlacer 注入第三方下单网关（server 启动时调用；nil 时 real 走本地降级）。
func (s *Service) SetGuajiBetPlacer(p GuajiBetPlacer) {
	if s == nil {
		return
	}
	s.guajiBets = p
}

func (s *Service) SetFormalBetSubmitter(submitter FormalBetSubmitter) {
	if s == nil {
		return
	}
	s.formalBets = submitter
}

func (s *Service) guajiRealEnabled() bool {
	return s != nil && s.guajiBets != nil && s.guajiBets.Enabled()
}
