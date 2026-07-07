package httpapi

import (
	"context"
	"errors"
	"net/http"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

var errLoginRequired = errors.New("login required")

func writeValidationError(w http.ResponseWriter) {
	sharedhttp.WriteErrorCode(w, http.StatusBadRequest, 1001, "Invalid request")
}

type errorOperation string

const (
	errorOperationCreatePost      errorOperation = "createPost"
	errorOperationSaveDraftBody   errorOperation = "saveDraftBody"
	errorOperationPublishPost     errorOperation = "publishPost"
	errorOperationPostLifecycle   errorOperation = "postLifecycle"
	errorOperationGetPostBody     errorOperation = "getPostBody"
	errorOperationPublicPostQuery errorOperation = "publicPostQuery"
	errorOperationAuthorWorkbench errorOperation = "authorWorkbench"
	errorOperationAdminOutbox     errorOperation = "adminOutbox"
)

func writeMappedError(w http.ResponseWriter, err error, operation ...errorOperation) {
	op := errorOperation("")
	if len(operation) > 0 {
		op = operation[0]
	}
	status, code, message, details := errorMapping(err, op)
	opts := make([]sharedhttp.ErrorOption, 0, 1)
	if len(details) > 0 {
		opts = append(opts, sharedhttp.WithDetails(details))
	}
	sharedhttp.WriteErrorCode(w, status, code, message, opts...)
}

func errorMapping(err error, operation errorOperation) (int, int, string, []sharedhttp.ErrorDetail) {
	var validationErr *application.BodyValidationError
	if errors.As(err, &validationErr) {
		status, code, message := bodyValidationMapping(validationErr)
		return status, code, message, validationDetails(validationErr.Details)
	}

	switch {
	case errors.Is(err, errLoginRequired), errors.Is(err, application.ErrLoginRequired):
		return http.StatusUnauthorized, 2006, "Authentication required", nil
	case errors.Is(err, application.ErrInvalidArgument):
		return http.StatusBadRequest, 1001, "Invalid request", nil
	case errors.Is(err, application.ErrTaxonomyReferenceNotFound):
		return http.StatusNotFound, 4012, "Category not found", nil
	case errors.Is(err, application.ErrMediaRefInvalid):
		return http.StatusBadRequest, 4021, "Media reference invalid", nil
	case errors.Is(err, application.ErrCoverUnavailable):
		return http.StatusBadRequest, 4023, "Cover unavailable", nil
	case errors.Is(err, application.ErrDependencyUnavailable):
		return http.StatusServiceUnavailable, 1004, "Service unavailable", nil
	case errors.Is(err, application.ErrRateLimited):
		return http.StatusTooManyRequests, 1003, "Request too frequent", nil
	case errors.Is(err, application.ErrRoleRequired):
		return http.StatusForbidden, 2007, "Role required", nil
	case errors.Is(err, application.ErrBodySchemaUnsupported):
		if operation == errorOperationSaveDraftBody || operation == errorOperationCreatePost {
			return http.StatusBadRequest, 4024, "Body schema unsupported", nil
		}
		return http.StatusInternalServerError, 4024, "Body schema unsupported", nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable, 1004, "Service unavailable", nil
	case errors.Is(err, application.ErrPostNotFound):
		return http.StatusNotFound, 4001, "Post not found", nil
	case errors.Is(err, application.ErrOutboxEventNotFound):
		return http.StatusNotFound, 1005, "Data not found", nil
	case errors.Is(err, application.ErrForbidden):
		return http.StatusForbidden, 2008, "Forbidden", nil
	case errors.Is(err, application.ErrPostAlreadyPublished):
		return http.StatusConflict, 4002, "Post already published", nil
	case errors.Is(err, application.ErrPostNotPublished):
		return http.StatusConflict, 4003, "Post not published", nil
	case errors.Is(err, application.ErrPostDeleted):
		return http.StatusConflict, 4004, "Post deleted", nil
	case errors.Is(err, application.ErrTitleRequired):
		return http.StatusBadRequest, 4005, "Post title is required", nil
	case errors.Is(err, application.ErrBodyRequired):
		return http.StatusBadRequest, 4006, "Post body is required", nil
	case errors.Is(err, application.ErrTitleTooLong):
		return http.StatusBadRequest, 4007, "Post title is too long", nil
	case errors.Is(err, application.ErrBodyTooShort):
		return http.StatusBadRequest, 4016, "Post body text is too short", nil
	case errors.Is(err, application.ErrDraftConflict):
		return http.StatusConflict, 4017, "Draft conflict", nil
	case errors.Is(err, application.ErrBodyUnavailable):
		return http.StatusInternalServerError, 4018, "Body unavailable", nil
	case errors.Is(err, application.ErrBodyInconsistent):
		return http.StatusConflict, 4019, "Body inconsistent", nil
	default:
		return http.StatusInternalServerError, 1000, "Internal server error", nil
	}
}

func bodyValidationMapping(err *application.BodyValidationError) (int, int, string) {
	if err.Truncated {
		return http.StatusBadRequest, 4022, "Too many validation errors"
	}
	for _, detail := range err.Details {
		switch detail.Code {
		case "BODY_TOO_LARGE", "BODY_TEXT_TOO_LONG", "BODY_BLOCK_COUNT_EXCEEDED", "BODY_INLINE_NODE_COUNT_EXCEEDED", "BODY_EXTERNAL_LINK_COUNT_EXCEEDED":
			return http.StatusBadRequest, 4015, "Body too large"
		case "BODY_SCHEMA_UNSUPPORTED":
			return http.StatusBadRequest, 4024, "Body schema unsupported"
		case "BLOCK_TYPE_NOT_ENABLED":
			return http.StatusBadRequest, 4014, "Block type not enabled"
		case "MEDIA_REF_INVALID":
			return http.StatusBadRequest, 4021, "Media reference invalid"
		case "EXTERNAL_EMBED_PROVIDER_NOT_ALLOWED":
			return http.StatusBadRequest, 4020, "External embed provider not allowed"
		}
	}
	return http.StatusBadRequest, 4013, "Body schema invalid"
}

func validationDetails(details []application.ValidationDetail) []sharedhttp.ErrorDetail {
	if len(details) == 0 {
		return nil
	}
	mapped := make([]sharedhttp.ErrorDetail, 0, len(details))
	for _, detail := range details {
		mapped = append(mapped, sharedhttp.ErrorDetail{Path: detail.Path, Code: detail.Code})
	}
	return mapped
}
