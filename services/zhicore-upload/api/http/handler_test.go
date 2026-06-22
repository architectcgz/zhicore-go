package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	uploadhttp "github.com/architectcgz/zhicore-go/services/zhicore-upload/api/http"
	"github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/application"
	"github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/ports"
)

func TestUploadImageUsesPublicAccessAndReturnsJavaCompatibleEnvelope(t *testing.T) {
	fileService := &fakeFileService{
		uploadResult: ports.UploadResult{
			FileID:        "file_123",
			URL:           "https://cdn.example.com/file_123.jpg",
			FileSize:      12,
			AccessLevel:   ports.AccessLevelPublic,
			OriginalName:  "avatar.jpg",
			ContentType:   "image/jpeg",
			UploadTime:    time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC),
			InstantUpload: false,
		},
	}
	handler := uploadhttp.NewHandler(application.NewService(fileService, application.DefaultConfig()))

	req := multipartRequest(t, http.MethodPost, "/api/v1/upload/image", "file", "avatar.jpg", "image/jpeg", []byte("image-bytes"), nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body uploadEnvelope[fileResponse]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if body.Code != 200 || body.Message != "操作成功" {
		t.Fatalf("envelope = (%d, %q), want (200, 操作成功)", body.Code, body.Message)
	}
	if body.Data.FileID != "file_123" || body.Data.OriginalName != "avatar.jpg" || body.Data.AccessLevel != "PUBLIC" {
		t.Fatalf("unexpected data: %+v", body.Data)
	}
	if fileService.lastAccessLevel != ports.AccessLevelPublic {
		t.Fatalf("access level = %q, want PUBLIC", fileService.lastAccessLevel)
	}
	if fileService.lastFile.OriginalName != "avatar.jpg" || fileService.lastFile.ContentType != "image/jpeg" {
		t.Fatalf("uploaded file metadata = %+v", fileService.lastFile)
	}
}

func TestUploadImageRejectsUnsupportedContentType(t *testing.T) {
	fileService := &fakeFileService{}
	handler := uploadhttp.NewHandler(application.NewService(fileService, application.DefaultConfig()))

	req := multipartRequest(t, http.MethodPost, "/api/v1/upload/image", "file", "notes.txt", "text/plain", []byte("plain"), nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	var body uploadEnvelope[json.RawMessage]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if body.Code != 400 {
		t.Fatalf("code = %d, want 400", body.Code)
	}
	if !strings.Contains(body.Message, "文件类型") {
		t.Fatalf("message = %q, want contains 文件类型", body.Message)
	}
	if fileService.uploadCalled {
		t.Fatal("file service should not be called for invalid content type")
	}
}

func TestUploadImageWithAccessPassesPrivateAccess(t *testing.T) {
	fileService := &fakeFileService{
		uploadResult: ports.UploadResult{
			FileID:       "file_private",
			URL:          "https://cdn.example.com/private.jpg",
			FileSize:     7,
			AccessLevel:  ports.AccessLevelPrivate,
			OriginalName: "private.jpg",
			ContentType:  "image/jpeg",
			UploadTime:   time.Date(2026, 6, 22, 10, 1, 0, 0, time.UTC),
		},
	}
	handler := uploadhttp.NewHandler(application.NewService(fileService, application.DefaultConfig()))

	req := multipartRequest(t, http.MethodPost, "/api/v1/upload/image/with-access", "file", "private.jpg", "image/jpeg", []byte("private"), map[string]string{
		"accessLevel": "PRIVATE",
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if fileService.lastAccessLevel != ports.AccessLevelPrivate {
		t.Fatalf("access level = %q, want PRIVATE", fileService.lastAccessLevel)
	}
}

func TestUploadImageWithAccessRequiresAccessLevel(t *testing.T) {
	fileService := &fakeFileService{}
	handler := uploadhttp.NewHandler(application.NewService(fileService, application.DefaultConfig()))

	req := multipartRequest(t, http.MethodPost, "/api/v1/upload/image/with-access", "file", "private.jpg", "image/jpeg", []byte("private"), nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	var body uploadEnvelope[json.RawMessage]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if !strings.Contains(body.Message, "accessLevel") {
		t.Fatalf("message = %q, want contains accessLevel", body.Message)
	}
	if fileService.uploadCalled {
		t.Fatal("file service should not be called when accessLevel is missing")
	}
}

func TestGetFileURLAndDeleteFileUsePathFileID(t *testing.T) {
	fileService := &fakeFileService{fileURL: "https://cdn.example.com/file_123.jpg"}
	handler := uploadhttp.NewHandler(application.NewService(fileService, application.DefaultConfig()))

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/upload/file/file_123/url", nil)
	getRR := httptest.NewRecorder()
	handler.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body=%s", getRR.Code, http.StatusOK, getRR.Body.String())
	}
	var getBody uploadEnvelope[string]
	decodeJSON(t, getRR.Body.Bytes(), &getBody)
	if getBody.Data != "https://cdn.example.com/file_123.jpg" || fileService.lastURLFileID != "file_123" {
		t.Fatalf("GET data=%q lastURLFileID=%q", getBody.Data, fileService.lastURLFileID)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/upload/file/file_123", nil)
	deleteRR := httptest.NewRecorder()
	handler.ServeHTTP(deleteRR, deleteReq)

	if deleteRR.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want %d; body=%s", deleteRR.Code, http.StatusOK, deleteRR.Body.String())
	}
	var deleteBody uploadEnvelope[json.RawMessage]
	decodeJSON(t, deleteRR.Body.Bytes(), &deleteBody)
	if deleteBody.Code != 200 || deleteBody.Message != "操作成功" {
		t.Fatalf("DELETE envelope = (%d, %q), want success", deleteBody.Code, deleteBody.Message)
	}
	if fileService.lastDeleteFileID != "file_123" {
		t.Fatalf("deleted file id = %q, want file_123", fileService.lastDeleteFileID)
	}
}

type fakeFileService struct {
	uploadResult ports.UploadResult
	fileURL      string

	uploadCalled     bool
	lastFile         ports.FilePayload
	lastAccessLevel  ports.AccessLevel
	lastURLFileID    string
	lastDeleteFileID string
}

func (f *fakeFileService) Upload(ctx context.Context, file ports.FilePayload, accessLevel ports.AccessLevel) (ports.UploadResult, error) {
	f.uploadCalled = true
	f.lastFile = file
	f.lastAccessLevel = accessLevel
	return f.uploadResult, nil
}

func (f *fakeFileService) GetFileURL(ctx context.Context, fileID string) (string, error) {
	f.lastURLFileID = fileID
	return f.fileURL, nil
}

func (f *fakeFileService) DeleteFile(ctx context.Context, fileID string) error {
	f.lastDeleteFileID = fileID
	return nil
}

type uploadEnvelope[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	Timestamp int64  `json:"timestamp"`
	TraceID   string `json:"traceId,omitempty"`
}

type fileResponse struct {
	FileID        string `json:"fileId"`
	URL           string `json:"url"`
	FileSize      int64  `json:"fileSize"`
	FileHash      string `json:"fileHash,omitempty"`
	InstantUpload bool   `json:"instantUpload"`
	UploadTime    string `json:"uploadTime"`
	AccessLevel   string `json:"accessLevel"`
	OriginalName  string `json:"originalName"`
	ContentType   string `json:"contentType"`
}

func multipartRequest(t *testing.T, method, target, fieldName, fileName, contentType string, data []byte, fields map[string]string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}

	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="` + fieldName + `"; filename="` + fileName + `"`},
		"Content-Type":        {contentType},
	})
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func decodeJSON(t *testing.T, data []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode response %s: %v", string(data), err)
	}
}
