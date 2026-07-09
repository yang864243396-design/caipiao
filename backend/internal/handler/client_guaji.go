package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/member"
)

func (h *Handler) GuajiAuthStatus(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		st, err := h.guajiAccounts.AuthStatus(r.Context(), account)
		if err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, st)
	})
}

func (h *Handler) GuajiListAccounts(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		items, err := h.guajiAccounts.List(r.Context(), account)
		if err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, map[string]any{"items": items})
	})
}

type guajiBindRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	LoginKey   string `json:"loginKey"`
	GoogleCode string `json:"googleCode"`
	EmailCode  string `json:"emailCode"`
	PhoneCode  string `json:"phoneCode"`
}

func (h *Handler) GuajiBindAccount(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	var req guajiBindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.guajiAccounts.Bind(r.Context(), account, accountsvc.BindInput{
			Username:   req.Username,
			Password:   req.Password,
			LoginKey:   req.LoginKey,
			GoogleCode: req.GoogleCode,
			EmailCode:  req.EmailCode,
			PhoneCode:  req.PhoneCode,
		})
		if err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) GuajiActivateAccount(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	id, err := parseGuajiAccountID(r)
	if err != nil {
		apix.Validation(w, "无效的账号 id")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		acct, err := h.guajiAccounts.Activate(r.Context(), account, id)
		if err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, acct)
	})
}

func (h *Handler) GuajiReauthAccount(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	id, err := parseGuajiAccountID(r)
	if err != nil {
		apix.Validation(w, "无效的账号 id")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		acct, err := h.guajiAccounts.Reauth(r.Context(), account, id)
		if err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, acct)
	})
}

func (h *Handler) GuajiDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	id, err := parseGuajiAccountID(r)
	if err != nil {
		apix.Validation(w, "无效的账号 id")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		if err := h.guajiAccounts.Unbind(r.Context(), account, id); err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, map[string]bool{"ok": true})
	})
}

func (h *Handler) GuajiGetPrimaryCurrency(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		cur, err := h.guajiAccounts.PrimaryCurrency(r.Context(), account)
		if err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, map[string]string{"currency": cur})
	})
}

type guajiPrimaryCurrencyRequest struct {
	Currency string `json:"currency"`
}

func (h *Handler) GuajiSetPrimaryCurrency(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	var req guajiPrimaryCurrencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		cur, err := h.guajiAccounts.SetPrimaryCurrency(r.Context(), account, req.Currency)
		if err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, map[string]string{"currency": cur})
	})
}

func (h *Handler) GuajiBalance(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		bal, err := h.guajiAccounts.Balance(r.Context(), account)
		if err != nil {
			h.handleGuajiErr(w, err)
			return
		}
		apix.OK(w, bal)
	})
}

func (h *Handler) AdminGuajiHealth(w http.ResponseWriter, r *http.Request) {
	if h.guajiAccounts == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务未就绪")
		return
	}
	stats, err := h.guajiAccounts.Health(r.Context())
	if err != nil {
		apix.Internal(w)
		return
	}
	apix.OK(w, stats)
}

func (h *Handler) AdminMemberGuajiAccounts(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.guajiAccounts.AdminList(r.Context(), memberID)
	if err != nil {
		h.handleGuajiErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func parseGuajiAccountID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	if raw == "" {
		return 0, errors.New("missing id")
	}
	return strconv.ParseInt(raw, 10, 64)
}

func (h *Handler) handleGuajiErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
	case errors.Is(err, accountsvc.ErrAccountNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "授权账号不存在")
	case errors.Is(err, accountsvc.ErrUsernameTaken):
		apix.Fail(w, http.StatusOK, apix.CodeValidation, "该第三方账号已被其他会员绑定，须先让对方解绑")
	case errors.Is(err, accountsvc.ErrGuajiDisabled):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "第三方对接未启用")
	case errors.Is(err, accountsvc.ErrCredentialsKey):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "服务端未配置 GUAJI_CREDENTIALS_KEY")
	case errors.Is(err, accountsvc.ErrNoActiveAccount):
		apix.Fail(w, http.StatusOK, apix.CodeValidation, "无启用中的授权账号")
	case errors.Is(err, accountsvc.ErrTokenInvalid):
		apix.Fail(w, http.StatusOK, apix.CodeUnauthorized, "授权已失效，请在授权列表页重新授权")
	case errors.Is(err, accountsvc.ErrReauthNeedsBind):
		apix.Fail(w, http.StatusOK, apix.CodeValidation, "自动重新授权失败，请前往绑定页重填密码")
	case errors.Is(err, accountsvc.ErrInvalidCredentials):
		apix.Fail(w, http.StatusOK, apix.CodeUnauthorized, "第三方账号或密码错误")
	case errors.Is(err, accountsvc.ErrInvalidCurrency):
		apix.Fail(w, http.StatusOK, apix.CodeValidation, "主币种仅支持 USDT / TRX / CNY")
	case errors.Is(err, accountsvc.ErrGuajiUpstream):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "第三方服务暂时不可用，请稍后重试")
	default:
		apix.Internal(w)
	}
}
