package application

import (
	"errors"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

var (
	ErrLoginRequired             = errors.New("login required")
	ErrInvalidArgument           = errors.New("invalid argument")
	ErrDependencyUnavailable     = errors.New("dependency unavailable")
	ErrRoleRequired              = errors.New("role required")
	ErrBodySchemaUnsupported     = errors.New("body schema unsupported")
	ErrTaxonomyReferenceNotFound = ports.ErrTaxonomyReferenceNotFound
	ErrMediaRefInvalid           = ports.ErrMediaRefInvalid
	ErrCoverUnavailable          = ports.ErrCoverUnavailable
	ErrPostNotFound              = domain.ErrPostNotFound
	ErrForbidden                 = domain.ErrForbidden
	ErrPostAlreadyPublished      = domain.ErrPostAlreadyPublished
	ErrPostDeleted               = domain.ErrPostDeleted
	ErrTitleRequired             = domain.ErrTitleRequired
	ErrTitleTooLong              = domain.ErrTitleTooLong
	ErrBodyRequired              = domain.ErrBodyRequired
	ErrBodyTooShort              = domain.ErrBodyTooShort
	ErrDraftConflict             = domain.ErrDraftConflict
	ErrBodyUnavailable           = domain.ErrBodyUnavailable
	ErrBodyInconsistent          = domain.ErrBodyInconsistent
	ErrOutboxEventNotFound       = ports.ErrOutboxEventNotFound
)

type Actor struct {
	UserID int64
	Roles  []string
}

// Application exposes body DTO aliases for inbound adapters so HTTP handlers
// do not depend on internal ports while preserving the parser-owned schema.

type Blocks = ports.Blocks

type BodyValidationError = ports.BodyValidationError

type ValidationDetail = ports.ValidationDetail

type Service struct {
	posts   ports.PostRepository
	queries ports.PostQueryRepository
	bodies  ports.PostContentStore
	cleanup ports.BodyCleanupTaskStore
	repair  ports.BodyRepairTaskStore
	outbox  ports.OutboxPublisher
	admin   ports.OutboxAdminRepository
	users   ports.UserProfileClient
	files   ports.FileResourceClient
	tx      ports.TransactionRunner
	parser  ports.BodyParserRegistry
	clock   ports.Clock
}

type Deps struct {
	Posts   ports.PostRepository
	Queries ports.PostQueryRepository
	Bodies  ports.PostContentStore
	Cleanup ports.BodyCleanupTaskStore
	Repair  ports.BodyRepairTaskStore
	Outbox  ports.OutboxPublisher
	Admin   ports.OutboxAdminRepository
	Users   ports.UserProfileClient
	Files   ports.FileResourceClient
	Tx      ports.TransactionRunner
	Parser  ports.BodyParserRegistry
	Clock   ports.Clock
}

func NewService(deps Deps) *Service {
	return &Service{
		posts:   deps.Posts,
		queries: deps.Queries,
		bodies:  deps.Bodies,
		cleanup: deps.Cleanup,
		repair:  deps.Repair,
		outbox:  deps.Outbox,
		admin:   deps.Admin,
		users:   deps.Users,
		files:   deps.Files,
		tx:      deps.Tx,
		parser:  deps.Parser,
		clock:   deps.Clock,
	}
}
