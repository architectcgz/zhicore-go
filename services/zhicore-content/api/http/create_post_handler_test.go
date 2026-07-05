package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestCreatePostRequiresTrustedUserID(t *testing.T) {
	service := &fakeContentService{}
	rr := httptest.NewRecorder()
	req := withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBufferString(`{"title":"Draft"}`)))

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
	if service.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", service.createCalls)
	}
}

func TestCreatePostCreatesEmptyDraftAndUsesTrustedIdentity(t *testing.T) {
	service := &fakeContentService{createResult: application.CreatePostResult{PostID: "post_pub_1", PostVersion: 1}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(
		http.MethodPost,
		"/api/v1/posts",
		bytes.NewBufferString(`{"userId":999,"ownerId":999,"actor":{"userId":999},"title":"  Draft  ","tags":["go"]}`),
	)), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", service.createCalls)
	}
	if service.createCmd.Actor == nil || service.createCmd.Actor.UserID != 42 {
		t.Fatalf("actor = %#v, want trusted user 42", service.createCmd.Actor)
	}
	if service.createCmd.Title != "  Draft  " || service.createCmd.Body != nil || len(service.createCmd.Tags) != 1 || service.createCmd.Tags[0] != "go" {
		t.Fatalf("create command = %#v", service.createCmd)
	}

	var body envelope[createPostResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Message != "操作成功" || body.Timestamp <= 0 {
		t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
	}
	if body.Data.PostID != "post_pub_1" || body.Data.PostVersion != 1 {
		t.Fatalf("data = %#v", body.Data)
	}
}

func TestCreatePostCreatesDraftWithBody(t *testing.T) {
	service := &fakeContentService{createResult: application.CreatePostResult{PostID: "post_pub_2", PostVersion: 1}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBufferString(`{
		"title":"Body draft",
		"body":{"schemaVersion":1,"blocks":[{"type":"paragraph","children":[{"type":"text","text":"hello"}]}]}
	}`))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.createCmd.Body == nil {
		t.Fatal("create body = nil, want body forwarded")
	}
	if service.createCmd.Body.SchemaVersion != 1 || len(service.createCmd.Body.Blocks) != 1 {
		t.Fatalf("create body = %#v", service.createCmd.Body)
	}
	assertSuccessCode(t, rr)
}

func TestCreatePostMapsTitleTooLong(t *testing.T) {
	service := &fakeContentService{createErr: domain.ErrTitleTooLong}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBufferString(`{"title":"too long"}`))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusBadRequest, 4007)
}

func TestCreatePostMapsReferenceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "taxonomy reference missing", err: application.ErrTaxonomyReferenceNotFound, wantStatus: http.StatusNotFound, wantCode: 4012},
		{name: "media reference invalid", err: application.ErrMediaRefInvalid, wantStatus: http.StatusBadRequest, wantCode: 4021},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeContentService{createErr: tc.err}
			rr := httptest.NewRecorder()
			req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBufferString(`{"title":"draft"}`))), "42")

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestCreatePostMapsInvalidBodyDetails(t *testing.T) {
	service := &fakeContentService{createErr: &ports.BodyValidationError{Details: []ports.ValidationDetail{{Path: "blocks[0]", Code: "BODY_SCHEMA_INVALID"}}}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBufferString(`{
		"body":{"schemaVersion":1,"blocks":[{"type":"paragraph"}]}
	}`))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusBadRequest, 4013)
	assertErrorDetail(t, rr, "blocks[0]", "BODY_SCHEMA_INVALID")
}

type fakeContentService struct {
	createCalls  int
	createCmd    application.CreatePostCommand
	createResult application.CreatePostResult
	createErr    error

	saveCalls  int
	saveCmd    application.SaveDraftBodyCommand
	saveCtxErr error
	saveResult application.SaveDraftBodyResult
	saveErr    error

	publishCalls  int
	publishCmd    application.PublishPostCommand
	publishResult application.PublishPostResult
	publishErr    error

	getBodyCalls  int
	getBodyQuery  application.GetPublishedPostBodyQuery
	getBodyResult application.GetPublishedPostBodyResult
	getBodyErr    error

	listPublishedCalls  int
	listPublishedQuery  application.ListPublishedPostsQuery
	listPublishedResult application.ListPublishedPostsResult
	listPublishedErr    error

	getDetailCalls  int
	getDetailQuery  application.GetPostDetailQuery
	getDetailResult application.GetPostDetailResult
	getDetailErr    error

	batchCalls  int
	batchQuery  application.BatchGetPublishedPostsQuery
	batchResult application.BatchGetPublishedPostsResult
	batchErr    error

	listOutboxCalls  int
	listOutboxQuery  application.ListAdminOutboxEventsQuery
	listOutboxResult application.ListAdminOutboxEventsResult
	listOutboxErr    error

	retryOutboxCalls   int
	retryOutboxCommand application.RetryAdminOutboxEventCommand
	retryOutboxResult  application.RetryAdminOutboxEventResult
	retryOutboxErr     error
}

func (f *fakeContentService) CreatePost(ctx context.Context, cmd application.CreatePostCommand) (application.CreatePostResult, error) {
	f.createCalls++
	f.createCmd = cmd
	if f.createErr != nil {
		return application.CreatePostResult{}, f.createErr
	}
	return f.createResult, nil
}

func (f *fakeContentService) SaveDraftBody(ctx context.Context, cmd application.SaveDraftBodyCommand) (application.SaveDraftBodyResult, error) {
	f.saveCalls++
	f.saveCmd = cmd
	f.saveCtxErr = ctx.Err()
	if f.saveErr != nil {
		return application.SaveDraftBodyResult{}, f.saveErr
	}
	return f.saveResult, nil
}

func (f *fakeContentService) PublishPost(ctx context.Context, cmd application.PublishPostCommand) (application.PublishPostResult, error) {
	f.publishCalls++
	f.publishCmd = cmd
	if f.publishErr != nil {
		return application.PublishPostResult{}, f.publishErr
	}
	return f.publishResult, nil
}

func (f *fakeContentService) GetPublishedPostBody(ctx context.Context, query application.GetPublishedPostBodyQuery) (application.GetPublishedPostBodyResult, error) {
	f.getBodyCalls++
	f.getBodyQuery = query
	if f.getBodyErr != nil {
		return application.GetPublishedPostBodyResult{}, f.getBodyErr
	}
	return f.getBodyResult, nil
}

func (f *fakeContentService) ListPublishedPosts(ctx context.Context, query application.ListPublishedPostsQuery) (application.ListPublishedPostsResult, error) {
	f.listPublishedCalls++
	f.listPublishedQuery = query
	if f.listPublishedErr != nil {
		return application.ListPublishedPostsResult{}, f.listPublishedErr
	}
	return f.listPublishedResult, nil
}

func (f *fakeContentService) GetPostDetail(ctx context.Context, query application.GetPostDetailQuery) (application.GetPostDetailResult, error) {
	f.getDetailCalls++
	f.getDetailQuery = query
	if f.getDetailErr != nil {
		return application.GetPostDetailResult{}, f.getDetailErr
	}
	return f.getDetailResult, nil
}

func (f *fakeContentService) BatchGetPublishedPosts(ctx context.Context, query application.BatchGetPublishedPostsQuery) (application.BatchGetPublishedPostsResult, error) {
	f.batchCalls++
	f.batchQuery = query
	if f.batchErr != nil {
		return application.BatchGetPublishedPostsResult{}, f.batchErr
	}
	return f.batchResult, nil
}

func (f *fakeContentService) ListAdminOutboxEvents(ctx context.Context, query application.ListAdminOutboxEventsQuery) (application.ListAdminOutboxEventsResult, error) {
	f.listOutboxCalls++
	f.listOutboxQuery = query
	if f.listOutboxErr != nil {
		return application.ListAdminOutboxEventsResult{}, f.listOutboxErr
	}
	return f.listOutboxResult, nil
}

func (f *fakeContentService) RetryAdminOutboxEvent(ctx context.Context, command application.RetryAdminOutboxEventCommand) (application.RetryAdminOutboxEventResult, error) {
	f.retryOutboxCalls++
	f.retryOutboxCommand = command
	if f.retryOutboxErr != nil {
		return application.RetryAdminOutboxEventResult{}, f.retryOutboxErr
	}
	return f.retryOutboxResult, nil
}

type envelope[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

type errorData struct {
	Details []struct {
		Path string `json:"path"`
		Code string `json:"code"`
	} `json:"details"`
}

func withJSON(req *http.Request) *http.Request {
	req.Header.Set("Content-Type", "application/json")
	return req
}

func withUserID(req *http.Request, userID string) *http.Request {
	req.Header.Set("X-User-Id", userID)
	return req
}

func withRoles(req *http.Request, roles string) *http.Request {
	req.Header.Set("X-User-Roles", roles)
	return req
}

func decodeJSON(t *testing.T, payload []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; payload=%s", err, string(payload))
	}
}

func assertSuccessCode(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	var body envelope[json.RawMessage]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Timestamp <= 0 {
		t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
	}
}

func assertErrorEnvelope(t *testing.T, rr *httptest.ResponseRecorder, wantStatus, wantCode int) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var body envelope[json.RawMessage]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if body.Code != wantCode || body.Timestamp <= 0 {
		t.Fatalf("error envelope = %#v, want code=%d timestamp>0", body, wantCode)
	}
}

func assertErrorDetail(t *testing.T, rr *httptest.ResponseRecorder, wantPath, wantCode string) {
	t.Helper()
	var body envelope[errorData]
	decodeJSON(t, rr.Body.Bytes(), &body)
	for _, detail := range body.Data.Details {
		if detail.Path == wantPath && detail.Code == wantCode {
			return
		}
	}
	t.Fatalf("details = %#v, want path=%q code=%q", body.Data.Details, wantPath, wantCode)
}

func isContextCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}
