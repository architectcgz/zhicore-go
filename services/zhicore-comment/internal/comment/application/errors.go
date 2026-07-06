package application

import (
	"errors"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func mapCommentLookupError(err error) error {
	if errors.Is(err, domain.ErrCommentNotFound) || errors.Is(err, domain.ErrParentCommentNotFound) || errors.Is(err, domain.ErrRootCommentNotFound) {
		return ErrCommentNotFound
	}
	return mapDomainValidationError(err)
}

func mapDomainValidationError(err error) error {
	switch {
	case errors.Is(err, domain.ErrCommentContentRequired):
		return ErrCommentContentRequired
	case errors.Is(err, domain.ErrCommentContentTooLong):
		return ErrCommentContentTooLong
	case errors.Is(err, domain.ErrCommentMediaInvalid), errors.Is(err, domain.ErrPostIDInvalid), errors.Is(err, domain.ErrUserIDInvalid):
		return ErrInvalidRequest
	case errors.Is(err, domain.ErrParentCommentNotFound), errors.Is(err, domain.ErrCommentNotFound):
		return ErrParentCommentNotFound
	case errors.Is(err, domain.ErrRootCommentNotFound):
		return ErrRootCommentNotFound
	case errors.Is(err, domain.ErrCommentIDInvalid):
		return ErrCommentIDInvalid
	default:
		return err
	}
}

func mapGuardError(err error) error {
	switch {
	case errors.Is(err, ports.ErrDependencyUnavailable):
		return ErrDependencyUnavailable
	case errors.Is(err, ports.ErrPostNotFound):
		return ErrPostNotFound
	case errors.Is(err, ports.ErrUserUnavailable):
		return ErrForbidden
	case errors.Is(err, ports.ErrInteractionBlocked):
		return ErrInteractionBlocked
	default:
		return err
	}
}
