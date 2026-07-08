package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestTaxonomyQueries(t *testing.T) {
	t.Run("lists tags with normalized limit and cursor", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.taxonomy = &fakeTaxonomyRepository{
			listTagsResult: []ports.TagRecord{
				tagRecord(1, "go", 8),
				tagRecord(2, "ddd", 3),
				tagRecord(3, "vue", 2),
			},
		}
		service := NewService(deps.asDeps())

		got, err := service.ListTags(context.Background(), ListTagsQuery{Limit: 2})
		if err != nil {
			t.Fatalf("ListTags() error = %v", err)
		}
		if deps.taxonomy.listTagsQuery.Limit != 3 {
			t.Fatalf("repository limit = %d, want limit+1", deps.taxonomy.listTagsQuery.Limit)
		}
		if len(got.Items) != 2 || got.Items[0].Slug != "go" || got.Items[1].PostCount != 3 || !got.HasMore || got.Limit != 2 {
			t.Fatalf("result = %+v", got)
		}
	})

	t.Run("maps missing tag detail to taxonomy reference error", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.taxonomy = &fakeTaxonomyRepository{getTagErr: ports.ErrTaxonomyReferenceNotFound}
		service := NewService(deps.asDeps())

		_, err := service.GetTag(context.Background(), GetTagQuery{Slug: "missing"})

		if !errors.Is(err, ErrTaxonomyReferenceNotFound) {
			t.Fatalf("GetTag() error = %v, want ErrTaxonomyReferenceNotFound", err)
		}
	})

	t.Run("lists posts by tag after proving tag exists", func(t *testing.T) {
		publishedAt := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
		deps := newCreatePostDeps()
		deps.taxonomy = &fakeTaxonomyRepository{
			getTagResult: tagRecord(1, "go", 8),
			listPostsResult: []ports.PostSummaryRecord{{
				PostID:      "post_1",
				AuthorID:    42,
				Title:       "Published",
				Status:      domain.PostStatusPublished,
				PublishedAt: publishedAt,
				CreatedAt:   publishedAt.Add(-time.Hour),
				UpdatedAt:   publishedAt,
			}},
		}
		service := NewService(deps.asDeps())

		got, err := service.ListPostsByTag(context.Background(), ListPostsByTagQuery{Slug: " Go ", Limit: 10})
		if err != nil {
			t.Fatalf("ListPostsByTag() error = %v", err)
		}
		if deps.taxonomy.getTagSlug != "go" || deps.taxonomy.listPostsQuery.Slug != "go" {
			t.Fatalf("slug normalization get=%q list=%+v", deps.taxonomy.getTagSlug, deps.taxonomy.listPostsQuery)
		}
		if len(got.Items) != 1 || got.Items[0].PostID != "post_1" {
			t.Fatalf("posts = %+v", got.Items)
		}
	})
}

func TestUpdatePostTagsNormalizesAndChecksOwnership(t *testing.T) {
	deps := newCreatePostDeps()
	deps.posts.getResult = ports.PostRecord{
		ID:          10,
		PublicID:    "post_1",
		OwnerID:     42,
		Status:      domain.PostStatusDraft,
		PostVersion: 3,
	}
	deps.taxonomy = &fakeTaxonomyRepository{
		replaceResult: ports.PostTagsMutationRecord{
			PostID:      "post_1",
			PostVersion: 4,
			Tags:        []ports.TagRecord{tagRecord(1, "go", 8), tagRecord(2, "ddd", 3)},
			UpdatedAt:   deps.clock.now,
		},
	}
	service := NewService(deps.asDeps())

	got, err := service.UpdatePostTags(context.Background(), UpdatePostTagsCommand{
		Actor:           &Actor{UserID: 42},
		PostID:          "post_1",
		BasePostVersion: 3,
		Tags:            []string{" Go ", "go", "DDD"},
	})

	if err != nil {
		t.Fatalf("UpdatePostTags() error = %v", err)
	}
	if deps.posts.getCalls != 1 || deps.taxonomy.replaceCalls != 1 {
		t.Fatalf("calls get=%d replace=%d", deps.posts.getCalls, deps.taxonomy.replaceCalls)
	}
	if got.PostVersion != 4 || len(got.Tags) != 2 || got.Tags[0].Slug != "go" || got.Tags[1].Slug != "ddd" {
		t.Fatalf("result = %+v", got)
	}
	if input := deps.taxonomy.replaceInput; input.PostInternalID != 10 || input.ActorID != 42 ||
		len(input.Slugs) != 2 || input.Slugs[0] != "go" || input.Slugs[1] != "ddd" {
		t.Fatalf("replace input = %+v", input)
	}
}

func TestUpdatePostTagsRejectsInvalidStates(t *testing.T) {
	t.Run("non author", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.getResult = ports.PostRecord{PublicID: "post_1", OwnerID: 99, Status: domain.PostStatusDraft, PostVersion: 3}
		service := NewService(deps.asDeps())

		_, err := service.UpdatePostTags(context.Background(), UpdatePostTagsCommand{Actor: &Actor{UserID: 42}, PostID: "post_1", BasePostVersion: 3, Tags: []string{"go"}})

		if !errors.Is(err, ErrForbidden) || deps.taxonomy.replaceCalls != 0 {
			t.Fatalf("error = %v replaceCalls=%d, want forbidden before repository mutation", err, deps.taxonomy.replaceCalls)
		}
	})

	t.Run("stale version", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.getResult = ports.PostRecord{PublicID: "post_1", OwnerID: 42, Status: domain.PostStatusDraft, PostVersion: 4}
		service := NewService(deps.asDeps())

		_, err := service.UpdatePostTags(context.Background(), UpdatePostTagsCommand{Actor: &Actor{UserID: 42}, PostID: "post_1", BasePostVersion: 3, Tags: []string{"go"}})

		if !errors.Is(err, ErrDraftConflict) || deps.taxonomy.replaceCalls != 0 {
			t.Fatalf("error = %v replaceCalls=%d, want draft conflict before repository mutation", err, deps.taxonomy.replaceCalls)
		}
	})
}

func TestDeletePostTag(t *testing.T) {
	deps := newCreatePostDeps()
	deps.posts.getResult = ports.PostRecord{
		ID:          10,
		PublicID:    "post_1",
		OwnerID:     42,
		Status:      domain.PostStatusPublished,
		PostVersion: 5,
	}
	deps.taxonomy = &fakeTaxonomyRepository{
		removeResult: ports.PostTagsMutationRecord{
			PostID:      "post_1",
			PostVersion: 6,
			Tags:        []ports.TagRecord{tagRecord(2, "ddd", 3)},
			UpdatedAt:   deps.clock.now,
		},
	}
	service := NewService(deps.asDeps())

	got, err := service.DeletePostTag(context.Background(), DeletePostTagCommand{
		Actor:           &Actor{UserID: 42},
		PostID:          "post_1",
		BasePostVersion: 5,
		Slug:            " Go ",
	})

	if err != nil {
		t.Fatalf("DeletePostTag() error = %v", err)
	}
	if got.PostVersion != 6 || len(got.Tags) != 1 || deps.taxonomy.removeInput.Slug != "go" {
		t.Fatalf("result=%+v removeInput=%+v", got, deps.taxonomy.removeInput)
	}
}

func tagRecord(id int64, slug string, postCount int64) ports.TagRecord {
	return ports.TagRecord{
		ID:        id,
		PublicID:  "tag_" + slug,
		Name:      slug,
		Slug:      slug,
		PostCount: postCount,
	}
}
