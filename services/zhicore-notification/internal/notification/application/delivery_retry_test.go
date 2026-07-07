package application

import (
	"context"
	"errors"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestRetryDeliveryDecodesPublicIDAndAllowsOwnerOrAdminOnly(t *testing.T) {
	deps := newDeliveryRetryDeps()
	deps.ids.decoded = 9001
	deps.deliveries.retryResult = ports.DeliveryRetryResult{PublicID: "d1retry", RecipientID: 42, Status: "WEBSOCKET_PENDING", Retried: true}
	service := mustNewService(t, deps.dependencies())

	if _, err := service.RetryDelivery(context.Background(), RetryDeliveryCommand{Actor: Actor{UserID: 42}, DeliveryID: "d1retry"}); err != nil {
		t.Fatalf("owner RetryDelivery() error = %v", err)
	}
	if deps.deliveries.lastRetry.DeliveryID != 9001 {
		t.Fatalf("decoded retry delivery id = %d, want 9001", deps.deliveries.lastRetry.DeliveryID)
	}
	if _, err := service.RetryDelivery(context.Background(), RetryDeliveryCommand{Actor: Actor{UserID: 7, Roles: []string{"admin"}}, DeliveryID: "d1retry"}); err != nil {
		t.Fatalf("admin RetryDelivery() error = %v", err)
	}

	deps.deliveries.retryResult = ports.DeliveryRetryResult{PublicID: "d1retry", RecipientID: 43, Status: "WEBSOCKET_PENDING", Retried: true}
	_, err := service.RetryDelivery(context.Background(), RetryDeliveryCommand{Actor: Actor{UserID: 42}, DeliveryID: "d1retry"})
	if !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("other user RetryDelivery() error = %v, want %v", err, ErrNotificationNotFound)
	}
}

func TestRetryDeliveryRejectsInvalidPublicIDBeforeRepositoryLookup(t *testing.T) {
	deps := newDeliveryRetryDeps()
	deps.ids.decodeErr = errors.New("bad checksum")
	service := mustNewService(t, deps.dependencies())

	_, err := service.RetryDelivery(context.Background(), RetryDeliveryCommand{Actor: Actor{UserID: 42}, DeliveryID: "bad-id"})

	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("RetryDelivery() error = %v, want %v", err, ErrInvalidRequest)
	}
	if deps.deliveries.retryCalls != 0 {
		t.Fatalf("retry calls = %d, want 0", deps.deliveries.retryCalls)
	}
}

type deliveryRetryDeps struct {
	readTestDeps
	deliveries *fakeDeliveryRepository
}

func newDeliveryRetryDeps() deliveryRetryDeps {
	readDeps := newReadTestDeps()
	return deliveryRetryDeps{
		readTestDeps: readDeps,
		deliveries:   &fakeDeliveryRepository{},
	}
}

func (d deliveryRetryDeps) dependencies() Dependencies {
	deps := d.readTestDeps.dependencies()
	deps.Deliveries = d.deliveries
	return deps
}

type fakeDeliveryRepository struct {
	retryResult ports.DeliveryRetryResult
	listResult  ports.DeliveryPage
	lastRetry   ports.RetryDeliveryInput
	retryCalls  int
}

func (f *fakeDeliveryRepository) ListDeliveries(ctx context.Context, query ports.ListDeliveriesQuery) (ports.DeliveryPage, error) {
	return f.listResult, nil
}

func (f *fakeDeliveryRepository) RetryDelivery(ctx context.Context, input ports.RetryDeliveryInput) (ports.DeliveryRetryResult, error) {
	f.retryCalls++
	f.lastRetry = input
	if f.retryResult.PublicID == "" {
		f.retryResult = ports.DeliveryRetryResult{PublicID: "d1retry", RecipientID: input.RequesterID, Status: "WEBSOCKET_PENDING", Retried: true}
	}
	return f.retryResult, nil
}
