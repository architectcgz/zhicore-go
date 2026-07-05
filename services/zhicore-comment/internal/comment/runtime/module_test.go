package runtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	commenthttp "github.com/architectcgz/zhicore-go/services/zhicore-comment/api/http"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/application"
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
	if module.HTTPHandler == nil || module.LiveHandler == nil || module.ReadyHandler == nil {
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

type stubWorker struct{}

func (stubWorker) Run(context.Context) error { return nil }

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
