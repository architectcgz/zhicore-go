package application

type UserID int64

type AccountID int64

type PublicID string

type UserStatus string

const (
	UserStatusActive      UserStatus = "ACTIVE"
	UserStatusDeactivated UserStatus = "DEACTIVATED"
	UserStatusDeleted     UserStatus = "DELETED"
)

type UserPair struct {
	ActorID  UserID
	TargetID UserID
}
