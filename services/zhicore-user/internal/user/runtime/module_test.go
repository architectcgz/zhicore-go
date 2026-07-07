package runtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	userhttp "github.com/architectcgz/zhicore-go/services/zhicore-user/api/http"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
)

func TestBuildRejectsMissingServiceDependency(t *testing.T) {
	_, err := Build(Deps{})
	if err == nil || !strings.Contains(err.Error(), "Service") {
		t.Fatalf("Build() error = %v, want mention Service", err)
	}
}

func TestBuildReturnsUserAndHealthHandlers(t *testing.T) {
	module, err := Build(Deps{Service: stubService{profile: testProfile()}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if module.HTTPHandler == nil {
		t.Fatal("Build() returned nil HTTPHandler")
	}

	for _, path := range []string{"/health/live", "/health/ready"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		module.HTTPHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("X-User-Id", "42")
	rec := httptest.NewRecorder()
	module.HTTPHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get me status = %d, want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, usercontract.BatchAvailabilityPath, strings.NewReader(`{"userIds":[42]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-Service", "zhicore-comment")
	req.Header.Set("X-Caller-Operation", usercontract.OperationCommentCheckUserAvailability)
	rec = httptest.NewRecorder()
	module.HTTPHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("internal availability status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

type stubService struct {
	profile application.Profile
}

func (s stubService) GetMyProfile(context.Context, application.UserID) (application.Profile, error) {
	return s.profile, nil
}

func (s stubService) GetUserProfileByPublicID(context.Context, application.PublicID) (application.Profile, error) {
	return application.Profile{}, errors.New("not implemented")
}

func (s stubService) UpdateProfile(context.Context, application.UpdateProfileCommand) (application.Profile, error) {
	return application.Profile{}, errors.New("not implemented")
}

func (s stubService) BlockUser(context.Context, application.BlockUserCommand) error {
	return errors.New("not implemented")
}

func (s stubService) UnblockUser(context.Context, application.UnblockUserCommand) error {
	return errors.New("not implemented")
}

func (s stubService) ListBlockedUsers(context.Context, application.ListBlockedUsersQuery) (application.RelationshipProfilePage, error) {
	return application.RelationshipProfilePage{}, errors.New("not implemented")
}

func (s stubService) FollowUser(context.Context, application.FollowUserCommand) error {
	return errors.New("not implemented")
}

func (s stubService) UnfollowUser(context.Context, application.UnfollowUserCommand) error {
	return errors.New("not implemented")
}

func (s stubService) ListFollowers(context.Context, application.ListFollowersQuery) (application.RelationshipProfilePage, error) {
	return application.RelationshipProfilePage{}, errors.New("not implemented")
}

func (s stubService) ListFollowing(context.Context, application.ListFollowingQuery) (application.RelationshipProfilePage, error) {
	return application.RelationshipProfilePage{}, errors.New("not implemented")
}

func (s stubService) BatchGetUserSimple(context.Context, []application.UserID) (application.BatchUserSimpleResult, error) {
	return application.BatchUserSimpleResult{}, nil
}

func (s stubService) BatchGetUserAvailability(context.Context, []application.UserID) ([]application.UserAvailability, error) {
	return []application.UserAvailability{{UserID: 42, Available: true, Status: application.UserStatusActive}}, nil
}

func (s stubService) BatchCheckBlocked(context.Context, []application.UserPair) (map[application.UserPair]bool, error) {
	return map[application.UserPair]bool{}, nil
}

func (s stubService) ListFollowerShard(context.Context, application.ListFollowerShardQuery) (application.FollowerShardPage, error) {
	return application.FollowerShardPage{}, nil
}

func testProfile() application.Profile {
	profile, err := application.NewProfile(application.ProfileSeed{
		UserID:                 42,
		PublicID:               "u_42",
		AccountID:              84,
		Nickname:               "alice",
		StrangerMessageAllowed: true,
		Status:                 application.UserStatusActive,
	})
	if err != nil {
		panic(err)
	}
	return profile
}

var _ userhttp.Service = stubService{}
