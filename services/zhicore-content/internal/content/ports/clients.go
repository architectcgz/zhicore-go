package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTaxonomyReferenceNotFound = errors.New("taxonomy reference not found")
	ErrMediaRefInvalid           = errors.New("media reference invalid")
	ErrCoverUnavailable          = errors.New("cover unavailable")
	ErrDependencyUnavailable     = errors.New("dependency unavailable")
)

type UserProfileClient interface {
	GetOwnerSnapshot(ctx context.Context, userID int64) (OwnerSnapshot, error)
}

type FileResourceClient interface {
	ValidateBodyMediaRefs(ctx context.Context, refs []MediaRef) error
	ValidateCoverFile(ctx context.Context, fileID string) error
}

type OwnerSnapshot struct {
	PublicID       string
	DisplayName    string
	AvatarFileID   string
	AvatarURL      string
	ProfileVersion int64
}

type Clock interface {
	Now() time.Time
}
