package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/content"
)

type cmsUploadResponse struct {
	URL string `json:"url"`
}

func (h *Handler) AdminUploadCMSImage(w http.ResponseWriter, r *http.Request) {
	if h.cmsUploads == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "上传服务未就绪")
		return
	}
	if r.Method != http.MethodPost {
		apix.Fail(w, http.StatusMethodNotAllowed, apix.CodeValidation, "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, content.MaxCMSUploadBytes+512)
	if err := r.ParseMultipartForm(content.MaxCMSUploadBytes); err != nil {
		apix.Validation(w, "文件过大或格式无效")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		apix.Validation(w, "请选择要上传的图片")
		return
	}
	defer file.Close()

	name, err := h.cmsUploads.SaveImage(file, content.MaxCMSUploadBytes)
	if err != nil {
		h.handleCMSUploadErr(w, err)
		return
	}
	h.writeAudit(r, "上传 CMS 图片 "+name)
	apix.OK(w, cmsUploadResponse{URL: publicCMSUploadURL(r, name)})
}

func (h *Handler) PublicCMSUpload(w http.ResponseWriter, r *http.Request) {
	if h.cmsUploads == nil {
		http.NotFound(w, r)
		return
	}
	name := r.PathValue("filename")
	path, err := h.cmsUploads.FilePath(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func publicCMSUploadURL(r *http.Request, filename string) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/api/v1/public/cms-uploads/" + filename
}

func (h *Handler) handleCMSUploadErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, content.ErrUploadTooLarge):
		apix.Validation(w, "图片大小不能超过 5MB")
	case errors.Is(err, content.ErrUploadInvalidType):
		apix.Validation(w, "仅支持 JPG、PNG、GIF、WebP 图片")
	case errors.Is(err, content.ErrUploadUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "上传服务未就绪")
	default:
		apix.Internal(w)
	}
}
