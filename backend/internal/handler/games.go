package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/games"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
)

func (h *Handler) GameDetail(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	code := r.PathValue("code")
	if code == "" {
		apix.Validation(w, "code 不能为空")
		return
	}
	q := games.DetailQuery{
		LotteryCode: code,
		SchemeName:  r.URL.Query().Get("schemeName"),
		PlayMethod:  r.URL.Query().Get("playMethod"),
		SnapshotID:  r.URL.Query().Get("snapshotId"),
		Board:       r.URL.Query().Get("board"),
		PlayTypeID:  r.URL.Query().Get("playTypeId"),
		SubPlayID:   r.URL.Query().Get("subPlayId"),
	}
	result, err := h.games.Detail(r.Context(), q)
	if err != nil {
		h.handleGamesErr(w, err)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) GameDraws(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	code := r.PathValue("code")
	if code == "" {
		apix.Validation(w, "code 不能为空")
		return
	}
	result, err := h.games.Draws(r.Context(), games.DrawsQuery{
		LotteryCode: code,
		Cursor:      r.URL.Query().Get("cursor"),
		Limit:       queryInt(r, "limit", 20),
	})
	if err != nil {
		h.handleGamesErr(w, err)
		return
	}
	apix.OK(w, result)
}

type placeGameBetRequest struct {
	IssueNo    string               `json:"issueNo"`
	Amount     float64              `json:"amount"`
	Multiplier int                  `json:"multiplier"`
	BetMode    string               `json:"betMode"`
	PlayMethod string               `json:"playMethod"`
	RunMode    string               `json:"runMode"`
	BetPayload schemes.BetPayload   `json:"betPayload"`
}

func (h *Handler) GamePlaceBet(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	code := r.PathValue("code")
	if code == "" {
		apix.Validation(w, "code 不能为空")
		return
	}
	var req placeGameBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.games.PlaceBet(r.Context(), account, code, games.PlaceBetInput{
			IssueNo:    req.IssueNo,
			Amount:     req.Amount,
			Multiplier: req.Multiplier,
			BetMode:    req.BetMode,
			PlayMethod: req.PlayMethod,
			RunMode:    req.RunMode,
			BetPayload: req.BetPayload,
		})
		if err != nil {
			h.handleGamesErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) handleGamesErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, games.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, games.ErrInvalidQuery), errors.Is(err, games.ErrInvalidBet):
		apix.Validation(w, err.Error())
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
	case errors.Is(err, member.ErrInsufficientFunds):
		apix.Fail(w, http.StatusOK, apix.CodeForbidden, "可用余额不足")
	case errors.Is(err, games.ErrLotteryMaintenance):
		apix.Fail(w, http.StatusForbidden, apix.CodeForbidden, "该彩种维护中")
	case errors.Is(err, games.ErrLotteryNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "彩种不存在")
	case errors.Is(err, games.ErrGuajiNoActiveAuth):
		apix.Fail(w, http.StatusOK, apix.CodeForbidden, "无启用中的授权账号，请先启用授权")
	case errors.Is(err, games.ErrGuajiTokenInvalid):
		apix.Fail(w, http.StatusOK, apix.CodeUnauthorized, "授权已失效，请在授权列表页重新授权")
	case errors.Is(err, games.ErrGuajiInsufficient):
		apix.Fail(w, http.StatusOK, apix.CodeForbidden, "可用余额不足，请前往第三方平台充值")
	case errors.Is(err, games.ErrGuajiPlaceRejected):
		apix.Fail(w, http.StatusOK, apix.CodeInternal, "第三方接单失败，请稍后再试")
	case errors.Is(err, games.ErrGuajiUpstream):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "第三方服务暂时不可用，请稍后重试")
	default:
		apix.Internal(w)
	}
}
