package delivery

const (
	StatusInApp            = "IN_APP"
	StatusWebsocketPending = "WEBSOCKET_PENDING"
	StatusDigestPending    = "DIGEST_PENDING"
	StatusSkipped          = "SKIPPED"
	StatusFailed           = "FAILED"
)

func CanRetry(status string) bool {
	switch status {
	case StatusWebsocketPending, StatusDigestPending, StatusFailed:
		return true
	default:
		return false
	}
}
