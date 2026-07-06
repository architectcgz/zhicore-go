package httpapi

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-file/internal/file/application"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *application.Service
	router  *gin.Engine
}

func NewHandler(service *application.Service) *gin.Engine {
	h := &Handler{
		service: service,
		router:  gin.New(),
	}
	h.routes()
	return h.router
}

func (h *Handler) routes() {
	h.router.POST("/api/v1/files/image", h.uploadImage)
	h.router.POST("/api/v1/files/audio", h.uploadAudio)
	h.router.POST("/api/v1/files/image/with-access", h.uploadImageWithAccess)
	h.router.POST("/api/v1/files/images/batch", h.uploadImagesBatch)
	h.router.GET("/api/v1/files/:fileId/url", h.getFileURL)
	h.router.DELETE("/api/v1/files/:fileId", h.deleteFile)
}

func (h *Handler) uploadImage(c *gin.Context) {
	w, r := c.Writer, c.Request
	limitMultipartBody(w, r, h.service.MaxImageSize())
	// 大文件 multipart 会落到临时文件，必须等应用层完成校验和上传读取后再清理。
	defer cleanupMultipartForm(r)
	file, err := filePayloadFromRequest(r, "file")
	if err != nil {
		sharedhttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.UploadImage(r.Context(), file, application.AccessLevelPublic)
	if err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, responseFromUploadResult(result))
}

func (h *Handler) uploadAudio(c *gin.Context) {
	w, r := c.Writer, c.Request
	limitMultipartBody(w, r, h.service.MaxAudioSize())
	// 音频上传同样可能使用临时文件，提前清理会让后续存储适配器读不到内容。
	defer cleanupMultipartForm(r)
	file, err := filePayloadFromRequest(r, "file")
	if err != nil {
		sharedhttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.UploadAudio(r.Context(), file, application.AccessLevelPublic)
	if err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, responseFromUploadResult(result))
}

func (h *Handler) uploadImageWithAccess(c *gin.Context) {
	w, r := c.Writer, c.Request
	limitMultipartBody(w, r, h.service.MaxImageSize())
	// 带权限上传仍由应用层最终读取文件，临时文件生命周期要覆盖整个 handler。
	defer cleanupMultipartForm(r)
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

func (h *Handler) uploadImagesBatch(c *gin.Context) {
	w, r := c.Writer, c.Request
	limitMultipartBody(w, r, h.service.MaxBatchImageSize())
	if err := r.ParseMultipartForm(h.service.MaxBatchImageSize()); err != nil {
		sharedhttp.WriteError(w, http.StatusBadRequest, "无效的 multipart 请求")
		return
	}
	defer cleanupMultipartForm(r)
	headers := r.MultipartForm.File["files"]
	files := make([]application.FilePayload, 0, len(headers))
	for _, header := range headers {
		files = append(files, filePayloadFromHeader(header))
	}
	accessLevel := parseAccessLevel(r.FormValue("accessLevel"))
	if accessLevel == "" {
		accessLevel = application.AccessLevelPublic
	}
	results, err := h.service.UploadImagesBatch(r.Context(), files, accessLevel)
	if err != nil {
		writeError(w, err)
		return
	}
	responses := make([]UploadFileResp, 0, len(results))
	for _, result := range results {
		responses = append(responses, responseFromUploadResult(result))
	}
	sharedhttp.WriteSuccess(w, responses)
}

func (h *Handler) getFileURL(c *gin.Context) {
	w, r := c.Writer, c.Request
	url, err := h.service.GetFileURL(r.Context(), c.Param("fileId"))
	if err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, url)
}

func (h *Handler) deleteFile(c *gin.Context) {
	w, r := c.Writer, c.Request
	if err := h.service.DeleteFile(r.Context(), c.Param("fileId")); err != nil {
		writeError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, nil)
}

func responseFromUploadResult(result application.UploadResult) UploadFileResp {
	uploadTime := ""
	if !result.UploadTime.IsZero() {
		uploadTime = result.UploadTime.Format(time.RFC3339)
	}
	return UploadFileResp{
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

func filePayloadFromRequest(r *http.Request, fieldName string) (application.FilePayload, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return application.FilePayload{}, errors.New("文件不能为空")
	}
	defer file.Close()
	return filePayloadFromHeader(header), nil
}

func filePayloadFromHeader(header *multipart.FileHeader) application.FilePayload {
	return application.FilePayload{
		OriginalName: header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Size:         header.Size,
		Open: func() (io.ReadCloser, error) {
			return header.Open()
		},
	}
}

func parseAccessLevel(value string) application.AccessLevel {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "PUBLIC":
		return application.AccessLevelPublic
	case "PRIVATE":
		return application.AccessLevelPrivate
	default:
		return application.AccessLevel(value)
	}
}

func writeError(w http.ResponseWriter, err error) {
	if appErr, ok := application.AsError(err); ok {
		sharedhttp.WriteError(w, statusFromApplicationCode(appErr.Code), appErr.Message)
		return
	}
	sharedhttp.WriteError(w, http.StatusInternalServerError, "系统内部错误，请稍后重试")
}

func statusFromApplicationCode(code application.Code) int {
	// 应用层只表达业务错误码，HTTP 语义集中在入站适配器映射，避免 use case 反向依赖传输协议。
	switch code {
	case application.CodeInvalidArgument:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
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
