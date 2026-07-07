package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestStoreMutateEngagementLikeInsertsRelationshipAndIncrementsStats(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	runner := NewTransactionRunner(db)
	occurredAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(mutateLikeEngagementSQL)).
		WithArgs("post_1", int64(42), occurredAt).
		WillReturnRows(engagementMutationRows().AddRow(
			int64(10), "post_1", int64(7), int64(42), true, true, false, int64(4),
			int64(0), int64(1), int64(0), int64(0),
		))
	mock.ExpectCommit()

	var got ports.EngagementMutationRecord
	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		var mutateErr error
		got, mutateErr = store.MutateEngagement(ctx, tx, ports.EngagementMutationInput{
			PostID:     "post_1",
			ActorID:    42,
			Action:     ports.EngagementActionLike,
			OccurredAt: occurredAt,
		})
		return mutateErr
	})
	if err != nil {
		t.Fatalf("MutateEngagement(like) error = %v", err)
	}
	if !got.Changed || !got.Liked || got.Stats.LikeCount != 1 || got.AggregateVersion != 4 {
		t.Fatalf("mutation = %+v, want changed like count 1 version 4", got)
	}
	assertExpectations(t, mock)
}

func TestStoreMutateEngagementDuplicateLikeDoesNotIncrementStats(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	runner := NewTransactionRunner(db)
	occurredAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(mutateLikeEngagementSQL)).
		WithArgs("post_1", int64(42), occurredAt).
		WillReturnRows(engagementMutationRows().AddRow(
			int64(10), "post_1", int64(7), int64(42), false, true, false, int64(4),
			int64(0), int64(1), int64(0), int64(0),
		))
	mock.ExpectCommit()

	var got ports.EngagementMutationRecord
	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		var mutateErr error
		got, mutateErr = store.MutateEngagement(ctx, tx, ports.EngagementMutationInput{
			PostID:     "post_1",
			ActorID:    42,
			Action:     ports.EngagementActionLike,
			OccurredAt: occurredAt,
		})
		return mutateErr
	})
	if err != nil {
		t.Fatalf("MutateEngagement(duplicate like) error = %v", err)
	}
	if got.Changed || !got.Liked || got.Stats.LikeCount != 1 {
		t.Fatalf("mutation = %+v, want unchanged existing like count", got)
	}
	assertExpectations(t, mock)
}

func TestStoreMutateEngagementDispatchesAllActions(t *testing.T) {
	testCases := []struct {
		name      string
		action    ports.EngagementAction
		wantSQL   string
		liked     bool
		favorited bool
	}{
		{name: "unlike", action: ports.EngagementActionUnlike, wantSQL: mutateUnlikeEngagementSQL, liked: false, favorited: true},
		{name: "favorite", action: ports.EngagementActionFavorite, wantSQL: mutateFavoriteEngagementSQL, liked: true, favorited: true},
		{name: "unfavorite", action: ports.EngagementActionUnfavorite, wantSQL: mutateUnfavoriteEngagementSQL, liked: true, favorited: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			store := NewStore(db, StoreConfig{})
			runner := NewTransactionRunner(db)
			occurredAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(tc.wantSQL)).
				WithArgs("post_1", int64(42), occurredAt).
				WillReturnRows(engagementMutationRows().AddRow(
					int64(10), "post_1", int64(7), int64(42), true, tc.liked, tc.favorited, int64(5),
					int64(0), int64(1), int64(1), int64(0),
				))
			mock.ExpectCommit()

			var got ports.EngagementMutationRecord
			err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
				var mutateErr error
				got, mutateErr = store.MutateEngagement(ctx, tx, ports.EngagementMutationInput{
					PostID:     "post_1",
					ActorID:    42,
					Action:     tc.action,
					OccurredAt: occurredAt,
				})
				return mutateErr
			})
			if err != nil {
				t.Fatalf("MutateEngagement(%s) error = %v", tc.action, err)
			}
			if got.Liked != tc.liked || got.Favorited != tc.favorited || got.AggregateVersion != 5 {
				t.Fatalf("mutation = %+v", got)
			}
			assertExpectations(t, mock)
		})
	}
}

func TestStoreGetPostEngagementReturnsStats(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})

	mock.ExpectQuery(regexp.QuoteMeta(getPostEngagementSQL)).
		WithArgs("post_1").
		WillReturnRows(sqlmock.NewRows([]string{"post_id", "view_count", "like_count", "favorite_count", "comment_count"}).
			AddRow("post_1", int64(10), int64(2), int64(3), int64(4)))

	got, err := store.GetPostEngagement(context.Background(), "post_1")
	if err != nil {
		t.Fatalf("GetPostEngagement() error = %v", err)
	}
	if got.PostID != "post_1" || got.Stats.ViewCount != 10 || got.Stats.FavoriteCount != 3 {
		t.Fatalf("engagement = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreGetPostEngagementMapsMissingPublishedPost(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})

	mock.ExpectQuery(regexp.QuoteMeta(getPostEngagementSQL)).
		WithArgs("post_missing").
		WillReturnError(sql.ErrNoRows)

	_, err := store.GetPostEngagement(context.Background(), "post_missing")
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("GetPostEngagement() error = %v, want ErrPostNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestStoreBatchGetViewerStatusUsesSingleBatchQuery(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})

	mock.ExpectQuery(regexp.QuoteMeta(batchGetViewerEngagementStatusSQL)).
		WithArgs(int64(42), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"post_id", "liked", "favorited"}).
			AddRow("post_1", true, false).
			AddRow("post_2", false, true))

	got, err := store.BatchGetViewerStatus(context.Background(), 42, []string{"post_1", "post_2"})
	if err != nil {
		t.Fatalf("BatchGetViewerStatus() error = %v", err)
	}
	if len(got) != 2 || !got[0].Liked || !got[1].Favorited {
		t.Fatalf("status = %+v", got)
	}
	assertExpectations(t, mock)
}

func engagementMutationRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"post_internal_id",
		"post_id",
		"author_id",
		"actor_id",
		"changed",
		"liked",
		"favorited",
		"aggregate_version",
		"view_count",
		"like_count",
		"favorite_count",
		"comment_count",
	})
}
