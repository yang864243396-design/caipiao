package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/dashboard"
)

func (h *Handler) AdminDashboardKpi(w http.ResponseWriter, r *http.Request) {
	if h.dashboard == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	kpi, err := h.dashboard.AdminKpi(r.Context())
	if err != nil {
		if errors.Is(err, dashboard.ErrUnavailable) {
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
			return
		}
		apix.Internal(w)
		return
	}
	apix.OK(w, kpi)
}
