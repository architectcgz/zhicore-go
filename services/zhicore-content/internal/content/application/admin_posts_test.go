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

func TestAdminPostsUseCases(t *testing.T) {
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	t.Run("lists posts for admin with page filters", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.adminPosts = &fakeAdminPostRepository{
			listResult: ports.AdminPostPage{
				Items: []ports.AdminPostRecord{{
					PostID:             "post_1",
					AuthorID:           42,
					AuthorName:         "architect",
					AuthorAvatarFileID: "file_avatar",
					Title:              "Published title",
					Summary:            "summary",
					CoverFileID:        "file_cover",
					Status:             domain.PostStatusPublished,
					PostVersion:        6,
					PublishedAt:        publishedAt,
					CreatedAt:          publishedAt.Add(-time.Hour),
					UpdatedAt:          publishedAt.Add(time.Minute),
					ViewCount:          10,
					LikeCount:          2,
					FavoriteCount:      1,
					CommentCount:       3,
				}},
				Page:  2,
				Size:  20,
				Total: 21,
			},
		}
		service := NewService(deps.asDeps())

		result, err := service.ListAdminPosts(context.Background(), ListAdminPostsQuery{
			Actor:    &Actor{UserID: 1001, Roles: []string{"writer", "admin"}},
			Status:   "published",
			AuthorID: 42,
			Page:     2,
			Size:     20,
		})
		if err != nil {
			t.Fatalf("ListAdminPosts() error = %v", err)
		}
		if deps.adminPosts.listQuery.Status != "PUBLISHED" || deps.adminPosts.listQuery.AuthorID != 42 ||
			deps.adminPosts.listQuery.Page != 2 || deps.adminPosts.listQuery.Size != 20 {
			t.Fatalf("list query = %+v", deps.adminPosts.listQuery)
		}
		if result.Page != 2 || result.Size != 20 || result.Total != 21 || len(result.Items) != 1 {
			t.Fatalf("result = %+v", result)
		}
		got := result.Items[0]
		if got.PostID != "post_1" || got.AuthorID != "42" || got.Title != "Published title" ||
			got.Stats.ViewCount != 10 || got.PublishedAt != publishedAt {
			t.Fatalf("item = %+v", got)
		}
	})

	t.Run("rejects non-admin and invalid filters before repository", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.adminPosts = &fakeAdminPostRepository{}
		service := NewService(deps.asDeps())

		_, err := service.ListAdminPosts(context.Background(), ListAdminPostsQuery{
			Actor:  &Actor{UserID: 1001, Roles: []string{"writer"}},
			Status: "published",
		})
		if !errors.Is(err, ErrRoleRequired) {
			t.Fatalf("ListAdminPosts(non-admin) error = %v, want ErrRoleRequired", err)
		}

		_, err = service.ListAdminPosts(context.Background(), ListAdminPostsQuery{
			Actor:  &Actor{UserID: 1001, Roles: []string{"admin"}},
			Status: "hidden",
		})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("ListAdminPosts(invalid status) error = %v, want ErrInvalidArgument", err)
		}

		_, err = service.ListAdminPosts(context.Background(), ListAdminPostsQuery{
			Actor:    &Actor{UserID: 1001, Roles: []string{"admin"}},
			AuthorID: -1,
		})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("ListAdminPosts(invalid author) error = %v, want ErrInvalidArgument", err)
		}
		if deps.adminPosts.listCalls != 0 {
			t.Fatalf("list calls = %d, want none", deps.adminPosts.listCalls)
		}
	})

	t.Run("deletes post with audit context and visibility event in one transaction", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.adminPosts = &fakeAdminPostRepository{
			deleteResult: ports.AdminPostDeleteRecord{
				Before: ports.PostRecord{
					ID:          10,
					PublicID:    "post_1",
					OwnerID:     42,
					Status:      domain.PostStatusPublished,
					PostVersion: 5,
				},
				After: ports.PostRecord{
					ID:          10,
					PublicID:    "post_1",
					OwnerID:     42,
					Status:      domain.PostStatusDeleted,
					PostVersion: 6,
				},
			},
		}
		service := NewService(deps.asDeps())

		result, err := service.DeleteAdminPost(context.Background(), DeleteAdminPostCommand{
			Actor:  &Actor{UserID: 1001, Roles: []string{"ROLE_ADMIN"}},
			PostID: "post_1",
			Reason: "policy violation",
		})
		if err != nil {
			t.Fatalf("DeleteAdminPost() error = %v", err)
		}
		if deps.tx.calls != 1 || deps.adminPosts.deleteCalls != 1 || deps.outbox.appendCalls != 1 {
			t.Fatalf("tx/delete/outbox calls = %d/%d/%d, want 1/1/1", deps.tx.calls, deps.adminPosts.deleteCalls, deps.outbox.appendCalls)
		}
		if deps.adminPosts.deleteCommand.AdminUserID != 1001 ||
			deps.adminPosts.deleteCommand.Reason != "policy violation" ||
			!deps.adminPosts.deleteCommand.DeletedAt.Equal(deps.clock.now) {
			t.Fatalf("delete command = %+v", deps.adminPosts.deleteCommand)
		}
		event := deps.outbox.events[0]
		if event.EventType != "content.post.visibility_changed" || event.AggregateID != "post_1" || event.AggregateVersion != 6 {
			t.Fatalf("outbox event = %+v", event)
		}
		payload := string(event.PayloadJSON)
		for _, fragment := range []string{
			`"oldVisibility":"PUBLIC"`,
			`"newVisibility":"DELETED"`,
			`"publicVisible":false`,
			`"reason":"ADMIN_DELETED"`,
		} {
			if !strings.Contains(payload, fragment) {
				t.Fatalf("payload %s missing %s", payload, fragment)
			}
		}
		if result.PostID != "post_1" || result.Status != "DELETED" {
			t.Fatalf("result = %+v", result)
		}
	})

	t.Run("maps repeat delete without writing visibility event", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.adminPosts = &fakeAdminPostRepository{deleteErr: domain.ErrPostDeleted}
		service := NewService(deps.asDeps())

		_, err := service.DeleteAdminPost(context.Background(), DeleteAdminPostCommand{
			Actor:  &Actor{UserID: 1001, Roles: []string{"admin"}},
			PostID: "post_1",
		})
		if !errors.Is(err, ErrPostDeleted) {
			t.Fatalf("DeleteAdminPost() error = %v, want ErrPostDeleted", err)
		}
		if deps.outbox.appendCalls != 0 {
			t.Fatalf("outbox calls = %d, want none", deps.outbox.appendCalls)
		}
	})
}

type fakeAdminPostRepository struct {
	listCalls     int
	listQuery     ports.AdminPostListQuery
	listResult    ports.AdminPostPage
	listErr       error
	deleteCalls   int
	deleteTx      ports.Tx
	deleteCommand ports.AdminPostDeleteCommand
	deleteResult  ports.AdminPostDeleteRecord
	deleteErr     error
}

func (f *fakeAdminPostRepository) ListAdminPosts(ctx context.Context, query ports.AdminPostListQuery) (ports.AdminPostPage, error) {
	f.listCalls++
	f.listQuery = query
	if f.listErr != nil {
		return ports.AdminPostPage{}, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeAdminPostRepository) DeleteAdminPost(ctx context.Context, tx ports.Tx, command ports.AdminPostDeleteCommand) (ports.AdminPostDeleteRecord, error) {
	f.deleteCalls++
	f.deleteTx = tx
	f.deleteCommand = command
	if f.deleteErr != nil {
		return ports.AdminPostDeleteRecord{}, f.deleteErr
	}
	return f.deleteResult, nil
}
