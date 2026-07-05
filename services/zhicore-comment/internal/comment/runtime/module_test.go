package runtime

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	commenthttp "github.com/architectcgz/zhicore-go/services/zhicore-comment/api/http"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/application"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestBuildRejectsMissingServiceDependency(t *testing.T) {
	_, err := Build(Deps{})
	if err == nil || !strings.Contains(err.Error(), "Service") {
		t.Fatalf("Build() error = %v, want mention Service", err)
	}
}

func TestBuildReturnsCommentHealthHandlersAndWorkers(t *testing.T) {
	worker := stubWorker{}
	module, err := Build(Deps{Service: stubService{}, Workers: []Worker{worker}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if module.HTTPHandler == nil {
		t.Fatalf("module handlers = %#v", module)
	}
	if len(module.Workers) != 1 {
		t.Fatalf("workers = %d, want 1", len(module.Workers))
	}

	for _, path := range []string{"/health/live", "/health/ready"} {
		rec := httptest.NewRecorder()
		module.HTTPHandler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}
	}
}

func TestBuildRejectsIncompleteOutboxRuntimeDependencies(t *testing.T) {
	_, err := Build(Deps{
		Service: stubService{},
		Outbox:  OutboxConfig{Enabled: true, DispatcherID: "zhicore-comment:outbox-dispatcher:test"},
	})
	if err == nil || !strings.Contains(err.Error(), "PostgresDB") {
		t.Fatalf("Build() error = %v, want mention PostgresDB", err)
	}

	db, _ := newMockDB(t)
	_, err = Build(Deps{
		Service:    stubService{},
		PostgresDB: db,
		Outbox:     OutboxConfig{Enabled: true, DispatcherID: "zhicore-comment:outbox-dispatcher:test"},
	})
	if err == nil || !strings.Contains(err.Error(), "RabbitMQChannel") {
		t.Fatalf("Build() error = %v, want mention RabbitMQChannel", err)
	}

	_, err = Build(Deps{
		Service:         stubService{},
		PostgresDB:      db,
		RabbitMQChannel: &stubAMQPChannel{},
		Outbox:          OutboxConfig{Enabled: true},
	})
	if err == nil || !strings.Contains(err.Error(), "DispatcherID") {
		t.Fatalf("Build() error = %v, want mention DispatcherID", err)
	}
}

func TestBuildWiresOutboxDispatcherWorker(t *testing.T) {
	db, mock := newMockDB(t)
	module, err := Build(Deps{
		Service:         stubService{},
		PostgresDB:      db,
		RabbitMQChannel: &stubAMQPChannel{},
		Clock:           fixedClock{now: time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)},
		Outbox: OutboxConfig{
			Enabled:      true,
			DispatcherID: "zhicore-comment:outbox-dispatcher:test",
			BatchSize:    10,
			PollInterval: time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(module.Workers) != 1 {
		t.Fatalf("workers = %d, want outbox worker", len(module.Workers))
	}

	mock.ExpectQuery("WITH picked AS").WillReturnRows(sqlmock.NewRows([]string{
		"id", "event_id", "event_type", "payload_version", "aggregate_type", "aggregate_id", "payload_json", "occurred_at", "attempt_count",
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err = module.Workers[0].Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("outbox worker Run() error = %v, want context deadline exceeded", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

type stubWorker struct{}

func (stubWorker) Run(context.Context) error { return nil }

type stubAMQPChannel struct{}

func (*stubAMQPChannel) PublishWithContext(context.Context, string, string, bool, bool, amqp.Publishing) error {
	return nil
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

type stubService struct{}

func (stubService) CreateComment(context.Context, application.CreateCommentCommand) (application.CreateCommentResult, error) {
	return application.CreateCommentResult{}, errors.New("not implemented")
}

func (stubService) ListTopLevelCommentsByPage(context.Context, application.ListTopLevelCommentsQuery) (application.TopLevelCommentPage, error) {
	return application.TopLevelCommentPage{}, errors.New("not implemented")
}

func (stubService) GetCommentDetail(context.Context, application.GetCommentDetailQuery) (application.CommentItem, error) {
	return application.CommentItem{}, errors.New("not implemented")
}

func (stubService) ListRepliesByPage(context.Context, application.ListRepliesByPageQuery) (application.CommentPage, error) {
	return application.CommentPage{}, errors.New("not implemented")
}

func (stubService) DeleteComment(context.Context, application.DeleteCommentCommand) (application.DeleteCommentResult, error) {
	return application.DeleteCommentResult{}, errors.New("not implemented")
}

func (stubService) AdminDeleteComment(context.Context, application.AdminDeleteCommentCommand) (application.DeleteCommentResult, error) {
	return application.DeleteCommentResult{}, errors.New("not implemented")
}

func (stubService) LikeComment(context.Context, application.LikeCommentCommand) (application.LikeCommentResult, error) {
	return application.LikeCommentResult{}, errors.New("not implemented")
}

func (stubService) UnlikeComment(context.Context, application.UnlikeCommentCommand) (application.LikeCommentResult, error) {
	return application.LikeCommentResult{}, errors.New("not implemented")
}

func (stubService) GetLikeStatus(context.Context, application.GetLikeStatusQuery) (application.LikeStatusResult, error) {
	return application.LikeStatusResult{PostID: "post_pub_1", CommentID: "c1", Liked: true}, nil
}

var _ commenthttp.Service = stubService{}
