package domain

import "time"

type RefreshSessionPersistencePolicy string

const (
	RefreshSessionPersistenceStandard   RefreshSessionPersistencePolicy = "STANDARD"
	RefreshSessionPersistenceRemembered RefreshSessionPersistencePolicy = "REMEMBERED"
)

type RefreshSession struct {
	SessionID         string
	AccountID         AccountID
	CurrentTokenID    string
	CurrentTokenHash  string
	PersistencePolicy RefreshSessionPersistencePolicy
	LastAccessJTI     string
	CreatedAt         time.Time
	LastUsedAt        time.Time
	ExpiresAt         time.Time
}

type SecurityOperation string

const (
	SecurityOperationRegister SecurityOperation = "REGISTER"
	SecurityOperationLogin    SecurityOperation = "LOGIN"
)
