package clients

import (
	"context"
	"errors"
	"net/http"
	"time"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	"github.com/architectcgz/zhicore-go/libs/kit/httpclient"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

const callerServiceNotification = "zhicore-notification"

type UserFollowerClientConfig struct {
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

type UserFollowerClient struct {
	client *httpclient.Client
}

func NewUserFollowerClient(config UserFollowerClientConfig) *UserFollowerClient {
	return &UserFollowerClient{
		client: httpclient.New(httpclient.Config{
			BaseURL:       config.BaseURL,
			CallerService: callerServiceNotification,
			HTTPClient:    config.HTTPClient,
			Timeout:       config.Timeout,
		}),
	}
}

func (c *UserFollowerClient) ListFollowerShard(ctx context.Context, input ports.ListFollowerShardInput) (ports.FollowerShardPage, error) {
	req := usercontract.ListFollowerShardRequest{
		FollowingID:   input.FollowingID,
		AudienceClass: input.AudienceClass,
		Cursor:        input.Cursor,
		Limit:         input.Limit,
	}
	if input.ActiveSince != nil {
		req.ActiveSince = input.ActiveSince.UTC().Format(time.RFC3339)
	}
	var resp usercontract.ListFollowerShardResponse
	err := c.client.DoJSON(ctx, http.MethodPost, usercontract.ListFollowerShardPath, usercontract.OperationNotificationListFollowerShard, req, &resp)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return ports.FollowerShardPage{}, err
		}
		return ports.FollowerShardPage{}, ports.ErrDependencyUnavailable
	}
	return ports.FollowerShardPage{
		FollowerIDs: resp.FollowerIDs,
		NextCursor:  resp.NextCursor,
		HasMore:     resp.HasMore,
	}, nil
}
