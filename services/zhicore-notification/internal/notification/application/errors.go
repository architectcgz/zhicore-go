package application

import (
	"errors"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func mapPortsError(err error) error {
	switch {
	case errors.Is(err, ports.ErrInvalidQuery):
		return ErrInvalidRequest
	case errors.Is(err, ports.ErrNotificationNotFound):
		return ErrNotificationNotFound
	case errors.Is(err, ports.ErrDependencyUnavailable):
		return ErrDependencyUnavailable
	default:
		return err
	}
}
