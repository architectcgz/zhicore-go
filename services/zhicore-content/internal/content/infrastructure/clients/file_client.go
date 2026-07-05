package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	filecontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/file"
	"github.com/architectcgz/zhicore-go/libs/kit/httpclient"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type FileClientConfig struct {
	BaseURL     string
	HTTPClient  *http.Client
	Timeout     time.Duration
	MaxAttempts int
}

type FileClient struct {
	client      *httpclient.Client
	maxAttempts int
}

func NewFileClient(config FileClientConfig) *FileClient {
	maxAttempts := config.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	return &FileClient{
		client: httpclient.New(httpclient.Config{
			BaseURL:       config.BaseURL,
			CallerService: callerServiceContent,
			HTTPClient:    config.HTTPClient,
			Timeout:       config.Timeout,
		}),
		maxAttempts: maxAttempts,
	}
}

func (c *FileClient) ValidateBodyMediaRefs(ctx context.Context, refs []ports.MediaRef) error {
	fileRefs := make([]filecontract.FileRef, 0, len(refs))
	for _, ref := range refs {
		fileID := strings.TrimSpace(ref.FileID)
		if fileID == "" {
			return ports.ErrMediaRefInvalid
		}
		fileRefs = append(fileRefs, filecontract.FileRef{FileID: fileID})
	}
	if len(fileRefs) == 0 {
		return nil
	}
	return c.validateRefs(
		ctx,
		filecontract.OperationContentValidateBodyMediaRefs,
		filecontract.UsageContentBodyMedia,
		fileRefs,
		ports.ErrMediaRefInvalid,
	)
}

func (c *FileClient) ValidateCoverFile(ctx context.Context, fileID string) error {
	trimmed := strings.TrimSpace(fileID)
	if trimmed == "" {
		return ports.ErrCoverUnavailable
	}
	return c.validateRefs(
		ctx,
		filecontract.OperationContentValidateCover,
		filecontract.UsageContentCover,
		[]filecontract.FileRef{{FileID: trimmed}},
		ports.ErrCoverUnavailable,
	)
}

func (c *FileClient) validateRefs(ctx context.Context, operation string, usage string, refs []filecontract.FileRef, semantic error) error {
	request := filecontract.ValidateRefsRequest{Refs: refs, Usage: usage}
	var lastErr error
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		var response filecontract.ValidateRefsResponse
		err := c.client.DoJSON(ctx, http.MethodPost, filecontract.ValidateRefsPath, operation, request, &response)
		if err == nil {
			if len(response.InvalidFileIDs) > 0 {
				return semantic
			}
			return nil
		}
		mapped := mapFileProviderError(err, semantic)
		if errors.Is(mapped, semantic) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return mapped
		}
		lastErr = mapped
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("%w: validate file references", ports.ErrDependencyUnavailable)
	}
	return lastErr
}

func mapFileProviderError(err error, semantic error) error {
	var providerErr httpclient.ProviderError
	if errors.As(err, &providerErr) {
		if providerErr.StatusCode == http.StatusNotFound || providerErr.StatusCode == http.StatusGone {
			return semantic
		}
		if providerErr.StatusCode >= http.StatusInternalServerError || providerErr.Code == 1004 {
			return fmt.Errorf("%w: file provider unavailable", ports.ErrDependencyUnavailable)
		}
		return fmt.Errorf("%w: file provider rejected request", ports.ErrDependencyUnavailable)
	}
	return mapDependencyError(err, "validate file references")
}

var _ ports.FileResourceClient = (*FileClient)(nil)
