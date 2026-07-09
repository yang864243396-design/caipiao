package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
)

func (h *Handler) CreateScheme(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	var req createSchemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.CreateDefinition(r.Context(), account, schemes.CreateDefinitionInput{
			Kind:        req.Kind,
			SchemeName:  req.SchemeName,
			LotteryCode: req.LotteryCode,
			RunTypeID:   req.RunTypeID,
			PlayTypeID:  req.PlayTypeID,
			SubPlayID:   req.SubPlayID,
		})
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

type createSchemeRequest struct {
	Kind        string `json:"kind"`
	SchemeName  string `json:"schemeName"`
	LotteryCode string `json:"lotteryCode"`
	RunTypeID   string `json:"runTypeId"`
	PlayTypeID  string `json:"playTypeId"`
	SubPlayID   string `json:"subPlayId"`
}

func (h *Handler) ListSchemes(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.ListDefinitions(r.Context(), account, r.URL.Query().Get("kind"))
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) CheckSchemeName(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		apix.Validation(w, "name 不能为空")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.CheckSchemeName(r.Context(), account, name)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) GetScheme(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := r.PathValue("definitionId")
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.GetDefinition(r.Context(), account, definitionID)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) PutBetMultiplier(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := r.PathValue("definitionId")
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	var payload json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.PutBetMultiplier(r.Context(), account, definitionID, payload)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

type putRoundsRequest struct {
	Rounds json.RawMessage `json:"rounds"`
}

func (h *Handler) PutRounds(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := r.PathValue("definitionId")
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	var req putRoundsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.PutRounds(r.Context(), account, definitionID, req.Rounds)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) ShareCatalog(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	result, err := h.schemes.ShareCatalog(r.Context(), schemes.ShareCatalogQuery{
		Keyword: r.URL.Query().Get("keyword"),
		Cursor:  r.URL.Query().Get("cursor"),
		Limit:   queryInt(r, "limit", 50),
	})
	if err != nil {
		h.handleSchemeErr(w, err)
		return
	}
	apix.OK(w, result)
}

type shareAddToCloudRequest struct {
	BetMultiplier map[string]interface{} `json:"betMultiplier"`
}

func (h *Handler) ShareAddToCloud(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	snapshotID := r.PathValue("snapshotId")
	if snapshotID == "" {
		apix.Validation(w, "snapshotId 不能为空")
		return
	}
	var req shareAddToCloudRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apix.Validation(w, "请求体须为 JSON")
			return
		}
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.ShareAddToCloud(r.Context(), account, snapshotID, schemes.ShareAddToCloudInput{
			BetMultiplier: req.BetMultiplier,
		})
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

type shareFollowBetRequest struct {
	LotteryCode  string `json:"lotteryCode"`
	PlayMethod   string `json:"playMethod"`
	PlayTemplate string `json:"playTemplate"`
	TypeID       string `json:"typeId"`
	SubID        string `json:"subId"`
}

func (h *Handler) ShareFollowBet(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	snapshotID := r.PathValue("snapshotId")
	if snapshotID == "" {
		apix.Validation(w, "snapshotId 不能为空")
		return
	}
	var req shareFollowBetRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apix.Validation(w, "请求体须为 JSON")
			return
		}
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.ShareFollowBet(r.Context(), account, snapshotID, schemes.ShareFollowBetInput{
			LotteryCode:  req.LotteryCode,
			PlayMethod:   req.PlayMethod,
			PlayTemplate: req.PlayTemplate,
			TypeID:       req.TypeID,
			SubID:        req.SubID,
		})
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

type contraryBetRequest struct {
	LotteryCode        string `json:"lotteryCode"`
	PlanInverseNumbers string `json:"planInverseNumbers"`
	PlayMethod         string `json:"playMethod"`
	PlayTemplate       string `json:"playTemplate"`
	TypeID             string `json:"typeId"`
	SubID              string `json:"subId"`
}

func (h *Handler) ContraryBet(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	var req contraryBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.ContraryBet(r.Context(), account, schemes.ContraryBetInput{
			LotteryCode:        req.LotteryCode,
			PlanInverseNumbers: req.PlanInverseNumbers,
			PlayMethod:         req.PlayMethod,
			PlayTemplate:       req.PlayTemplate,
			TypeID:             req.TypeID,
			SubID:              req.SubID,
		})
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) AddDefinitionToCloud(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := r.PathValue("definitionId")
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	var req addToCloudRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.AddDefinitionToCloud(r.Context(), account, definitionID, req.ShareStatus, schemes.AddToCloudConfigPatch{
			RunMode:      runModeFromAddCloudRequest(req),
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
		})
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) ForkDefinitionToCloud(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := r.PathValue("definitionId")
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	var req addToCloudRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.ForkDefinitionToCloud(r.Context(), account, definitionID, schemes.AddToCloudConfigPatch{
			RunMode:      runModeFromAddCloudRequest(req),
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
		})
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

type addToCloudRequest struct {
	ShareStatus  string   `json:"shareStatus"`
	RunMode      string   `json:"runMode"`
	SimBet       *bool    `json:"simBet"`
	SchemeFunds  string   `json:"schemeFunds"`
	StartTime    string   `json:"startTime"`
	EndTime      string   `json:"endTime"`
	SchemeGroups []string `json:"schemeGroups"`
	StopLoss     string   `json:"stopLoss"`
	TakeProfit   string   `json:"takeProfit"`
	BetUnit      string   `json:"betUnit"`
	BetMode      string   `json:"betMode"`
	PlayTemplate string   `json:"playTemplate"`
	TypeID       string   `json:"typeId"`
	SubID        string   `json:"subId"`
}

func runModeFromAddCloudRequest(req addToCloudRequest) string {
	if req.SimBet != nil {
		if *req.SimBet {
			return "sim"
		}
		return "real"
	}
	return req.RunMode
}

func (h *Handler) PatchScheme(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := r.PathValue("definitionId")
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	patch, err := schemes.ParseUpdatePatch(raw)
	if err != nil {
		if errors.Is(err, schemes.ErrInvalidUpdatePatch) {
			apix.Validation(w, err.Error())
			return
		}
		apix.Validation(w, "更新参数无效")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.schemes.UpdateDefinition(r.Context(), account, definitionID, patch)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) DeleteScheme(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := r.PathValue("definitionId")
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		if err := h.schemes.DeleteDefinition(r.Context(), account, definitionID); err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, struct{}{})
	})
}

type schemeFavoriteRequest struct {
	SnapshotID string `json:"snapshotId"`
}

func (h *Handler) SchemeFavoritesList(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		items, err := h.schemes.ListFavorites(r.Context(), account)
		if err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, map[string]any{"items": items})
	})
}

func (h *Handler) SchemeFavoriteAdd(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	var req schemeFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		if err := h.schemes.AddFavorite(r.Context(), account, req.SnapshotID); err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, map[string]bool{"ok": true})
	})
}

func (h *Handler) SchemeFavoriteDelete(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	snapshotID := r.PathValue("snapshotId")
	if snapshotID == "" {
		apix.Validation(w, "snapshotId 不能为空")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		if err := h.schemes.RemoveFavorite(r.Context(), account, snapshotID); err != nil {
			h.handleSchemeErr(w, err)
			return
		}
		apix.OK(w, map[string]bool{"ok": true})
	})
}

func (h *Handler) handleSchemeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, schemes.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, schemes.ErrFavoriteRequired):
		apix.Fail(w, http.StatusOK, apix.CodeValidation, "请先在跟单大厅收藏该方案")
	case errors.Is(err, schemes.ErrSnapshotNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "分享方案不存在")
	case errors.Is(err, schemes.ErrDefinitionNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "方案不存在")
	case errors.Is(err, schemes.ErrAlreadyHasInstance):
		apix.Fail(w, http.StatusConflict, 40901, "方案已有云端实例，请使用复制上云")
	case errors.Is(err, schemes.ErrNoInstanceForFork):
		apix.Fail(w, http.StatusConflict, 40901, "方案尚无云端实例，请直接添加至云端")
	case errors.Is(err, schemes.ErrAddCloudTooFast):
		apix.Fail(w, http.StatusTooManyRequests, 42901, "操作过于频繁，请 1 秒后再试")
	case errors.Is(err, schemes.ErrShareNotAllowed):
		apix.Fail(w, http.StatusOK, 42202, "该方案类型不可公开分享")
	case errors.Is(err, schemes.ErrDeleteWhileRunning):
		apix.Fail(w, http.StatusConflict, 40902, "运行中的方案不可删除")
	case errors.Is(err, schemes.ErrPatchWhileRunning):
		apix.Fail(w, http.StatusConflict, 40902, "运行中不可修改倍投或期次设定")
	case errors.Is(err, schemes.ErrPatchSimBetWhileRunning):
		apix.Fail(w, http.StatusConflict, 40902, "运行中不可修改投注通道")
	case errors.Is(err, schemes.ErrInvalidUpdatePatch):
		apix.Validation(w, "更新参数无效")
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
	case errors.Is(err, schemes.ErrInvalidKind):
		apix.Validation(w, "kind 须为 custom、contrary 或 follow")
	case errors.Is(err, schemes.ErrInvalidCreateRequest):
		apix.Validation(w, err.Error())
	case errors.Is(err, schemes.ErrNameDuplicate):
		apix.Fail(w, http.StatusOK, 42201, "方案名称已存在，请更换名称")
	default:
		apix.Internal(w)
	}
}

func (h *Handler) LotterySchemeOptions(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	code := r.PathValue("code")
	if code == "" {
		apix.Validation(w, "code 不能为空")
		return
	}
	result, err := h.schemes.GetSchemeOptions(r.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, schemes.ErrLotteryOptionsNotFound):
			apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "彩种选项不存在")
		case errors.Is(err, schemes.ErrUnavailable):
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		default:
			apix.Internal(w)
		}
		return
	}
	apix.OK(w, result)
}
