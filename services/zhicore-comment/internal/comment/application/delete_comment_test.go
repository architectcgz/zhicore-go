package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestDeleteCommentDeletesAuthorOwnedSubtreeAndPublishesSingleEvent(t *testing.T) {
	now := time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	root := store.seedComment(t, domain.CommentSeed{ID: 6101, PostID: "post_pub_delete", ContentInternalID: 9901, AuthorID: 42, Content: "root", Status: domain.CommentStatusNormal, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)})
	reply := store.seedComment(t, domain.CommentSeed{ID: 6102, PostID: "post_pub_delete", ContentInternalID: 9901, AuthorID: 43, RootID: root.ID, ParentID: root.ID, Content: "reply", Status: domain.CommentStatusNormal, CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now.Add(-30 * time.Minute)})
	store.stats[root.ID] = domain.CommentStats{CommentID: root.ID, ReplyCount: 1}
	store.stats[reply.ID] = domain.CommentStats{CommentID: reply.ID}
	store.postStats[root.PostID] = domain.CommentPostStats{PostID: root.PostID, TotalComments: 2, TotalTopLevelComments: 1}
	store.hotRanks[root.ID] = true
	store.recommendedRanks[root.ID] = true
	outbox := &fakeOutboxPublisher{}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: root.PostID, ContentInternalID: 9901, AuthorID: 42}},
		UserProfiles:  &fakeUserProfileClient{},
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{store: store},
		Outbox:        outbox,
		Clock:         fixedClock{now: now},
	})

	result, err := service.DeleteComment(context.Background(), DeleteCommentCommand{
		ActorUserID: 42,
		PostID:      "post_pub_delete",
		CommentID:   "c6101",
	})
	if err != nil {
		t.Fatalf("DeleteComment() error = %v", err)
	}

	if result.CommentID != "c6101" || result.AffectedCount != 2 || result.DeletedByRole != DeletedByRoleAuthor || result.AlreadyDeleted {
		t.Fatalf("DeleteComment() result = %#v", result)
	}
	if store.comments[root.ID].Status != domain.CommentStatusDeleted || store.comments[reply.ID].Status != domain.CommentStatusDeleted {
		t.Fatalf("comments after delete: root=%#v reply=%#v", store.comments[root.ID], store.comments[reply.ID])
	}
	if store.postStats[root.PostID].TotalComments != 0 || store.postStats[root.PostID].TotalTopLevelComments != 0 {
		t.Fatalf("post stats after delete = %#v", store.postStats[root.PostID])
	}
	if store.hotRanks[root.ID] || store.recommendedRanks[root.ID] {
		t.Fatalf("top level ranks should be hidden: hot=%v recommended=%v", store.hotRanks[root.ID], store.recommendedRanks[root.ID])
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("outbox messages = %d, want 1", len(outbox.messages))
	}
	assertOutboxPayload(t, outbox.messages[0], "comment.deleted", map[string]any{
		"commentId":     float64(root.ID),
		"publicId":      string(root.PostID),
		"internalId":    float64(root.ContentInternalID),
		"authorId":      float64(root.AuthorID),
		"deletedBy":     float64(42),
		"deletedByRole": "AUTHOR",
		"isRoot":        true,
		"affectedCount": float64(2),
	})
}

func TestDeleteCommentRejectsNonAuthorAndMissingComments(t *testing.T) {
	now := time.Date(2026, 7, 5, 9, 30, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		actor UserID
		seed  bool
		want  error
	}{
		{name: "non author", actor: 99, seed: true, want: ErrForbidden},
		{name: "missing", actor: 42, seed: false, want: ErrCommentNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeCommentStore()
			if tc.seed {
				store.seedComment(t, domain.CommentSeed{ID: 6201, PostID: "post_pub_delete", ContentInternalID: 9901, AuthorID: 42, Content: "root", Status: domain.CommentStatusNormal, CreatedAt: now, UpdatedAt: now})
			}
			outbox := &fakeOutboxPublisher{}
			service := mustNewService(t, Dependencies{
				Commands:      store,
				Queries:       store,
				Stats:         store,
				PostStats:     store,
				ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_delete", ContentInternalID: 9901, AuthorID: 42}},
				UserProfiles:  &fakeUserProfileClient{},
				UserRelations: &fakeUserRelationClient{},
				Files:         &fakeFileReferenceClient{},
				IDs:           publicIDCodec{},
				RateLimiter:   &fakeRateLimiter{},
				TxRunner:      &fakeTransactionRunner{store: store},
				Outbox:        outbox,
				Clock:         fixedClock{now: now},
			})

			_, err := service.DeleteComment(context.Background(), DeleteCommentCommand{ActorUserID: tc.actor, PostID: "post_pub_delete", CommentID: "c6201"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("DeleteComment() error = %v, want %v", err, tc.want)
			}
			if len(outbox.messages) != 0 {
				t.Fatalf("outbox messages = %d, want 0", len(outbox.messages))
			}
		})
	}
}

func TestAdminDeleteCommentIsIdempotentForAlreadyDeletedComment(t *testing.T) {
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	deleted := store.seedComment(t, domain.CommentSeed{ID: 6301, PostID: "post_pub_delete", ContentInternalID: 9901, AuthorID: 42, Content: "deleted", Status: domain.CommentStatusDeleted, CreatedAt: now, UpdatedAt: now})
	outbox := &fakeOutboxPublisher{}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: deleted.PostID, ContentInternalID: 9901, AuthorID: 42}},
		UserProfiles:  &fakeUserProfileClient{},
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{store: store},
		Outbox:        outbox,
		Clock:         fixedClock{now: now},
	})

	result, err := service.AdminDeleteComment(context.Background(), AdminDeleteCommentCommand{
		ActorUserID: 7,
		PostID:      "post_pub_delete",
		CommentID:   "c6301",
		Reason:      "spam",
	})
	if err != nil {
		t.Fatalf("AdminDeleteComment() error = %v", err)
	}

	if result.AffectedCount != 0 || !result.AlreadyDeleted || result.DeletedByRole != DeletedByRoleAdmin {
		t.Fatalf("AdminDeleteComment() result = %#v", result)
	}
	if len(outbox.messages) != 0 {
		t.Fatalf("outbox messages = %d, want 0", len(outbox.messages))
	}
}

func assertOutboxPayload(t *testing.T, message ports.OutboxMessage, eventType string, want map[string]any) {
	t.Helper()
	if message.EventType != eventType {
		t.Fatalf("event type = %q, want %q", message.EventType, eventType)
	}
	var payload map[string]any
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	for key, wantValue := range want {
		if got := payload[key]; got != wantValue {
			t.Fatalf("payload[%s] = %#v, want %#v; payload=%#v", key, got, wantValue, payload)
		}
	}
}
