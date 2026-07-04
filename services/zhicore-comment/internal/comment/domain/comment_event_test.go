package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
)

func TestTopLevelCommentCreatedCarriesOnlyCreatedComment(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	comment := mustComment(t, domain.CommentSeed{
		ID:                1001,
		PostID:            "post_pub_1",
		ContentInternalID: 9001,
		AuthorID:          501,
		Content:           "root",
		Status:            domain.CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})

	event, err := domain.NewTopLevelCommentCreated(comment)
	if err != nil {
		t.Fatalf("NewTopLevelCommentCreated() error = %v", err)
	}
	if !reflect.DeepEqual(event.CreatedComment(), comment) {
		t.Fatalf("CreatedComment() = %+v, want %+v", event.CreatedComment(), comment)
	}
	if _, ok := event.RootComment(); ok {
		t.Fatalf("RootComment() ok = true, want false for top-level comment")
	}
	if _, ok := event.ParentComment(); ok {
		t.Fatalf("ParentComment() ok = true, want false for top-level comment")
	}
}

func TestReplyCreatedCarriesRootAndParentFacts(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	root := mustComment(t, domain.CommentSeed{
		ID:                2001,
		PostID:            "post_pub_2",
		ContentInternalID: 9002,
		AuthorID:          601,
		Content:           "root",
		Status:            domain.CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	parent := mustComment(t, domain.CommentSeed{
		ID:                2002,
		PostID:            root.PostID,
		ContentInternalID: root.ContentInternalID,
		AuthorID:          602,
		RootID:            root.ID,
		ParentID:          root.ID,
		Content:           "parent",
		Status:            domain.CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	reply := mustComment(t, domain.CommentSeed{
		ID:                2003,
		PostID:            root.PostID,
		ContentInternalID: root.ContentInternalID,
		AuthorID:          603,
		RootID:            root.ID,
		ParentID:          parent.ID,
		Content:           "reply",
		Status:            domain.CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})

	event, err := domain.NewReplyCreated(reply, root, parent)
	if err != nil {
		t.Fatalf("NewReplyCreated() error = %v", err)
	}
	if !reflect.DeepEqual(event.CreatedComment(), reply) {
		t.Fatalf("CreatedComment() = %+v, want %+v", event.CreatedComment(), reply)
	}
	if got, ok := event.RootComment(); !ok || !reflect.DeepEqual(got, root) {
		t.Fatalf("RootComment() = %+v, %v; want %+v, true", got, ok, root)
	}
	if got, ok := event.ParentComment(); !ok || !reflect.DeepEqual(got, parent) {
		t.Fatalf("ParentComment() = %+v, %v; want %+v, true", got, ok, parent)
	}
}

func TestReplyCreatedRejectsMismatchedTreeFacts(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	root := mustComment(t, domain.CommentSeed{
		ID:                3001,
		PostID:            "post_pub_3",
		ContentInternalID: 9003,
		AuthorID:          701,
		Content:           "root",
		Status:            domain.CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	parent := mustComment(t, domain.CommentSeed{
		ID:                3002,
		PostID:            root.PostID,
		ContentInternalID: root.ContentInternalID,
		AuthorID:          702,
		RootID:            root.ID,
		ParentID:          root.ID,
		Content:           "parent",
		Status:            domain.CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	reply := mustComment(t, domain.CommentSeed{
		ID:                3003,
		PostID:            root.PostID,
		ContentInternalID: root.ContentInternalID,
		AuthorID:          703,
		RootID:            root.ID,
		ParentID:          root.ID,
		Content:           "reply",
		Status:            domain.CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})

	if _, err := domain.NewReplyCreated(reply, root, parent); err == nil {
		t.Fatalf("NewReplyCreated() error = nil, want tree mismatch error")
	}
}

func mustComment(t *testing.T, seed domain.CommentSeed) domain.Comment {
	t.Helper()
	comment, err := domain.NewComment(seed)
	if err != nil {
		t.Fatalf("NewComment() error = %v", err)
	}
	return comment
}
