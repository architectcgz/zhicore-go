package auth

// AccountRegisteredPayload is the version 1 payload for auth.account.registered.
// It intentionally excludes credentials, tokens, headers, and raw request body.
type AccountRegisteredPayload struct {
	AccountID  int64  `json:"accountId"`
	UserID     int64  `json:"userId"`
	Email      string `json:"email"`
	OccurredAt string `json:"occurredAt"`
}
