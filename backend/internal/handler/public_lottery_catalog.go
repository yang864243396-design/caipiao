package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/games"
)

func (h *Handler) PublicLotteries(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.games.PublicListLotteries(r.Context())
	if err != nil {
		if errors.Is(err, games.ErrUnavailable) {
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
			return
		}
		apix.Internal(w)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) PublicLotteryRouteStatus(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	code := r.PathValue("code")
	apix.OK(w, h.games.PublicLotteryRouteStatus(r.Context(), code))
}

func (h *Handler) MemberLotteryFilterOptions(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.games.MemberLotteryFilterOptions(r.Context())
	if err != nil {
		if errors.Is(err, games.ErrUnavailable) {
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
			return
		}
		apix.Internal(w)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) PublicLotteryPlayTree(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	code := r.PathValue("code")
	tree, err := h.games.PublicPlayTree(r.Context(), code)
	if err != nil {
		if errors.Is(err, games.ErrLotteryMaintenance) {
			apix.Fail(w, http.StatusForbidden, apix.CodeForbidden, "该彩种维护中")
			return
		}
		if errors.Is(err, games.ErrLotteryNotFound) {
			apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "彩种不存在或未上架")
			return
		}
		if errors.Is(err, games.ErrUnavailable) {
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
			return
		}
		apix.Internal(w)
		return
	}
	apix.OK(w, tree)
}
