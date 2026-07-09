package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/audit"
	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/cloud/betrecords"
	"caipiao/backend/internal/cloud/instances"
	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/content"
	"caipiao/backend/internal/copyhall"
	"caipiao/backend/internal/dashboard"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/games"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/maintenance"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/middleware"
	ordersadmin "caipiao/backend/internal/orders/admin"
	"caipiao/backend/internal/orders/bets"
	"caipiao/backend/internal/orders/chases"
	"caipiao/backend/internal/reports"
	"caipiao/backend/internal/schemes"
	"caipiao/backend/internal/ws"
)

type Handler struct {
	auth        *auth.Service
	maintenance *maintenance.Service
	betRecords  *betrecords.Service
	instances   *instances.Store
	lookback    *lookback.Service
	db          *db.Pool
	members     *member.Service
	bets        *bets.Service
	chases      *chases.Service
	copyHall    *copyhall.Service
	schemes     *schemes.Service
	games       *games.Service
	content     *content.Service
	audit       *audit.Service
	dashboard   *dashboard.Service
	ordersAdmin *ordersadmin.Service
	reports     *reports.Service
	wsHub       *ws.Hub
	guaji       *guaji.Client
	guajiAccounts *accountsvc.Service
	cmsUploads    *content.UploadStore
	maintenanceResume schemes.MaintenanceResumeScheduler
}

func New(
	authSvc *auth.Service,
	maint *maintenance.Service,
	betSvc *betrecords.Service,
	inst *instances.Store,
	lb *lookback.Service,
	pool *db.Pool,
	memberSvc *member.Service,
	betOrders *bets.Service,
	chaseSvc *chases.Service,
	copyHallSvc *copyhall.Service,
	schemesSvc *schemes.Service,
	gamesSvc *games.Service,
	contentSvc *content.Service,
	auditSvc *audit.Service,
	dashboardSvc *dashboard.Service,
	ordersAdminSvc *ordersadmin.Service,
	reportsSvc *reports.Service,
	wsHub *ws.Hub,
	guajiClient *guaji.Client,
	guajiAccounts *accountsvc.Service,
	cmsUploads *content.UploadStore,
) *Handler {
	return &Handler{
		auth: authSvc, maintenance: maint, betRecords: betSvc,
		instances: inst, lookback: lb, db: pool, members: memberSvc, bets: betOrders,
		chases: chaseSvc, copyHall: copyHallSvc,
		schemes: schemesSvc, games: gamesSvc, content: contentSvc, audit: auditSvc,
		dashboard: dashboardSvc, ordersAdmin: ordersAdminSvc, reports: reportsSvc,
		wsHub: wsHub, guaji: guajiClient, guajiAccounts: guajiAccounts,
		cmsUploads: cmsUploads,
	}
}

// SetMaintenanceResumeScheduler 注入 Scheme Worker，用于维护/彩种上架后主动触发续投扫描。
func (h *Handler) SetMaintenanceResumeScheduler(s schemes.MaintenanceResumeScheduler) {
	if h == nil {
		return
	}
	h.maintenanceResume = s
}

func (h *Handler) scheduleMaintenanceResume() {
	if h == nil || h.maintenanceResume == nil {
		return
	}
	scheduler := h.maintenanceResume
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		scheduler.TickMaintenanceResume(ctx)
	}()
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{
		"status": "ok",
		"db":     "disabled",
	}
	if h.db == nil {
		if h.guaji != nil {
			payload["guaji"] = h.guajiHealth(r.Context())
		}
		apix.OK(w, payload)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		payload["db"] = "down"
		payload["dbError"] = err.Error()
	} else {
		payload["db"] = "up"
		if version, err := h.db.ServerVersion(ctx); err == nil {
			payload["postgres"] = version
		}
	}
	if h.guaji != nil {
		gctx, gcancel := context.WithTimeout(r.Context(), 8*time.Second)
		payload["guaji"] = h.guajiHealth(gctx)
		gcancel()
	}
	apix.OK(w, payload)
}

func (h *Handler) guajiHealth(ctx context.Context) guaji.ProbeResult {
	if h.guaji == nil {
		return guaji.ProbeResult{Enabled: false}
	}
	return h.guaji.Probe(ctx)
}

func (h *Handler) PublicMaintenance(w http.ResponseWriter, r *http.Request) {
	if h.maintenance == nil {
		apix.OK(w, maintenance.State{})
		return
	}
	state, err := h.maintenance.PublicGet(r.Context())
	if err != nil {
		h.handleMaintenanceErr(w, err)
		return
	}
	apix.OK(w, state)
}

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

func (h *Handler) ClientLogin(w http.ResponseWriter, r *http.Request) {
	h.login(w, r, h.auth.LoginClient)
}

func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON，且包含 account / password")
		return
	}
	if req.Account == "" || req.Password == "" {
		apix.Validation(w, "account 与 password 不能为空")
		return
	}
	result, err := h.auth.LoginAdmin(req.Account, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			apix.Fail(w, http.StatusOK, apix.CodeUnauthorized, "账号或密码错误")
		case errors.Is(err, auth.ErrAccountFrozen):
			apix.Fail(w, http.StatusOK, apix.CodeForbidden, "账号已冻结")
		default:
			apix.Internal(w)
		}
		return
	}
	if h.audit != nil {
		_ = h.audit.Append(r.Context(), result.Account, "登录后台", clientIP(r))
	}
	apix.OK(w, map[string]string{
		"accessToken": result.AccessToken,
		"expiresAt":   result.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		"account":     result.Account,
		"displayName": result.DisplayName,
		"roleId":      result.RoleID,
	})
}

func (h *Handler) AdminSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		apix.Unauthorized(w, "")
		return
	}
	apix.OK(w, map[string]string{
		"account":     claims.Subject,
		"displayName": claims.DisplayName,
		"roleId":      claims.AdminRoleID,
	})
}

type loginFn func(account, password string) (auth.TokenResult, error)

func (h *Handler) login(w http.ResponseWriter, r *http.Request, fn loginFn) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON，且包含 account / password")
		return
	}
	if req.Account == "" || req.Password == "" {
		apix.Validation(w, "account 与 password 不能为空")
		return
	}
	result, err := fn(req.Account, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			apix.Fail(w, http.StatusOK, apix.CodeUnauthorized, "账号或密码错误")
		case errors.Is(err, auth.ErrAccountFrozen):
			apix.Fail(w, http.StatusOK, apix.CodeForbidden, "账号已冻结")
		default:
			apix.Internal(w)
		}
		return
	}
	apix.OK(w, map[string]string{
		"accessToken": result.AccessToken,
		"expiresAt":   result.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		"account":     result.Account,
		"displayName": result.DisplayName,
	})
}

func (h *Handler) BetRecordGroups(w http.ResponseWriter, r *http.Request) {
	h.withMember(w, r, func(members *member.Service, account string) {
		m, err := members.GetByAccount(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		filter := betrecords.GroupsFilter{
			Mode:        strings.TrimSpace(r.URL.Query().Get("mode")),
			Days:        queryInt(r, "days", 3),
			DateFrom:    strings.TrimSpace(r.URL.Query().Get("dateFrom")),
			DateTo:      strings.TrimSpace(r.URL.Query().Get("dateTo")),
			LotteryCode: strings.TrimSpace(r.URL.Query().Get("lotteryCode")),
		}
		if err := filter.Validate(); err != nil {
			apix.Validation(w, err.Error())
			return
		}
		filter.Cursor = strings.TrimSpace(r.URL.Query().Get("cursor"))
		filter.Limit = queryInt(r, "limit", 20)
		result, err := h.betRecords.GroupsWithFilter(r.Context(), m.ID, filter)
		if err != nil {
			apix.Validation(w, "cursor 无效")
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) BetRecordDetail(w http.ResponseWriter, r *http.Request) {
	schemeID := r.PathValue("schemeId")
	if schemeID == "" {
		apix.Validation(w, "schemeId 不能为空")
		return
	}
	h.withMember(w, r, func(members *member.Service, account string) {
		m, err := members.GetByAccount(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		mode := betrecords.ParseMode(r.URL.Query().Get("mode"))
		days := queryInt(r, "days", 3)
		limit := queryInt(r, "limit", 50)
		cursor := r.URL.Query().Get("cursor")

		result, ok, err := h.betRecords.Detail(r.Context(), m.ID, schemeID, mode, days, limit, cursor)
		if err != nil {
			apix.Validation(w, "cursor 无效")
			return
		}
		if !ok {
			apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "方案不存在")
			return
		}
		apix.OK(w, result)
	})
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
