package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/content"
)

func (h *Handler) AdminListBanners(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var enabled *bool
	switch strings.TrimSpace(r.URL.Query().Get("enabled")) {
	case "true":
		v := true
		enabled = &v
	case "false":
		v := false
		enabled = &v
	}
	result, err := svc.AdminListBanners(r.Context(), content.AdminBannerListQuery{
		Page:        queryInt(r, "page", 1),
		PageSize:    queryInt(r, "pageSize", 10),
		Enabled:     enabled,
		CreatedFrom: parseOptionalDateQuery(r, "createdFrom", false),
		CreatedTo:   parseOptionalDateQuery(r, "createdTo", true),
	})
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) AdminSaveBanner(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var body struct {
		ID       string `json:"id"`
		Remark   string `json:"remark"`
		ImageUrl string `json:"imageUrl"`
		LinkUrl  string `json:"linkUrl"`
		Sort     int    `json:"sort"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminSaveBanner(r.Context(), content.SaveBannerInput{
		ID: body.ID, Remark: body.Remark, ImageUrl: body.ImageUrl,
		LinkUrl: body.LinkUrl, Sort: body.Sort, Enabled: body.Enabled,
	})
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	action := "创建"
	if strings.TrimSpace(body.ID) != "" {
		action = "编辑"
	}
	h.writeAudit(r, fmt.Sprintf("%s Banner %s", action, item.ID))
	apix.OK(w, item)
}

func (h *Handler) AdminSetBannerEnabled(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id := r.PathValue("id")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminSetBannerEnabled(r.Context(), id, body.Enabled)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	state := "禁用"
	if body.Enabled {
		state = "启用"
	}
	h.writeAudit(r, fmt.Sprintf("%s Banner %s", state, item.ID))
	apix.OK(w, item)
}

func (h *Handler) AdminDeleteBanner(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id := r.PathValue("id")
	if err := svc.AdminDeleteBanner(r.Context(), id); err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("删除 Banner %s", id))
	apix.OK(w, map[string]string{"id": id})
}

func (h *Handler) PublicBanners(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.content.PublicBanners(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func parseOptionalDateQuery(r *http.Request, key string, endOfDay bool) *time.Time {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil
	}
	var t time.Time
	var err error
	if len(raw) == 10 {
		t, err = time.Parse("2006-01-02", raw)
	} else {
		t, err = time.Parse(time.RFC3339, raw)
	}
	if err != nil {
		return nil
	}
	if endOfDay && len(raw) == 10 {
		t = t.Add(24*time.Hour - time.Nanosecond)
	}
	utc := t.UTC()
	return &utc
}
