package application

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/ports"
)

func TestUploadImagesBatchReturnsErrorWhenAnyFileFailsValidation(t *testing.T) {
	service := NewService(&fakeFileService{}, DefaultConfig())

	_, err := service.UploadImagesBatch(context.Background(), []ports.FilePayload{
		filePayload("ok.jpg", "image/jpeg", jpegHeaderBytes()),
		filePayload("bad.txt", "text/plain", []byte("plain")),
	}, ports.AccessLevelPublic)

	if err == nil {
		t.Fatal("error = nil, want batch validation error")
	}
	if !strings.Contains(err.Error(), "批量上传存在失败文件") {
		t.Fatalf("error = %q, want contains 批量上传存在失败文件", err.Error())
	}
}

type fakeFileService struct{}

func (f *fakeFileService) Upload(ctx context.Context, file ports.FilePayload, accessLevel ports.AccessLevel) (ports.UploadResult, error) {
	return ports.UploadResult{}, nil
}

func (f *fakeFileService) GetFileURL(ctx context.Context, fileID string) (string, error) {
	return "", nil
}

func (f *fakeFileService) DeleteFile(ctx context.Context, fileID string) error {
	return nil
}

func filePayload(name string, contentType string, data []byte) ports.FilePayload {
	return ports.FilePayload{
		OriginalName: name,
		ContentType:  contentType,
		Size:         int64(len(data)),
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
}

func jpegHeaderBytes() []byte {
	return []byte{
		0xff, 0xd8, 0xff, 0xdb,
		0x00, 0x43, 0x00,
		0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07,
		0x07, 0x07, 0x09, 0x09, 0x08, 0x0a, 0x0c, 0x14,
		0x0d, 0x0c, 0x0b, 0x0b,
	}
}
