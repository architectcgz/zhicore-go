package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestLikeCommentWritesLikeDeltaAndOutboxOnce(t *testing.T) {
	now := time.Date(2026, 7, 5, 11, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	comment := store.seedComment(t, domain.CommentSeed{ID: 7101, PostID: "post_pub_like", ContentInternalID: 9911, AuthorID: 501, Content: "comment", Status: domain.CommentStatusNormal, CreatedAt: now, UpdatedAt: now})
	outbox := &fakeOutboxPublisher{}
	relations := &fakeUserRelationClient{}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: comment.PostID, ContentInternalID: 9911, AuthorID: 900}},
		UserProfiles:  &fakeUserProfileClient{},
		UserRelations: relations,
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{store: store},
		Outbox:        outbox,
		Clock:         fixedClock{now: now},
	})

	first, err := service.LikeComment(context.Background(), LikeCommentCommand{ActorUserID: 77, PostID: "post_pub_like", CommentID: "c7101"})
	if err != nil {
		t.Fatalf("LikeComment() first error = %v", err)
	}
	second, err := service.LikeComment(context.Background(), LikeCommentCommand{ActorUserID: 77, PostID: "post_pub_like", CommentID: "c7101"})
	if err != nil {
		t.Fatalf("LikeComment() second error = %v", err)
	}

	if !first.Liked || !first.Changed || !second.Liked || second.Changed {
		t.Fatalf("like results: first=%#v second=%#v", first, second)
	}
	if len(store.counterDeltas) != 1 || store.counterDeltas[0].DeltaValue != 1 {
		t.Fatalf("counter deltas = %#v, want single +1", store.counterDeltas)
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("outbox messages = %d, want 1", len(outbox.messages))
	}
	if !relations.checkedPair(comment.AuthorID, 77) {
		t.Fatalf("like must check comment author block relation, pairs=%#v", relations.pairs)
	}
	assertOutboxPayload(t, outbox.messages[0], "comment.liked", map[string]any{
		"commentId":       float64(comment.ID),
		"publicId":        string(comment.PostID),
		"internalId":      float64(comment.ContentInternalID),
		"commentAuthorId": float64(comment.AuthorID),
		"likedBy":         float64(77),
	})
}

func TestLikeCommentFailsClosedWhenRelationUnavailableOrBlocked(t *testing.T) {
	for _, tc := range []struct {
		name     string
		relation *fakeUserRelationClient
		want     error
	}{
		{name: "relation unavailable", relation: &fakeUserRelationClient{err: ports.ErrDependencyUnavailable}, want: ErrDependencyUnavailable},
		{name: "blocked", relation: &fakeUserRelationClient{blocked: true}, want: ErrInteractionBlocked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 7, 5, 11, 30, 0, 0, time.UTC)
			store := newFakeCommentStore()
			comment := store.seedComment(t, domain.CommentSeed{ID: 7201, PostID: "post_pub_like", ContentInternalID: 9911, AuthorID: 501, Content: "comment", Status: domain.CommentStatusNormal, CreatedAt: now, UpdatedAt: now})
			outbox := &fakeOutboxPublisher{}
			service := mustNewService(t, Dependencies{
				Commands:      store,
				Queries:       store,
				Stats:         store,
				PostStats:     store,
				ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: comment.PostID, ContentInternalID: 9911, AuthorID: 900}},
				UserProfiles:  &fakeUserProfileClient{},
				UserRelations: tc.relation,
				Files:         &fakeFileReferenceClient{},
				IDs:           publicIDCodec{},
				RateLimiter:   &fakeRateLimiter{},
				TxRunner:      &fakeTransactionRunner{store: store},
				Outbox:        outbox,
				Clock:         fixedClock{now: now},
			})

			_, err := service.LikeComment(context.Background(), LikeCommentCommand{ActorUserID: 77, PostID: "post_pub_like", CommentID: "c7201"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("LikeComment() error = %v, want %v", err, tc.want)
			}
			if len(store.counterDeltas) != 0 || len(outbox.messages) != 0 {
				t.Fatalf("mutation after failure: deltas=%#v outbox=%#v", store.counterDeltas, outbox.messages)
			}
		})
	}
}

func TestUnlikeCommentDeletesLikeDeltaAndOutboxOnceWithoutRelationGuard(t *testing.T) {
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	comment := store.seedComment(t, domain.CommentSeed{ID: 7301, PostID: "post_pub_like", ContentInternalID: 9911, AuthorID: 501, Content: "comment", Status: domain.CommentStatusNormal, CreatedAt: now, UpdatedAt: now})
	store.likes[comment.ID] = map[domain.UserID]bool{77: true}
	outbox := &fakeOutboxPublisher{}
	relations := &fakeUserRelationClient{err: ports.ErrDependencyUnavailable}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: comment.PostID, ContentInternalID: 9911, AuthorID: 900}},
		UserProfiles:  &fakeUserProfileClient{},
		UserRelations: relations,
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{store: store},
		Outbox:        outbox,
		Clock:         fixedClock{now: now},
	})

	first, err := service.UnlikeComment(context.Background(), UnlikeCommentCommand{ActorUserID: 77, PostID: "post_pub_like", CommentID: "c7301"})
	if err != nil {
		t.Fatalf("UnlikeComment() first error = %v", err)
	}
	second, err := service.UnlikeComment(context.Background(), UnlikeCommentCommand{ActorUserID: 77, PostID: "post_pub_like", CommentID: "c7301"})
	if err != nil {
		t.Fatalf("UnlikeComment() second error = %v", err)
	}

	if first.Liked || !first.Changed || second.Liked || second.Changed {
		t.Fatalf("unlike results: first=%#v second=%#v", first, second)
	}
	if len(relations.pairs) != 0 {
		t.Fatalf("unlike must not check relation guard, pairs=%#v", relations.pairs)
	}
	if len(store.counterDeltas) != 1 || store.counterDeltas[0].DeltaValue != -1 {
		t.Fatalf("counter deltas = %#v, want single -1", store.counterDeltas)
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("outbox messages = %d, want 1", len(outbox.messages))
	}
	assertOutboxPayload(t, outbox.messages[0], "comment.unliked", map[string]any{
		"commentId":       float64(comment.ID),
		"publicId":        string(comment.PostID),
		"internalId":      float64(comment.ContentInternalID),
		"commentAuthorId": float64(comment.AuthorID),
		"unlikedBy":       float64(77),
	})
}
