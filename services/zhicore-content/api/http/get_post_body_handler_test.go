package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
)

func TestGetPostBodyMapsInvisibleAndBodyErrors(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "draft invisible", err: domain.ErrPostNotFound, wantStatus: http.StatusNotFound, wantCode: 4001},
		{name: "deleted invisible", err: domain.ErrPostNotFound, wantStatus: http.StatusNotFound, wantCode: 4001},
		{name: "body miss", err: domain.ErrBodyUnavailable, wantStatus: http.StatusInternalServerError, wantCode: 4018},
		{name: "hash conflict", err: domain.ErrBodyInconsistent, wantStatus: http.StatusConflict, wantCode: 4019},
		{name: "schema unreadable", err: application.ErrBodySchemaUnsupported, wantStatus: http.StatusInternalServerError, wantCode: 4024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeContentService{getBodyErr: tc.err}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_pub_1/body", nil)

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestGetPostBodyReturnsSuccessEnvelope(t *testing.T) {
	createdAt := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	blocks := json.RawMessage(`[{"type":"paragraph","children":[{"type":"text","text":"hello"}]}]`)
	canonical := json.RawMessage(`{"schemaVersion":1,"blocks":[{"type":"paragraph","children":[{"type":"text","text":"hello"}]}]}`)
	service := &fakeContentService{getBodyResult: application.GetPublishedPostBodyResult{
		BodyID:        "body_pub_1",
		SchemaVersion: 1,
		CanonicalJSON: canonical,
		PlainText:     "hello",
		ContentHash:   "sha256:body",
		SizeBytes:     len(canonical),
		CreatedAt:     createdAt,
	}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_pub_1/body", nil)

	NewHandler(service).ServeHTTP(rr, req)

	if service.getBodyQuery.PostID != "post_pub_1" {
		t.Fatalf("query = %#v", service.getBodyQuery)
	}
	var body envelope[postBodyResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Timestamp <= 0 {
		t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
	}
	if body.Data.BodyID != "body_pub_1" || body.Data.SchemaVersion != 1 || body.Data.Format != "blocks" ||
		body.Data.PlainText != "hello" || body.Data.ContentHash != "sha256:body" || body.Data.SizeBytes != len(canonical) ||
		body.Data.CreatedAt != "2026-07-05T10:00:00Z" {
		t.Fatalf("data = %#v", body.Data)
	}
	if string(body.Data.Blocks) != string(blocks) {
		t.Fatalf("blocks = %s, want %s", body.Data.Blocks, blocks)
	}
}

func TestGetPostBodyRejectsMalformedCanonicalBody(t *testing.T) {
	service := &fakeContentService{getBodyResult: application.GetPublishedPostBodyResult{
		BodyID:        "body_pub_1",
		SchemaVersion: 1,
		CanonicalJSON: json.RawMessage(`{"schemaVersion":1}`),
		PlainText:     "hello",
		ContentHash:   "sha256:body",
		SizeBytes:     19,
		CreatedAt:     time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC),
	}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_pub_1/body", nil)

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusInternalServerError, 4024)
}
