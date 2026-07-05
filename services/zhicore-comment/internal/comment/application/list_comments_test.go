package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestListTopLevelCommentsUsesRequestedSortAndReturnsPageTotals(t *testing.T) {
	now := time.Date(2026, 7, 4, 14, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	first := store.seedComment(t, domain.CommentSeed{ID: 4101, PostID: "post_pub_5", ContentInternalID: 9501, AuthorID: 601, Content: "recommended first", Status: domain.CommentStatusNormal, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute)})
	second := store.seedComment(t, domain.CommentSeed{ID: 4102, PostID: "post_pub_5", ContentInternalID: 9501, AuthorID: 602, Content: "recommended second", Status: domain.CommentStatusNormal, CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)})
	store.stats[first.ID] = domain.CommentStats{CommentID: first.ID, LikeCount: 2, ReplyCount: 3}
	store.stats[second.ID] = domain.CommentStats{CommentID: second.ID, LikeCount: 1, ReplyCount: 0}
	store.postStats["post_pub_5"] = domain.CommentPostStats{PostID: "post_pub_5", TotalComments: 5, TotalTopLevelComments: 2}
	store.queryResults[domain.CommentSortRecommended] = []domain.Comment{first, second}
	users := &fakeUserProfileClient{summaries: map[domain.UserID]ports.AuthorSummary{
		601: {UserID: 601, PublicID: "user_pub_601", DisplayName: "Alice"},
		602: {UserID: 602, PublicID: "user_pub_602", DisplayName: "Bob"},
	}}
	store.viewerLiked = map[domain.CommentID]bool{first.ID: true}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_5", ContentInternalID: 9501, AuthorID: 601}},
		UserProfiles:  users,
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        &fakeOutboxPublisher{},
		Clock:         fixedClock{now: now},
	})

	page, err := service.ListTopLevelCommentsByPage(context.Background(), ListTopLevelCommentsQuery{
		PostID:       "post_pub_5",
		ViewerUserID: 88,
		Page:         1,
		Size:         20,
		Sort:         CommentSortRecommended,
	})
	if err != nil {
		t.Fatalf("ListTopLevelCommentsByPage() error = %v", err)
	}

	if store.lastListQuery.Sort != domain.CommentSortRecommended || store.lastListQuery.Page != 1 || store.lastListQuery.Size != 20 {
		t.Fatalf("last list query = %#v", store.lastListQuery)
	}
	if page.TotalComments != 5 || page.TotalTopLevelComments != 2 || page.Pages != 1 {
		t.Fatalf("page totals = %#v", page)
	}
	if len(page.Items) != 2 {
		t.Fatalf("item count = %d, want 2", len(page.Items))
	}
	if page.Items[0].CommentID != "c4101" || page.Items[0].Author.PublicID != "user_pub_601" || page.Items[0].Stats.ReplyCount != 3 {
		t.Fatalf("first item = %#v", page.Items[0])
	}
	if page.Items[0].Viewer == nil || !page.Items[0].Viewer.Liked || page.Items[1].Viewer == nil || page.Items[1].Viewer.Liked {
		t.Fatalf("viewer states = %#v %#v", page.Items[0].Viewer, page.Items[1].Viewer)
	}
}

func TestListTopLevelCommentsSupportsHotAndTimeSorts(t *testing.T) {
	for _, sort := range []CommentSort{CommentSortHot, CommentSortTime} {
		t.Run(string(sort), func(t *testing.T) {
			store := newFakeCommentStore()
			store.postStats["post_pub_6"] = domain.CommentPostStats{PostID: "post_pub_6"}
			service := mustNewService(t, Dependencies{
				Commands:      store,
				Queries:       store,
				Stats:         store,
				PostStats:     store,
				ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_6", ContentInternalID: 9601, AuthorID: 601}},
				UserProfiles:  &fakeUserProfileClient{},
				UserRelations: &fakeUserRelationClient{},
				Files:         &fakeFileReferenceClient{},
				IDs:           publicIDCodec{},
				RateLimiter:   &fakeRateLimiter{},
				TxRunner:      &fakeTransactionRunner{},
				Outbox:        &fakeOutboxPublisher{},
				Clock:         fixedClock{now: time.Now()},
			})

			_, err := service.ListTopLevelCommentsByPage(context.Background(), ListTopLevelCommentsQuery{PostID: "post_pub_6", Page: 1, Size: 10, Sort: sort})
			if err != nil {
				t.Fatalf("ListTopLevelCommentsByPage() error = %v", err)
			}
			if store.lastListQuery.Sort != domainCommentSort(sort) {
				t.Fatalf("sort = %q, want %q", store.lastListQuery.Sort, domainCommentSort(sort))
			}
		})
	}
}

func TestListTopLevelCommentsDefaultsAndValidatesPagination(t *testing.T) {
	store := newFakeCommentStore()
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_7", ContentInternalID: 9701, AuthorID: 601}},
		UserProfiles:  &fakeUserProfileClient{},
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        &fakeOutboxPublisher{},
		Clock:         fixedClock{now: time.Now()},
	})

	_, err := service.ListTopLevelCommentsByPage(context.Background(), ListTopLevelCommentsQuery{PostID: "post_pub_7"})
	if err != nil {
		t.Fatalf("ListTopLevelCommentsByPage() default query error = %v", err)
	}
	if store.lastListQuery.Page != 1 || store.lastListQuery.Size != 20 || store.lastListQuery.Sort != domain.CommentSortRecommended {
		t.Fatalf("default query = %#v", store.lastListQuery)
	}

	for _, query := range []ListTopLevelCommentsQuery{
		{PostID: "post_pub_7", Page: 1, Size: 101, Sort: CommentSortRecommended},
		{PostID: "post_pub_7", Page: 1, Size: 20, Sort: "BAD"},
	} {
		_, err := service.ListTopLevelCommentsByPage(context.Background(), query)
		if !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("ListTopLevelCommentsByPage(%#v) error = %v, want %v", query, err, ErrInvalidRequest)
		}
	}
}

func TestListTopLevelCommentsDegradesUnavailableAuthorSummaryWithoutForgingPublicID(t *testing.T) {
	now := time.Date(2026, 7, 4, 15, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	comment := store.seedComment(t, domain.CommentSeed{ID: 5101, PostID: "post_pub_8", ContentInternalID: 9801, AuthorID: 701, Content: "author degraded", Status: domain.CommentStatusNormal, CreatedAt: now, UpdatedAt: now})
	store.queryResults[domain.CommentSortRecommended] = []domain.Comment{comment}
	store.stats[comment.ID] = domain.CommentStats{CommentID: comment.ID}
	users := &fakeUserProfileClient{batchErr: ports.ErrDependencyUnavailable}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_8", ContentInternalID: 9801, AuthorID: 701}},
		UserProfiles:  users,
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        &fakeOutboxPublisher{},
		Clock:         fixedClock{now: now},
	})

	page, err := service.ListTopLevelCommentsByPage(context.Background(), ListTopLevelCommentsQuery{PostID: "post_pub_8"})
	if err != nil {
		t.Fatalf("ListTopLevelCommentsByPage() error = %v", err)
	}
	if len(page.Items) != 1 || !page.Items[0].Author.Unavailable || page.Items[0].Author.PublicID != "" {
		t.Fatalf("degraded author = %#v", page.Items)
	}
}

func TestGetCommentDetailReturnsAuthorAndViewerState(t *testing.T) {
	now := time.Date(2026, 7, 5, 13, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	comment := store.seedComment(t, domain.CommentSeed{ID: 6101, PostID: "post_pub_detail", ContentInternalID: 9901, AuthorID: 701, Content: "detail", Status: domain.CommentStatusNormal, CreatedAt: now, UpdatedAt: now})
	store.stats[comment.ID] = domain.CommentStats{CommentID: comment.ID, LikeCount: 9, ReplyCount: 1}
	store.viewerLiked = map[domain.CommentID]bool{comment.ID: true}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: "post_pub_detail", ContentInternalID: 9901, AuthorID: 701}},
		UserProfiles:  &fakeUserProfileClient{summaries: map[domain.UserID]ports.AuthorSummary{701: {UserID: 701, PublicID: "user_pub_701", DisplayName: "Detail Author"}}},
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        &fakeOutboxPublisher{},
		Clock:         fixedClock{now: now},
	})

	item, err := service.GetCommentDetail(context.Background(), GetCommentDetailQuery{PostID: "post_pub_detail", CommentID: "c6101", ViewerUserID: 88})
	if err != nil {
		t.Fatalf("GetCommentDetail() error = %v", err)
	}
	if item.CommentID != "c6101" || item.Author.PublicID != "user_pub_701" || item.Stats.LikeCount != 9 {
		t.Fatalf("item = %#v", item)
	}
	if item.Viewer == nil || !item.Viewer.Liked {
		t.Fatalf("viewer state = %#v", item.Viewer)
	}
}

func TestListRepliesByPageDefaultsToHotAndUsesRootReplyTotal(t *testing.T) {
	now := time.Date(2026, 7, 5, 14, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	root := store.seedComment(t, domain.CommentSeed{ID: 6201, PostID: "post_pub_replies", ContentInternalID: 9902, AuthorID: 701, Content: "root", Status: domain.CommentStatusNormal, CreatedAt: now, UpdatedAt: now})
	first := store.seedComment(t, domain.CommentSeed{ID: 6202, PostID: root.PostID, ContentInternalID: root.ContentInternalID, AuthorID: 702, RootID: root.ID, ParentID: root.ID, Content: "first", Status: domain.CommentStatusNormal, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)})
	second := store.seedComment(t, domain.CommentSeed{ID: 6203, PostID: root.PostID, ContentInternalID: root.ContentInternalID, AuthorID: 703, RootID: root.ID, ParentID: root.ID, Content: "second", Status: domain.CommentStatusNormal, CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute)})
	store.stats[first.ID] = domain.CommentStats{CommentID: first.ID, LikeCount: 1}
	store.stats[second.ID] = domain.CommentStats{CommentID: second.ID, LikeCount: 10}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: root.PostID, ContentInternalID: root.ContentInternalID, AuthorID: root.AuthorID}},
		UserProfiles:  &fakeUserProfileClient{},
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        &fakeOutboxPublisher{},
		Clock:         fixedClock{now: now},
	})

	page, err := service.ListRepliesByPage(context.Background(), ListRepliesByPageQuery{PostID: PostID(root.PostID), RootCommentID: "c6201"})
	if err != nil {
		t.Fatalf("ListRepliesByPage() error = %v", err)
	}
	if store.lastReplyQuery.Sort != domain.CommentSortHot || store.lastReplyQuery.Page != 1 || store.lastReplyQuery.Size != 20 {
		t.Fatalf("last reply query = %#v", store.lastReplyQuery)
	}
	if page.Total != 2 || page.Pages != 1 || len(page.Items) != 2 {
		t.Fatalf("page = %#v", page)
	}
	if page.Items[0].CommentID != "c6203" || page.Items[0].RootCommentID != "c6201" || page.Items[0].ParentCommentID != "c6201" {
		t.Fatalf("first reply = %#v", page.Items[0])
	}
}

func TestGetLikeStatusReadsStrongViewerLikedFact(t *testing.T) {
	now := time.Date(2026, 7, 5, 15, 0, 0, 0, time.UTC)
	store := newFakeCommentStore()
	comment := store.seedComment(t, domain.CommentSeed{ID: 6301, PostID: "post_pub_like_status", ContentInternalID: 9903, AuthorID: 701, Content: "liked", Status: domain.CommentStatusNormal, CreatedAt: now, UpdatedAt: now})
	store.viewerLiked = map[domain.CommentID]bool{comment.ID: true}
	service := mustNewService(t, Dependencies{
		Commands:      store,
		Queries:       store,
		Stats:         store,
		PostStats:     store,
		ContentPosts:  &fakeContentPostClient{post: ports.CommentablePost{PostID: comment.PostID, ContentInternalID: comment.ContentInternalID, AuthorID: comment.AuthorID}},
		UserProfiles:  &fakeUserProfileClient{},
		UserRelations: &fakeUserRelationClient{},
		Files:         &fakeFileReferenceClient{},
		IDs:           publicIDCodec{},
		RateLimiter:   &fakeRateLimiter{},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        &fakeOutboxPublisher{},
		Clock:         fixedClock{now: now},
	})

	status, err := service.GetLikeStatus(context.Background(), GetLikeStatusQuery{PostID: "post_pub_like_status", CommentID: "c6301", ViewerUserID: 88})
	if err != nil {
		t.Fatalf("GetLikeStatus() error = %v", err)
	}
	if status.CommentID != "c6301" || !status.Liked {
		t.Fatalf("status = %#v", status)
	}
}
