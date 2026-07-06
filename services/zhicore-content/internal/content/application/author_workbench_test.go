package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestAuthorWorkbench(t *testing.T) {
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	t.Run("lists current author's posts with capped cursor pagination", func(t *testing.T) {
		deps := newCreatePostDeps()
		for i := 0; i < maxAuthorPostLimit+1; i++ {
			deps.posts.listAuthorResult = append(deps.posts.listAuthorResult, publishedSummary("post_author", 1001, publishedAt.Add(-time.Duration(i)*time.Second)))
		}
		service := NewService(deps.asDeps())

		got, err := service.ListAuthorPosts(context.Background(), ListAuthorPostsQuery{
			Actor: &Actor{UserID: 1001},
			Limit: 500,
		})
		if err != nil {
			t.Fatalf("ListAuthorPosts() error = %v", err)
		}
		if deps.posts.listAuthorQuery.OwnerID != 1001 || deps.posts.listAuthorQuery.Limit != maxAuthorPostLimit+1 {
			t.Fatalf("query = %+v, want owner 1001 capped limit+1", deps.posts.listAuthorQuery)
		}
		if len(got.Items) != maxAuthorPostLimit || !got.HasMore || got.NextCursor == "" {
			t.Fatalf("page = %+v, want capped author page with cursor", got)
		}
	})

	t.Run("rejects author list without actor and invalid status", func(t *testing.T) {
		service := NewService(newCreatePostDeps().asDeps())

		if _, err := service.ListAuthorPosts(context.Background(), ListAuthorPostsQuery{}); !errors.Is(err, ErrLoginRequired) {
			t.Fatalf("missing actor error = %v, want ErrLoginRequired", err)
		}
		if _, err := service.ListAuthorPosts(context.Background(), ListAuthorPostsQuery{
			Actor:  &Actor{UserID: 1001},
			Status: "unknown",
		}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("invalid status error = %v, want ErrInvalidArgument", err)
		}
	})

	t.Run("lists drafts through the same owner query", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.listAuthorResult = []ports.PostSummaryRecord{draftSummary("post_draft", 1001, publishedAt)}
		service := NewService(deps.asDeps())

		got, err := service.ListAuthorDrafts(context.Background(), ListAuthorDraftsQuery{Actor: &Actor{UserID: 1001}})
		if err != nil {
			t.Fatalf("ListAuthorDrafts() error = %v", err)
		}
		if deps.posts.listAuthorQuery.Status != string(domain.PostStatusDraft) {
			t.Fatalf("status = %q, want DRAFT", deps.posts.listAuthorQuery.Status)
		}
		if len(got.Items) != 1 || got.Items[0].PostID != "post_draft" {
			t.Fatalf("items = %+v", got.Items)
		}
	})

	t.Run("gets own draft with body", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.posts.draftResult = ports.DraftPostRecord{
			Post: ports.PostRecord{
				ID:             10,
				PublicID:       "post_1",
				OwnerID:        1001,
				Status:         domain.PostStatusDraft,
				PostVersion:    5,
				DraftTitle:     "Draft",
				DraftSummary:   "summary",
				DraftBodyID:    "body_draft",
				DraftBodyHash:  "sha256:published",
				DraftSizeBytes: 36,
			},
			CreatedAt: publishedAt.Add(-time.Hour),
			UpdatedAt: publishedAt,
		}
		service := NewService(deps.asDeps())

		got, err := service.GetAuthorDraft(context.Background(), GetAuthorDraftQuery{
			Actor:  &Actor{UserID: 1001},
			PostID: "post_1",
		})
		if err != nil {
			t.Fatalf("GetAuthorDraft() error = %v", err)
		}
		if got.PostID != "post_1" || got.PostVersion != 5 || got.Body == nil || got.Body.BodyID != "body_published" {
			t.Fatalf("draft = %+v", got)
		}
	})

	t.Run("forbids reading another author's draft", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.draftResult = ports.DraftPostRecord{Post: ports.PostRecord{PublicID: "post_1", OwnerID: 2002, Status: domain.PostStatusDraft}}
		service := NewService(deps.asDeps())

		_, err := service.GetAuthorDraft(context.Background(), GetAuthorDraftQuery{Actor: &Actor{UserID: 1001}, PostID: "post_1"})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("error = %v, want ErrForbidden", err)
		}
	})

	t.Run("records repair task when draft body is missing", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.posts.draftResult = ports.DraftPostRecord{
			Post: ports.PostRecord{
				ID:            10,
				PublicID:      "post_1",
				OwnerID:       1001,
				Status:        domain.PostStatusDraft,
				PostVersion:   5,
				DraftBodyID:   "body_draft",
				DraftBodyHash: "sha256:draft",
			},
			CreatedAt: publishedAt.Add(-time.Hour),
			UpdatedAt: publishedAt,
		}
		deps.bodies.readErr = domain.ErrBodyUnavailable
		service := NewService(deps.asDeps())

		_, err := service.GetAuthorDraft(context.Background(), GetAuthorDraftQuery{Actor: &Actor{UserID: 1001}, PostID: "post_1"})
		if !errors.Is(err, domain.ErrBodyUnavailable) {
			t.Fatalf("error = %v, want ErrBodyUnavailable", err)
		}
		if deps.repair.appendOutsideCalls != 1 || deps.repair.outsideTasks[0].TaskType != "draft_body_missing" {
			t.Fatalf("repair tasks = %+v, want draft_body_missing", deps.repair.outsideTasks)
		}
	})

	t.Run("records repair task when draft body hash mismatches", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.posts.draftResult = ports.DraftPostRecord{
			Post: ports.PostRecord{
				ID:            10,
				PublicID:      "post_1",
				OwnerID:       1001,
				Status:        domain.PostStatusDraft,
				PostVersion:   5,
				DraftBodyID:   "body_draft",
				DraftBodyHash: "sha256:expected",
			},
			CreatedAt: publishedAt.Add(-time.Hour),
			UpdatedAt: publishedAt,
		}
		deps.bodies.readResult.ContentHash = "sha256:actual"
		service := NewService(deps.asDeps())

		_, err := service.GetAuthorDraft(context.Background(), GetAuthorDraftQuery{Actor: &Actor{UserID: 1001}, PostID: "post_1"})
		if !errors.Is(err, domain.ErrBodyInconsistent) {
			t.Fatalf("error = %v, want ErrBodyInconsistent", err)
		}
		if deps.repair.appendOutsideCalls != 1 || deps.repair.outsideTasks[0].TaskType != "body_hash_mismatch" {
			t.Fatalf("repair tasks = %+v, want body_hash_mismatch", deps.repair.outsideTasks)
		}
		if got := deps.repair.outsideTasks[0]; got.ExpectedHash != "sha256:expected" || got.ObservedHash != "sha256:actual" {
			t.Fatalf("repair task = %+v, want expected/observed hash", got)
		}
	})

	t.Run("updates draft metadata after owner and version checks", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.posts.updateMetaResult = ports.PostRecord{
			PublicID:         "post_1",
			OwnerID:          1001,
			Status:           domain.PostStatusDraft,
			PostVersion:      6,
			DraftTitle:       "Next title",
			DraftSummary:     "next summary",
			DraftCoverFileID: "file_cover",
		}
		service := NewService(deps.asDeps())
		title := " Next title "
		summary := "next summary"
		cover := "file_cover"

		got, err := service.UpdateDraftMeta(context.Background(), UpdateDraftMetaCommand{
			Actor:           &Actor{UserID: 1001},
			PostID:          "post_1",
			BasePostVersion: 5,
			Title:           &title,
			Summary:         &summary,
			CoverFileID:     &cover,
		})
		if err != nil {
			t.Fatalf("UpdateDraftMeta() error = %v", err)
		}
		if deps.files.validateCoverCalls != 1 || deps.files.coverFileID != "file_cover" {
			t.Fatalf("cover validation = %d/%q, want file_cover", deps.files.validateCoverCalls, deps.files.coverFileID)
		}
		if deps.posts.updateMetaInput.Title == nil || *deps.posts.updateMetaInput.Title != "Next title" {
			t.Fatalf("update input title = %#v", deps.posts.updateMetaInput.Title)
		}
		if deps.posts.updateMetaInput.UpdatedAt != deps.clock.now {
			t.Fatalf("updated_at = %v, want fake clock %v", deps.posts.updateMetaInput.UpdatedAt, deps.clock.now)
		}
		if got.PostVersion != 6 || got.Title != "Next title" || got.UpdatedAt != deps.clock.now {
			t.Fatalf("result = %+v", got)
		}
	})

	t.Run("clears title and summary when explicitly set to empty", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.posts.updateMetaResult = ports.PostRecord{PublicID: "post_1", OwnerID: 1001, Status: domain.PostStatusDraft, PostVersion: 6}
		service := NewService(deps.asDeps())
		empty := "  "

		_, err := service.UpdateDraftMeta(context.Background(), UpdateDraftMetaCommand{
			Actor:           &Actor{UserID: 1001},
			PostID:          "post_1",
			BasePostVersion: 5,
			Title:           &empty,
			Summary:         &empty,
		})
		if err != nil {
			t.Fatalf("UpdateDraftMeta() error = %v", err)
		}
		if deps.posts.updateMetaInput.Title == nil || *deps.posts.updateMetaInput.Title != "" {
			t.Fatalf("title update = %#v, want explicit empty string", deps.posts.updateMetaInput.Title)
		}
		if deps.posts.updateMetaInput.Summary == nil || *deps.posts.updateMetaInput.Summary != "" {
			t.Fatalf("summary update = %#v, want explicit empty string", deps.posts.updateMetaInput.Summary)
		}
	})

	t.Run("rejects taxonomy metadata until taxonomy slice owns persistence", func(t *testing.T) {
		deps := newSaveDraftDeps()
		service := NewService(deps.asDeps())
		category := "cat_1"

		_, err := service.UpdateDraftMeta(context.Background(), UpdateDraftMetaCommand{
			Actor:           &Actor{UserID: 1001},
			PostID:          "post_1",
			BasePostVersion: 5,
			CategoryID:      &category,
		})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("error = %v, want ErrInvalidArgument", err)
		}
		if deps.posts.updateMetaCalls != 0 {
			t.Fatalf("update calls = %d, want none", deps.posts.updateMetaCalls)
		}
	})

	t.Run("deletes draft and schedules old draft body cleanup", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.posts.deleteDraftResult = ports.PostRecord{
			ID:          10,
			PublicID:    "post_1",
			OwnerID:     1001,
			Status:      domain.PostStatusDraft,
			PostVersion: 6,
		}
		service := NewService(deps.asDeps())

		got, err := service.DeleteAuthorDraft(context.Background(), DeleteAuthorDraftCommand{
			Actor:  &Actor{UserID: 1001},
			PostID: "post_1",
		})
		if err != nil {
			t.Fatalf("DeleteAuthorDraft() error = %v", err)
		}
		if got.PostID != "post_1" || got.PostVersion != 6 {
			t.Fatalf("result = %+v", got)
		}
		if deps.tx.calls != 1 || deps.posts.getCalls != 1 || deps.posts.deleteDraftCalls != 1 {
			t.Fatalf("tx/get/delete calls = %d/%d/%d, want 1/1/1", deps.tx.calls, deps.posts.getCalls, deps.posts.deleteDraftCalls)
		}
		if deps.posts.deleteDraftInput.UpdatedAt != deps.clock.now {
			t.Fatalf("delete updated_at = %v, want fake clock %v", deps.posts.deleteDraftInput.UpdatedAt, deps.clock.now)
		}
		if deps.cleanup.appendCalls != 1 || deps.cleanup.tasks[0].BodyID != "body_old" {
			t.Fatalf("cleanup tasks = %+v, want old draft body cleanup", deps.cleanup.tasks)
		}
		if deps.posts.getTx != deps.posts.deleteDraftTx || deps.posts.getTx != deps.cleanup.appendTxs[0] {
			t.Fatalf("tx mismatch get=%#v delete=%#v cleanup=%#v", deps.posts.getTx, deps.posts.deleteDraftTx, deps.cleanup.appendTxs[0])
		}
	})
}

func draftSummary(postID string, ownerID int64, updatedAt time.Time) ports.PostSummaryRecord {
	record := publishedSummary(postID, ownerID, updatedAt)
	record.Title = "Draft " + postID
	record.Status = domain.PostStatusDraft
	record.PublishedAt = time.Time{}
	return record
}
