package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPostFactoryCreateDraft(t *testing.T) {
	factory := PostFactory{}

	t.Run("creates draft with trimmed title and post created event", func(t *testing.T) {
		post, err := factory.CreateDraft(CreateDraftInput{
			PublicID: PublicPostID("post_1"),
			OwnerID:  OwnerID(1001),
			Title:    "  first draft  ",
			Owner: OwnerSnapshot{
				DisplayName:    "architect",
				AvatarFileID:   "file_avatar",
				ProfileVersion: 3,
			},
		})
		if err != nil {
			t.Fatalf("CreateDraft returned error: %v", err)
		}

		if got := post.Status(); got != PostStatusDraft {
			t.Fatalf("status = %s, want %s", got, PostStatusDraft)
		}
		if got := post.Title(); got != PostTitle("first draft") {
			t.Fatalf("title = %q, want trimmed title", got)
		}
		if got := post.OwnerID(); got != OwnerID(1001) {
			t.Fatalf("owner = %d, want 1001", got)
		}

		events := post.PullEvents()
		if len(events) != 1 {
			t.Fatalf("events = %d, want 1", len(events))
		}
		if _, ok := events[0].(PostCreated); !ok {
			t.Fatalf("event = %T, want PostCreated", events[0])
		}
	})

	t.Run("allows empty title while draft is unpublished", func(t *testing.T) {
		post, err := factory.CreateDraft(CreateDraftInput{
			PublicID: PublicPostID("post_2"),
			OwnerID:  OwnerID(1001),
			Title:    "  ",
		})
		if err != nil {
			t.Fatalf("CreateDraft returned error: %v", err)
		}
		if got := post.Title(); got != "" {
			t.Fatalf("title = %q, want empty draft title", got)
		}
	})

	t.Run("rejects too long title", func(t *testing.T) {
		_, err := factory.CreateDraft(CreateDraftInput{
			PublicID: PublicPostID("post_3"),
			OwnerID:  OwnerID(1001),
			Title:    strings.Repeat("字", MaxPostTitleRunes+1),
		})
		if !errors.Is(err, ErrTitleTooLong) {
			t.Fatalf("error = %v, want ErrTitleTooLong", err)
		}
	})
}

func TestPostPublish(t *testing.T) {
	factory := PostFactory{}
	policy := NewPostPublishPolicy(5)
	publishedAt := time.Date(2026, 7, 5, 10, 30, 0, 0, time.UTC)

	t.Run("rejects empty title", func(t *testing.T) {
		post := mustDraftPost(t, factory, CreateDraftInput{
			PublicID: PublicPostID("post_empty_title"),
			OwnerID:  OwnerID(1001),
			Title:    " ",
			DraftBody: &BodyPointer{
				ID:              "body_draft",
				Hash:            "sha256:abc",
				PlainTextLength: 12,
				SizeBytes:       100,
			},
		})

		err := post.Publish(policy, PublishInput{
			DraftBody:   post.DraftBody(),
			PublishedAt: publishedAt,
		})
		if !errors.Is(err, ErrTitleRequired) {
			t.Fatalf("error = %v, want ErrTitleRequired", err)
		}
	})

	t.Run("rejects missing body", func(t *testing.T) {
		post := mustDraftPost(t, factory, CreateDraftInput{
			PublicID: PublicPostID("post_missing_body"),
			OwnerID:  OwnerID(1001),
			Title:    "Ready",
		})

		err := post.Publish(policy, PublishInput{PublishedAt: publishedAt})
		if !errors.Is(err, ErrBodyRequired) {
			t.Fatalf("error = %v, want ErrBodyRequired", err)
		}
	})

	t.Run("rejects body with insufficient text", func(t *testing.T) {
		post := mustDraftPost(t, factory, CreateDraftInput{
			PublicID: PublicPostID("post_short_body"),
			OwnerID:  OwnerID(1001),
			Title:    "Ready",
			DraftBody: &BodyPointer{
				ID:              "body_short",
				Hash:            "sha256:short",
				PlainTextLength: 3,
				SizeBytes:       100,
			},
		})

		err := post.Publish(policy, PublishInput{
			DraftBody:   post.DraftBody(),
			PublishedAt: publishedAt,
		})
		if !errors.Is(err, ErrBodyTooShort) {
			t.Fatalf("error = %v, want ErrBodyTooShort", err)
		}
	})

	t.Run("deleted post cannot save draft", func(t *testing.T) {
		post := mustDraftPost(t, factory, CreateDraftInput{
			PublicID: PublicPostID("post_deleted"),
			OwnerID:  OwnerID(1001),
			Title:    "Deleted",
		})
		post.Delete(time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC))

		err := post.SaveDraftBody(BodyPointer{
			ID:              "body_after_delete",
			Hash:            "sha256:after-delete",
			PlainTextLength: 20,
			SizeBytes:       100,
		})
		if !errors.Is(err, ErrPostDeleted) {
			t.Fatalf("error = %v, want ErrPostDeleted", err)
		}
	})

	t.Run("published post cannot be published again", func(t *testing.T) {
		post := mustPublishablePost(t, factory, "post_repeat")
		if err := post.Publish(policy, PublishInput{DraftBody: post.DraftBody(), PublishedAt: publishedAt}); err != nil {
			t.Fatalf("first Publish returned error: %v", err)
		}

		err := post.Publish(policy, PublishInput{DraftBody: post.PublishedBody(), PublishedAt: publishedAt.Add(time.Minute)})
		if !errors.Is(err, ErrPostAlreadyPublished) {
			t.Fatalf("error = %v, want ErrPostAlreadyPublished", err)
		}
	})

	t.Run("scheduled post cannot use normal publish path", func(t *testing.T) {
		post, err := HydratePost(HydratePostInput{
			PublicID: PublicPostID("post_scheduled"),
			OwnerID:  OwnerID(1001),
			Title:    "Scheduled",
			Status:   PostStatusScheduled,
			DraftBody: &BodyPointer{
				ID:              "body_scheduled",
				Hash:            "sha256:scheduled",
				PlainTextLength: 12,
				SizeBytes:       100,
			},
		})
		if err != nil {
			t.Fatalf("HydratePost returned error: %v", err)
		}

		err = post.Publish(policy, PublishInput{DraftBody: post.DraftBody(), PublishedAt: publishedAt})
		if !errors.Is(err, ErrDraftConflict) {
			t.Fatalf("error = %v, want ErrDraftConflict", err)
		}
	})

	t.Run("publishes draft and records event", func(t *testing.T) {
		post := mustPublishablePost(t, factory, "post_publish")
		post.PullEvents()

		err := post.Publish(policy, PublishInput{DraftBody: post.DraftBody(), PublishedAt: publishedAt})
		if err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		if got := post.Status(); got != PostStatusPublished {
			t.Fatalf("status = %s, want %s", got, PostStatusPublished)
		}
		if got := post.PublishedBody(); got.ID != "body_publish" {
			t.Fatalf("published body = %+v, want body_publish", got)
		}

		events := post.PullEvents()
		if len(events) != 1 {
			t.Fatalf("events = %d, want 1", len(events))
		}
		published, ok := events[0].(PostPublished)
		if !ok {
			t.Fatalf("event = %T, want PostPublished", events[0])
		}
		if published.PublicID != PublicPostID("post_publish") {
			t.Fatalf("event public id = %s, want post_publish", published.PublicID)
		}
	})
}

func mustPublishablePost(t *testing.T, factory PostFactory, publicID string) *Post {
	t.Helper()
	return mustDraftPost(t, factory, CreateDraftInput{
		PublicID: PublicPostID(publicID),
		OwnerID:  OwnerID(1001),
		Title:    "Publishable",
		DraftBody: &BodyPointer{
			ID:              "body_publish",
			Hash:            "sha256:publish",
			PlainTextLength: 12,
			SizeBytes:       100,
		},
	})
}

func mustDraftPost(t *testing.T, factory PostFactory, input CreateDraftInput) *Post {
	t.Helper()
	post, err := factory.CreateDraft(input)
	if err != nil {
		t.Fatalf("CreateDraft returned error: %v", err)
	}
	return post
}
