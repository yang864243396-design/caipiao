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
		"toggle_freeze":        "禁用/启用",
	}[result.Action]
	if label == "" {
		label = result.Action
	}
	h.writeAudit(r, fmt.Sprintf("会员 %d · %s", memberID, label))
	apix.OK(w, result)
}

type adminMemberCreateRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

func (h *Handler) AdminCreateMember(w http.ResponseWriter, r *http.Request) {
	if h.members == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	var req adminMemberCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := h.members.AdminCreateMember(r.Context(), member.AdminMemberCreateInput{
		Account:  req.Account,
		Password: req.Password,
		Status:   req.Status,
	})
	if err != nil {
		h.handleAdminMemberWriteErr(w, err)
		return
	}
	h.enrichAdminMemberGuajiBalances(r.Context(), &item)
	h.writeAudit(r, fmt.Sprintf("新增会员 %s（%s）", item.Account, item.ID))
	apix.OK(w, item)
}

type adminMemberUpdateRequest struct {
	Password string `json:"password"`
	Status   string `json:"status"`
}

func (h *Handler) AdminUpdateMember(w http.ResponseWriter, r *http.Request) {
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
	var req adminMemberUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := h.members.AdminUpdateMember(r.Context(), memberID, member.AdminMemberUpdateInput{
		Password: req.Password,
		Status:   req.Status,
	})
	if err != nil {
		h.handleAdminMemberWriteErr(w, err)
		return
	}
	h.enrichAdminMemberGuajiBalances(r.Context(), &item)
	h.writeAudit(r, fmt.Sprintf("编辑会员 %d", memberID))
	apix.OK(w, item)
}

func (h *Handler) AdminClearMemberGuajiAuth(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
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
	paused, cleared, err := h.guajiAccounts.AdminClearAllAuth(r.Context(), memberID)
	if err != nil {
		if errors.Is(err, member.ErrNotFound) {
			apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
			return
		}
		apix.Internal(w)
		return
	}
	h.writeAudit(r, fmt.Sprintf("会员 %d · 清空授权（暂停方案 %d，清除授权 %d）", memberID, paused, cleared))
	apix.OK(w, map[string]any{
		"pausedSchemes": paused,
		"clearedAuths":  cleared,
		"message":       fmt.Sprintf("已停止 %d 个方案并清空 %d 条授权", paused, cleared),
	})
}

func (h *Handler) handleAdminMemberWriteErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
	case errors.Is(err, member.ErrDuplicateAccount):
		apix.Validation(w, "会员账号已存在")
	case errors.Is(err, member.ErrPasswordTooShort):
		apix.Validation(w, "密码至少 6 位")
	case errors.Is(err, member.ErrInvalidInput):
		apix.Validation(w, err.Error())
	case errors.Is(err, member.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	default:
		apix.Internal(w)
	}
}
