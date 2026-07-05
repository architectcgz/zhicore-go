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
}

type createPostDeps struct {
	posts   *fakePostRepository
	bodies  *fakeBodyStore
	cleanup *fakeCleanupTaskStore
	repair  *fakeRepairTaskStore
	outbox  *fakeOutboxPublisher
	users   *fakeUserProfileClient
	files   *fakeFileResourceClient
	tx      *fakeTxRunner
	parser  *fakeBodyParser
	clock   fakeClock
}

func newCreatePostDeps() createPostDeps {
	return createPostDeps{
		posts:   &fakePostRepository{},
		bodies:  &fakeBodyStore{},
		cleanup: &fakeCleanupTaskStore{},
		repair:  &fakeRepairTaskStore{},
		outbox:  &fakeOutboxPublisher{},
		files:   &fakeFileResourceClient{},
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
		Posts:   d.posts,
		Queries: d.posts,
		Bodies:  d.bodies,
		Cleanup: d.cleanup,
		Repair:  d.repair,
		Outbox:  d.outbox,
		Users:   d.users,
		Files:   d.files,
		Tx:      d.tx,
		Parser:  d.parser,
		Clock:   d.clock,
	}
}

type fakePostRepository struct {
	createCalls       int
	createInput       ports.CreateDraftPost
	createResult      ports.PostRecord
	createErr         error
	getCalls          int
	getPublicID       string
	getResult         ports.PostRecord
	getErr            error
	saveCalls         int
	saveInput         ports.SaveDraftBodyUpdate
	saveResult        ports.PostRecord
	saveErr           error
	publishCalls      int
	publishInput      ports.PublishPostUpdate
	publishResult     ports.PostRecord
	publishErr        error
	bodyPointerCalls  int
	bodyPointerPublic string
	bodyPointerResult ports.PublishedBodyPointer
	bodyPointerErr    error
}

func (f *fakePostRepository) CreateDraft(ctx context.Context, tx ports.Tx, input ports.CreateDraftPost) (ports.PostRecord, error) {
	f.createCalls++
	f.createInput = input
	if f.createErr != nil {
		return ports.PostRecord{}, f.createErr
	}
	return f.createResult, nil
}

func (f *fakePostRepository) GetForUpdate(ctx context.Context, tx ports.Tx, publicID string) (ports.PostRecord, error) {
	f.getCalls++
	f.getPublicID = publicID
	if f.getErr != nil {
		return ports.PostRecord{}, f.getErr
	}
	return f.getResult, nil
}

func (f *fakePostRepository) SaveDraftBody(ctx context.Context, tx ports.Tx, input ports.SaveDraftBodyUpdate) (ports.PostRecord, error) {
	f.saveCalls++
	f.saveInput = input
	if f.saveErr != nil {
		return ports.PostRecord{}, f.saveErr
	}
	return f.saveResult, nil
}

func (f *fakePostRepository) Publish(ctx context.Context, tx ports.Tx, input ports.PublishPostUpdate) (ports.PostRecord, error) {
	f.publishCalls++
	f.publishInput = input
	if f.publishErr != nil {
		return ports.PostRecord{}, f.publishErr
	}
	return f.publishResult, nil
}

func (f *fakePostRepository) GetPublishedBodyPointer(ctx context.Context, publicID string) (ports.PublishedBodyPointer, error) {
	f.bodyPointerCalls++
	f.bodyPointerPublic = publicID
	if f.bodyPointerErr != nil {
		return ports.PublishedBodyPointer{}, f.bodyPointerErr
	}
	return f.bodyPointerResult, nil
}

type fakeBodyStore struct {
	writeDraftCalls    int
	writeInput         ports.WriteBodyInput
	writeSnapshotCalls int
	draftResult        ports.StoredBody
	snapshotResult     ports.StoredBody
	readCalls          int
	readBodyID         string
	readResult         ports.StoredBody
	deleteCalls        int
	deleteBodyID       string
	deleteErr          error
	writeDraftErr      error
	writeSnapshotErr   error
	readErr            error
}

func (f *fakeBodyStore) WriteDraftBody(ctx context.Context, input ports.WriteBodyInput) (ports.StoredBody, error) {
	f.writeDraftCalls++
	f.writeInput = input
	if f.writeDraftErr != nil {
		return ports.StoredBody{}, f.writeDraftErr
	}
	return f.draftResult, nil
}

func (f *fakeBodyStore) WriteSnapshotBody(ctx context.Context, input ports.WriteBodyInput) (ports.StoredBody, error) {
	f.writeSnapshotCalls++
	f.writeInput = input
	if f.writeSnapshotErr != nil {
		return ports.StoredBody{}, f.writeSnapshotErr
	}
	return f.snapshotResult, nil
}

func (f *fakeBodyStore) ReadBody(ctx context.Context, bodyID string) (ports.StoredBody, error) {
	f.readCalls++
	f.readBodyID = bodyID
	if f.readErr != nil {
		return ports.StoredBody{}, f.readErr
	}
	return f.readResult, nil
}

func (f *fakeBodyStore) DeleteBody(ctx context.Context, bodyID string) error {
	f.deleteCalls++
	f.deleteBodyID = bodyID
	return f.deleteErr
}

type fakeUserProfileClient struct {
	calls           int
	requestedUserID int64
	snapshot        ports.OwnerSnapshot
	err             error
}

type fakeFileResourceClient struct {
	validateMediaCalls int
	mediaRefs          []ports.MediaRef
	validateCoverCalls int
	coverFileID        string
	err                error
}

func (f *fakeFileResourceClient) ValidateBodyMediaRefs(ctx context.Context, refs []ports.MediaRef) error {
	f.validateMediaCalls++
	f.mediaRefs = append([]ports.MediaRef(nil), refs...)
	return f.err
}

func (f *fakeFileResourceClient) ValidateCoverFile(ctx context.Context, fileID string) error {
	f.validateCoverCalls++
	f.coverFileID = fileID
	return f.err
}

type fakeCleanupTaskStore struct {
	appendCalls        int
	appendOutsideCalls int
	tasks              []ports.BodyCleanupTask
	outsideTasks       []ports.BodyCleanupTask
	err                error
}

type fakeRepairTaskStore struct {
	appendCalls        int
	appendOutsideCalls int
	tasks              []ports.BodyRepairTask
	outsideTasks       []ports.BodyRepairTask
	err                error
}

func (f *fakeRepairTaskStore) Append(ctx context.Context, tx ports.Tx, task ports.BodyRepairTask) error {
	f.appendCalls++
	f.tasks = append(f.tasks, task)
	return f.err
}

func (f *fakeRepairTaskStore) AppendOutsideTx(ctx context.Context, task ports.BodyRepairTask) error {
	f.appendOutsideCalls++
	f.outsideTasks = append(f.outsideTasks, task)
	return f.err
}

type fakeOutboxPublisher struct {
	appendCalls int
	events      []ports.OutboxEvent
	err         error
}

func (f *fakeOutboxPublisher) Append(ctx context.Context, tx ports.Tx, event ports.OutboxEvent) error {
	f.appendCalls++
	f.events = append(f.events, event)
	return f.err
}

func (f *fakeCleanupTaskStore) Append(ctx context.Context, tx ports.Tx, task ports.BodyCleanupTask) error {
	f.appendCalls++
	f.tasks = append(f.tasks, task)
	return f.err
}

func (f *fakeCleanupTaskStore) AppendOutsideTx(ctx context.Context, task ports.BodyCleanupTask) error {
	f.appendOutsideCalls++
	f.outsideTasks = append(f.outsideTasks, task)
	return f.err
}

func (f *fakeUserProfileClient) GetOwnerSnapshot(ctx context.Context, userID int64) (ports.OwnerSnapshot, error) {
	f.calls++
	f.requestedUserID = userID
	if f.err != nil {
		return ports.OwnerSnapshot{}, f.err
	}
	return f.snapshot, nil
}

type fakeTxRunner struct {
	calls int
	err   error
}

func (f *fakeTxRunner) WithinTx(ctx context.Context, fn func(ctx context.Context, tx ports.Tx) error) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	return fn(ctx, struct{}{})
}

type fakeBodyParser struct {
	calls      int
	input      ports.PostBodyWriteInput
	normalized ports.NormalizedBody
	err        error
}

func (f *fakeBodyParser) Parse(ctx context.Context, input ports.PostBodyWriteInput) (ports.NormalizedBody, error) {
	f.calls++
	f.input = input
	if f.err != nil {
		return ports.NormalizedBody{}, f.err
	}
	return f.normalized, nil
}

type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time {
	return f.now
}
