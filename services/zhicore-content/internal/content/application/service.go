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
	ErrRateLimited               = errors.New("rate limited")
	ErrRoleRequired              = errors.New("role required")
	ErrBodySchemaUnsupported     = errors.New("body schema unsupported")
	ErrTaxonomyReferenceNotFound = ports.ErrTaxonomyReferenceNotFound
	ErrMediaRefInvalid           = ports.ErrMediaRefInvalid
	ErrCoverUnavailable          = ports.ErrCoverUnavailable
	ErrPostNotFound              = domain.ErrPostNotFound
	ErrForbidden                 = domain.ErrForbidden
	ErrPostAlreadyPublished      = domain.ErrPostAlreadyPublished
	ErrPostNotPublished          = domain.ErrPostNotPublished
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
	posts           ports.PostRepository
	queries         ports.PostQueryRepository
	bodies          ports.PostContentStore
	cleanup         ports.BodyCleanupTaskStore
	repair          ports.BodyRepairTaskStore
	outbox          ports.OutboxPublisher
	admin           ports.OutboxAdminRepository
	adminPosts      ports.AdminPostRepository
	taxonomy        ports.TaxonomyRepository
	engagement      ports.EngagementRepository
	engagementStats ports.EngagementStatsTaskStore
	engagementCache ports.EngagementCacheStore
	users           ports.UserProfileClient
	files           ports.FileResourceClient
	tx              ports.TransactionRunner
	parser          ports.BodyParserRegistry
	limiter         ports.RateLimiter
	observe         ports.ContentObserver
	clock           ports.Clock
}

type Deps struct {
	Posts           ports.PostRepository
	Queries         ports.PostQueryRepository
	Bodies          ports.PostContentStore
	Cleanup         ports.BodyCleanupTaskStore
	Repair          ports.BodyRepairTaskStore
	Outbox          ports.OutboxPublisher
	Admin           ports.OutboxAdminRepository
	AdminPosts      ports.AdminPostRepository
	Taxonomy        ports.TaxonomyRepository
	Engagement      ports.EngagementRepository
	EngagementStats ports.EngagementStatsTaskStore
	EngagementCache ports.EngagementCacheStore
	Users           ports.UserProfileClient
	Files           ports.FileResourceClient
	Tx              ports.TransactionRunner
	Parser          ports.BodyParserRegistry
	Limiter         ports.RateLimiter
	Observe         ports.ContentObserver
	Clock           ports.Clock
}

func NewService(deps Deps) *Service {
	return &Service{
		posts:           deps.Posts,
		queries:         deps.Queries,
		bodies:          deps.Bodies,
		cleanup:         deps.Cleanup,
		repair:          deps.Repair,
		outbox:          deps.Outbox,
		admin:           deps.Admin,
		adminPosts:      deps.AdminPosts,
		taxonomy:        deps.Taxonomy,
		engagement:      deps.Engagement,
		engagementStats: deps.EngagementStats,
		engagementCache: deps.EngagementCache,
		users:           deps.Users,
		files:           deps.Files,
		tx:              deps.Tx,
		parser:          deps.Parser,
		limiter:         deps.Limiter,
		observe:         deps.Observe,
		clock:           deps.Clock,
	}
}
