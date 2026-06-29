package httpapi

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/application"
	"github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/ports"
)

type Handler struct {
	service *application.Service
	mux     *http.ServeMux
}

func NewHandler(service *application.Service) http.Handler {
	h := &Handler{
		service: service,
		mux:     http.NewServeMux(),
	}
	h.routes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	h.mux.HandleFunc("POST /api/v1/upload/image", h.uploadImage)
	h.mux.HandleFunc("POST /api/v1/upload/audio", h.uploadAudio)
	h.mux.HandleFunc("POST /api/v1/upload/image/with-access", h.uploadImageWithAccess)
	h.mux.HandleFunc("POST /api/v1/upload/images/batch", h.uploadImagesBatch)
	h.mux.HandleFunc("GET /api/v1/upload/file/{fileId}/url", h.getFileURL)
	h.mux.HandleFunc("DELETE /api/v1/upload/file/{fileId}", h.deleteFile)
}

func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request) {
	limitMultipartBody(w, r, h.service.MaxImageSize())
	file, err := filePayloadFromRequest(r, "file")
	if err != nil {
		sharedhttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.UploadImage(r.Context(), file, ports.AccessLevelPublic)
	if err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, responseFromUploadResult(result))
}

func (h *Handler) uploadAudio(w http.ResponseWriter, r *http.Request) {
	limitMultipartBody(w, r, h.service.MaxAudioSize())
	file, err := filePayloadFromRequest(r, "file")
	if err != nil {
		sharedhttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.UploadAudio(r.Context(), file, ports.AccessLevelPublic)
	if err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, responseFromUploadResult(result))
}

func (h *Handler) uploadImageWithAccess(w http.ResponseWriter, r *http.Request) {
	limitMultipartBody(w, r, h.service.MaxImageSize())
	file, err := filePayloadFromRequest(r, "file")
	if err != nil {
		sharedhttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	rawAccessLevel := strings.TrimSpace(r.FormValue("accessLevel"))
	if rawAccessLevel == "" {
		sharedhttp.WriteError(w, http.StatusBadRequest, "accessLevel 不能为空")
		return
	}
	accessLevel := parseAccessLevel(rawAccessLevel)
	result, err := h.service.UploadImage(r.Context(), file, accessLevel)
	if err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, responseFromUploadResult(result))
}

func (h *Handler) uploadImagesBatch(w http.ResponseWriter, r *http.Request) {
	limitMultipartBody(w, r, h.service.MaxBatchImageSize())
	if err := r.ParseMultipartForm(h.service.MaxBatchImageSize()); err != nil {
		sharedhttp.WriteError(w, http.StatusBadRequest, "无效的 multipart 请求")
		return
	}
	defer cleanupMultipartForm(r)
	headers := r.MultipartForm.File["files"]
	files := make([]ports.FilePayload, 0, len(headers))
	for _, header := range headers {
		files = append(files, filePayloadFromHeader(header))
	}
	accessLevel := parseAccessLevel(r.FormValue("accessLevel"))
	if accessLevel == "" {
		accessLevel = ports.AccessLevelPublic
	}
	results, err := h.service.UploadImagesBatch(r.Context(), files, accessLevel)
	if err != nil {
		writeError(w, err)
		return
	}
	responses := make([]uploadResponse, 0, len(results))
	for _, result := range results {
		responses = append(responses, responseFromUploadResult(result))
	}
	sharedhttp.WriteSuccess(w, responses)
}

func (h *Handler) getFileURL(w http.ResponseWriter, r *http.Request) {
	url, err := h.service.GetFileURL(r.Context(), r.PathValue("fileId"))
	if err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, url)
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteFile(r.Context(), r.PathValue("fileId")); err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, nil)
}

type uploadResponse struct {
	FileID        string `json:"fileId"`
	URL           string `json:"url"`
	FileSize      int64  `json:"fileSize"`
	FileHash      string `json:"fileHash,omitempty"`
	InstantUpload bool   `json:"instantUpload"`
	UploadTime    string `json:"uploadTime,omitempty"`
	AccessLevel   string `json:"accessLevel"`
	OriginalName  string `json:"originalName"`
	ContentType   string `json:"contentType"`
}

func responseFromUploadResult(result ports.UploadResult) uploadResponse {
	uploadTime := ""
	if !result.UploadTime.IsZero() {
		uploadTime = result.UploadTime.Format(time.RFC3339)
	}
	return uploadResponse{
		FileID:        result.FileID,
		URL:           result.URL,
		FileSize:      result.FileSize,
		FileHash:      result.FileHash,
		InstantUpload: result.InstantUpload,
		UploadTime:    uploadTime,
		AccessLevel:   string(result.AccessLevel),
		OriginalName:  result.OriginalName,
		ContentType:   result.ContentType,
	}
}

func filePayloadFromRequest(r *http.Request, fieldName string) (ports.FilePayload, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return ports.FilePayload{}, errors.New("文件不能为空")
	}
	defer file.Close()
	defer cleanupMultipartForm(r)
	return filePayloadFromHeader(header), nil
}

func filePayloadFromHeader(header *multipart.FileHeader) ports.FilePayload {
	return ports.FilePayload{
		OriginalName: header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Size:         header.Size,
		Open: func() (io.ReadCloser, error) {
			return header.Open()
		},
	}
}

func parseAccessLevel(value string) ports.AccessLevel {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "PUBLIC":
		return ports.AccessLevelPublic
	case "PRIVATE":
		return ports.AccessLevelPrivate
	default:
		return ports.AccessLevel(value)
	}
}

func writeError(w http.ResponseWriter, err error) {
	if appErr, ok := application.AsError(err); ok {
		sharedhttp.WriteError(w, appErr.Status, appErr.Message)
		return
	}
	sharedhttp.WriteError(w, http.StatusInternalServerError, "系统内部错误，请稍后重试")
}

func limitMultipartBody(w http.ResponseWriter, r *http.Request, maxFileSize int64) {
	const multipartOverhead = 1 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize+multipartOverhead)
}

func cleanupMultipartForm(r *http.Request) {
	if r.MultipartForm != nil {
		_ = r.MultipartForm.RemoveAll()
	}
}
