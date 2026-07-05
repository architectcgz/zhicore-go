package ports

import (
	"context"
	"time"
)

type UserProfileClient interface {
	GetOwnerSnapshot(ctx context.Context, userID int64) (OwnerSnapshot, error)
}

type FileResourceClient interface {
	ValidateBodyMediaRefs(ctx context.Context, refs []MediaRef) error
	ValidateCoverFile(ctx context.Context, fileID string) error
}

type OwnerSnapshot struct {
	DisplayName    string
	AvatarFileID   string
	ProfileVersion int64
}

type Clock interface {
	Now() time.Time
}
