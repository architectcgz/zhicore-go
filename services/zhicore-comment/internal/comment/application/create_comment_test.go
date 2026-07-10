package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestCreateCommentCreatesTopLevelCommentWithStatsRanksAndOutbox(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	outbox := &fakeOutboxPublisher{}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_1", ContentInternalID: 9001, AuthorID: 501}},
		UserProfiles:  &fakeUserProfileClient{summaries: map[domain.UserID]ports.AuthorSummary{42: {UserID: 42, PublicID: "user_pub_42", DisplayName: "Alice", AvatarURL: "https://cdn.example/avatar.png"}}},
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        outbox,
		Clock:         fixedClock{now: now},
	})

	result, err := service.CreateComment(context.Background(), CreateCommentCommand{
		ActorUserID: 42,
		PostID:      "post_pub_1",
		Content:     " 第一条评论 ",
		ImageFileIDs: []string{
			"img_1",
		},
	})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}

	if result.PostID != "post_pub_1" || result.CommentID != "c1001" || result.RootCommentID != "" || result.ParentCommentID != "" || !result.CreatedAt.Equal(now) {
		t.Fatalf("CreateComment() result = %#v", result)
	}
	created := store.mustComment(t, 1001)
	if !created.IsTopLevel() || created.Content != "第一条评论" || created.ContentInternalID != 9001 || created.AuthorID != 42 {
		t.Fatalf("created comment = %#v", created)
	}
	if store.stats[1001].ReplyCount != 0 || store.stats[1001].LikeCount != 0 {
		t.Fatalf("stats = %#v, want zero counts", store.stats[1001])
	}
	if store.postStats["post_pub_1"].TotalComments != 1 || store.postStats["post_pub_1"].TotalTopLevelComments != 1 {
		t.Fatalf("post stats = %#v", store.postStats["post_pub_1"])
	}
	if !store.hotRanks[1001] || !store.recommendedRanks[1001] {
		t.Fatalf("rank initialization missing: hot=%v recommended=%v", store.hotRanks[1001], store.recommendedRanks[1001])
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("outbox message count = %d, want 1", len(outbox.messages))
	}
	assertCreatedOutboxPayload(t, outbox.messages[0], map[string]any{
		"commentId":    float64(1001),
		"publicId":     "post_pub_1",
		"internalId":   float64(9001),
		"authorId":     float64(42),
		"postAuthorId": float64(501),
		"hasImages":    true,
		"hasVoice":     false,
		"actor":        map[string]any{"publicId": "user_pub_42", "displayName": "Alice", "avatarUrl": "https://cdn.example/avatar.png"},
	})
}

func TestCreateCommentCreatesReplyAfterCheckingParentInsideTransaction(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 30, 0, 0, time.UTC)
	store := newFakeCommentStore()
	root := store.seedComment(t, domain.CommentSeed{ID: 2001, PostID: "post_pub_2", ContentInternalID: 9002, AuthorID: 501, Content: "root", Status: domain.CommentStatusNormal, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)})
	parent := store.seedComment(t, domain.CommentSeed{ID: 2002, PostID: "post_pub_2", ContentInternalID: 9002, AuthorID: 502, RootID: root.ID, ParentID: root.ID, Content: "parent", Status: domain.CommentStatusNormal, CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now.Add(-30 * time.Minute)})
	store.nextID = 2003
	outbox := &fakeOutboxPublisher{}
	relations := &fakeUserRelationClient{}
	txRunner := &fakeTransactionRunner{store: store}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_2", ContentInternalID: 9002, AuthorID: 501}},
		UserProfiles:  &fakeUserProfileClient{summaries: map[domain.UserID]ports.AuthorSummary{44: {UserID: 44, PublicID: "user_pub_44", DisplayName: "Bob"}}},
		UserRelations: relations,
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      txRunner,
		Outbox:        outbox,
		Clock:         fixedClock{now: now},
	})

	result, err := service.CreateComment(context.Background(), CreateCommentCommand{
		ActorUserID:     44,
		PostID:          "post_pub_2",
		ParentCommentID: "c2002",
		Content:         "reply",
	})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}

	if result.CommentID != "c2003" || result.RootCommentID != "c2001" || result.ParentCommentID != "c2002" {
		t.Fatalf("CreateComment() result = %#v", result)
	}
	reply := store.mustComment(t, 2003)
	if reply.RootID != root.ID || reply.ParentID != parent.ID || !reply.IsReply() {
		t.Fatalf("reply tree = %#v", reply)
	}
	if store.stats[root.ID].ReplyCount != 1 {
		t.Fatalf("root reply count = %d, want 1", store.stats[root.ID].ReplyCount)
	}
	if store.postStats["post_pub_2"].TotalComments != 1 || store.postStats["post_pub_2"].TotalTopLevelComments != 0 {
		t.Fatalf("post stats = %#v", store.postStats["post_pub_2"])
	}
	if txRunner.calledCount != 1 || !store.parentLoadedInTx {
		t.Fatalf("parent load transaction state: txCalls=%d loadedInTx=%v", txRunner.calledCount, store.parentLoadedInTx)
	}
	if !relations.checkedPair(501, 44) || !relations.checkedPair(502, 44) {
		t.Fatalf("blocked relation checks = %#v, want post author and parent author", relations.pairs)
	}
	assertCreatedOutboxPayload(t, outbox.messages[0], map[string]any{
		"commentId":      float64(2003),
		"rootId":         float64(2001),
		"parentId":       float64(2002),
		"rootAuthorId":   float64(501),
		"parentAuthorId": float64(502),
	})
}

func TestCreateCommentRejectsInvalidContentBeforeExternalGuards(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  CreateCommentCommand
		want error
	}{
		{name: "empty post", cmd: CreateCommentCommand{ActorUserID: 1, Content: "hello"}, want: ErrInvalidRequest},
		{name: "empty", cmd: CreateCommentCommand{ActorUserID: 1, PostID: "post", Content: " \t "}, want: ErrCommentContentRequired},
		{name: "too long", cmd: CreateCommentCommand{ActorUserID: 1, PostID: "post", Content: strings.Repeat("界", 2001)}, want: ErrCommentContentTooLong},
		{name: "too many images", cmd: CreateCommentCommand{ActorUserID: 1, PostID: "post", ImageFileIDs: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}}, want: ErrInvalidRequest},
		{name: "voice and images", cmd: CreateCommentCommand{ActorUserID: 1, PostID: "post", ImageFileIDs: []string{"img"}, VoiceFileID: "voice", VoiceDuration: 8}, want: ErrInvalidRequest},
		{name: "voice duration missing", cmd: CreateCommentCommand{ActorUserID: 1, PostID: "post", VoiceFileID: "voice"}, want: ErrInvalidRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contentClient := &fakeContentPostClient{post: ports.CommentablePost{PostID: "post", ContentInternalID: 1, AuthorID: 2}}
			txRunner := &fakeTransactionRunner{}
			service := mustNewService(t, Dependencies{
				Commands:      newFakeCommentStore(),
				Queries:       newFakeCommentStore(),
				Stats:         newFakeCommentStore(),
				PostStats:     newFakeCommentStore(),
				ContentPosts:  contentClient,
				UserProfiles:  &fakeUserProfileClient{},
				UserRelations: &fakeUserRelationClient{},
				Files:         &fakeFileReferenceClient{},
				IDs:           publicIDCodec{},
				RateLimiter:   &fakeRateLimiter{},
				TxRunner:      txRunner,
				Outbox:        &fakeOutboxPublisher{},
				Clock:         fixedClock{now: time.Now()},
			})

			_, err := service.CreateComment(context.Background(), tc.cmd)
			if !errors.Is(err, tc.want) {
				t.Fatalf("CreateComment() error = %v, want %v", err, tc.want)
			}
			if contentClient.calls != 0 || txRunner.calledCount != 0 {
				t.Fatalf("external guard calls=%d txCalls=%d, want 0", contentClient.calls, txRunner.calledCount)
			}
		})
	}
}

func TestCreateCommentRejectsDeletedParentWithoutWriting(t *testing.T) {
	now := time.Date(2026, 7, 4, 13, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	deleted := store.seedComment(t, domain.CommentSeed{ID: 3001, PostID: "post_pub_3", ContentInternalID: 9003, AuthorID: 501, Content: "deleted", Status: domain.CommentStatusDeleted, CreatedAt: now, UpdatedAt: now})
	store.nextID = 3002
	txRunner := &fakeTransactionRunner{store: store}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_3", ContentInternalID: 9003, AuthorID: 501}},
		UserProfiles:  &fakeUserProfileClient{},
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      txRunner,
		Outbox:        &fakeOutboxPublisher{},
		Clock:         fixedClock{now: now},
	})

	_, err := service.CreateComment(context.Background(), CreateCommentCommand{
		ActorUserID:     44,
		PostID:          "post_pub_3",
		ParentCommentID: PublicCommentID(publicIDCodec{}.Encode(deleted.ID)),
		Content:         "reply",
	})
	if !errors.Is(err, ErrParentCommentNotFound) {
		t.Fatalf("CreateComment() error = %v, want %v", err, ErrParentCommentNotFound)
	}
	if store.createCalls != 0 || len(store.comments) != 1 || len(store.postStats) != 0 {
		t.Fatalf("store mutated after deleted parent: createCalls=%d stats=%#v postStats=%#v", store.createCalls, store.stats, store.postStats)
	}
	if txRunner.calledCount != 1 || !store.parentLoadedInTx {
		t.Fatalf("deleted parent must be checked inside transaction: txCalls=%d loadedInTx=%v", txRunner.calledCount, store.parentLoadedInTx)
	}
}

func TestCreateCommentFailsClosedWhenExternalGuardsFail(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakeContentPostClient, *fakeUserProfileClient, *fakeUserRelationClient, *fakeFileReferenceClient)
		want  error
	}{
		{name: "content unavailable", setup: func(c *fakeContentPostClient, _ *fakeUserProfileClient, _ *fakeUserRelationClient, _ *fakeFileReferenceClient) {
			c.err = ports.ErrDependencyUnavailable
		}, want: ErrDependencyUnavailable},
		{name: "post not found", setup: func(c *fakeContentPostClient, _ *fakeUserProfileClient, _ *fakeUserRelationClient, _ *fakeFileReferenceClient) {
			c.err = ErrPostNotFound
		}, want: ErrPostNotFound},
		{name: "user unavailable", setup: func(_ *fakeContentPostClient, u *fakeUserProfileClient, _ *fakeUserRelationClient, _ *fakeFileReferenceClient) {
			u.err = ports.ErrDependencyUnavailable
		}, want: ErrDependencyUnavailable},
		{name: "blocked", setup: func(_ *fakeContentPostClient, _ *fakeUserProfileClient, r *fakeUserRelationClient, _ *fakeFileReferenceClient) {
			r.blocked = true
		}, want: ErrInteractionBlocked},
		{name: "file unavailable", setup: func(_ *fakeContentPostClient, _ *fakeUserProfileClient, _ *fakeUserRelationClient, f *fakeFileReferenceClient) {
			f.err = ports.ErrDependencyUnavailable
		}, want: ErrDependencyUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeCommentStore()
			contentClient := &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_4", ContentInternalID: 9004, AuthorID: 501}}
			userClient := &fakeUserProfileClient{}
			relationClient := &fakeUserRelationClient{}
			fileClient := &fakeFileReferenceClient{}
			tc.setup(contentClient, userClient, relationClient, fileClient)
			txRunner := &fakeTransactionRunner{}
			service := mustNewService(t, Dependencies{
				Commands:      store,
				Queries:       store,
				Stats:         store,
				PostStats:     store,
				ContentPosts:  contentClient,
				UserProfiles:  userClient,
				UserRelations: relationClient,
				Files:         fileClient,
				IDs:           publicIDCodec{},
				RateLimiter:   &fakeRateLimiter{},
				TxRunner:      txRunner,
				Outbox:        &fakeOutboxPublisher{},
				Clock:         fixedClock{now: time.Now()},
			})

			_, err := service.CreateComment(context.Background(), CreateCommentCommand{ActorUserID: 44, PostID: "post_pub_4", Content: "hello", ImageFileIDs: []string{"img_1"}})
			if !errors.Is(err, tc.want) {
				t.Fatalf("CreateComment() error = %v, want %v", err, tc.want)
			}
			if txRunner.calledCount != 0 || store.createCalls != 0 {
				t.Fatalf("guard failure wrote local state: txCalls=%d createCalls=%d", txRunner.calledCount, store.createCalls)
			}
		})
	}
}

func assertCreatedOutboxPayload(t *testing.T, message ports.OutboxMessage, want map[string]any) {
	t.Helper()
	if message.EventType != "comment.created" {
		t.Fatalf("event type = %q, want comment.created", message.EventType)
	}
	var payload map[string]any
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	for key, wantValue := range want {
		if got := payload[key]; !reflect.DeepEqual(got, wantValue) {
			t.Fatalf("payload[%s] = %#v, want %#v; payload=%#v", key, got, wantValue, payload)
		}
	}
}

func mustNewService(t *testing.T, deps Dependencies) *Service {
	t.Helper()
	service, err := NewService(deps)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fakeCommentStore struct {
	nextID             int64
	comments           map[domain.CommentID]domain.Comment
	stats              map[domain.CommentID]domain.CommentStats
	postStats          map[domain.PostID]domain.CommentPostStats
	hotRanks           map[domain.CommentID]bool
	recommendedRanks   map[domain.CommentID]bool
	likes              map[domain.CommentID]map[domain.UserID]bool
	counterDeltas      []ports.CommentCounterDelta
	queryResults       map[domain.CommentSort][]domain.Comment
	viewerLiked        map[domain.CommentID]bool
	lastListQuery      ports.TopLevelCommentPageQuery
	lastReplyQuery     ports.ReplyCommentPageQuery
	createCalls        int
	inTx               bool
	parentLoadedInTx   bool
	failCreate         error
	failInitializeRank error
}

func newFakeCommentStore() *fakeCommentStore {
	return &fakeCommentStore{
		nextID:           1001,
		comments:         map[domain.CommentID]domain.Comment{},
		stats:            map[domain.CommentID]domain.CommentStats{},
		postStats:        map[domain.PostID]domain.CommentPostStats{},
		hotRanks:         map[domain.CommentID]bool{},
		recommendedRanks: map[domain.CommentID]bool{},
		likes:            map[domain.CommentID]map[domain.UserID]bool{},
		queryResults:     map[domain.CommentSort][]domain.Comment{},
		viewerLiked:      map[domain.CommentID]bool{},
	}
}

func (s *fakeCommentStore) seedComment(t *testing.T, seed domain.CommentSeed) domain.Comment {
	t.Helper()
	comment, err := domain.NewComment(seed)
	if err != nil {
		t.Fatalf("domain.NewComment() error = %v", err)
	}
	s.comments[comment.ID] = comment
	s.stats[comment.ID] = domain.CommentStats{CommentID: comment.ID}
	if int64(comment.ID) >= s.nextID {
		s.nextID = int64(comment.ID) + 1
	}
	return comment
}

func (s *fakeCommentStore) mustComment(t *testing.T, id domain.CommentID) domain.Comment {
	t.Helper()
	comment, ok := s.comments[id]
	if !ok {
		t.Fatalf("comment %d missing", id)
	}
	return comment
}

func (s *fakeCommentStore) FindReplyTarget(ctx context.Context, postID domain.PostID, parentID domain.CommentID) (ports.ReplyTarget, error) {
	if s.inTx {
		s.parentLoadedInTx = true
	}
	comment, ok := s.comments[parentID]
	if !ok || comment.PostID != postID || comment.Status != domain.CommentStatusNormal {
		return ports.ReplyTarget{}, domain.ErrParentCommentNotFound
	}
	root := comment
	if comment.IsReply() {
		rootComment, ok := s.comments[comment.RootID]
		if !ok || rootComment.PostID != postID || rootComment.Status != domain.CommentStatusNormal || !rootComment.IsTopLevel() {
			return ports.ReplyTarget{}, domain.ErrRootCommentNotFound
		}
		root = rootComment
	}
	return ports.ReplyTarget{Parent: comment, Root: root}, nil
}

func (s *fakeCommentStore) FindReplyGuardPreview(ctx context.Context, postID domain.PostID, parentID domain.CommentID) (ports.ReplyGuardPreview, bool, error) {
	comment, ok := s.comments[parentID]
	if !ok || comment.PostID != postID || comment.Status != domain.CommentStatusNormal {
		return ports.ReplyGuardPreview{}, false, nil
	}
	return ports.ReplyGuardPreview{ParentAuthorID: comment.AuthorID}, true, nil
}

func (s *fakeCommentStore) Create(ctx context.Context, draft domain.Comment) (domain.Comment, error) {
	s.createCalls++
	if s.failCreate != nil {
		return domain.Comment{}, s.failCreate
	}
	draft.ID = domain.CommentID(s.nextID)
	s.nextID++
	s.comments[draft.ID] = draft
	return draft, nil
}

func (s *fakeCommentStore) FindCommentForMutation(ctx context.Context, postID domain.PostID, commentID domain.CommentID) (domain.Comment, error) {
	comment, ok := s.comments[commentID]
	if !ok || comment.PostID != postID {
		return domain.Comment{}, domain.ErrCommentNotFound
	}
	return comment, nil
}

func (s *fakeCommentStore) SoftDeleteSubtree(ctx context.Context, input ports.DeleteSubtreeInput) (ports.DeleteSubtreeResult, error) {
	entry, ok := s.comments[input.CommentID]
	if !ok || entry.PostID != input.PostID {
		return ports.DeleteSubtreeResult{}, domain.ErrCommentNotFound
	}
	if entry.Status == domain.CommentStatusDeleted {
		if !input.AllowDeleted {
			return ports.DeleteSubtreeResult{}, domain.ErrCommentNotFound
		}
		rootID := entry.RootID
		if entry.IsTopLevel() {
			rootID = entry.ID
		}
		return ports.DeleteSubtreeResult{Entry: entry, RootID: rootID, AlreadyDeleted: true}, nil
	}
	affected := 0
	for id, comment := range s.comments {
		if comment.PostID != input.PostID || comment.Status != domain.CommentStatusNormal {
			continue
		}
		if id != entry.ID && comment.RootID != entry.ID && comment.ParentID != entry.ID && comment.RootID != entry.RootID {
			continue
		}
		if entry.IsReply() && id != entry.ID && comment.RootID != entry.RootID {
			continue
		}
		comment.Status = domain.CommentStatusDeleted
		s.comments[id] = comment
		affected++
	}
	rootID := entry.RootID
	if entry.IsTopLevel() {
		rootID = entry.ID
	}
	return ports.DeleteSubtreeResult{Entry: entry, RootID: rootID, AffectedCount: affected}, nil
}

func (s *fakeCommentStore) Initialize(ctx context.Context, commentID domain.CommentID, now time.Time) error {
	s.stats[commentID] = domain.CommentStats{CommentID: commentID}
	return nil
}

func (s *fakeCommentStore) IncrementReplyCount(ctx context.Context, rootID domain.CommentID, now time.Time) error {
	stats := s.stats[rootID]
	stats.CommentID = rootID
	stats.ReplyCount++
	s.stats[rootID] = stats
	return nil
}

func (s *fakeCommentStore) DecrementReplyCount(ctx context.Context, rootID domain.CommentID, by int, now time.Time) error {
	stats := s.stats[rootID]
	stats.CommentID = rootID
	stats.ReplyCount -= int64(by)
	if stats.ReplyCount < 0 {
		stats.ReplyCount = 0
	}
	s.stats[rootID] = stats
	return nil
}

func (s *fakeCommentStore) IncrementForTopLevel(ctx context.Context, postID domain.PostID, now time.Time) error {
	stats := s.postStats[postID]
	stats.PostID = postID
	stats.TotalComments++
	stats.TotalTopLevelComments++
	s.postStats[postID] = stats
	return nil
}

func (s *fakeCommentStore) IncrementForReply(ctx context.Context, postID domain.PostID, now time.Time) error {
	stats := s.postStats[postID]
	stats.PostID = postID
	stats.TotalComments++
	s.postStats[postID] = stats
	return nil
}

func (s *fakeCommentStore) DecrementForDelete(ctx context.Context, postID domain.PostID, affectedCount int, topLevelDeleted bool, now time.Time) error {
	stats := s.postStats[postID]
	stats.PostID = postID
	stats.TotalComments -= int64(affectedCount)
	if stats.TotalComments < 0 {
		stats.TotalComments = 0
	}
	if topLevelDeleted {
		stats.TotalTopLevelComments--
		if stats.TotalTopLevelComments < 0 {
			stats.TotalTopLevelComments = 0
		}
	}
	s.postStats[postID] = stats
	return nil
}

func (s *fakeCommentStore) InitializeTopLevelRanks(ctx context.Context, comment domain.Comment, now time.Time) error {
	if s.failInitializeRank != nil {
		return s.failInitializeRank
	}
	s.hotRanks[comment.ID] = true
	s.recommendedRanks[comment.ID] = true
	return nil
}

func (s *fakeCommentStore) HideTopLevelRanks(ctx context.Context, commentID domain.CommentID, now time.Time) error {
	s.hotRanks[commentID] = false
	s.recommendedRanks[commentID] = false
	return nil
}

func (s *fakeCommentStore) UpsertLike(ctx context.Context, input ports.LikeMutationInput) (bool, error) {
	if _, err := s.FindCommentForMutation(ctx, input.PostID, input.CommentID); err != nil {
		return false, err
	}
	if s.likes[input.CommentID] == nil {
		s.likes[input.CommentID] = map[domain.UserID]bool{}
	}
	if s.likes[input.CommentID][input.UserID] {
		return false, nil
	}
	s.likes[input.CommentID][input.UserID] = true
	return true, nil
}

func (s *fakeCommentStore) DeleteLike(ctx context.Context, input ports.LikeMutationInput) (bool, error) {
	if _, err := s.FindCommentForMutation(ctx, input.PostID, input.CommentID); err != nil {
		return false, err
	}
	if s.likes[input.CommentID] == nil || !s.likes[input.CommentID][input.UserID] {
		return false, nil
	}
	delete(s.likes[input.CommentID], input.UserID)
	return true, nil
}

func (s *fakeCommentStore) AppendCounterDelta(ctx context.Context, delta ports.CommentCounterDelta) error {
	s.counterDeltas = append(s.counterDeltas, delta)
	return nil
}

func (s *fakeCommentStore) ListTopLevelComments(ctx context.Context, query ports.TopLevelCommentPageQuery) (ports.TopLevelCommentPage, error) {
	s.lastListQuery = query
	comments := s.queryResults[query.Sort]
	items := make([]ports.TopLevelCommentRecord, 0, len(comments))
	for _, comment := range comments {
		items = append(items, ports.TopLevelCommentRecord{Comment: comment, Stats: s.stats[comment.ID]})
	}
	return ports.TopLevelCommentPage{Items: items}, nil
}

func (s *fakeCommentStore) GetCommentDetail(ctx context.Context, postID domain.PostID, commentID domain.CommentID) (ports.TopLevelCommentRecord, error) {
	comment, ok := s.comments[commentID]
	if !ok || comment.PostID != postID || comment.Status != domain.CommentStatusNormal {
		return ports.TopLevelCommentRecord{}, domain.ErrCommentNotFound
	}
	return ports.TopLevelCommentRecord{Comment: comment, Stats: s.stats[comment.ID]}, nil
}

func (s *fakeCommentStore) ListRepliesByPage(ctx context.Context, query ports.ReplyCommentPageQuery) (ports.ReplyCommentPage, error) {
	s.lastReplyQuery = query
	root, ok := s.comments[query.RootID]
	if !ok || root.PostID != query.PostID || !root.IsTopLevel() || root.Status != domain.CommentStatusNormal {
		return ports.ReplyCommentPage{}, domain.ErrRootCommentNotFound
	}

	replies := make([]domain.Comment, 0)
	for _, comment := range s.comments {
		if comment.PostID == query.PostID && comment.RootID == query.RootID && comment.IsReply() && comment.Status == domain.CommentStatusNormal {
			replies = append(replies, comment)
		}
	}
	sort.Slice(replies, func(i, j int) bool {
		if query.Sort == domain.CommentSortHot {
			left := s.stats[replies[i].ID].LikeCount
			right := s.stats[replies[j].ID].LikeCount
			if left != right {
				return left > right
			}
		}
		return replies[i].ID < replies[j].ID
	})

	page := query.Page
	if page < 1 {
		page = 1
	}
	size := query.Size
	if size < 1 {
		size = 20
	}
	offset := (page - 1) * size
	if offset > len(replies) {
		offset = len(replies)
	}
	end := offset + size
	if end > len(replies) {
		end = len(replies)
	}
	items := make([]ports.TopLevelCommentRecord, 0, end-offset)
	for _, comment := range replies[offset:end] {
		items = append(items, ports.TopLevelCommentRecord{Comment: comment, Stats: s.stats[comment.ID]})
	}
	return ports.ReplyCommentPage{Items: items, Total: int64(len(replies))}, nil
}

func (s *fakeCommentStore) BatchGetViewerLiked(ctx context.Context, viewerID domain.UserID, commentIDs []domain.CommentID) (map[domain.CommentID]bool, error) {
	result := map[domain.CommentID]bool{}
	for _, id := range commentIDs {
		result[id] = s.viewerLiked[id]
	}
	return result, nil
}

func (s *fakeCommentStore) Get(ctx context.Context, postID domain.PostID) (domain.CommentPostStats, error) {
	return s.postStats[postID], nil
}

type fakeContentPostClient struct {
	post  ports.CommentablePost
	err   error
	calls int
}

func (f *fakeContentPostClient) CheckPostCommentable(ctx context.Context, postID domain.PostID) (ports.CommentablePost, error) {
	f.calls++
	if f.err != nil {
		return ports.CommentablePost{}, f.err
	}
	return f.post, nil
}

type fakeUserProfileClient struct {
	err       error
	batchErr  error
	calls     int
	summaries map[domain.UserID]ports.AuthorSummary
}

func (f *fakeUserProfileClient) EnsureUserCanComment(ctx context.Context, userID domain.UserID) error {
	f.calls++
	return f.err
}

func (f *fakeUserProfileClient) BatchGetAuthorSummaries(ctx context.Context, userIDs []domain.UserID) (map[domain.UserID]ports.AuthorSummary, error) {
	if f.batchErr != nil {
		return nil, f.batchErr
	}
	result := map[domain.UserID]ports.AuthorSummary{}
	for _, userID := range userIDs {
		if summary, ok := f.summaries[userID]; ok {
			result[userID] = summary
			continue
		}
		result[userID] = ports.AuthorSummary{UserID: userID, Unavailable: true}
	}
	return result, nil
}

type fakeUserRelationClient struct {
	pairs   []ports.BlockPair
	blocked bool
	err     error
}

func (f *fakeUserRelationClient) BatchCheckBlocked(ctx context.Context, pairs []ports.BlockPair) (map[ports.BlockPair]bool, error) {
	f.pairs = append(f.pairs, pairs...)
	if f.err != nil {
		return nil, f.err
	}
	result := map[ports.BlockPair]bool{}
	for _, pair := range pairs {
		result[pair] = f.blocked
	}
	return result, nil
}

func (f *fakeUserRelationClient) checkedPair(blocker, blocked domain.UserID) bool {
	for _, pair := range f.pairs {
		if pair.BlockerID == blocker && pair.BlockedID == blocked {
			return true
		}
	}
	return false
}

type fakeFileReferenceClient struct {
	err   error
	input ports.CommentMediaReferences
}

func (f *fakeFileReferenceClient) EnsureCommentMediaReferenced(ctx context.Context, input ports.CommentMediaReferences) error {
	f.input = input
	return f.err
}

type fakeRateLimiter struct{ err error }

func (f *fakeRateLimiter) AllowCreateComment(ctx context.Context, input ports.CreateCommentRateLimitInput) error {
	return f.err
}

type fakeTransactionRunner struct {
	calledCount int
	store       *fakeCommentStore
}

func (f *fakeTransactionRunner) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	f.calledCount++
	if f.store != nil {
		f.store.inTx = true
		defer func() { f.store.inTx = false }()
	}
	return fn(ctx)
}

type fakeOutboxPublisher struct {
	messages []ports.OutboxMessage
}

func (f *fakeOutboxPublisher) Publish(ctx context.Context, message ports.OutboxMessage) error {
	f.messages = append(f.messages, message)
	return nil
}

type publicIDCodec struct{}

func (publicIDCodec) Encode(id domain.CommentID) domain.PublicCommentID {
	return domain.PublicCommentID("c" + strconv.FormatInt(int64(id), 10))
}

func (publicIDCodec) Decode(publicID domain.PublicCommentID) (domain.CommentID, error) {
	raw := strings.TrimPrefix(string(publicID), "c")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ErrCommentIDInvalid
	}
	return domain.CommentID(id), nil
}
