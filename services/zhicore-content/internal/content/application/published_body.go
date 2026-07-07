package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
)

func (s *Service) GetPublishedPostBody(ctx context.Context, query GetPublishedPostBodyQuery) (GetPublishedPostBodyResult, error) {
	if err := s.enforceRateLimit(ctx, publicRateLimitRequest(query.PostID, "get_published_post_body")); err != nil {
		return GetPublishedPostBodyResult{}, err
	}
	pointer, err := s.queries.GetPublishedBodyPointer(ctx, query.PostID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return GetPublishedPostBodyResult{}, err
		}
		return GetPublishedPostBodyResult{}, fmt.Errorf("%w: get published pointer", ErrDependencyUnavailable)
	}
	if pointer.Status != domain.PostStatusPublished || pointer.PublishedBodyID == "" {
		return GetPublishedPostBodyResult{}, domain.ErrPostNotFound
	}

	body, err := s.readPublishedBody(ctx, pointer.PostID, pointer.PublishedBodyID, pointer.PublishedBodyHash)
	if err != nil {
		return GetPublishedPostBodyResult{}, err
	}

	return GetPublishedPostBodyResult{
		BodyID:        body.BodyID,
		SchemaVersion: body.SchemaVersion,
		CanonicalJSON: append([]byte(nil), body.CanonicalJSON...),
		PlainText:     body.PlainText,
		ContentHash:   body.ContentHash,
		SizeBytes:     body.SizeBytes,
		CreatedAt:     body.CreatedAt,
	}, nil
}
