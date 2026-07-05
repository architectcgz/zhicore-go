package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestSaveDraftBodyRequiresAuthorAndMapsForbidden(t *testing.T) {
	t.Run("missing trusted user", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_pub_1/draft/body", bytes.NewBufferString(validSaveDraftBodyJSON())))

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
		if service.saveCalls != 0 {
			t.Fatalf("saveCalls = %d, want 0", service.saveCalls)
		}
	})

	t.Run("non owner", func(t *testing.T) {
		service := &fakeContentService{saveErr: domain.ErrForbidden}
		rr := httptest.NewRecorder()
		req := withUserID(withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_pub_1/draft/body", bytes.NewBufferString(validSaveDraftBodyJSON()))), "42")

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusForbidden, 2008)
	})
}

func TestSaveDraftBodyMapsVersionConflict(t *testing.T) {
	service := &fakeContentService{saveErr: domain.ErrDraftConflict}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_pub_1/draft/body", bytes.NewBufferString(validSaveDraftBodyJSON()))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusConflict, 4017)
}

func TestSaveDraftBodyMapsBodySchemaErrorAndTooLarge(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantCode   int
		wantStatus int
		detailCode string
	}{
		{
			name:       "schema",
			err:        &ports.BodyValidationError{Details: []ports.ValidationDetail{{Path: "blocks", Code: "BODY_SCHEMA_INVALID"}}},
			wantCode:   4013,
			wantStatus: http.StatusBadRequest,
			detailCode: "BODY_SCHEMA_INVALID",
		},
		{
			name:       "body too large",
			err:        &ports.BodyValidationError{Details: []ports.ValidationDetail{{Path: "blocks", Code: "BODY_TOO_LARGE"}}},
			wantCode:   4015,
			wantStatus: http.StatusBadRequest,
			detailCode: "BODY_TOO_LARGE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeContentService{saveErr: tc.err}
			rr := httptest.NewRecorder()
			req := withUserID(withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_pub_1/draft/body", bytes.NewBufferString(validSaveDraftBodyJSON()))), "42")

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
			assertErrorDetail(t, rr, "blocks", tc.detailCode)
		})
	}
}

func TestSaveDraftBodyRejectsRequestBodyOverLimit(t *testing.T) {
	service := &fakeContentService{}
	rr := httptest.NewRecorder()
	oversized := `{"basePostVersion":1,"schemaVersion":1,"blocks":[{"type":"code_block","code":"` +
		strings.Repeat("x", maxJSONRequestBodyBytes) + `"}]}`
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_pub_1/draft/body", strings.NewReader(oversized))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusRequestEntityTooLarge, 4015)
	if service.saveCalls != 0 {
		t.Fatalf("saveCalls = %d, want 0", service.saveCalls)
	}
}

func TestSaveDraftBodyPropagatesRequestContextCancellation(t *testing.T) {
	service := &fakeContentService{saveErr: context.Canceled}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_pub_1/draft/body", bytes.NewBufferString(validSaveDraftBodyJSON())).WithContext(ctx)), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if !isContextCanceled(service.saveCtxErr) && !isContextCanceled(service.saveErr) {
		t.Fatalf("context cancellation was not observable")
	}
	assertErrorEnvelope(t, rr, http.StatusServiceUnavailable, 1004)
}

func TestSaveDraftBodyReturnsSuccessEnvelope(t *testing.T) {
	savedAt := time.Date(2026, 7, 5, 8, 30, 0, 0, time.UTC)
	service := &fakeContentService{saveResult: application.SaveDraftBodyResult{
		PostID:        "post_pub_1",
		PostVersion:   3,
		DraftBodyID:   "body_draft_2",
		DraftBodyHash: "sha256:abc",
		SavedAt:       savedAt,
		WordCount:     2,
	}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_pub_1/draft/body", bytes.NewBufferString(validSaveDraftBodyJSON()))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.saveCmd.Actor == nil || service.saveCmd.Actor.UserID != 42 || service.saveCmd.PostID != "post_pub_1" {
		t.Fatalf("save command = %#v", service.saveCmd)
	}
	if service.saveCmd.BasePostVersion != 1 || service.saveCmd.BaseDraftBodyID != "body_draft_1" || service.saveCmd.BaseDraftBodyHash != "sha256:old" {
		t.Fatalf("save base fields = %#v", service.saveCmd)
	}
	var body envelope[saveDraftBodyResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Timestamp <= 0 {
		t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
	}
	if body.Data.PostID != "post_pub_1" || body.Data.PostVersion != 3 || body.Data.DraftBodyID != "body_draft_2" ||
		body.Data.DraftBodyHash != "sha256:abc" || body.Data.SavedAt != "2026-07-05T08:30:00Z" || body.Data.WordCount != 2 {
		t.Fatalf("data = %#v", body.Data)
	}
}

func validSaveDraftBodyJSON() string {
	return `{"basePostVersion":1,"baseDraftBodyId":"body_draft_1","baseDraftBodyHash":"sha256:old","schemaVersion":1,"blocks":[{"type":"paragraph","children":[{"type":"text","text":"hello world"}]}]}`
}
