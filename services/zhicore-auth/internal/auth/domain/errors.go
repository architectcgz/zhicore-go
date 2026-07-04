package domain

import "errors"

var (
	ErrEmailInvalid          = errors.New("email is invalid")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrLoginIdentifierExists = errors.New("login identifier already exists")
	ErrAccountNotFound       = errors.New("account not found")
	ErrCredentialNotFound    = errors.New("credential not found")
	ErrAccountDisabled       = errors.New("account is disabled")
	ErrAccountBanned         = errors.New("account is banned")
	ErrAccountLocked         = errors.New("account is locked")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrRateLimitUnavailable  = errors.New("rate limit unavailable")
)
