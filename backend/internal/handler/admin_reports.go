package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/reports"
)

func (h *Handler) AdminLotteryStatReport(w http.ResponseWriter, r *http.Request) {
	if h.reports == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	result, err := h.reports.AdminLotteryStat(r.Context(), reports.Query{
		DateFrom: r.URL.Query().Get("dateFrom"),
		DateTo:   r.URL.Query().Get("dateTo"),
	})
	if err != nil {
		h.handleReportsErr(w, err)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) AdminPnlReport(w http.ResponseWriter, r *http.Request) {
	if h.reports == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	result, err := h.reports.AdminPnlReport(r.Context(), reports.Query{
		DateFrom: r.URL.Query().Get("dateFrom"),
		DateTo:   r.URL.Query().Get("dateTo"),
	})
	if err != nil {
		h.handleReportsErr(w, err)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) AdminDailyLotteryReport(w http.ResponseWriter, r *http.Request) {
	if h.reports == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	result, err := h.reports.AdminDailyLotteryReport(r.Context(), reports.Query{
		DateFrom:    r.URL.Query().Get("dateFrom"),
		DateTo:      r.URL.Query().Get("dateTo"),
		LotteryCode: r.URL.Query().Get("lotteryCode"),
	})
	if err != nil {
		h.handleReportsErr(w, err)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) handleReportsErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, reports.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, reports.ErrInvalidQuery):
		apix.Validation(w, "日期参数无效")
	default:
		apix.Internal(w)
	}
}
