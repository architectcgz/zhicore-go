package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteSuccess(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteSuccess(rr, map[string]string{"id": "123"})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := decodeResponseBody(t, rr)

	if got := decodeIntField(t, body, "code"); got != CodeSuccess {
		t.Fatalf("expected code %d, got %d", CodeSuccess, got)
	}

	if got := decodeStringField(t, body, "message"); got != MessageSuccess {
		t.Fatalf("expected message %q, got %q", MessageSuccess, got)
	}

	var data struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body["data"], &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}

	if data.ID != "123" {
		t.Fatalf("expected data id %q, got %q", "123", data.ID)
	}

	if got := decodeInt64Field(t, body, "timestamp"); got == 0 {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestWriteErrorCodeUsesBusinessCode(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteErrorCode(rr, http.StatusNotFound, 5001, "Comment not found")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	body := decodeResponseBody(t, rr)

	if got := decodeIntField(t, body, "code"); got != 5001 {
		t.Fatalf("expected business code 5001, got %d", got)
	}

	if got := decodeStringField(t, body, "message"); got != "Comment not found" {
		t.Fatalf("expected message %q, got %q", "Comment not found", got)
	}

	if got := decodeInt64Field(t, body, "timestamp"); got == 0 {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestWriteErrorPreservesLegacyStatusCodeBehavior(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteError(rr, http.StatusBadRequest, "bad request")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	body := decodeResponseBody(t, rr)

	if got := decodeIntField(t, body, "code"); got != http.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", http.StatusBadRequest, got)
	}

	if got := decodeStringField(t, body, "message"); got != "bad request" {
		t.Fatalf("expected message %q, got %q", "bad request", got)
	}
}

func TestWriteErrorCodeIncludesTraceIDAndDetails(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteErrorCode(
		rr,
		http.StatusBadRequest,
		1001,
		"参数校验失败",
		WithTraceID("trace-123"),
		WithDetails([]ErrorDetail{
			{
				Path:       "blocks[3].children[1].latex",
				Code:       "MATH_LATEX_TOO_LONG",
				MessageKey: "content.body.math_latex_too_long",
			},
		}),
	)

	body := decodeResponseBody(t, rr)

	if got := decodeStringField(t, body, "traceId"); got != "trace-123" {
		t.Fatalf("expected traceId %q, got %q", "trace-123", got)
	}

	var data struct {
		Details []struct {
			Path       string `json:"path"`
			Code       string `json:"code"`
			MessageKey string `json:"messageKey"`
		} `json:"details"`
	}
	if err := json.Unmarshal(body["data"], &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}

	if len(data.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(data.Details))
	}

	if data.Details[0].Path != "blocks[3].children[1].latex" ||
		data.Details[0].Code != "MATH_LATEX_TOO_LONG" ||
		data.Details[0].MessageKey != "content.body.math_latex_too_long" {
		t.Fatalf("unexpected detail: %#v", data.Details[0])
	}
}

func TestWriteErrorCodeOmitsEmptyOptionalFields(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteErrorCode(
		rr,
		http.StatusBadRequest,
		1001,
		"参数校验失败",
		WithTraceID(""),
		WithDetails(nil),
	)

	body := decodeResponseBody(t, rr)

	if _, ok := body["traceId"]; ok {
		t.Fatal("expected traceId to be omitted")
	}

	if _, ok := body["data"]; ok {
		t.Fatal("expected data to be omitted")
	}
}

func decodeResponseBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	return body
}

func decodeStringField(t *testing.T, body map[string]json.RawMessage, field string) string {
	t.Helper()

	raw, ok := body[field]
	if !ok {
		t.Fatalf("expected %s field", field)
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", field, err)
	}

	return value
}

func decodeIntField(t *testing.T, body map[string]json.RawMessage, field string) int {
	t.Helper()

	raw, ok := body[field]
	if !ok {
		t.Fatalf("expected %s field", field)
	}

	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", field, err)
	}

	return value
}

func decodeInt64Field(t *testing.T, body map[string]json.RawMessage, field string) int64 {
	t.Helper()

	raw, ok := body[field]
	if !ok {
		t.Fatalf("expected %s field", field)
	}

	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", field, err)
	}

	return value
}
