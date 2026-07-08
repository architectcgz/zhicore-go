package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestCreatePost(t *testing.T) {
	t.Run("requires actor", func(t *testing.T) {
		deps := newCreatePostDeps()
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{Title: "draft"})
		if !errors.Is(err, ErrLoginRequired) {
			t.Fatalf("error = %v, want ErrLoginRequired", err)
		}
		if deps.tx.calls != 0 {
			t.Fatalf("tx calls = %d, want 0", deps.tx.calls)
		}
	})

	t.Run("creates empty draft with owner snapshot and initial stats", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.createResult = ports.PostRecord{PublicID: "post_empty", PostVersion: 1}
		service := NewService(deps.asDeps())

		got, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor: &Actor{UserID: 1001},
			Title: "  draft title  ",
		})
		if err != nil {
			t.Fatalf("CreatePost returned error: %v", err)
		}
		if got.PostID != "post_empty" || got.PostVersion != 1 {
			t.Fatalf("result = %+v, want post_empty version 1", got)
		}
		if deps.parser.calls != 0 || deps.bodies.writeDraftCalls != 0 {
			t.Fatalf("body path calls parser=%d write=%d, want none", deps.parser.calls, deps.bodies.writeDraftCalls)
		}
		if deps.users.requestedUserID != 1001 {
			t.Fatalf("user lookup id = %d, want 1001", deps.users.requestedUserID)
		}
		if deps.posts.createInput.OwnerDisplayName != "architect" {
			t.Fatalf("owner display name = %q, want snapshot", deps.posts.createInput.OwnerDisplayName)
		}
		if deps.posts.createInput.Title != "draft title" {
			t.Fatalf("title = %q, want trimmed", deps.posts.createInput.Title)
		}
		if deps.tx.calls != 1 || deps.posts.createCalls != 1 {
			t.Fatalf("tx/create calls = %d/%d, want 1/1", deps.tx.calls, deps.posts.createCalls)
		}
	})

	t.Run("creates draft with parsed body before PostgreSQL transaction", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.createResult = ports.PostRecord{PublicID: "post_body", PostVersion: 1}
		deps.parser.normalized = ports.NormalizedBody{
			PlainText:     "hello world",
			CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
			ContentHash:   "sha256:body",
			SizeBytes:     35,
			BlockCount:    1,
		}
		deps.bodies.draftResult = ports.StoredBody{ID: "body_1"}
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor: &Actor{UserID: 1001},
			Title: "body draft",
			Body: &PostBodyInput{
				SchemaVersion: 1,
				Blocks: ports.Blocks{
					&ports.ParagraphBlock{Children: []ports.InlineNode{{Type: "text", Text: "hello world"}}},
				},
			},
		})
		if err != nil {
			t.Fatalf("CreatePost returned error: %v", err)
		}
		if deps.parser.calls != 1 || deps.bodies.writeDraftCalls != 1 {
			t.Fatalf("body path calls parser=%d write=%d, want 1/1", deps.parser.calls, deps.bodies.writeDraftCalls)
		}
		if deps.tx.calls != 1 {
			t.Fatalf("tx calls = %d, want body write before one PG tx", deps.tx.calls)
		}
		if deps.posts.createInput.DraftBodyID != "body_1" || deps.posts.createInput.DraftBodyHash != "sha256:body" {
			t.Fatalf("draft pointer = %s/%s, want body_1/sha256:body", deps.posts.createInput.DraftBodyID, deps.posts.createInput.DraftBodyHash)
		}
		if deps.posts.createInput.DraftPlainTextLength != len([]rune("hello world")) {
			t.Fatalf("plain text length = %d, want hello world length", deps.posts.createInput.DraftPlainTextLength)
		}
	})

	t.Run("rejects invalid title before side effects", func(t *testing.T) {
		deps := newCreatePostDeps()
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor: &Actor{UserID: 1001},
			Title: strings.Repeat("字", domain.MaxPostTitleRunes+1),
		})
		if !errors.Is(err, domain.ErrTitleTooLong) {
			t.Fatalf("error = %v, want ErrTitleTooLong", err)
		}
		if deps.users.calls != 0 || deps.tx.calls != 0 || deps.bodies.writeDraftCalls != 0 {
			t.Fatalf("side effects users=%d tx=%d body=%d, want none", deps.users.calls, deps.tx.calls, deps.bodies.writeDraftCalls)
		}
	})

	t.Run("returns body validation error before writing body", func(t *testing.T) {
		deps := newCreatePostDeps()
		parseErr := &ports.BodyValidationError{Details: []ports.ValidationDetail{{Path: "blocks[0]", Code: "BODY_SCHEMA_INVALID"}}}
		deps.parser.err = parseErr
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor: &Actor{UserID: 1001},
			Title: "draft",
			Body:  &PostBodyInput{SchemaVersion: 1, Blocks: ports.Blocks{}},
		})
		if err != parseErr {
			t.Fatalf("error = %v, want parser error", err)
		}
		if deps.bodies.writeDraftCalls != 0 || deps.tx.calls != 0 {
			t.Fatalf("body/tx calls = %d/%d, want none", deps.bodies.writeDraftCalls, deps.tx.calls)
		}
	})

	t.Run("does not create visible post when body write fails", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.bodies.writeDraftErr = errors.New("mongo unavailable")
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor: &Actor{UserID: 1001},
			Title: "draft",
			Body:  &PostBodyInput{SchemaVersion: 1, Blocks: ports.Blocks{}},
		})
		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.tx.calls != 0 || deps.posts.createCalls != 0 {
			t.Fatalf("tx/create calls = %d/%d, want no visible post", deps.tx.calls, deps.posts.createCalls)
		}
	})

	t.Run("records orphan cleanup when PostgreSQL create fails after body write", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.parser.normalized = ports.NormalizedBody{
			PlainText:     "hello world",
			CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
			ContentHash:   "sha256:body",
			SizeBytes:     35,
			BlockCount:    1,
		}
		deps.bodies.draftResult = ports.StoredBody{ID: "body_orphan"}
		deps.posts.createErr = errors.New("pg failed")
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor: &Actor{UserID: 1001},
			Title: "body draft",
			Body:  &PostBodyInput{SchemaVersion: 1, Blocks: ports.Blocks{}},
		})
		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.cleanup.appendOutsideCalls != 1 {
			t.Fatalf("outside cleanup calls = %d, want 1", deps.cleanup.appendOutsideCalls)
		}
		if got := deps.cleanup.outsideTasks[0]; got.BodyID != "body_orphan" || got.TaskType != "ORPHAN_DRAFT" {
			t.Fatalf("outside cleanup = %+v, want orphan draft", got)
		}
	})

	t.Run("returns taxonomy reference error without treating it as dependency outage", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.createErr = ports.ErrTaxonomyReferenceNotFound
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor:      &Actor{UserID: 1001},
			Title:      "draft",
			CategoryID: "cat_missing",
		})

		if !errors.Is(err, ErrTaxonomyReferenceNotFound) {
			t.Fatalf("error = %v, want ErrTaxonomyReferenceNotFound", err)
		}
		if errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, must not be dependency unavailable", err)
		}
	})

	t.Run("returns media reference error before body write", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.parser.normalized = ports.NormalizedBody{
			PlainText:     "hello world",
			CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
			ContentHash:   "sha256:body",
			SizeBytes:     35,
			BlockCount:    1,
			MediaRefs:     []ports.MediaRef{{FileID: "file_missing"}},
		}
		deps.files.err = ports.ErrMediaRefInvalid
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor: &Actor{UserID: 1001},
			Title: "draft",
			Body:  &PostBodyInput{SchemaVersion: 1, Blocks: ports.Blocks{}},
		})

		if !errors.Is(err, ErrMediaRefInvalid) {
			t.Fatalf("error = %v, want ErrMediaRefInvalid", err)
		}
		if deps.bodies.writeDraftCalls != 0 || deps.tx.calls != 0 {
			t.Fatalf("body/tx calls = %d/%d, want none", deps.bodies.writeDraftCalls, deps.tx.calls)
		}
	})

	t.Run("returns dependency unavailable for file service outage", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.parser.normalized = ports.NormalizedBody{
			PlainText:     "hello world",
			CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
			ContentHash:   "sha256:body",
			SizeBytes:     35,
			BlockCount:    1,
			MediaRefs:     []ports.MediaRef{{FileID: "file_1"}},
		}
		deps.files.err = ports.ErrDependencyUnavailable
		service := NewService(deps.asDeps())

		_, err := service.CreatePost(context.Background(), CreatePostCommand{
			Actor: &Actor{UserID: 1001},
			Title: "draft",
			Body:  &PostBodyInput{SchemaVersion: 1, Blocks: ports.Blocks{}},
		})

		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.bodies.writeDraftCalls != 0 || deps.tx.calls != 0 {
			t.Fatalf("body/tx calls = %d/%d, want none", deps.bodies.writeDraftCalls, deps.tx.calls)
		}
	})
}

type createPostDeps struct {
	posts           *fakePostRepository
	bodies          *fakeBodyStore
	cleanup         *fakeCleanupTaskStore
	repair          *fakeRepairTaskStore
	outbox          *fakeOutboxPublisher
	outboxAdmin     *fakeOutboxAdminRepository
	adminPosts      *fakeAdminPostRepository
	taxonomy        *fakeTaxonomyRepository
	engagement      *fakeEngagementRepository
	engagementStats *fakeEngagementStatsTaskStore
	engagementCache *fakeEngagementCache
	users           *fakeUserProfileClient
	files           *fakeFileResourceClient
	tx              *fakeTxRunner
	parser          *fakeBodyParser
	clock           fakeClock
}

func newCreatePostDeps() createPostDeps {
	return createPostDeps{
		posts:           &fakePostRepository{},
		bodies:          &fakeBodyStore{},
		cleanup:         &fakeCleanupTaskStore{},
		repair:          &fakeRepairTaskStore{},
		outbox:          &fakeOutboxPublisher{},
		outboxAdmin:     &fakeOutboxAdminRepository{},
		adminPosts:      &fakeAdminPostRepository{},
		taxonomy:        &fakeTaxonomyRepository{},
		engagement:      &fakeEngagementRepository{},
		engagementStats: &fakeEngagementStatsTaskStore{},
		engagementCache: &fakeEngagementCache{},
		files:           &fakeFileResourceClient{},
		users: &fakeUserProfileClient{snapshot: ports.OwnerSnapshot{
			DisplayName:    "architect",
			AvatarFileID:   "file_avatar",
			ProfileVersion: 3,
		}},
		tx:     &fakeTxRunner{},
		parser: &fakeBodyParser{},
		clock:  fakeClock{now: time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)},
	}
}

func (d createPostDeps) asDeps() Deps {
	return Deps{
		Posts:           d.posts,
		Queries:         d.posts,
		Bodies:          d.bodies,
		Cleanup:         d.cleanup,
		Repair:          d.repair,
		Outbox:          d.outbox,
		Admin:           d.outboxAdmin,
		AdminPosts:      d.adminPosts,
		Taxonomy:        d.taxonomy,
		Engagement:      d.engagement,
		EngagementStats: d.engagementStats,
		EngagementCache: d.engagementCache,
		Users:           d.users,
		Files:           d.files,
		Tx:              d.tx,
		Parser:          d.parser,
		Clock:           d.clock,
	}
}
