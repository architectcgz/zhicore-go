package domain

import "time"

type RefreshSession struct {
	SessionID        string
	AccountID        AccountID
	CurrentTokenID   string
	CurrentTokenHash string
	LastAccessJTI    string
	CreatedAt        time.Time
	LastUsedAt       time.Time
	ExpiresAt        time.Time
}

type SecurityOperation string

const (
	SecurityOperationRegister SecurityOperation = "REGISTER"
	SecurityOperationLogin    SecurityOperation = "LOGIN"
)
