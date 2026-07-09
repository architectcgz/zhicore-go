package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Service) GetPublishedPostBody(ctx context.Context, query GetPublishedPostBodyQuery) (GetPublishedPostBodyResult, error) {
	if err := s.enforceRateLimit(ctx, bodyReadRateLimitRequest(query)); err != nil {
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

func bodyReadRateLimitRequest(query GetPublishedPostBodyQuery) ports.RateLimitRequest {
	callerService := strings.TrimSpace(query.CallerService)
	callerOperation := strings.TrimSpace(query.CallerOperation)
	if callerService != "" && callerOperation != "" {
		return ports.RateLimitRequest{
			LimitType: ports.RateLimitTypeInternalClient,
			Subject:   "caller:" + callerService + ":" + callerOperation,
			Resource:  strings.TrimSpace(query.PostID),
			Operation: "get_published_post_body",
		}
	}
	return publicRateLimitRequest(query.RateLimitSubject, query.PostID, "get_published_post_body")
}
