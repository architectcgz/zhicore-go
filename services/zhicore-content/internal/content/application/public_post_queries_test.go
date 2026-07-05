package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestPublicPostQueries(t *testing.T) {
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	olderPublishedAt := publishedAt.Add(-time.Minute)

	t.Run("lists published posts with default limit and stable next cursor", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.listPublishedResult = []ports.PostSummaryRecord{
			publishedSummary("post_new", 42, publishedAt),
			publishedSummary("post_old", 42, olderPublishedAt),
		}
		service := NewService(deps.asDeps())

		got, err := service.ListPublishedPosts(context.Background(), ListPublishedPostsQuery{AuthorID: "42"})
		if err != nil {
			t.Fatalf("ListPublishedPosts() error = %v", err)
		}
		if deps.posts.listPublishedQuery.AuthorID != 42 || deps.posts.listPublishedQuery.Limit != defaultPublicPostLimit+1 {
			t.Fatalf("query = %+v, want author 42 limit+1", deps.posts.listPublishedQuery)
		}
		if len(got.Items) != 2 || got.Items[0].PostID != "post_new" || got.Items[1].PostID != "post_old" {
			t.Fatalf("items = %+v", got.Items)
		}
		if got.Limit != defaultPublicPostLimit || got.HasMore || got.NextCursor != "" {
			t.Fatalf("page = %+v, want no cursor", got)
		}
	})

	t.Run("caps limit and returns opaque cursor from extra row", func(t *testing.T) {
		deps := newCreatePostDeps()
		for i := 0; i < maxPublicPostLimit+1; i++ {
			deps.posts.listPublishedResult = append(deps.posts.listPublishedResult, publishedSummary("post_cursor", 42, publishedAt.Add(-time.Duration(i)*time.Second)))
		}
		service := NewService(deps.asDeps())

		got, err := service.ListPublishedPosts(context.Background(), ListPublishedPostsQuery{Limit: 500})
		if err != nil {
			t.Fatalf("ListPublishedPosts() error = %v", err)
		}
		if deps.posts.listPublishedQuery.Limit != maxPublicPostLimit+1 {
			t.Fatalf("repo limit = %d, want cap+1", deps.posts.listPublishedQuery.Limit)
		}
		if len(got.Items) != maxPublicPostLimit || got.Limit != maxPublicPostLimit || !got.HasMore || got.NextCursor == "" {
			t.Fatalf("page = %+v, want capped page with next cursor", got)
		}
	})

	t.Run("rejects invalid cursor and author filter", func(t *testing.T) {
		service := NewService(newCreatePostDeps().asDeps())

		if _, err := service.ListPublishedPosts(context.Background(), ListPublishedPostsQuery{Cursor: "not-base64"}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("invalid cursor error = %v, want ErrInvalidArgument", err)
		}
		if _, err := service.ListPublishedPosts(context.Background(), ListPublishedPostsQuery{AuthorID: "not-number"}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("invalid author error = %v, want ErrInvalidArgument", err)
		}
		if _, err := service.ListPublishedPosts(context.Background(), ListPublishedPostsQuery{Tag: "go"}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("tag filter before task 8 error = %v, want ErrInvalidArgument", err)
		}
	})

	t.Run("maps list repository failure to dependency unavailable", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.listPublishedErr = errors.New("postgres down")
		service := NewService(deps.asDeps())

		_, err := service.ListPublishedPosts(context.Background(), ListPublishedPostsQuery{})
		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("list error = %v, want ErrDependencyUnavailable", err)
		}
	})

	t.Run("gets published detail with validated body", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.posts.detailResult = ports.PostDetailRecord{
			Summary:         publishedSummary("post_1", 42, publishedAt),
			PublishedBodyID: "body_published",
			PublishedHash:   "sha256:published",
		}
		service := NewService(deps.asDeps())

		got, err := service.GetPostDetail(context.Background(), GetPostDetailQuery{PostID: "post_1"})
		if err != nil {
			t.Fatalf("GetPostDetail() error = %v", err)
		}
		if got.Post.PostID != "post_1" || got.Post.Status != string(domain.PostStatusPublished) {
			t.Fatalf("post = %+v", got.Post)
		}
		if got.Body == nil || got.Body.BodyID != "body_published" || got.Body.PlainText != "published body" {
			t.Fatalf("body = %+v", got.Body)
		}
	})

	t.Run("hides missing draft and deleted posts in detail", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.detailErr = domain.ErrPostNotFound
		service := NewService(deps.asDeps())

		_, err := service.GetPostDetail(context.Background(), GetPostDetailQuery{PostID: "post_missing"})
		if !errors.Is(err, domain.ErrPostNotFound) {
			t.Fatalf("detail error = %v, want ErrPostNotFound", err)
		}
	})

	t.Run("records detail repair task with internal post id", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.posts.detailResult = ports.PostDetailRecord{
			InternalPostID:  10,
			Summary:         publishedSummary("post_1", 42, publishedAt),
			PublishedBodyID: "body_published",
			PublishedHash:   "sha256:published",
		}
		deps.bodies.readErr = domain.ErrBodyUnavailable
		service := NewService(deps.asDeps())

		_, err := service.GetPostDetail(context.Background(), GetPostDetailQuery{PostID: "post_1"})
		if !errors.Is(err, domain.ErrBodyUnavailable) {
			t.Fatalf("GetPostDetail() error = %v, want ErrBodyUnavailable", err)
		}
		if len(deps.repair.outsideTasks) != 1 || deps.repair.outsideTasks[0].PostID != 10 {
			t.Fatalf("repair tasks = %+v, want internal post id 10", deps.repair.outsideTasks)
		}
	})

	t.Run("batch returns visible summaries and missing ids in request order", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.batchResult = []ports.PostSummaryRecord{
			publishedSummary("post_2", 43, olderPublishedAt),
			publishedSummary("post_1", 42, publishedAt),
		}
		service := NewService(deps.asDeps())

		got, err := service.BatchGetPublishedPosts(context.Background(), BatchGetPublishedPostsQuery{
			PostIDs: []string{"post_1", "post_missing", "post_2"},
		})
		if err != nil {
			t.Fatalf("BatchGetPublishedPosts() error = %v", err)
		}
		if len(deps.posts.batchIDs) != 3 || deps.posts.batchIDs[0] != "post_1" || deps.posts.batchIDs[2] != "post_2" {
			t.Fatalf("batch ids = %+v", deps.posts.batchIDs)
		}
		if len(got.Items) != 2 || got.Items[0].PostID != "post_1" || got.Items[1].PostID != "post_2" {
			t.Fatalf("items = %+v, want request order for visible posts", got.Items)
		}
		if len(got.MissingPostIDs) != 1 || got.MissingPostIDs[0] != "post_missing" {
			t.Fatalf("missing = %+v", got.MissingPostIDs)
		}
	})

	t.Run("rejects empty or oversized batch", func(t *testing.T) {
		service := NewService(newCreatePostDeps().asDeps())

		if _, err := service.BatchGetPublishedPosts(context.Background(), BatchGetPublishedPostsQuery{}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("empty batch error = %v, want ErrInvalidArgument", err)
		}
		ids := make([]string, maxPublicBatchPostIDs+1)
		for i := range ids {
			ids[i] = "post"
		}
		if _, err := service.BatchGetPublishedPosts(context.Background(), BatchGetPublishedPostsQuery{PostIDs: ids}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("large batch error = %v, want ErrInvalidArgument", err)
		}
	})

	t.Run("maps batch repository failure to dependency unavailable", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.posts.batchErr = errors.New("postgres down")
		service := NewService(deps.asDeps())

		_, err := service.BatchGetPublishedPosts(context.Background(), BatchGetPublishedPostsQuery{PostIDs: []string{"post_1"}})
		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("batch error = %v, want ErrDependencyUnavailable", err)
		}
	})
}

func publishedSummary(postID string, ownerID int64, publishedAt time.Time) ports.PostSummaryRecord {
	return ports.PostSummaryRecord{
		PostID:             postID,
		AuthorID:           ownerID,
		AuthorName:         "architect",
		AuthorAvatarFileID: "file_avatar",
		Title:              "Published " + postID,
		Summary:            "summary",
		CoverFileID:        "file_cover",
		Status:             domain.PostStatusPublished,
		PostVersion:        3,
		PublishedAt:        publishedAt,
		CreatedAt:          publishedAt.Add(-time.Hour),
		UpdatedAt:          publishedAt,
		ViewCount:          10,
		LikeCount:          2,
		FavoriteCount:      1,
		CommentCount:       4,
	}
}
