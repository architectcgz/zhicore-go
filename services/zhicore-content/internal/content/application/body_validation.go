package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Service) appendRepairTask(ctx context.Context, task ports.BodyRepairTask) {
	if s.repair == nil {
		return
	}
	_ = s.repair.AppendOutsideTx(ctx, task)
}

func mapFileValidationError(err error, semantic error) error {
	if errors.Is(err, semantic) || errors.Is(err, ports.ErrMediaRefInvalid) || errors.Is(err, ports.ErrCoverUnavailable) {
		return err
	}
	if errors.Is(err, ports.ErrDependencyUnavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: validate file reference", ErrDependencyUnavailable)
	}
	// File adapters own transport details. Unknown adapter errors are treated as
	// dependency failures so Content never branches on provider error text.
	return fmt.Errorf("%w: validate file reference: %w", ErrDependencyUnavailable, err)
}

func (s *Service) validateStoredBody(ctx context.Context, body ports.StoredBody) (ports.NormalizedBody, error) {
	if body.SchemaVersion != 1 {
		return ports.NormalizedBody{}, ErrBodySchemaUnsupported
	}
	normalized, err := s.parser.Parse(ctx, ports.PostBodyWriteInput{
		SchemaVersion: body.SchemaVersion,
		Blocks:        body.Blocks,
	})
	if err != nil {
		return ports.NormalizedBody{}, ErrBodySchemaUnsupported
	}
	if normalized.ContentHash != body.ContentHash {
		return ports.NormalizedBody{}, domain.ErrBodyInconsistent
	}
	return normalized, nil
}
