package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/member"
)

func (h *Handler) AdminListMembers(w http.ResponseWriter, r *http.Request) {
	if h.members == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	result, err := h.members.AdminListMembers(r.Context(), member.AdminMemberListQuery{
		Keyword:     r.URL.Query().Get("keyword"),
		SearchField: r.URL.Query().Get("searchField"),
		Page:        queryInt(r, "page", 1),
		PageSize:    queryInt(r, "pageSize", 10),
	})
	if err != nil {
		h.handleMemberErr(w, err)
		return
	}
	h.enrichAdminMemberListGuajiBalances(r.Context(), &result)
	apix.OK(w, result)
}

func (h *Handler) enrichAdminMemberListGuajiBalances(ctx context.Context, result *member.AdminMemberListResult) {
	if result == nil || len(result.Items) == 0 || h.guajiAccounts == nil {
		return
	}
	ids := make([]int64, 0, len(result.Items))
	for _, item := range result.Items {
		id, err := member.ParseMemberID(item.ID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}
	balances := h.guajiAccounts.AdminMultiCurrencyBalances(ctx, ids)
	for i := range result.Items {
		id, err := member.ParseMemberID(result.Items[i].ID)
		if err != nil {
			continue
		}
		b := balances[id]
		result.Items[i].GuajiBalances = member.AdminGuajiBalances{
			USDT: b.USDT,
			TRX:  b.TRX,
			CNY:  b.CNY,
		}
	}
}

func (h *Handler) AdminGetMember(w http.ResponseWriter, r *http.Request) {
	if h.members == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	memberID, err := member.ParseMemberID(r.PathValue("memberId"))
	if err != nil {
		if errors.Is(err, member.ErrNotFound) {
			apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
			return
		}
		apix.Validation(w, err.Error())
		return
	}
	item, err := h.members.AdminGetMember(r.Context(), memberID)
	if err != nil {
		h.handleMemberErr(w, err)
		return
	}
	h.enrichAdminMemberGuajiBalances(r.Context(), &item)
	apix.OK(w, item)
}

func (h *Handler) enrichAdminMemberGuajiBalances(ctx context.Context, item *member.AdminMemberRow) {
	if item == nil || h.guajiAccounts == nil {
		return
	}
	id, err := member.ParseMemberID(item.ID)
	if err != nil {
		return
	}
	balances := h.guajiAccounts.SyncGuajiBalancesForMemberID(ctx, id)
	item.GuajiBalances = member.AdminGuajiBalances{
		USDT: balances.USDT,
		TRX:  balances.TRX,
		CNY:  balances.CNY,
	}
}

func (h *Handler) AdminMemberFundRecords(w http.ResponseWriter, r *http.Request) {
	if h.members == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	memberID, err := member.ParseMemberID(r.PathValue("memberId"))
	if err != nil {
		if errors.Is(err, member.ErrNotFound) {
			apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
			return
		}
		apix.Validation(w, err.Error())
		return
	}
	result, err := h.members.AdminFundRecordsForMemberID(r.Context(), memberID, member.AdminFundRecordsQuery{
		DateFrom: r.URL.Query().Get("dateFrom"),
		DateTo:   r.URL.Query().Get("dateTo"),
		FlowType: r.URL.Query().Get("flowType"),
		Currency: r.URL.Query().Get("currency"),
		Page:     queryInt(r, "page", 1),
		PageSize: queryInt(r, "pageSize", 10),
	})
	if err != nil {
		h.handleMemberErr(w, err)
		return
	}
	apix.OK(w, result)
}

type adminMemberOpRequest struct {
	Action string `json:"action"`
}

func (h *Handler) AdminMemberOp(w http.ResponseWriter, r *http.Request) {
	if h.members == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	memberID, err := member.ParseMemberID(r.PathValue("memberId"))
	if err != nil {
		if errors.Is(err, member.ErrNotFound) {
			apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
			return
		}
		apix.Validation(w, err.Error())
		return
	}
	var req adminMemberOpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	result, err := h.members.AdminApplyOp(r.Context(), memberID, member.AdminMemberOpInput{
		Action: req.Action,
	})
	if err != nil {
		switch {
		case errors.Is(err, member.ErrInvalidOp):
			apix.Validation(w, err.Error())
		default:
			h.handleMemberErr(w, err)
		}
		return
	}
	label := map[string]string{
		"reset_login_password": "重置登录密码",
		"toggle_freeze":        "冻结/解冻",
	}[result.Action]
	if label == "" {
		label = result.Action
	}
	h.writeAudit(r, fmt.Sprintf("会员 %d · %s", memberID, label))
	apix.OK(w, result)
}
