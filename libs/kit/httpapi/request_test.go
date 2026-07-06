package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDecodeJSONBodyRejectsMalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":`))

	var body struct {
		Name string `json:"name"`
	}

	if err := DecodeJSONBody(req, &body); err == nil {
		t.Fatal("expected malformed JSON to fail")
	}
}

func TestDecodeJSONBodyRejectsTrailingJSONValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"first"} {"name":"second"}`))

	var body struct {
		Name string `json:"name"`
	}

	if err := DecodeJSONBody(req, &body); err == nil {
		t.Fatal("expected trailing JSON value to fail")
	}
}

func TestDecodeJSONBodyAcceptsSingleJSONValueWithWhitespace(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("  {\"name\":\"first\"} \n\t"))

	var body struct {
		Name string `json:"name"`
	}

	if err := DecodeJSONBody(req, &body); err != nil {
		t.Fatalf("decode JSON body: %v", err)
	}

	if body.Name != "first" {
		t.Fatalf("expected name %q, got %q", "first", body.Name)
	}
}

func TestDecodeJSONBodyLimitedPreservesMaxBytesError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"too large"}`))
	rr := httptest.NewRecorder()

	var body struct {
		Name string `json:"name"`
	}

	err := DecodeJSONBodyLimited(rr, req, 4, &body)
	var maxBytesErr *http.MaxBytesError
	if !errors.As(err, &maxBytesErr) {
		t.Fatalf("expected MaxBytesError, got %T: %v", err, err)
	}
}

func TestParsePositiveIntUsesDefaultForEmptyInput(t *testing.T) {
	value, err := ParsePositiveInt(" \t", 20, 100)
	if err != nil {
		t.Fatalf("parse positive int: %v", err)
	}
	if value != 20 {
		t.Fatalf("expected default value 20, got %d", value)
	}
}

func TestParsePositiveIntRejectsInvalidNonPositiveAndTooLargeValues(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		max  int
	}{
		{name: "non integer", raw: "abc", max: 100},
		{name: "zero", raw: "0", max: 100},
		{name: "negative", raw: "-1", max: 100},
		{name: "above max", raw: "101", max: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParsePositiveInt(tt.raw, 20, tt.max); err == nil {
				t.Fatal("expected parse error")
			}
		})
	}
}

func TestParsePositiveIntAcceptsPositiveValueAndIgnoresNonPositiveMax(t *testing.T) {
	value, err := ParsePositiveInt(" 101 ", 20, 0)
	if err != nil {
		t.Fatalf("parse positive int: %v", err)
	}
	if value != 101 {
		t.Fatalf("expected parsed value 101, got %d", value)
	}
}

func TestParsePositiveIntAcceptsValueEqualToMax(t *testing.T) {
	value, err := ParsePositiveInt("100", 20, 100)
	if err != nil {
		t.Fatalf("parse positive int: %v", err)
	}
	if value != 100 {
		t.Fatalf("expected parsed value 100, got %d", value)
	}
}

func TestFormatRFC3339UTC(t *testing.T) {
	local := time.Date(2026, 7, 6, 15, 30, 0, 0, time.FixedZone("CST", 8*60*60))

	got := FormatRFC3339UTC(local)
	if got != "2026-07-06T07:30:00Z" {
		t.Fatalf("expected UTC RFC3339 time, got %q", got)
	}
}

func TestFormatRFC3339UTCReturnsEmptyForZeroTime(t *testing.T) {
	if got := FormatRFC3339UTC(time.Time{}); got != "" {
		t.Fatalf("expected empty zero time, got %q", got)
	}
}
