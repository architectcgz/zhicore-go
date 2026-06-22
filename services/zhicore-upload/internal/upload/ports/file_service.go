package ports

import (
	"context"
	"io"
	"time"
)

type AccessLevel string

const (
	AccessLevelPublic  AccessLevel = "PUBLIC"
	AccessLevelPrivate AccessLevel = "PRIVATE"
)

type FilePayload struct {
	OriginalName string
	ContentType  string
	Size         int64
	Open         func() (io.ReadCloser, error)
}

type UploadResult struct {
	FileID        string
	URL           string
	FileSize      int64
	FileHash      string
	InstantUpload bool
	UploadTime    time.Time
	AccessLevel   AccessLevel
	OriginalName  string
	ContentType   string
}

type FileService interface {
	Upload(ctx context.Context, file FilePayload, accessLevel AccessLevel) (UploadResult, error)
	GetFileURL(ctx context.Context, fileID string) (string, error)
	DeleteFile(ctx context.Context, fileID string) error
}
