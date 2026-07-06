package application

import (
	"context"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type fakePostRepository struct {
	createCalls          int
	createTx             ports.Tx
	createInput          ports.CreateDraftPost
	createResult         ports.PostRecord
	createErr            error
	getCalls             int
	getTx                ports.Tx
	getPublicID          string
	getResult            ports.PostRecord
	getErr               error
	saveCalls            int
	saveTx               ports.Tx
	saveInput            ports.SaveDraftBodyUpdate
	saveResult           ports.PostRecord
	saveErr              error
	publishCalls         int
	publishTx            ports.Tx
	publishInput         ports.PublishPostUpdate
	publishResult        ports.PostRecord
	publishErr           error
	unpublishCalls       int
	unpublishTx          ports.Tx
	unpublishInput       ports.PostLifecycleUpdate
	unpublishResult      ports.PostRecord
	unpublishErr         error
	deletePostCalls      int
	deletePostTx         ports.Tx
	deletePostInput      ports.PostLifecycleUpdate
	deletePostResult     ports.PostRecord
	deletePostErr        error
	restoreCalls         int
	restoreTx            ports.Tx
	restoreInput         ports.PostLifecycleUpdate
	restoreResult        ports.PostRecord
	restoreErr           error
	scheduleCalls        int
	scheduleTx           ports.Tx
	scheduleInput        ports.SchedulePostUpdate
	scheduleResult       ports.PostRecord
	scheduleErr          error
	cancelScheduleCalls  int
	cancelScheduleTx     ports.Tx
	cancelScheduleInput  ports.PostLifecycleUpdate
	cancelScheduleResult ports.PostRecord
	cancelScheduleErr    error
	updateMetaCalls      int
	updateMetaTx         ports.Tx
	updateMetaInput      ports.UpdateDraftMetaUpdate
	updateMetaResult     ports.PostRecord
	updateMetaErr        error
	deleteDraftCalls     int
	deleteDraftTx        ports.Tx
	deleteDraftInput     ports.DeleteDraftUpdate
	deleteDraftResult    ports.PostRecord
	deleteDraftErr       error
	bodyPointerCalls     int
	bodyPointerPublic    string
	bodyPointerResult    ports.PublishedBodyPointer
	bodyPointerErr       error
	listPublishedCalls   int
	listPublishedQuery   ports.PostListQuery
	listPublishedResult  []ports.PostSummaryRecord
	listPublishedErr     error
	detailCalls          int
	detailPublicID       string
	detailResult         ports.PostDetailRecord
	detailErr            error
	batchCalls           int
	batchIDs             []string
	batchResult          []ports.PostSummaryRecord
	batchErr             error
	listAuthorCalls      int
	listAuthorQuery      ports.AuthorPostListQuery
	listAuthorResult     []ports.PostSummaryRecord
	listAuthorErr        error
	draftCalls           int
	draftPublicID        string
	draftResult          ports.DraftPostRecord
	draftErr             error
	referenceChecks      int
	referenceBodyID      string
	bodyReferenced       bool
	bodyReferenceErr     error
}

func (f *fakePostRepository) CreateDraft(ctx context.Context, tx ports.Tx, input ports.CreateDraftPost) (ports.PostRecord, error) {
	f.createCalls++
	f.createTx = tx
	f.createInput = input
	if f.createErr != nil {
		return ports.PostRecord{}, f.createErr
	}
	return f.createResult, nil
}

func (f *fakePostRepository) GetForUpdate(ctx context.Context, tx ports.Tx, publicID string) (ports.PostRecord, error) {
	f.getCalls++
	f.getTx = tx
	f.getPublicID = publicID
	if f.getErr != nil {
		return ports.PostRecord{}, f.getErr
	}
	return f.getResult, nil
}

func (f *fakePostRepository) SaveDraftBody(ctx context.Context, tx ports.Tx, input ports.SaveDraftBodyUpdate) (ports.PostRecord, error) {
	f.saveCalls++
	f.saveTx = tx
	f.saveInput = input
	if f.saveErr != nil {
		return ports.PostRecord{}, f.saveErr
	}
	return f.saveResult, nil
}

func (f *fakePostRepository) Publish(ctx context.Context, tx ports.Tx, input ports.PublishPostUpdate) (ports.PostRecord, error) {
	f.publishCalls++
	f.publishTx = tx
	f.publishInput = input
	if f.publishErr != nil {
		return ports.PostRecord{}, f.publishErr
	}
	return f.publishResult, nil
}

func (f *fakePostRepository) Unpublish(ctx context.Context, tx ports.Tx, input ports.PostLifecycleUpdate) (ports.PostRecord, error) {
	f.unpublishCalls++
	f.unpublishTx = tx
	f.unpublishInput = input
	if f.unpublishErr != nil {
		return ports.PostRecord{}, f.unpublishErr
	}
	return f.unpublishResult, nil
}

func (f *fakePostRepository) DeletePost(ctx context.Context, tx ports.Tx, input ports.PostLifecycleUpdate) (ports.PostRecord, error) {
	f.deletePostCalls++
	f.deletePostTx = tx
	f.deletePostInput = input
	if f.deletePostErr != nil {
		return ports.PostRecord{}, f.deletePostErr
	}
	return f.deletePostResult, nil
}

func (f *fakePostRepository) RestorePost(ctx context.Context, tx ports.Tx, input ports.PostLifecycleUpdate) (ports.PostRecord, error) {
	f.restoreCalls++
	f.restoreTx = tx
	f.restoreInput = input
	if f.restoreErr != nil {
		return ports.PostRecord{}, f.restoreErr
	}
	return f.restoreResult, nil
}

func (f *fakePostRepository) SchedulePost(ctx context.Context, tx ports.Tx, input ports.SchedulePostUpdate) (ports.PostRecord, error) {
	f.scheduleCalls++
	f.scheduleTx = tx
	f.scheduleInput = input
	if f.scheduleErr != nil {
		return ports.PostRecord{}, f.scheduleErr
	}
	return f.scheduleResult, nil
}

func (f *fakePostRepository) CancelSchedule(ctx context.Context, tx ports.Tx, input ports.PostLifecycleUpdate) (ports.PostRecord, error) {
	f.cancelScheduleCalls++
	f.cancelScheduleTx = tx
	f.cancelScheduleInput = input
	if f.cancelScheduleErr != nil {
		return ports.PostRecord{}, f.cancelScheduleErr
	}
	return f.cancelScheduleResult, nil
}

func (f *fakePostRepository) UpdateDraftMeta(ctx context.Context, tx ports.Tx, input ports.UpdateDraftMetaUpdate) (ports.PostRecord, error) {
	f.updateMetaCalls++
	f.updateMetaTx = tx
	f.updateMetaInput = input
	if f.updateMetaErr != nil {
		return ports.PostRecord{}, f.updateMetaErr
	}
	return f.updateMetaResult, nil
}

func (f *fakePostRepository) DeleteDraft(ctx context.Context, tx ports.Tx, input ports.DeleteDraftUpdate) (ports.PostRecord, error) {
	f.deleteDraftCalls++
	f.deleteDraftTx = tx
	f.deleteDraftInput = input
	if f.deleteDraftErr != nil {
		return ports.PostRecord{}, f.deleteDraftErr
	}
	return f.deleteDraftResult, nil
}

func (f *fakePostRepository) GetPublishedBodyPointer(ctx context.Context, publicID string) (ports.PublishedBodyPointer, error) {
	f.bodyPointerCalls++
	f.bodyPointerPublic = publicID
	if f.bodyPointerErr != nil {
		return ports.PublishedBodyPointer{}, f.bodyPointerErr
	}
	return f.bodyPointerResult, nil
}

func (f *fakePostRepository) ListPublishedPosts(ctx context.Context, query ports.PostListQuery) ([]ports.PostSummaryRecord, error) {
	f.listPublishedCalls++
	f.listPublishedQuery = query
	if f.listPublishedErr != nil {
		return nil, f.listPublishedErr
	}
	return append([]ports.PostSummaryRecord(nil), f.listPublishedResult...), nil
}

func (f *fakePostRepository) GetPublishedPostDetail(ctx context.Context, publicID string) (ports.PostDetailRecord, error) {
	f.detailCalls++
	f.detailPublicID = publicID
	if f.detailErr != nil {
		return ports.PostDetailRecord{}, f.detailErr
	}
	return f.detailResult, nil
}

func (f *fakePostRepository) BatchGetPublishedPostSummaries(ctx context.Context, publicIDs []string) ([]ports.PostSummaryRecord, error) {
	f.batchCalls++
	f.batchIDs = append([]string(nil), publicIDs...)
	if f.batchErr != nil {
		return nil, f.batchErr
	}
	return append([]ports.PostSummaryRecord(nil), f.batchResult...), nil
}

func (f *fakePostRepository) ListAuthorPosts(ctx context.Context, query ports.AuthorPostListQuery) ([]ports.PostSummaryRecord, error) {
	f.listAuthorCalls++
	f.listAuthorQuery = query
	if f.listAuthorErr != nil {
		return nil, f.listAuthorErr
	}
	return append([]ports.PostSummaryRecord(nil), f.listAuthorResult...), nil
}

func (f *fakePostRepository) GetDraftPost(ctx context.Context, publicID string) (ports.DraftPostRecord, error) {
	f.draftCalls++
	f.draftPublicID = publicID
	if f.draftErr != nil {
		return ports.DraftPostRecord{}, f.draftErr
	}
	return f.draftResult, nil
}

func (f *fakePostRepository) IsBodyReferenced(ctx context.Context, bodyID string) (bool, error) {
	f.referenceChecks++
	f.referenceBodyID = bodyID
	return f.bodyReferenced, f.bodyReferenceErr
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
	afterDelete        func()
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
	if f.afterDelete != nil {
		f.afterDelete()
	}
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
	appendTxs          []ports.Tx
	appendOutsideCalls int
	tasks              []ports.BodyCleanupTask
	outsideTasks       []ports.BodyCleanupTask
	claimCalls         int
	claimRequests      []ports.TaskClaimRequest
	claimResults       [][]ports.BodyCleanupTaskClaim
	succeeded          []fakeTaskSuccess
	failed             []ports.TaskFailure
	err                error
}

type fakeRepairTaskStore struct {
	appendCalls        int
	appendOutsideCalls int
	tasks              []ports.BodyRepairTask
	outsideTasks       []ports.BodyRepairTask
	claimCalls         int
	claimRequests      []ports.TaskClaimRequest
	claimResults       [][]ports.BodyRepairTaskClaim
	succeeded          []fakeTaskSuccess
	failed             []ports.TaskFailure
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

func (f *fakeRepairTaskStore) Claim(ctx context.Context, request ports.TaskClaimRequest) ([]ports.BodyRepairTaskClaim, error) {
	f.claimCalls++
	f.claimRequests = append(f.claimRequests, request)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.claimResults) == 0 {
		return nil, nil
	}
	tasks := f.claimResults[0]
	f.claimResults = f.claimResults[1:]
	return tasks, nil
}

func (f *fakeRepairTaskStore) MarkSucceeded(ctx context.Context, taskID int64, workerID string, resolvedAt time.Time) error {
	f.succeeded = append(f.succeeded, fakeTaskSuccess{
		taskID: taskID,
		worker: workerID,
		at:     resolvedAt,
	})
	return f.err
}

func (f *fakeRepairTaskStore) MarkFailed(ctx context.Context, failure ports.TaskFailure) error {
	f.failed = append(f.failed, failure)
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
	f.appendTxs = append(f.appendTxs, tx)
	f.tasks = append(f.tasks, task)
	return f.err
}

func (f *fakeCleanupTaskStore) AppendOutsideTx(ctx context.Context, task ports.BodyCleanupTask) error {
	f.appendOutsideCalls++
	f.outsideTasks = append(f.outsideTasks, task)
	return f.err
}

func (f *fakeCleanupTaskStore) Claim(ctx context.Context, request ports.TaskClaimRequest) ([]ports.BodyCleanupTaskClaim, error) {
	f.claimCalls++
	f.claimRequests = append(f.claimRequests, request)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.claimResults) == 0 {
		return nil, nil
	}
	tasks := f.claimResults[0]
	f.claimResults = f.claimResults[1:]
	return tasks, nil
}

func (f *fakeCleanupTaskStore) MarkSucceeded(ctx context.Context, taskID int64, workerID string, completedAt time.Time) error {
	f.succeeded = append(f.succeeded, fakeTaskSuccess{
		taskID: taskID,
		worker: workerID,
		at:     completedAt,
	})
	return f.err
}

func (f *fakeCleanupTaskStore) MarkFailed(ctx context.Context, failure ports.TaskFailure) error {
	f.failed = append(f.failed, failure)
	return f.err
}

type fakeTaskSuccess struct {
	taskID int64
	worker string
	at     time.Time
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

type fakeTx struct {
	id int
}

func (f *fakeTxRunner) WithinTx(ctx context.Context, fn func(ctx context.Context, tx ports.Tx) error) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	return fn(ctx, fakeTx{id: f.calls})
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
