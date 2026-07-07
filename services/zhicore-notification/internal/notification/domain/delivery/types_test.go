package delivery

import "testing"

func TestDeliveryStatusNamesSeparateInAppAndRetryableStatuses(t *testing.T) {
	if StatusInApp != "IN_APP" || StatusSkipped != "SKIPPED" {
		t.Fatalf("in-app/skipped statuses = %q/%q", StatusInApp, StatusSkipped)
	}
	if CanRetry(StatusInApp) || CanRetry(StatusSkipped) {
		t.Fatalf("in-app and skipped delivery statuses must not be retryable")
	}
	if !CanRetry(StatusWebsocketPending) || !CanRetry(StatusDigestPending) || !CanRetry(StatusFailed) {
		t.Fatalf("pending and failed delivery statuses must stay retryable")
	}
}
