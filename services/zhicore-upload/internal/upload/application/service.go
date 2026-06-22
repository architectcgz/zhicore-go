package application

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-upload/internal/upload/ports"
)

const (
	defaultMaxImageSize = 50 * 1024 * 1024
	defaultMaxAudioSize = 10 * 1024 * 1024
)

type Config struct {
	MaxImageSize     int64
	MaxAudioSize     int64
	AllowedImageType []string
	AllowedAudioType []string
}

func DefaultConfig() Config {
	return Config{
		MaxImageSize: defaultMaxImageSize,
		MaxAudioSize: defaultMaxAudioSize,
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
	return &Service{files: files, config: config}
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
	results := make([]ports.UploadResult, 0, len(files))
	for _, file := range files {
		result, err := s.UploadImage(ctx, file, accessLevel)
		if err != nil {
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) GetFileURL(ctx context.Context, fileID string) (string, error) {
	if strings.TrimSpace(fileID) == "" {
		return "", errorf(http.StatusBadRequest, "文件ID不能为空")
	}
	return s.files.GetFileURL(ctx, fileID)
}

func (s *Service) DeleteFile(ctx context.Context, fileID string) error {
	if strings.TrimSpace(fileID) == "" {
		return errorf(http.StatusBadRequest, "文件ID不能为空")
	}
	return s.files.DeleteFile(ctx, fileID)
}

func validateAccessLevel(accessLevel ports.AccessLevel) error {
	switch accessLevel {
	case ports.AccessLevelPublic, ports.AccessLevelPrivate:
		return nil
	default:
		return errorf(http.StatusBadRequest, "访问级别必须是 PUBLIC 或 PRIVATE")
	}
}

func validateFile(file ports.FilePayload, allowedTypes []string, maxSize int64) error {
	if file.Size <= 0 || file.Open == nil {
		return errorf(http.StatusBadRequest, "文件不能为空")
	}
	if !containsContentType(file.ContentType, allowedTypes) {
		return errorf(http.StatusBadRequest, fmt.Sprintf("文件类型不允许: %s", file.ContentType))
	}
	if file.Size > maxSize {
		return errorf(http.StatusBadRequest, "文件大小超过限制")
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
