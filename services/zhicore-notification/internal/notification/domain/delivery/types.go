package delivery

const (
	StatusWebsocketPending = "WEBSOCKET_PENDING"
	StatusDigestPending    = "DIGEST_PENDING"
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
