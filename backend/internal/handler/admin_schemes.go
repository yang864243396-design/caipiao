package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/schemes"
)

func (h *Handler) AdminSchemeMonitorList(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "user"
	}
	simBetFilter := strings.TrimSpace(r.URL.Query().Get("simBet"))
	if simBetFilter == "" {
		simBetFilter = strings.TrimSpace(r.URL.Query().Get("runMode"))
	}
	query := schemes.AdminMonitorQuery{
		Scope:       scope,
		Keyword:     r.URL.Query().Get("keyword"),
		SearchField: r.URL.Query().Get("searchField"),
		Kind:        r.URL.Query().Get("kind"),
		Status:      r.URL.Query().Get("status"),
		SimBet:      simBetFilter,
		LotteryCode: r.URL.Query().Get("lotteryCode"),
		Limit:       queryInt(r, "limit", 200),
	}
	result, err := h.schemes.AdminMonitorList(r.Context(), query)
	if err != nil {
		h.handleSchemeAdminErr(w, err)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) AdminSchemeBetHistory(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	instanceID := r.PathValue("instanceId")
	days := queryInt(r, "days", 30)
	result, err := h.schemes.AdminBetHistory(r.Context(), instanceID, days)
	if err != nil {
		h.handleSchemeAdminErr(w, err)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) AdminSchemeForceStop(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	instanceID := r.PathValue("instanceId")
	inst, err := h.schemes.AdminForceStop(r.Context(), instanceID)
	if err != nil {
		h.handleSchemeAdminErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("强停方案实例 %s", instanceID))
	h.publishSchemeMonitorWS(instanceID, string(inst.Status), "force_stop")
	apix.OK(w, inst)
}

func (h *Handler) AdminSchemeReleaseStop(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	instanceID := r.PathValue("instanceId")
	inst, err := h.schemes.AdminReleaseStop(r.Context(), instanceID)
	if err != nil {
		h.handleSchemeAdminErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("解封方案实例 %s → 已暂停", instanceID))
	h.publishSchemeMonitorWS(instanceID, string(inst.Status), "release_stop")
	apix.OK(w, inst)
}

func (h *Handler) AdminPatchShareSnapshot(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	snapshotID := r.PathValue("snapshotId")
	var req createShareSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	snap, err := h.schemes.AdminUpdateShareSnapshot(r.Context(), snapshotID, adminShareSnapshotInputFromRequest(req))
	if err != nil {
		h.handleSchemeAdminErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("更新分享池快照 %s", snapshotID))
	apix.OK(w, snap)
}

func (h *Handler) AdminDeleteShareSnapshot(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	snapshotID := r.PathValue("snapshotId")
	if err := h.schemes.AdminDeleteShareSnapshot(r.Context(), snapshotID); err != nil {
		h.handleSchemeAdminErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("删除分享池快照 %s", snapshotID))
	apix.OK(w, map[string]string{"id": snapshotID})
}

func (h *Handler) AdminCreateShareSnapshot(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	var req createShareSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	snap, err := h.schemes.AdminCreateShareSnapshot(r.Context(), adminShareSnapshotInputFromRequest(req))
	if err != nil {
		h.handleSchemeAdminErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("新建分享池快照 %s（%s）", snap.ID, snap.SchemeName))
	apix.OK(w, snap)
}

type createShareSnapshotRequest struct {
	SchemeName    string                 `json:"schemeName"`
	LotteryCode   string                 `json:"lotteryCode"`
	RunTypeID     string                 `json:"runTypeId"`
	PlayTypeID    string                 `json:"playTypeId"`
	SubPlayID     string                 `json:"subPlayId"`
	RunMode       string                 `json:"runMode"`
	SchemeFunds   string                 `json:"schemeFunds"`
	StartTime     string                 `json:"startTime"`
	EndTime       string                 `json:"endTime"`
	SchemeGroups  []string               `json:"schemeGroups"`
	StopLoss      string                 `json:"stopLoss"`
	TakeProfit    string                 `json:"takeProfit"`
	BetUnit       string                 `json:"betUnit"`
	BetMode       string                 `json:"betMode"`
	PlayTemplate  string                 `json:"playTemplate"`
	TypeID        string                 `json:"typeId"`
	SubID         string                 `json:"subId"`
	MultCoeff     string                 `json:"multCoeff"`
	BetMultiplier map[string]interface{} `json:"betMultiplier"`
}

func adminShareSnapshotInputFromRequest(req createShareSnapshotRequest) schemes.AdminCreateShareSnapshotInput {
	return schemes.AdminCreateShareSnapshotInput{
		SchemeName:  req.SchemeName,
		LotteryCode: req.LotteryCode,
		RunTypeID:   req.RunTypeID,
		PlayTypeID:  req.PlayTypeID,
		SubPlayID:   req.SubPlayID,
		Patch: schemes.AddToCloudConfigPatch{
			RunMode:      req.RunMode,
			SchemeFunds:  req.SchemeFunds,
			StartTime:    req.StartTime,
			EndTime:      req.EndTime,
			SchemeGroups: req.SchemeGroups,
			StopLoss:     req.StopLoss,
			TakeProfit:   req.TakeProfit,
			BetUnit:      req.BetUnit,
			BetMode:      req.BetMode,
			PlayTemplate: req.PlayTemplate,
			TypeID:       req.TypeID,
			SubID:        req.SubID,
		},
		Extra: schemes.AdminShareConfigExtra{
			MultCoeff:     req.MultCoeff,
			BetMultiplier: req.BetMultiplier,
		},
	}
}

func (h *Handler) handleSchemeAdminErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, schemes.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, schemes.ErrInstanceNotFound), errors.Is(err, schemes.ErrSnapshotNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "资源不存在")
	case errors.Is(err, schemes.ErrInvalidAdminAction):
		apix.Validation(w, "当前状态不允许此操作")
	case errors.Is(err, schemes.ErrSnapshotKindImmutable):
		apix.Validation(w, "分享池快照类型不可变更")
	case errors.Is(err, schemes.ErrInvalidCreateRequest):
		apix.Validation(w, err.Error())
	default:
		slog.Error("admin scheme handler error", "err", err)
		apix.Internal(w)
	}
}
