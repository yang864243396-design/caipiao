package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/cloud/instances"
	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
	"caipiao/backend/internal/ws"
)

func parseInstanceRunModeFilter(r *http.Request) string {
	if simBetRaw := strings.TrimSpace(r.URL.Query().Get("simBet")); simBetRaw != "" {
		switch simBetRaw {
		case "true", "1":
			return "sim"
		case "false", "0":
			return "real"
		default:
			return "__invalid__"
		}
	}
	runMode := strings.TrimSpace(r.URL.Query().Get("runMode"))
	if runMode != "" && runMode != "real" && runMode != "sim" {
		return "__invalid__"
	}
	return runMode
}

func (h *Handler) CloudRunningSchemes(w http.ResponseWriter, r *http.Request) {
	if h.schemes != nil {
		h.withMember(w, r, func(_ *member.Service, account string) {
			runMode := parseInstanceRunModeFilter(r)
			if runMode == "__invalid__" {
				apix.Validation(w, "runMode 须为 real 或 sim；simBet 须为 true/false")
				return
			}
			idsParam := strings.TrimSpace(r.URL.Query().Get("ids"))
			if idsParam != "" {
				ids := splitCSVQuery(idsParam)
				if len(ids) == 0 {
					apix.Validation(w, "ids 不能为空")
					return
				}
				if len(ids) > 200 {
					apix.Validation(w, "ids 最多 200 个")
					return
				}
				result, err := h.schemes.ListInstancesQuery(r.Context(), account, schemes.InstanceListQuery{IDs: ids})
				if err != nil {
					h.handleSchemeErr(w, err)
					return
				}
				apix.OK(w, result)
				return
			}
			limit := parsePositiveIntQuery(r.URL.Query().Get("limit"), 0)
			cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
			search := strings.TrimSpace(r.URL.Query().Get("q"))
			if limit > 0 || cursor != "" || search != "" {
				if limit <= 0 {
					limit = 10
				}
				result, err := h.schemes.ListInstancesQuery(r.Context(), account, schemes.InstanceListQuery{
					RunMode: runMode,
					Limit:   limit,
					Cursor:  cursor,
					Search:  search,
				})
				if err != nil {
					h.handleSchemeErr(w, err)
					return
				}
				apix.OK(w, result)
				return
			}
			result, err := h.schemes.ListInstances(r.Context(), account, runMode)
			if err != nil {
				h.handleSchemeErr(w, err)
				return
			}
			apix.OK(w, result)
		})
		return
	}
	items := h.instances.List()
	apix.OK(w, map[string]interface{}{"items": items})
}

func (h *Handler) CloudCenterStats(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.GetCloudCenterStats(r.Context(), account)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) CloudLookbackGet(w http.ResponseWriter, r *http.Request) {
	if h.lookback == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	h.withMember(w, r, func(members *member.Service, account string) {
		m, err := members.GetByAccount(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		result, err := h.lookback.Get(r.Context(), m.ID)
		if err != nil {
			slog.Error("lookback get failed", "memberId", m.ID, "account", account, "err", err)
			apix.Internal(w)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) CloudLookbackPut(w http.ResponseWriter, r *http.Request) {
	if h.lookback == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	var body lookback.Settings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	body.RunModes = lookback.NormalizeRunModes(body.RunModes)
	if !body.ApplyFormal && !body.ApplySim && len(body.RunModes) > 0 {
		lookback.SyncApplyFlagsFromRunModes(&body)
	} else {
		lookback.SyncRunModesFromApplyFlags(&body)
	}
	for _, m := range body.RunModes {
		if m != lookback.RunModeReal && m != lookback.RunModeSim {
			apix.Validation(w, "runModes 仅允许 real、sim")
			return
		}
	}
	if body.Judgment != lookback.JudgmentNone &&
		body.Judgment != lookback.JudgmentIndividual &&
		body.Judgment != lookback.JudgmentOverall {
		apix.Validation(w, "judgment 须为空、individual 或 overall")
		return
	}
	h.withMember(w, r, func(members *member.Service, account string) {
		m, err := members.GetByAccount(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		result, err := h.lookback.Put(r.Context(), m.ID, body)
		if err != nil {
			if errors.Is(err, lookback.ErrInvalidSettings) {
				apix.Validation(w, err.Error())
				return
			}
			slog.Error("lookback put failed", "memberId", m.ID, "account", account, "err", err)
			apix.Internal(w)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) CloudGlobalSettingsGet(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.GetCloudGlobalSettings(r.Context(), account)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) CloudGlobalSettingsPut(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	var body schemes.CloudGlobalSettings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.PutCloudGlobalSettings(r.Context(), account, body)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) CloudInstanceStart(w http.ResponseWriter, r *http.Request) {
	h.cloudInstanceAction(w, r, func(ctx context.Context, account, id string) (interface{}, error) {
		if h.schemes != nil {
			return h.schemes.StartInstance(ctx, account, id)
		}
		return h.instances.Start(id)
	})
}

func (h *Handler) CloudInstanceStop(w http.ResponseWriter, r *http.Request) {
	h.cloudInstanceAction(w, r, func(ctx context.Context, account, id string) (interface{}, error) {
		if h.schemes != nil {
			return h.schemes.StopInstance(ctx, account, id)
		}
		return h.instances.Pause(id)
	})
}

func (h *Handler) CloudInstancePause(w http.ResponseWriter, r *http.Request) {
	h.CloudInstanceStop(w, r)
}

func (h *Handler) CloudInstanceResume(w http.ResponseWriter, r *http.Request) {
	h.cloudInstanceAction(w, r, func(ctx context.Context, account, id string) (interface{}, error) {
		if h.schemes != nil {
			return h.schemes.ResumeInstance(ctx, account, id)
		}
		return h.instances.Resume(id)
	})
}

type cloudInstanceMultiplierBody struct {
	Multiplier float64 `json:"multiplier"`
}

func (h *Handler) CloudInstanceMultiplierPut(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil && h.instances == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	id := r.PathValue("instanceId")
	if id == "" {
		apix.Validation(w, "instanceId 不能为空")
		return
	}
	var body cloudInstanceMultiplierBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	run := func(account string) {
		if h.schemes != nil {
			row, err := h.schemes.UpdateInstanceMultiplier(r.Context(), account, id, body.Multiplier)
			if err != nil {
				h.handleCloudInstanceErr(w, err)
				return
			}
			h.publishSchemeInstanceWS(account, row, "user_action")
			apix.OK(w, row)
			return
		}
		row, err := h.instances.UpdateMultiplier(id, body.Multiplier)
		if err != nil {
			h.handleCloudInstanceErr(w, err)
			return
		}
		apix.OK(w, row)
	}
	if h.schemes != nil {
		h.withMember(w, r, func(_ *member.Service, account string) { run(account) })
		return
	}
	run("")
}

type cloudInstanceSimBetBody struct {
	SimBet bool `json:"simBet"`
}

func (h *Handler) CloudInstanceSimBetPut(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	id := r.PathValue("instanceId")
	if id == "" {
		apix.Validation(w, "instanceId 不能为空")
		return
	}
	var body cloudInstanceSimBetBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		row, err := h.schemes.UpdateInstanceSimBet(r.Context(), account, id, body.SimBet)
		if err != nil {
			h.handleCloudInstanceErr(w, err)
			return
		}
		h.publishSchemeInstanceWS(account, row, "user_action")
		apix.OK(w, row)
	})
}

type cloudInstanceActionFn func(context.Context, string, string) (interface{}, error)

func (h *Handler) cloudInstanceAction(w http.ResponseWriter, r *http.Request, fn cloudInstanceActionFn) {
	id := r.PathValue("instanceId")
	if id == "" {
		apix.Validation(w, "instanceId 不能为空")
		return
	}
	run := func(account string) {
		row, err := fn(r.Context(), account, id)
		if err != nil {
			h.handleCloudInstanceErr(w, err)
			return
		}
		if inst, ok := row.(schemes.Instance); ok {
			h.publishSchemeInstanceWS(account, inst, "user_action")
		}
		apix.OK(w, row)
	}
	if h.schemes != nil {
		h.withMember(w, r, func(_ *member.Service, account string) { run(account) })
		return
	}
	run("")
}

func (h *Handler) publishSchemeInstanceWS(account string, inst schemes.Instance, reason string) {
	if h.wsHub == nil || account == "" {
		return
	}
	ws.PublishSchemeInstance(h.wsHub, account, ws.SchemeInstancePayload{
		InstanceID: inst.ID,
		RunMode:    inst.RunMode,
		SimBet:     inst.SimBet,
		Status:     inst.Status,
		Reason:     reason,
		Hint:       "refresh_running_list",
	})
}

func (h *Handler) handleCloudInstanceErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, instances.ErrNotFound), errors.Is(err, schemes.ErrDefinitionNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "实例不存在")
	case errors.Is(err, schemes.ErrStartTimeNotAfterNow), errors.Is(err, schemes.ErrEndTimeReached), errors.Is(err, schemes.ErrMinBetAmountTooLow),
		errors.Is(err, schemes.ErrSimSchemeConcurrentLimit), errors.Is(err, schemes.ErrSimSchemeDailyStartLimit):
		apix.Fail(w, http.StatusOK, apix.CodeValidation, err.Error())
	case errors.Is(err, schemes.ErrStartInsufficientFunds), errors.Is(err, guajibet.ErrInsufficient), errors.Is(err, member.ErrInsufficientFunds):
		apix.Fail(w, http.StatusOK, apix.CodeForbidden, "可用余额不足，请充值后再开启")
	case errors.Is(err, schemes.ErrMaintenanceResumeBlocked):
		apix.Fail(w, http.StatusOK, apix.CodeValidation, err.Error())
	case errors.Is(err, instances.ErrInvalidAction), errors.Is(err, schemes.ErrInvalidInstanceAction):
		apix.Fail(w, http.StatusOK, apix.CodeValidation, "当前状态不允许此操作")
	case errors.Is(err, schemes.ErrInstanceRunningSimBet):
		apix.Fail(w, http.StatusConflict, apix.CodeValidation, "方案运行中不可修改模拟投注")
	case errors.Is(err, guajibet.ErrNoActiveAuth):
		apix.Fail(w, http.StatusOK, apix.CodeForbidden, "无启用中的授权账号，请先启用授权")
	case errors.Is(err, guajibet.ErrTokenInvalid):
		apix.Fail(w, http.StatusOK, apix.CodeUnauthorized, "授权已失效，请在授权列表页重新授权")
	case errors.Is(err, guajibet.ErrUpstream):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "暂时无法校验余额，请稍后重试")
	case errors.Is(err, schemes.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
	default:
		apix.Internal(w)
	}
}

func splitCSVQuery(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parsePositiveIntQuery(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
