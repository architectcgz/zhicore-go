package httpapi_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	uploadhttp "github.com/architectcgz/zhicore-go/services/zhicore-upload/api/http"
	"github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/application"
	"github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/ports"
)

func TestUploadImagesBatchRejectsPartialFailureInsteadOfDroppingInvalidFiles(t *testing.T) {
	fileService := &fakeFileService{
		uploadResult: ports.UploadResult{
			FileID:       "file_ok",
			URL:          "https://cdn.example.com/file_ok.jpg",
			FileSize:     int64(len(jpegHeaderBytes())),
			AccessLevel:  ports.AccessLevelPublic,
			OriginalName: "ok.jpg",
			ContentType:  "image/jpeg",
		},
	}
	handler := uploadhttp.NewHandler(application.NewService(fileService, application.DefaultConfig()))

	req := batchMultipartRequest(t, http.MethodPost, "/api/v1/upload/images/batch", []multipartFile{
		{name: "ok.jpg", contentType: "image/jpeg", data: jpegHeaderBytes()},
		{name: "bad.txt", contentType: "text/plain", data: []byte("plain")},
	}, nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	var body uploadEnvelope[json.RawMessage]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if !strings.Contains(body.Message, "批量上传存在失败文件") {
		t.Fatalf("message = %q, want contains 批量上传存在失败文件", body.Message)
	}
}

type multipartFile struct {
	name        string
	contentType string
	data        []byte
}

func batchMultipartRequest(t *testing.T, method, target string, files []multipartFile, fields map[string]string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}

	for _, file := range files {
		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="files"; filename="` + file.name + `"`},
			"Content-Type":        {file.contentType},
		})
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(file.data); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
