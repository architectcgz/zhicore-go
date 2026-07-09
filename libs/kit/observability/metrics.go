package observability

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const (
	maxMetricNameLength = 128
	maxLabelNameLength  = 64
	maxLabelValueLength = 128
)

var (
	metricNamePattern = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	labelNamePattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	labelValuePattern = regexp.MustCompile(`^[a-zA-Z0-9_./:{}-]+$`)
)

// Labels carries metrics dimensions that must stay low-cardinality and free of
// raw identifiers, request metadata, secrets, and user supplied text.
type Labels map[string]string

type MetricsRecorder interface {
	IncrementCounter(ctx context.Context, name string, labels Labels) error
}

type NoopMetricsRecorder struct{}

func (NoopMetricsRecorder) IncrementCounter(_ context.Context, name string, labels Labels) error {
	if err := ValidateMetricName(name); err != nil {
		return err
	}
	return ValidateLowCardinalityLabels(labels)
}

func ValidateMetricName(name string) error {
	if name != strings.TrimSpace(name) {
		return fmt.Errorf("metric name %q must not contain surrounding whitespace", name)
	}
	if name == "" {
		return fmt.Errorf("metric name is required")
	}
	if len(name) > maxMetricNameLength {
		return fmt.Errorf("metric name %q exceeds %d characters", name, maxMetricNameLength)
	}
	if !metricNamePattern.MatchString(name) {
		return fmt.Errorf("metric name %q contains unsupported characters", name)
	}
	return nil
}

func ValidateLowCardinalityLabels(labels Labels) error {
	for name, value := range labels {
		normalizedName := strings.TrimSpace(name)
		normalizedValue := strings.TrimSpace(value)
		if name != normalizedName {
			return fmt.Errorf("metric label name %q must not contain surrounding whitespace", name)
		}
		if value != normalizedValue {
			return fmt.Errorf("metric label %q value must not contain surrounding whitespace", normalizedName)
		}
		if normalizedName == "" {
			return fmt.Errorf("metric label name is required")
		}
		if len(normalizedName) > maxLabelNameLength {
			return fmt.Errorf("metric label name %q exceeds %d characters", normalizedName, maxLabelNameLength)
		}
		if !labelNamePattern.MatchString(normalizedName) {
			return fmt.Errorf("metric label name %q contains unsupported characters", normalizedName)
		}
		if isForbiddenLabelName(normalizedName) {
			return fmt.Errorf("metric label %q is high-cardinality or sensitive", normalizedName)
		}
		if normalizedValue == "" {
			return fmt.Errorf("metric label %q value is required", normalizedName)
		}
		if len(normalizedValue) > maxLabelValueLength {
			return fmt.Errorf("metric label %q value exceeds %d characters", normalizedName, maxLabelValueLength)
		}
		if !labelValuePattern.MatchString(normalizedValue) {
			return fmt.Errorf("metric label %q value contains unsupported characters", normalizedName)
		}
	}
	return nil
}

func isForbiddenLabelName(name string) bool {
	lower := strings.ToLower(name)
	for _, forbidden := range []string{
		"userid",
		"user_id",
		"postid",
		"post_id",
		"fileid",
		"file_id",
		"requestid",
		"request_id",
		"traceid",
		"trace_id",
		"ip",
		"url",
		"token",
		"cookie",
		"authorization",
	} {
		if lower == forbidden {
			return true
		}
	}
	return false
}
