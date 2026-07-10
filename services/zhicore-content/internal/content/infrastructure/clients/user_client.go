package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	"github.com/architectcgz/zhicore-go/libs/kit/httpclient"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const callerServiceContent = "zhicore-content"

type UserClientConfig struct {
	BaseURL     string
	HTTPClient  *http.Client
	Timeout     time.Duration
	MaxAttempts int
}

type UserClient struct {
	client      *httpclient.Client
	maxAttempts int
}

func NewUserClient(config UserClientConfig) *UserClient {
	maxAttempts := config.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	return &UserClient{
		client: httpclient.New(httpclient.Config{
			BaseURL:       config.BaseURL,
			CallerService: callerServiceContent,
			HTTPClient:    config.HTTPClient,
			Timeout:       config.Timeout,
		}),
		maxAttempts: maxAttempts,
	}
}

func (c *UserClient) GetOwnerSnapshot(ctx context.Context, userID int64) (ports.OwnerSnapshot, error) {
	var lastErr error
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		var payload usercontract.SimpleBatchResponse
		err := c.client.DoJSON(
			ctx,
			http.MethodPost,
			usercontract.BatchSimplePath,
			usercontract.OperationContentGetOwnerSnapshot,
			usercontract.IDsRequest{UserIDs: []int64{userID}},
			&payload,
		)
		if err != nil {
			mapped := mapDependencyError(err, "get owner snapshot")
			if !shouldRetryUserSnapshot(err) {
				return ports.OwnerSnapshot{}, mapped
			}
			lastErr = mapped
			continue
		}

		for _, item := range payload.Items {
			if item.UserID != userID {
				continue
			}
			return ports.OwnerSnapshot{
				PublicID:       item.PublicID,
				DisplayName:    item.Nickname,
				AvatarFileID:   item.AvatarFileID,
				AvatarURL:      item.AvatarURL,
				ProfileVersion: item.ProfileVersion,
			}, nil
		}
		return ports.OwnerSnapshot{}, fmt.Errorf("%w: user snapshot missing", ports.ErrDependencyUnavailable)
	}
	if lastErr != nil {
		return ports.OwnerSnapshot{}, lastErr
	}
	return ports.OwnerSnapshot{}, fmt.Errorf("%w: user snapshot missing", ports.ErrDependencyUnavailable)
}

func shouldRetryUserSnapshot(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var providerErr httpclient.ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.StatusCode >= http.StatusInternalServerError || providerErr.Code == 1004
	}
	return true
}

func mapDependencyError(err error, operation string) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %s: %w", ports.ErrDependencyUnavailable, operation, err)
	}
	return fmt.Errorf("%w: %s", ports.ErrDependencyUnavailable, operation)
}

var _ ports.UserProfileClient = (*UserClient)(nil)
