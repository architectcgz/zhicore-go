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
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

type UserClient struct {
	client *httpclient.Client
}

func NewUserClient(config UserClientConfig) *UserClient {
	return &UserClient{
		client: httpclient.New(httpclient.Config{
			BaseURL:       config.BaseURL,
			CallerService: callerServiceContent,
			HTTPClient:    config.HTTPClient,
			Timeout:       config.Timeout,
		}),
	}
}

func (c *UserClient) GetOwnerSnapshot(ctx context.Context, userID int64) (ports.OwnerSnapshot, error) {
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
		return ports.OwnerSnapshot{}, mapDependencyError(err, "get owner snapshot")
	}

	for _, item := range payload.Items {
		if item.UserID != userID {
			continue
		}
		return ports.OwnerSnapshot{
			DisplayName:    item.Nickname,
			AvatarFileID:   item.AvatarFileID,
			ProfileVersion: item.ProfileVersion,
		}, nil
	}
	return ports.OwnerSnapshot{}, fmt.Errorf("%w: user snapshot missing", ports.ErrDependencyUnavailable)
}

func mapDependencyError(err error, operation string) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %s: %w", ports.ErrDependencyUnavailable, operation, err)
	}
	return fmt.Errorf("%w: %s", ports.ErrDependencyUnavailable, operation)
}

var _ ports.UserProfileClient = (*UserClient)(nil)
