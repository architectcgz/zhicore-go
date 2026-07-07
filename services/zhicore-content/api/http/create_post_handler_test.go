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

	unpublishCalls  int
	unpublishCmd    application.PostLifecycleCommand
	unpublishResult application.PostLifecycleResult
	unpublishErr    error

	deletePostCalls  int
	deletePostCmd    application.PostLifecycleCommand
	deletePostResult application.PostLifecycleResult
	deletePostErr    error

	restoreCalls  int
	restoreCmd    application.PostLifecycleCommand
	restoreResult application.PostLifecycleResult
	restoreErr    error

	scheduleCalls  int
	scheduleCmd    application.SchedulePostCommand
	scheduleResult application.SchedulePostResult
	scheduleErr    error

	cancelScheduleCalls  int
	cancelScheduleCmd    application.PostLifecycleCommand
	cancelScheduleResult application.PostLifecycleResult
	cancelScheduleErr    error

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

	listAuthorPostsCalls  int
	listAuthorPostsQuery  application.ListAuthorPostsQuery
	listAuthorPostsResult application.AuthorPostPageResult
	listAuthorPostsErr    error

	listAuthorDraftsCalls  int
	listAuthorDraftsQuery  application.ListAuthorDraftsQuery
	listAuthorDraftsResult application.AuthorPostPageResult
	listAuthorDraftsErr    error

	getAuthorDraftCalls  int
	getAuthorDraftQuery  application.GetAuthorDraftQuery
	getAuthorDraftResult application.AuthorDraftResult
	getAuthorDraftErr    error

	updateDraftMetaCalls   int
	updateDraftMetaCommand application.UpdateDraftMetaCommand
	updateDraftMetaResult  application.DraftMutationResult
	updateDraftMetaErr     error

	deleteDraftCalls   int
	deleteDraftCommand application.DeleteAuthorDraftCommand
	deleteDraftResult  application.DraftMutationResult
	deleteDraftErr     error

	listOutboxCalls  int
	listOutboxQuery  application.ListAdminOutboxEventsQuery
	listOutboxResult application.ListAdminOutboxEventsResult
	listOutboxErr    error

	retryOutboxCalls   int
	retryOutboxCommand application.RetryAdminOutboxEventCommand
	retryOutboxResult  application.RetryAdminOutboxEventResult
	retryOutboxErr     error

	listTagsCalls  int
	listTagsQuery  application.ListTagsQuery
	listTagsResult application.TagPageResult
	listTagsErr    error

	getTagCalls  int
	getTagQuery  application.GetTagQuery
	getTagResult application.Tag
	getTagErr    error

	searchTagsCalls  int
	searchTagsQuery  application.SearchTagsQuery
	searchTagsResult []application.Tag
	searchTagsErr    error

	listHotTagsCalls  int
	listHotTagsQuery  application.ListHotTagsQuery
	listHotTagsResult []application.Tag
	listHotTagsErr    error

	listPostsByTagCalls  int
	listPostsByTagQuery  application.ListPostsByTagQuery
	listPostsByTagResult application.ListPublishedPostsResult
	listPostsByTagErr    error

	getPostTagsCalls  int
	getPostTagsQuery  application.GetPostTagsQuery
	getPostTagsResult []application.Tag
	getPostTagsErr    error

	updatePostTagsCalls  int
	updatePostTagsCmd    application.UpdatePostTagsCommand
	updatePostTagsResult application.PostTagsMutationResult
	updatePostTagsErr    error

	deletePostTagCalls  int
	deletePostTagCmd    application.DeletePostTagCommand
	deletePostTagResult application.PostTagsMutationResult
	deletePostTagErr    error

	likePostCalls  int
	likePostCmd    application.EngagementCommand
	likePostResult application.EngagementResult
	likePostErr    error

	unlikePostCalls  int
	unlikePostCmd    application.EngagementCommand
	unlikePostResult application.EngagementResult
	unlikePostErr    error

	favoritePostCalls  int
	favoritePostCmd    application.EngagementCommand
	favoritePostResult application.EngagementResult
	favoritePostErr    error

	unfavoritePostCalls  int
	unfavoritePostCmd    application.EngagementCommand
	unfavoritePostResult application.EngagementResult
	unfavoritePostErr    error

	getEngagementCalls  int
	getEngagementQuery  application.GetPostEngagementQuery
	getEngagementResult application.PostEngagementResult
	getEngagementErr    error

	batchEngagementCalls  int
	batchEngagementQuery  application.BatchGetEngagementStatusQuery
	batchEngagementResult application.BatchEngagementStatusResult
	batchEngagementErr    error
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

func (f *fakeContentService) UnpublishPost(ctx context.Context, cmd application.PostLifecycleCommand) (application.PostLifecycleResult, error) {
	f.unpublishCalls++
	f.unpublishCmd = cmd
	if f.unpublishErr != nil {
		return application.PostLifecycleResult{}, f.unpublishErr
	}
	return f.unpublishResult, nil
}

func (f *fakeContentService) DeletePost(ctx context.Context, cmd application.PostLifecycleCommand) (application.PostLifecycleResult, error) {
	f.deletePostCalls++
	f.deletePostCmd = cmd
	if f.deletePostErr != nil {
		return application.PostLifecycleResult{}, f.deletePostErr
	}
	return f.deletePostResult, nil
}

func (f *fakeContentService) RestorePost(ctx context.Context, cmd application.PostLifecycleCommand) (application.PostLifecycleResult, error) {
	f.restoreCalls++
	f.restoreCmd = cmd
	if f.restoreErr != nil {
		return application.PostLifecycleResult{}, f.restoreErr
	}
	return f.restoreResult, nil
}

func (f *fakeContentService) SchedulePost(ctx context.Context, cmd application.SchedulePostCommand) (application.SchedulePostResult, error) {
	f.scheduleCalls++
	f.scheduleCmd = cmd
	if f.scheduleErr != nil {
		return application.SchedulePostResult{}, f.scheduleErr
	}
	return f.scheduleResult, nil
}

func (f *fakeContentService) CancelSchedule(ctx context.Context, cmd application.PostLifecycleCommand) (application.PostLifecycleResult, error) {
	f.cancelScheduleCalls++
	f.cancelScheduleCmd = cmd
	if f.cancelScheduleErr != nil {
		return application.PostLifecycleResult{}, f.cancelScheduleErr
	}
	return f.cancelScheduleResult, nil
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

func (f *fakeContentService) ListAuthorPosts(ctx context.Context, query application.ListAuthorPostsQuery) (application.AuthorPostPageResult, error) {
	f.listAuthorPostsCalls++
	f.listAuthorPostsQuery = query
	if f.listAuthorPostsErr != nil {
		return application.AuthorPostPageResult{}, f.listAuthorPostsErr
	}
	return f.listAuthorPostsResult, nil
}

func (f *fakeContentService) ListAuthorDrafts(ctx context.Context, query application.ListAuthorDraftsQuery) (application.AuthorPostPageResult, error) {
	f.listAuthorDraftsCalls++
	f.listAuthorDraftsQuery = query
	if f.listAuthorDraftsErr != nil {
		return application.AuthorPostPageResult{}, f.listAuthorDraftsErr
	}
	return f.listAuthorDraftsResult, nil
}

func (f *fakeContentService) GetAuthorDraft(ctx context.Context, query application.GetAuthorDraftQuery) (application.AuthorDraftResult, error) {
	f.getAuthorDraftCalls++
	f.getAuthorDraftQuery = query
	if f.getAuthorDraftErr != nil {
		return application.AuthorDraftResult{}, f.getAuthorDraftErr
	}
	return f.getAuthorDraftResult, nil
}

func (f *fakeContentService) UpdateDraftMeta(ctx context.Context, command application.UpdateDraftMetaCommand) (application.DraftMutationResult, error) {
	f.updateDraftMetaCalls++
	f.updateDraftMetaCommand = command
	if f.updateDraftMetaErr != nil {
		return application.DraftMutationResult{}, f.updateDraftMetaErr
	}
	return f.updateDraftMetaResult, nil
}

func (f *fakeContentService) DeleteAuthorDraft(ctx context.Context, command application.DeleteAuthorDraftCommand) (application.DraftMutationResult, error) {
	f.deleteDraftCalls++
	f.deleteDraftCommand = command
	if f.deleteDraftErr != nil {
		return application.DraftMutationResult{}, f.deleteDraftErr
	}
	return f.deleteDraftResult, nil
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

func (f *fakeContentService) ListTags(ctx context.Context, query application.ListTagsQuery) (application.TagPageResult, error) {
	f.listTagsCalls++
	f.listTagsQuery = query
	if f.listTagsErr != nil {
		return application.TagPageResult{}, f.listTagsErr
	}
	return f.listTagsResult, nil
}

func (f *fakeContentService) GetTag(ctx context.Context, query application.GetTagQuery) (application.Tag, error) {
	f.getTagCalls++
	f.getTagQuery = query
	if f.getTagErr != nil {
		return application.Tag{}, f.getTagErr
	}
	return f.getTagResult, nil
}

func (f *fakeContentService) SearchTags(ctx context.Context, query application.SearchTagsQuery) ([]application.Tag, error) {
	f.searchTagsCalls++
	f.searchTagsQuery = query
	if f.searchTagsErr != nil {
		return nil, f.searchTagsErr
	}
	return append([]application.Tag(nil), f.searchTagsResult...), nil
}

func (f *fakeContentService) ListHotTags(ctx context.Context, query application.ListHotTagsQuery) ([]application.Tag, error) {
	f.listHotTagsCalls++
	f.listHotTagsQuery = query
	if f.listHotTagsErr != nil {
		return nil, f.listHotTagsErr
	}
	return append([]application.Tag(nil), f.listHotTagsResult...), nil
}

func (f *fakeContentService) ListPostsByTag(ctx context.Context, query application.ListPostsByTagQuery) (application.ListPublishedPostsResult, error) {
	f.listPostsByTagCalls++
	f.listPostsByTagQuery = query
	if f.listPostsByTagErr != nil {
		return application.ListPublishedPostsResult{}, f.listPostsByTagErr
	}
	return f.listPostsByTagResult, nil
}

func (f *fakeContentService) GetPostTags(ctx context.Context, query application.GetPostTagsQuery) ([]application.Tag, error) {
	f.getPostTagsCalls++
	f.getPostTagsQuery = query
	if f.getPostTagsErr != nil {
		return nil, f.getPostTagsErr
	}
	return append([]application.Tag(nil), f.getPostTagsResult...), nil
}

func (f *fakeContentService) UpdatePostTags(ctx context.Context, command application.UpdatePostTagsCommand) (application.PostTagsMutationResult, error) {
	f.updatePostTagsCalls++
	f.updatePostTagsCmd = command
	if f.updatePostTagsErr != nil {
		return application.PostTagsMutationResult{}, f.updatePostTagsErr
	}
	return f.updatePostTagsResult, nil
}

func (f *fakeContentService) DeletePostTag(ctx context.Context, command application.DeletePostTagCommand) (application.PostTagsMutationResult, error) {
	f.deletePostTagCalls++
	f.deletePostTagCmd = command
	if f.deletePostTagErr != nil {
		return application.PostTagsMutationResult{}, f.deletePostTagErr
	}
	return f.deletePostTagResult, nil
}

func (f *fakeContentService) LikePost(ctx context.Context, command application.EngagementCommand) (application.EngagementResult, error) {
	f.likePostCalls++
	f.likePostCmd = command
	if f.likePostErr != nil {
		return application.EngagementResult{}, f.likePostErr
	}
	return f.likePostResult, nil
}

func (f *fakeContentService) UnlikePost(ctx context.Context, command application.EngagementCommand) (application.EngagementResult, error) {
	f.unlikePostCalls++
	f.unlikePostCmd = command
	if f.unlikePostErr != nil {
		return application.EngagementResult{}, f.unlikePostErr
	}
	return f.unlikePostResult, nil
}

func (f *fakeContentService) FavoritePost(ctx context.Context, command application.EngagementCommand) (application.EngagementResult, error) {
	f.favoritePostCalls++
	f.favoritePostCmd = command
	if f.favoritePostErr != nil {
		return application.EngagementResult{}, f.favoritePostErr
	}
	return f.favoritePostResult, nil
}

func (f *fakeContentService) UnfavoritePost(ctx context.Context, command application.EngagementCommand) (application.EngagementResult, error) {
	f.unfavoritePostCalls++
	f.unfavoritePostCmd = command
	if f.unfavoritePostErr != nil {
		return application.EngagementResult{}, f.unfavoritePostErr
	}
	return f.unfavoritePostResult, nil
}

func (f *fakeContentService) GetPostEngagement(ctx context.Context, query application.GetPostEngagementQuery) (application.PostEngagementResult, error) {
	f.getEngagementCalls++
	f.getEngagementQuery = query
	if f.getEngagementErr != nil {
		return application.PostEngagementResult{}, f.getEngagementErr
	}
	return f.getEngagementResult, nil
}

func (f *fakeContentService) BatchGetEngagementStatus(ctx context.Context, query application.BatchGetEngagementStatusQuery) (application.BatchEngagementStatusResult, error) {
	f.batchEngagementCalls++
	f.batchEngagementQuery = query
	if f.batchEngagementErr != nil {
		return application.BatchEngagementStatusResult{}, f.batchEngagementErr
	}
	return f.batchEngagementResult, nil
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
