package observability

import (
	"context"
	"testing"
)

func TestMetricsNoopRecorderAcceptsValidCounter(t *testing.T) {
	labels := Labels{
		"service":   "zhicore-content",
		"operation": "content.post.list",
		"status":    "allowed",
	}

	if err := ValidateLowCardinalityLabels(labels); err != nil {
		t.Fatalf("ValidateLowCardinalityLabels() error = %v, want nil", err)
	}
	if err := (NoopMetricsRecorder{}).IncrementCounter(context.Background(), "zhicore_content_rate_limit_decisions_total", labels); err != nil {
		t.Fatalf("NoopMetricsRecorder.IncrementCounter() error = %v, want nil", err)
	}
}

func TestMetricsRejectsHighCardinalityOrSensitiveLabels(t *testing.T) {
	testCases := []struct {
		name   string
		labels Labels
	}{
		{
			name: "post id label name",
			labels: Labels{
				"service": "zhicore-content",
				"postId":  "post_123",
			},
		},
		{
			name: "request id label name",
			labels: Labels{
				"service":   "zhicore-content",
				"requestId": "req-123",
			},
		},
		{
			name: "raw user input label value",
			labels: Labels{
				"service": "zhicore-content",
				"reason":  "redis timeout for user supplied post title",
			},
		},
		{
			name: "overlong label value",
			labels: Labels{
				"service": "zhicore-content",
				"reason":  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateLowCardinalityLabels(tc.labels); err == nil {
				t.Fatalf("ValidateLowCardinalityLabels(%#v) error = nil, want rejection", tc.labels)
			}
		})
	}
}

func TestMetricsRejectsSurroundingWhitespace(t *testing.T) {
	if err := ValidateMetricName(" zhicore_content_rate_limit_decisions_total"); err == nil {
		t.Fatalf("ValidateMetricName() error = nil, want rejection for surrounding whitespace")
	}

	testCases := []struct {
		name   string
		labels Labels
	}{
		{
			name: "label name",
			labels: Labels{
				" service": "zhicore-content",
			},
		},
		{
			name: "label value",
			labels: Labels{
				"service": " zhicore-content ",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateLowCardinalityLabels(tc.labels); err == nil {
				t.Fatalf("ValidateLowCardinalityLabels(%#v) error = nil, want rejection for surrounding whitespace", tc.labels)
			}
		})
	}
}
