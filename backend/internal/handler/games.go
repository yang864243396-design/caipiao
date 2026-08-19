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
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u6570\u636e\u5e93\u672a\u5c31\u7eea")
		return
	}
	code := r.PathValue("code")
	if code == "" {
		apix.Validation(w, "code \u4e0d\u80fd\u4e3a\u7a7a")
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
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u6570\u636e\u5e93\u672a\u5c31\u7eea")
		return
	}
	code := r.PathValue("code")
	if code == "" {
		apix.Validation(w, "code \u4e0d\u80fd\u4e3a\u7a7a")
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
	RequestID  string             `json:"requestId"`
	IssueNo    string             `json:"issueNo"`
	Amount     float64            `json:"amount"`
	Multiplier int                `json:"multiplier"`
	BetMode    string             `json:"betMode"`
	PlayMethod string             `json:"playMethod"`
	RunMode    string             `json:"runMode"`
	BetPayload schemes.BetPayload `json:"betPayload"`
}

func (h *Handler) GamePlaceBet(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u6570\u636e\u5e93\u672a\u5c31\u7eea")
		return
	}
	code := r.PathValue("code")
	if code == "" {
		apix.Validation(w, "code \u4e0d\u80fd\u4e3a\u7a7a")
		return
	}
	var req placeGameBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "\u8bf7\u6c42\u4f53\u987b\u4e3a JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.games.PlaceBet(r.Context(), account, code, games.PlaceBetInput{
			RequestID:  req.RequestID,
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
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u6570\u636e\u5e93\u672a\u5c31\u7eea")
	case errors.Is(err, games.ErrInvalidQuery), errors.Is(err, games.ErrInvalidBet):
		apix.Validation(w, err.Error())
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "\u4f1a\u5458\u4e0d\u5b58\u5728")
	case errors.Is(err, member.ErrInsufficientFunds):
		apix.Fail(w, http.StatusOK, apix.CodeForbidden, "\u53ef\u7528\u4f59\u989d\u4e0d\u8db3")
	case errors.Is(err, games.ErrLotteryMaintenance):
		apix.Fail(w, http.StatusForbidden, apix.CodeForbidden, "\u8be5\u5f69\u79cd\u7ef4\u62a4\u4e2d")
	case errors.Is(err, games.ErrLotteryNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "\u5f69\u79cd\u4e0d\u5b58\u5728")
	case errors.Is(err, games.ErrGuajiNoActiveAuth):
		apix.Fail(w, http.StatusOK, apix.CodeForbidden, "\u65e0\u542f\u7528\u4e2d\u7684\u6388\u6743\u8d26\u53f7\uff0c\u8bf7\u5148\u542f\u7528\u6388\u6743")
	case errors.Is(err, games.ErrGuajiTokenInvalid):
		apix.Fail(w, http.StatusOK, apix.CodeUnauthorized, "\u6388\u6743\u5df2\u5931\u6548\uff0c\u8bf7\u5728\u6388\u6743\u5217\u8868\u9875\u91cd\u65b0\u6388\u6743")
	case errors.Is(err, games.ErrGuajiInsufficient):
		apix.Fail(w, http.StatusOK, apix.CodeForbidden, "\u53ef\u7528\u4f59\u989d\u4e0d\u8db3\uff0c\u8bf7\u524d\u5f80\u7b2c\u4e09\u65b9\u5e73\u53f0\u5145\u503c")
	case errors.Is(err, games.ErrGuajiPlaceRejected):
		apix.Fail(w, http.StatusOK, apix.CodeInternal, "\u7b2c\u4e09\u65b9\u63a5\u5355\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u518d\u8bd5")
	case errors.Is(err, games.ErrGuajiAcceptanceUnknown):
		apix.Fail(w, http.StatusConflict, apix.CodeInternal, "\u7b2c\u4e09\u65b9\u63a5\u5355\u72b6\u6001\u672a\u77e5\uff0c\u8bf7\u52ff\u91cd\u590d\u4e0b\u6ce8\uff0c\u8054\u7cfb\u5ba2\u670d\u6838\u5bf9")
	case errors.Is(err, games.ErrGuajiUpstream):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u7b2c\u4e09\u65b9\u670d\u52a1\u6682\u65f6\u4e0d\u53ef\u7528\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5")
	default:
		apix.Internal(w)
	}
}
