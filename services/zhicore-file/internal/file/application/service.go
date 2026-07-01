package application

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-file/internal/file/ports"
)

const (
	defaultMaxImageSize      = 50 * 1024 * 1024
	defaultMaxAudioSize      = 10 * 1024 * 1024
	defaultMaxBatchImageSize = 100 * 1024 * 1024
	fileSniffSize            = 512
)

type AccessLevel = ports.AccessLevel
type FilePayload = ports.FilePayload
type UploadResult = ports.UploadResult

const (
	AccessLevelPublic  = ports.AccessLevelPublic
	AccessLevelPrivate = ports.AccessLevelPrivate
)

type Config struct {
	MaxImageSize      int64
	MaxAudioSize      int64
	MaxBatchImageSize int64
	AllowedImageType  []string
	AllowedAudioType  []string
}

func DefaultConfig() Config {
	return Config{
		MaxImageSize:      defaultMaxImageSize,
		MaxAudioSize:      defaultMaxAudioSize,
		MaxBatchImageSize: defaultMaxBatchImageSize,
		AllowedImageType: []string{
			"image/jpeg",
			"image/jpg",
			"image/png",
			"image/gif",
			"image/webp",
		},
		AllowedAudioType: []string{
			"audio/mpeg",
			"audio/mp3",
			"audio/mp4",
			"audio/x-m4a",
			"audio/aac",
			"audio/wav",
			"audio/x-wav",
			"audio/ogg",
			"audio/webm",
		},
	}
}

type Service struct {
	files  ports.FileService
	config Config
}

func NewService(files ports.FileService, config Config) *Service {
	if config.MaxImageSize == 0 {
		config.MaxImageSize = defaultMaxImageSize
	}
	if config.MaxAudioSize == 0 {
		config.MaxAudioSize = defaultMaxAudioSize
	}
	if config.MaxBatchImageSize == 0 {
		config.MaxBatchImageSize = defaultMaxBatchImageSize
	}
	return &Service{files: files, config: config}
}

func (s *Service) MaxImageSize() int64 {
	return s.config.MaxImageSize
}

func (s *Service) MaxAudioSize() int64 {
	return s.config.MaxAudioSize
}

func (s *Service) MaxBatchImageSize() int64 {
	return s.config.MaxBatchImageSize
}

func (s *Service) UploadImage(ctx context.Context, file ports.FilePayload, accessLevel ports.AccessLevel) (ports.UploadResult, error) {
	if err := validateAccessLevel(accessLevel); err != nil {
		return ports.UploadResult{}, err
	}
	if err := validateFile(file, s.config.AllowedImageType, s.config.MaxImageSize); err != nil {
		return ports.UploadResult{}, err
	}
	return s.files.Upload(ctx, file, accessLevel)
}

func (s *Service) UploadAudio(ctx context.Context, file ports.FilePayload, accessLevel ports.AccessLevel) (ports.UploadResult, error) {
	if err := validateAccessLevel(accessLevel); err != nil {
		return ports.UploadResult{}, err
	}
	if err := validateFile(file, s.config.AllowedAudioType, s.config.MaxAudioSize); err != nil {
		return ports.UploadResult{}, err
	}
	return s.files.Upload(ctx, file, accessLevel)
}

func (s *Service) UploadImagesBatch(ctx context.Context, files []ports.FilePayload, accessLevel ports.AccessLevel) ([]ports.UploadResult, error) {
	if err := validateAccessLevel(accessLevel); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, invalidArgument("文件不能为空")
	}
	results := make([]ports.UploadResult, 0, len(files))
	for index, file := range files {
		result, err := s.UploadImage(ctx, file, accessLevel)
		if err != nil {
			// 批量上传不能静默丢弃失败文件，否则调用方会误以为整批资源已完成入库和可引用。
			if appErr, ok := AsError(err); ok {
				return nil, invalidArgument(fmt.Sprintf("批量上传存在失败文件: 第 %d 个文件 %s, %s", index+1, file.OriginalName, appErr.Message))
			}
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) GetFileURL(ctx context.Context, fileID string) (string, error) {
	if strings.TrimSpace(fileID) == "" {
		return "", invalidArgument("文件ID不能为空")
	}
	return s.files.GetFileURL(ctx, fileID)
}

func (s *Service) DeleteFile(ctx context.Context, fileID string) error {
	if strings.TrimSpace(fileID) == "" {
		return invalidArgument("文件ID不能为空")
	}
	return s.files.DeleteFile(ctx, fileID)
}

func validateAccessLevel(accessLevel ports.AccessLevel) error {
	switch accessLevel {
	case ports.AccessLevelPublic, ports.AccessLevelPrivate:
		return nil
	default:
		return invalidArgument("访问级别必须是 PUBLIC 或 PRIVATE")
	}
}

func validateFile(file ports.FilePayload, allowedTypes []string, maxSize int64) error {
	if file.Size <= 0 || file.Open == nil {
		return invalidArgument("文件不能为空")
	}
	if !containsContentType(file.ContentType, allowedTypes) {
		return invalidArgument(fmt.Sprintf("文件类型不允许: %s", file.ContentType))
	}
	if file.Size > maxSize {
		return invalidArgument("文件大小超过限制")
	}
	if !extensionAllowsContentType(file.OriginalName, file.ContentType) {
		return invalidArgument("文件类型不允许")
	}
	detectedType, err := detectContentType(file)
	if err != nil {
		return invalidArgument("文件不能为空")
	}
	if !contentTypesCompatible(file.ContentType, detectedType) || !containsContentType(detectedType, allowedTypes) {
		return invalidArgument(fmt.Sprintf("文件类型不允许: %s", detectedType))
	}
	return nil
}

func containsContentType(contentType string, allowedTypes []string) bool {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	for _, allowedType := range allowedTypes {
		if contentType == strings.ToLower(allowedType) {
			return true
		}
	}
	return false
}

func detectContentType(file ports.FilePayload) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	buffer := make([]byte, fileSniffSize)
	n, err := io.ReadFull(rc, buffer)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}
	if n == 0 {
		return "", io.EOF
	}
	return normalizeContentType(http.DetectContentType(buffer[:n])), nil
}

func extensionAllowsContentType(name string, contentType string) bool {
	contentType = normalizeContentType(contentType)
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return contentType == "image/jpeg"
	case ".png":
		return contentType == "image/png"
	case ".gif":
		return contentType == "image/gif"
	case ".webp":
		return contentType == "image/webp"
	case ".mp3":
		return contentType == "audio/mpeg" || contentType == "audio/mp3"
	case ".m4a", ".mp4":
		return contentType == "audio/mp4" || contentType == "audio/x-m4a" || contentType == "audio/aac"
	case ".aac":
		return contentType == "audio/aac" || contentType == "audio/mp4"
	case ".wav":
		return contentType == "audio/wav" || contentType == "audio/x-wav"
	case ".ogg":
		return contentType == "audio/ogg"
	case ".webm":
		return contentType == "audio/webm"
	default:
		return false
	}
}

func contentTypesCompatible(claimed string, detected string) bool {
	claimed = normalizeContentType(claimed)
	detected = normalizeContentType(detected)
	if claimed == detected {
		return true
	}
	return equivalentContentTypes(claimed, detected)
}

func equivalentContentTypes(a string, b string) bool {
	return (a == "image/jpg" && b == "image/jpeg") ||
		(a == "image/jpeg" && b == "image/jpg") ||
		(a == "audio/mp3" && b == "audio/mpeg") ||
		(a == "audio/mpeg" && b == "audio/mp3") ||
		(a == "audio/x-wav" && b == "audio/wav") ||
		(a == "audio/wav" && b == "audio/x-wav")
}

func normalizeContentType(contentType string) string {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	return contentType
}
