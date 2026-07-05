package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

var (
	ErrLoginRequired         = errors.New("login required")
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrDependencyUnavailable = errors.New("dependency unavailable")
	ErrBodySchemaUnsupported = errors.New("body schema unsupported")
	ErrPostNotFound          = domain.ErrPostNotFound
	ErrForbidden             = domain.ErrForbidden
	ErrPostAlreadyPublished  = domain.ErrPostAlreadyPublished
	ErrPostDeleted           = domain.ErrPostDeleted
	ErrTitleRequired         = domain.ErrTitleRequired
	ErrTitleTooLong          = domain.ErrTitleTooLong
	ErrBodyRequired          = domain.ErrBodyRequired
	ErrBodyTooShort          = domain.ErrBodyTooShort
	ErrDraftConflict         = domain.ErrDraftConflict
	ErrBodyUnavailable       = domain.ErrBodyUnavailable
	ErrBodyInconsistent      = domain.ErrBodyInconsistent
)

type Actor struct {
	UserID int64
}

// Application exposes body DTO aliases for inbound adapters so HTTP handlers
// do not depend on internal ports while preserving the parser-owned schema.
type Blocks = ports.Blocks

type BodyValidationError = ports.BodyValidationError

type ValidationDetail = ports.ValidationDetail

type PostBodyInput struct {
	SchemaVersion int
	Blocks        Blocks
}

type CreatePostCommand struct {
	Actor       *Actor
	Title       string
	Summary     string
	CoverFileID string
	TopicID     string
	CategoryID  string
	Tags        []string
	Body        *PostBodyInput
}

type CreatePostResult struct {
	PostID      string
	PostVersion int64
}

type SaveDraftBodyCommand struct {
	Actor             *Actor
	PostID            string
	BasePostVersion   int64
	BaseDraftBodyID   string
	BaseDraftBodyHash string
	Body              PostBodyInput
}

type SaveDraftBodyResult struct {
	PostID        string
	PostVersion   int64
	DraftBodyID   string
	DraftBodyHash string
	SavedAt       time.Time
	WordCount     int
}

type PublishPostCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	DraftBodyID     string
	DraftBodyHash   string
}

type PublishPostResult struct {
	PostID      string
	PostVersion int64
	PublishedAt time.Time
}

type GetPublishedPostBodyQuery struct {
	PostID string
}

type GetPublishedPostBodyResult struct {
	BodyID        string
	SchemaVersion int
	CanonicalJSON []byte
	PlainText     string
	ContentHash   string
	SizeBytes     int
	CreatedAt     time.Time
}

type Service struct {
	posts   ports.PostRepository
	queries ports.PostQueryRepository
	bodies  ports.PostContentStore
	cleanup ports.BodyCleanupTaskStore
	repair  ports.BodyRepairTaskStore
	outbox  ports.OutboxPublisher
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
		users:   deps.Users,
		files:   deps.Files,
		tx:      deps.Tx,
		parser:  deps.Parser,
		clock:   deps.Clock,
	}
}

func (s *Service) CreatePost(ctx context.Context, cmd CreatePostCommand) (CreatePostResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return CreatePostResult{}, ErrLoginRequired
	}

	title, err := domain.NewPostTitle(cmd.Title)
	if err != nil {
		return CreatePostResult{}, err
	}

	owner, err := s.users.GetOwnerSnapshot(ctx, cmd.Actor.UserID)
	if err != nil {
		return CreatePostResult{}, fmt.Errorf("%w: get owner snapshot", ErrDependencyUnavailable)
	}

	var draftBody ports.StoredBody
	var normalized ports.NormalizedBody
	if cmd.Body != nil {
		normalized, err = s.parser.Parse(ctx, ports.PostBodyWriteInput{
			SchemaVersion: cmd.Body.SchemaVersion,
			Blocks:        cmd.Body.Blocks,
		})
		if err != nil {
			return CreatePostResult{}, err
		}
		if s.files != nil && len(normalized.MediaRefs) > 0 {
			if err := s.files.ValidateBodyMediaRefs(ctx, normalized.MediaRefs); err != nil {
				return CreatePostResult{}, err
			}
		}

		// Body writes are copy-on-write candidates. The PostgreSQL row is only
		// created after MongoDB accepts the draft body, so a body-store failure
		// cannot create a visible post that points at missing content.
		draftBody, err = s.bodies.WriteDraftBody(ctx, ports.WriteBodyInput{
			OwnerID:       cmd.Actor.UserID,
			SchemaVersion: cmd.Body.SchemaVersion,
			Blocks:        cmd.Body.Blocks,
			CanonicalJSON: normalized.CanonicalJSON,
			PlainText:     normalized.PlainText,
			ContentHash:   normalized.ContentHash,
			SizeBytes:     normalized.SizeBytes,
			BlockCount:    normalized.BlockCount,
			CreatedAt:     s.clock.Now(),
		})
		if err != nil {
			return CreatePostResult{}, fmt.Errorf("%w: write draft body", ErrDependencyUnavailable)
		}
	}

	var created ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		input := ports.CreateDraftPost{
			OwnerID:              cmd.Actor.UserID,
			OwnerDisplayName:     owner.DisplayName,
			OwnerAvatarFileID:    owner.AvatarFileID,
			OwnerProfileVersion:  owner.ProfileVersion,
			Title:                string(title),
			Summary:              cmd.Summary,
			CoverFileID:          cmd.CoverFileID,
			TopicID:              cmd.TopicID,
			CategoryID:           cmd.CategoryID,
			Tags:                 append([]string(nil), cmd.Tags...),
			DraftBodyID:          draftBody.ID,
			DraftBodyHash:        normalized.ContentHash,
			DraftSizeBytes:       normalized.SizeBytes,
			DraftPlainTextLength: len([]rune(normalized.PlainText)),
		}
		var err error
		created, err = s.posts.CreateDraft(ctx, tx, input)
		return err
	})
	if err != nil {
		if draftBody.ID != "" && s.cleanup != nil {
			_ = s.cleanup.AppendOutsideTx(ctx, ports.BodyCleanupTask{
				BodyID:    draftBody.ID,
				TaskType:  "ORPHAN_DRAFT",
				Reason:    "draft_replaced",
				CreatedAt: s.clock.Now(),
			})
		}
		return CreatePostResult{}, fmt.Errorf("%w: create draft", ErrDependencyUnavailable)
	}

	return CreatePostResult{
		PostID:      created.PublicID,
		PostVersion: created.PostVersion,
	}, nil
}

func (s *Service) SaveDraftBody(ctx context.Context, cmd SaveDraftBodyCommand) (SaveDraftBodyResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return SaveDraftBodyResult{}, ErrLoginRequired
	}

	current, err := s.loadPostForDraftWrite(ctx, cmd.PostID)
	if err != nil {
		return SaveDraftBodyResult{}, err
	}
	if current.OwnerID != cmd.Actor.UserID {
		return SaveDraftBodyResult{}, domain.ErrForbidden
	}
	if current.Status == domain.PostStatusDeleted {
		return SaveDraftBodyResult{}, domain.ErrPostDeleted
	}
	if current.PostVersion != cmd.BasePostVersion ||
		current.DraftBodyID != cmd.BaseDraftBodyID ||
		current.DraftBodyHash != cmd.BaseDraftBodyHash {
		return SaveDraftBodyResult{}, domain.ErrDraftConflict
	}

	normalized, err := s.parser.Parse(ctx, ports.PostBodyWriteInput{
		SchemaVersion: cmd.Body.SchemaVersion,
		Blocks:        cmd.Body.Blocks,
	})
	if err != nil {
		return SaveDraftBodyResult{}, err
	}
	if s.files != nil && len(normalized.MediaRefs) > 0 {
		if err := s.files.ValidateBodyMediaRefs(ctx, normalized.MediaRefs); err != nil {
			return SaveDraftBodyResult{}, err
		}
	}

	now := s.clock.Now()
	wordCount := len([]rune(normalized.PlainText))
	if normalized.ContentHash == current.DraftBodyHash {
		return SaveDraftBodyResult{
			PostID:        current.PublicID,
			PostVersion:   current.PostVersion,
			DraftBodyID:   current.DraftBodyID,
			DraftBodyHash: current.DraftBodyHash,
			SavedAt:       now,
			WordCount:     wordCount,
		}, nil
	}

	newDraft, err := s.bodies.WriteDraftBody(ctx, ports.WriteBodyInput{
		PostPublicID:  cmd.PostID,
		OwnerID:       cmd.Actor.UserID,
		SchemaVersion: cmd.Body.SchemaVersion,
		Blocks:        cmd.Body.Blocks,
		CanonicalJSON: normalized.CanonicalJSON,
		PlainText:     normalized.PlainText,
		ContentHash:   normalized.ContentHash,
		SizeBytes:     normalized.SizeBytes,
		BlockCount:    normalized.BlockCount,
		CreatedAt:     now,
	})
	if err != nil {
		return SaveDraftBodyResult{}, fmt.Errorf("%w: write draft body", ErrDependencyUnavailable)
	}

	var saved ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		saved, err = s.posts.SaveDraftBody(ctx, tx, ports.SaveDraftBodyUpdate{
			PublicID:             cmd.PostID,
			OwnerID:              cmd.Actor.UserID,
			BasePostVersion:      cmd.BasePostVersion,
			BaseDraftBodyID:      cmd.BaseDraftBodyID,
			BaseDraftBodyHash:    cmd.BaseDraftBodyHash,
			NewDraftBodyID:       newDraft.ID,
			NewDraftBodyHash:     normalized.ContentHash,
			NewDraftSizeBytes:    normalized.SizeBytes,
			NewDraftPlainTextLen: wordCount,
		})
		if err != nil {
			return err
		}
		if current.DraftBodyID != "" && current.DraftBodyID != newDraft.ID && s.cleanup != nil {
			return s.cleanup.Append(ctx, tx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    current.DraftBodyID,
				TaskType:  "OLD_DRAFT",
				Reason:    "draft_replaced",
				CreatedAt: now,
			})
		}
		return nil
	})
	if err != nil {
		// The new MongoDB body is not referenced if the PostgreSQL conditional
		// update fails; record an orphan cleanup outside the rolled-back tx.
		if s.cleanup != nil {
			_ = s.cleanup.AppendOutsideTx(ctx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    newDraft.ID,
				TaskType:  "ORPHAN_DRAFT",
				Reason:    "draft_replaced",
				CreatedAt: now,
			})
		}
		if errors.Is(err, domain.ErrDraftConflict) || errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrPostDeleted) {
			return SaveDraftBodyResult{}, err
		}
		return SaveDraftBodyResult{}, fmt.Errorf("%w: save draft body", ErrDependencyUnavailable)
	}

	return SaveDraftBodyResult{
		PostID:        saved.PublicID,
		PostVersion:   saved.PostVersion,
		DraftBodyID:   saved.DraftBodyID,
		DraftBodyHash: saved.DraftBodyHash,
		SavedAt:       now,
		WordCount:     wordCount,
	}, nil
}

func (s *Service) loadPostForDraftWrite(ctx context.Context, publicID string) (ports.PostRecord, error) {
	var current ports.PostRecord
	err := s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		current, err = s.posts.GetForUpdate(ctx, tx, publicID)
		return err
	})
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return ports.PostRecord{}, err
		}
		return ports.PostRecord{}, fmt.Errorf("%w: load post", ErrDependencyUnavailable)
	}
	return current, nil
}

func (s *Service) PublishPost(ctx context.Context, cmd PublishPostCommand) (PublishPostResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return PublishPostResult{}, ErrLoginRequired
	}

	current, err := s.loadPostForDraftWrite(ctx, cmd.PostID)
	if err != nil {
		return PublishPostResult{}, err
	}
	if current.OwnerID != cmd.Actor.UserID {
		return PublishPostResult{}, domain.ErrForbidden
	}
	if current.Status == domain.PostStatusDeleted {
		return PublishPostResult{}, domain.ErrPostDeleted
	}
	if current.Status == domain.PostStatusPublished {
		return PublishPostResult{}, domain.ErrPostAlreadyPublished
	}
	if current.DraftBodyID == "" || current.DraftBodyHash == "" {
		return PublishPostResult{}, domain.ErrBodyRequired
	}
	if current.PostVersion != cmd.BasePostVersion ||
		current.DraftBodyID != cmd.DraftBodyID ||
		current.DraftBodyHash != cmd.DraftBodyHash {
		return PublishPostResult{}, domain.ErrDraftConflict
	}

	draftBody, err := s.bodies.ReadBody(ctx, current.DraftBodyID)
	if err != nil {
		if errors.Is(err, domain.ErrBodyUnavailable) {
			s.appendRepairTask(ctx, ports.BodyRepairTask{
				PostID:       current.ID,
				BodyID:       current.DraftBodyID,
				TaskType:     "draft_body_missing",
				ExpectedHash: current.DraftBodyHash,
				CreatedAt:    s.clock.Now(),
			})
			return PublishPostResult{}, err
		}
		return PublishPostResult{}, fmt.Errorf("%w: read draft body", ErrDependencyUnavailable)
	}
	if draftBody.ContentHash != current.DraftBodyHash || draftBody.ContentHash != cmd.DraftBodyHash {
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       current.ID,
			BodyID:       current.DraftBodyID,
			TaskType:     "body_hash_mismatch",
			ExpectedHash: current.DraftBodyHash,
			ObservedHash: draftBody.ContentHash,
			CreatedAt:    s.clock.Now(),
		})
		return PublishPostResult{}, domain.ErrBodyInconsistent
	}
	normalized, err := s.validateStoredBody(ctx, draftBody)
	if err != nil {
		if errors.Is(err, domain.ErrBodyInconsistent) {
			s.appendRepairTask(ctx, ports.BodyRepairTask{
				PostID:       current.ID,
				BodyID:       current.DraftBodyID,
				TaskType:     "body_hash_mismatch",
				ExpectedHash: current.DraftBodyHash,
				ObservedHash: draftBody.ContentHash,
				CreatedAt:    s.clock.Now(),
			})
		}
		return PublishPostResult{}, err
	}
	if s.files != nil {
		if len(normalized.MediaRefs) > 0 {
			if err := s.files.ValidateBodyMediaRefs(ctx, normalized.MediaRefs); err != nil {
				return PublishPostResult{}, err
			}
		}
		if current.DraftCoverFileID != "" {
			if err := s.files.ValidateCoverFile(ctx, current.DraftCoverFileID); err != nil {
				return PublishPostResult{}, err
			}
		}
	}

	post, err := domain.HydratePost(domain.HydratePostInput{
		ID:       domain.PostID(current.ID),
		PublicID: domain.PublicPostID(current.PublicID),
		OwnerID:  domain.OwnerID(current.OwnerID),
		Title:    current.DraftTitle,
		Summary:  current.DraftSummary,
		Status:   current.Status,
		DraftBody: &domain.BodyPointer{
			ID:              current.DraftBodyID,
			Hash:            current.DraftBodyHash,
			PlainTextLength: len([]rune(normalized.PlainText)),
			SizeBytes:       normalized.SizeBytes,
		},
	})
	if err != nil {
		return PublishPostResult{}, err
	}
	post.PullEvents()
	if err := post.Publish(domain.NewPostPublishPolicy(defaultMinPublishedPlainTextRunes), domain.PublishInput{
		DraftBody:   post.DraftBody(),
		PublishedAt: s.clock.Now(),
	}); err != nil {
		return PublishPostResult{}, err
	}
	domainEvents := post.PullEvents()

	publishedAt := s.clock.Now()
	snapshot, err := s.bodies.WriteSnapshotBody(ctx, ports.WriteBodyInput{
		PostPublicID:  cmd.PostID,
		OwnerID:       cmd.Actor.UserID,
		SchemaVersion: draftBody.SchemaVersion,
		Blocks:        draftBody.Blocks,
		CanonicalJSON: normalized.CanonicalJSON,
		PlainText:     normalized.PlainText,
		ContentHash:   normalized.ContentHash,
		SizeBytes:     normalized.SizeBytes,
		BlockCount:    normalized.BlockCount,
		CreatedAt:     publishedAt,
	})
	if err != nil {
		return PublishPostResult{}, fmt.Errorf("%w: write published snapshot", ErrDependencyUnavailable)
	}

	var published ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		published, err = s.posts.Publish(ctx, tx, ports.PublishPostUpdate{
			PublicID:                 cmd.PostID,
			OwnerID:                  cmd.Actor.UserID,
			BasePostVersion:          cmd.BasePostVersion,
			ExpectedDraftBodyID:      cmd.DraftBodyID,
			ExpectedDraftBodyHash:    cmd.DraftBodyHash,
			NewPublishedBodyID:       snapshot.ID,
			NewPublishedBodyHash:     normalized.ContentHash,
			NewPublishedPlainTextLen: len([]rune(normalized.PlainText)),
			PublishedAt:              publishedAt,
		})
		if err != nil {
			return err
		}
		if s.outbox != nil && hasPostPublishedEvent(domainEvents) {
			event, err := newPostPublishedOutboxEvent(current, published, snapshot.ID, normalized.ContentHash, publishedAt)
			if err != nil {
				return err
			}
			if err := s.outbox.Append(ctx, tx, event); err != nil {
				return err
			}
		}
		if current.DraftBodyID != "" && s.cleanup != nil {
			return s.cleanup.Append(ctx, tx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    current.DraftBodyID,
				TaskType:  "OLD_DRAFT",
				Reason:    "post_published",
				CreatedAt: publishedAt,
			})
		}
		return nil
	})
	if err != nil {
		// Snapshot candidates become live only after the PostgreSQL transaction
		// switches published_body_id. Commit outcome can be ambiguous, so the
		// application does not directly delete MongoDB bodies here; cleanup
		// workers must confirm PostgreSQL no longer references the body first.
		var cleanupErr error
		if s.cleanup != nil {
			cleanupErr = s.cleanup.AppendOutsideTx(ctx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    snapshot.ID,
				TaskType:  "ORPHAN_SNAPSHOT",
				Reason:    "publish_tx_failed",
				CreatedAt: publishedAt,
			})
		}
		if cleanupErr != nil {
			return PublishPostResult{}, fmt.Errorf("%w: publish post; register orphan snapshot cleanup: %w", ErrDependencyUnavailable, cleanupErr)
		}
		if errors.Is(err, domain.ErrDraftConflict) || errors.Is(err, domain.ErrForbidden) ||
			errors.Is(err, domain.ErrPostDeleted) || errors.Is(err, domain.ErrPostAlreadyPublished) {
			return PublishPostResult{}, err
		}
		return PublishPostResult{}, fmt.Errorf("%w: publish post", ErrDependencyUnavailable)
	}

	return PublishPostResult{
		PostID:      published.PublicID,
		PostVersion: published.PostVersion,
		PublishedAt: publishedAt,
	}, nil
}

const defaultMinPublishedPlainTextRunes = 10

func (s *Service) GetPublishedPostBody(ctx context.Context, query GetPublishedPostBodyQuery) (GetPublishedPostBodyResult, error) {
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

	body, err := s.bodies.ReadBody(ctx, pointer.PublishedBodyID)
	if err != nil {
		taskType := "mongo_read_error_after_pg_published"
		if errors.Is(err, domain.ErrBodyUnavailable) {
			taskType = "published_body_missing"
		}
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       pointer.PostID,
			BodyID:       pointer.PublishedBodyID,
			TaskType:     taskType,
			ExpectedHash: pointer.PublishedBodyHash,
			CreatedAt:    s.clock.Now(),
		})
		if errors.Is(err, domain.ErrBodyUnavailable) {
			return GetPublishedPostBodyResult{}, err
		}
		return GetPublishedPostBodyResult{}, fmt.Errorf("%w: read published body", ErrDependencyUnavailable)
	}

	if body.ContentHash != pointer.PublishedBodyHash {
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       pointer.PostID,
			BodyID:       pointer.PublishedBodyID,
			TaskType:     "body_hash_mismatch",
			ExpectedHash: pointer.PublishedBodyHash,
			ObservedHash: body.ContentHash,
			CreatedAt:    s.clock.Now(),
		})
		return GetPublishedPostBodyResult{}, domain.ErrBodyInconsistent
	}
	normalized, err := s.validateStoredBody(ctx, body)
	if err != nil {
		taskType := "mongo_read_error_after_pg_published"
		if errors.Is(err, domain.ErrBodyInconsistent) {
			taskType = "body_hash_mismatch"
		}
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       pointer.PostID,
			BodyID:       pointer.PublishedBodyID,
			TaskType:     taskType,
			ExpectedHash: pointer.PublishedBodyHash,
			ObservedHash: body.ContentHash,
			CreatedAt:    s.clock.Now(),
		})
		return GetPublishedPostBodyResult{}, err
	}
	if normalized.ContentHash != pointer.PublishedBodyHash {
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       pointer.PostID,
			BodyID:       pointer.PublishedBodyID,
			TaskType:     "body_hash_mismatch",
			ExpectedHash: pointer.PublishedBodyHash,
			ObservedHash: normalized.ContentHash,
			CreatedAt:    s.clock.Now(),
		})
		return GetPublishedPostBodyResult{}, domain.ErrBodyInconsistent
	}

	return GetPublishedPostBodyResult{
		BodyID:        body.ID,
		SchemaVersion: body.SchemaVersion,
		CanonicalJSON: append([]byte(nil), normalized.CanonicalJSON...),
		PlainText:     normalized.PlainText,
		ContentHash:   normalized.ContentHash,
		SizeBytes:     normalized.SizeBytes,
		CreatedAt:     body.CreatedAt,
	}, nil
}

func (s *Service) appendRepairTask(ctx context.Context, task ports.BodyRepairTask) {
	if s.repair == nil {
		return
	}
	_ = s.repair.AppendOutsideTx(ctx, task)
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

func hasPostPublishedEvent(events []domain.DomainEvent) bool {
	for _, event := range events {
		if _, ok := event.(domain.PostPublished); ok {
			return true
		}
	}
	return false
}

type postPublishedOutboxPayload struct {
	PublicID          string    `json:"publicId"`
	InternalID        int64     `json:"internalId"`
	AuthorID          int64     `json:"authorId"`
	Title             string    `json:"title"`
	Summary           string    `json:"summary,omitempty"`
	CoverFileID       string    `json:"coverFileId,omitempty"`
	PublishedAt       time.Time `json:"publishedAt"`
	PublishedBodyID   string    `json:"publishedBodyId,omitempty"`
	PublishedBodyHash string    `json:"publishedBodyHash,omitempty"`
}

// Application owns the mapping from domain publish facts to the cross-service
// outbox contract so the domain model stays free of MQ / JSON concerns.
func newPostPublishedOutboxEvent(current, published ports.PostRecord, publishedBodyID, publishedBodyHash string, publishedAt time.Time) (ports.OutboxEvent, error) {
	payloadJSON, err := json.Marshal(postPublishedOutboxPayload{
		PublicID:          current.PublicID,
		InternalID:        current.ID,
		AuthorID:          current.OwnerID,
		Title:             current.DraftTitle,
		Summary:           current.DraftSummary,
		CoverFileID:       current.DraftCoverFileID,
		PublishedAt:       publishedAt,
		PublishedBodyID:   publishedBodyID,
		PublishedBodyHash: publishedBodyHash,
	})
	if err != nil {
		return ports.OutboxEvent{}, err
	}

	return ports.OutboxEvent{
		EventType:        "content.post.published",
		PayloadVersion:   1,
		AggregateType:    "post",
		AggregateID:      current.PublicID,
		AggregateVersion: published.PostVersion,
		PayloadJSON:      payloadJSON,
		OccurredAt:       publishedAt,
	}, nil
}
