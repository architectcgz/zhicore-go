package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
)

func TestProfileGetMeRequiresTrustedUserID(t *testing.T) {
	service := &fakeProfileService{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)

	NewHandler(service, nil).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
	if service.getMyCalls != 0 {
		t.Fatalf("getMyCalls = %d, want 0", service.getMyCalls)
	}
}

func TestProfileGetMeReturnsProfileEnvelope(t *testing.T) {
	service := &fakeProfileService{
		getMyProfile: newProfile(t, application.ProfileSeed{
			UserID:                 42,
			PublicID:               "user_pub_42",
			AccountID:              1042,
			Nickname:               "Alice",
			AvatarFileID:           "avatar-file-42",
			Bio:                    "hello",
			StrangerMessageAllowed: true,
			Status:                 application.UserStatusActive,
			ProfileVersion:         7,
		}),
	}
	resolver := &fakeAvatarURLResolver{url: "https://cdn.example.com/avatar-file-42.jpg"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("X-User-Id", "42")

	NewHandler(service, resolver).ServeHTTP(rr, req)

	if service.getMyUserID != 42 {
		t.Fatalf("getMyUserID = %d, want 42", service.getMyUserID)
	}
	if resolver.fileID != "avatar-file-42" {
		t.Fatalf("resolver fileID = %q, want avatar-file-42", resolver.fileID)
	}

	var body envelope[profileResponse]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if body.Code != 200 || body.Message != "操作成功" || body.Timestamp <= 0 {
		t.Fatalf("envelope = %#v", body)
	}
	if body.Data.PublicID != "user_pub_42" || body.Data.Nickname != "Alice" || body.Data.AvatarURL != "https://cdn.example.com/avatar-file-42.jpg" {
		t.Fatalf("profile response = %#v", body.Data)
	}
}

func TestProfileGetMeMapsProfileNotFound(t *testing.T) {
	service := &fakeProfileService{getMyErr: application.ErrProfileNotFound}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("X-User-Id", "42")
	NewHandler(service, nil).ServeHTTP(rr, req)
	assertErrorEnvelope(t, rr, http.StatusNotFound, 3001)
}

func TestProfileGetPublicProfileRejectsInvalidPublicID(t *testing.T) {
	service := &fakeProfileService{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/%20", nil)

	NewHandler(service, nil).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
	if service.getPublicCalls != 0 {
		t.Fatalf("getPublicCalls = %d, want 0", service.getPublicCalls)
	}
}

func TestProfileGetPublicProfileMapsNotFound(t *testing.T) {
	service := &fakeProfileService{getPublicErr: application.ErrProfileNotFound}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user_pub_99", nil)

	NewHandler(service, nil).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusNotFound, 3001)
	if service.getPublicID != "user_pub_99" {
		t.Fatalf("getPublicID = %q, want user_pub_99", service.getPublicID)
	}
}

func TestProfileGetPublicProfileOmitsAvatarURLWhenResolverFails(t *testing.T) {
	service := &fakeProfileService{
		getPublicProfile: newProfile(t, application.ProfileSeed{
			UserID:                 77,
			PublicID:               "user_pub_77",
			AccountID:              1077,
			Nickname:               "Bob",
			AvatarFileID:           "avatar-file-77",
			Bio:                    "public profile",
			StrangerMessageAllowed: false,
			Status:                 application.UserStatusActive,
			ProfileVersion:         4,
		}),
	}
	resolver := &fakeAvatarURLResolver{err: errors.New("resolver unavailable")}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user_pub_77", nil)

	NewHandler(service, resolver).ServeHTTP(rr, req)

	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 {
		t.Fatalf("status=%d code=%d body=%s", rr.Code, body.Code, rr.Body.String())
	}
	if _, ok := body.Data["avatarUrl"]; ok {
		t.Fatalf("avatarUrl = %#v, want omitted; data=%#v", body.Data["avatarUrl"], body.Data)
	}
	if got := body.Data["avatarFileId"]; got != "avatar-file-77" {
		t.Fatalf("avatarFileId = %#v, want avatar-file-77", got)
	}
}

func TestProfileDependencyUnavailableMapsServiceUnavailable(t *testing.T) {
	testCases := []struct {
		name    string
		request *http.Request
		service *fakeProfileService
	}{
		{
			name:    "get me",
			request: withUserIDHeader(httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil), "88"),
			service: &fakeProfileService{getMyErr: application.ErrDependencyUnavailable},
		},
		{
			name:    "get profile",
			request: httptest.NewRequest(http.MethodGet, "/api/v1/users/user_pub_88", nil),
			service: &fakeProfileService{getPublicErr: application.ErrDependencyUnavailable},
		},
		{
			name:    "update profile",
			request: withUserIDHeader(withJSONHeader(httptest.NewRequest(http.MethodPatch, "/api/v1/users/me/profile", bytes.NewBufferString(`{"nickname":"Alice2"}`))), "88"),
			service: &fakeProfileService{updateErr: application.ErrDependencyUnavailable},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			NewHandler(tc.service, nil).ServeHTTP(rr, tc.request)
			assertErrorEnvelope(t, rr, http.StatusServiceUnavailable, 1004)
		})
	}
}

func TestProfileUpdateUsesTrustedIdentityAndForwardsOnlyPresentPatchFields(t *testing.T) {
	service := &fakeProfileService{
		updateProfile: newProfile(t, application.ProfileSeed{
			UserID:                 88,
			PublicID:               "user_pub_88",
			AccountID:              1088,
			Nickname:               "Alice2",
			AvatarFileID:           "avatar-old",
			Bio:                    "old bio",
			StrangerMessageAllowed: true,
			Status:                 application.UserStatusActive,
			ProfileVersion:         4,
		}),
	}
	rr := httptest.NewRecorder()
	req := withUserIDHeader(
		withJSONHeader(httptest.NewRequest(http.MethodPatch, "/api/v1/users/me/profile", bytes.NewBufferString(`{"userId":999,"actor":{"userId":999},"nickname":"Alice2"}`))),
		"88",
	)
	NewHandler(service, nil).ServeHTTP(rr, req)
	if service.updateCmd.UserID != 88 {
		t.Fatalf("update userID = %d, want 88", service.updateCmd.UserID)
	}
	assertOptionalString(t, service.updateCmd.Nickname, "Alice2")
	if service.updateCmd.AvatarFileID != nil || service.updateCmd.Bio != nil || service.updateCmd.StrangerMessageAllowed != nil {
		t.Fatalf("update command = %#v", service.updateCmd)
	}

	var body envelope[profileResponse]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Timestamp <= 0 {
		t.Fatalf("status=%d envelope=%#v", rr.Code, body)
	}
	if body.Data.Nickname != "Alice2" || body.Data.ProfileVersion != 4 {
		t.Fatalf("profile response = %#v", body.Data)
	}
}

func TestProfileUpdateDoesNotPrefetchCurrentProfileForPatchMerge(t *testing.T) {
	service := &fakeProfileService{
		getMyErr: errors.New("handler must not preload current profile"),
		updateProfile: newProfile(t, application.ProfileSeed{
			UserID:                 188,
			PublicID:               "user_pub_188",
			AccountID:              1188,
			Nickname:               "Alice2",
			AvatarFileID:           "avatar-from-application",
			Bio:                    "bio from application",
			StrangerMessageAllowed: true,
			Status:                 application.UserStatusActive,
			ProfileVersion:         5,
		}),
	}
	rr := httptest.NewRecorder()
	req := withUserIDHeader(
		withJSONHeader(httptest.NewRequest(http.MethodPatch, "/api/v1/users/me/profile", bytes.NewBufferString(`{"nickname":"Alice2","userId":999}`))),
		"188",
	)

	NewHandler(service, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if service.getMyCalls != 0 {
		t.Fatalf("getMyCalls = %d, want 0", service.getMyCalls)
	}
	if service.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", service.updateCalls)
	}
	if service.updateCmd.UserID != 188 {
		t.Fatalf("update command userID = %d, want 188", service.updateCmd.UserID)
	}
}

func TestProfileUpdateRequiresTrustedUserID(t *testing.T) {
	service := &fakeProfileService{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me/profile", bytes.NewBufferString(`{"nickname":"Alice2"}`))
	req.Header.Set("Content-Type", "application/json")

	NewHandler(service, nil).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
	if service.updateCalls != 0 {
		t.Fatalf("updateCalls = %d, want 0", service.updateCalls)
	}
}

func TestProfileUpdateMapsUserContractErrors(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "nickname taken", err: application.ErrNicknameTaken, wantStatus: http.StatusConflict, wantCode: 3005},
		{name: "nickname invalid", err: application.ErrNicknameInvalid, wantStatus: http.StatusBadRequest, wantCode: 3013},
		{name: "bio invalid", err: application.ErrBioInvalid, wantStatus: http.StatusBadRequest, wantCode: 3014},
		{name: "avatar invalid", err: application.ErrAvatarInvalid, wantStatus: http.StatusBadRequest, wantCode: 3015},
		{name: "user not active", err: application.ErrUserNotActive, wantStatus: http.StatusForbidden, wantCode: 3006},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeProfileService{updateErr: tc.err}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me/profile", bytes.NewBufferString(`{"nickname":"Alice2","bio":"new"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-Id", "91")

			NewHandler(service, nil).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

type fakeProfileService struct {
	getMyProfile     application.Profile
	getMyErr         error
	getPublicProfile application.Profile
	getPublicErr     error
	updateProfile    application.Profile
	updateErr        error
	relationshipPage application.RelationshipProfilePage
	blockErr         error
	unblockErr       error
	followErr        error
	unfollowErr      error
	listBlockedErr   error
	listFollowersErr error
	listFollowingErr error
	batchSimpleErr   error
	availabilityErr  error
	batchBlockedErr  error

	getMyCalls         int
	getPublicCalls     int
	updateCalls        int
	blockCalls         int
	unblockCalls       int
	followCalls        int
	unfollowCalls      int
	listBlockedCalls   int
	listFollowersCalls int
	listFollowingCalls int
	batchSimpleCalls   int
	availabilityCalls  int
	batchBlockedCalls  int

	getMyUserID        application.UserID
	getPublicID        application.PublicID
	updateCmd          application.UpdateProfileCommand
	blockCmd           application.BlockUserCommand
	unblockCmd         application.UnblockUserCommand
	followCmd          application.FollowUserCommand
	unfollowCmd        application.UnfollowUserCommand
	listBlockedQuery   application.ListBlockedUsersQuery
	listFollowersQuery application.ListFollowersQuery
	listFollowingQuery application.ListFollowingQuery
	batchSimpleIDs     []application.UserID
	availabilityIDs    []application.UserID
	batchBlockedPairs  []application.UserPair
	batchSimpleResult  application.BatchUserSimpleResult
	availabilityItems  []application.UserAvailability
	batchBlockedResult map[application.UserPair]bool
}

func (f *fakeProfileService) GetMyProfile(_ context.Context, userID application.UserID) (application.Profile, error) {
	f.getMyCalls++
	f.getMyUserID = userID
	if f.getMyErr != nil {
		return application.Profile{}, f.getMyErr
	}
	return f.getMyProfile, nil
}

func (f *fakeProfileService) GetUserProfileByPublicID(_ context.Context, publicID application.PublicID) (application.Profile, error) {
	f.getPublicCalls++
	f.getPublicID = publicID
	if f.getPublicErr != nil {
		return application.Profile{}, f.getPublicErr
	}
	return f.getPublicProfile, nil
}

func (f *fakeProfileService) UpdateProfile(_ context.Context, cmd application.UpdateProfileCommand) (application.Profile, error) {
	f.updateCalls++
	f.updateCmd = cmd
	if f.updateErr != nil {
		return application.Profile{}, f.updateErr
	}
	return f.updateProfile, nil
}

func (f *fakeProfileService) BlockUser(_ context.Context, cmd application.BlockUserCommand) error {
	f.blockCalls++
	f.blockCmd = cmd
	return f.blockErr
}

func (f *fakeProfileService) UnblockUser(_ context.Context, cmd application.UnblockUserCommand) error {
	f.unblockCalls++
	f.unblockCmd = cmd
	return f.unblockErr
}

func (f *fakeProfileService) FollowUser(_ context.Context, cmd application.FollowUserCommand) error {
	f.followCalls++
	f.followCmd = cmd
	return f.followErr
}

func (f *fakeProfileService) UnfollowUser(_ context.Context, cmd application.UnfollowUserCommand) error {
	f.unfollowCalls++
	f.unfollowCmd = cmd
	return f.unfollowErr
}

func (f *fakeProfileService) ListBlockedUsers(_ context.Context, query application.ListBlockedUsersQuery) (application.RelationshipProfilePage, error) {
	f.listBlockedCalls++
	f.listBlockedQuery = query
	if f.listBlockedErr != nil {
		return application.RelationshipProfilePage{}, f.listBlockedErr
	}
	return f.relationshipPage, nil
}

func (f *fakeProfileService) ListFollowers(_ context.Context, query application.ListFollowersQuery) (application.RelationshipProfilePage, error) {
	f.listFollowersCalls++
	f.listFollowersQuery = query
	if f.listFollowersErr != nil {
		return application.RelationshipProfilePage{}, f.listFollowersErr
	}
	return f.relationshipPage, nil
}

func (f *fakeProfileService) ListFollowing(_ context.Context, query application.ListFollowingQuery) (application.RelationshipProfilePage, error) {
	f.listFollowingCalls++
	f.listFollowingQuery = query
	if f.listFollowingErr != nil {
		return application.RelationshipProfilePage{}, f.listFollowingErr
	}
	return f.relationshipPage, nil
}

func (f *fakeProfileService) BatchGetUserSimple(_ context.Context, userIDs []application.UserID) (application.BatchUserSimpleResult, error) {
	f.batchSimpleCalls++
	f.batchSimpleIDs = append([]application.UserID(nil), userIDs...)
	if f.batchSimpleErr != nil {
		return application.BatchUserSimpleResult{}, f.batchSimpleErr
	}
	return f.batchSimpleResult, nil
}

func (f *fakeProfileService) BatchGetUserAvailability(_ context.Context, userIDs []application.UserID) ([]application.UserAvailability, error) {
	f.availabilityCalls++
	f.availabilityIDs = append([]application.UserID(nil), userIDs...)
	if f.availabilityErr != nil {
		return nil, f.availabilityErr
	}
	return f.availabilityItems, nil
}

func (f *fakeProfileService) BatchCheckBlocked(_ context.Context, pairs []application.UserPair) (map[application.UserPair]bool, error) {
	f.batchBlockedCalls++
	f.batchBlockedPairs = append([]application.UserPair(nil), pairs...)
	if f.batchBlockedErr != nil {
		return nil, f.batchBlockedErr
	}
	return f.batchBlockedResult, nil
}

type fakeAvatarURLResolver struct {
	url    string
	err    error
	fileID string
}

func (f *fakeAvatarURLResolver) ResolveAvatarURL(_ context.Context, fileID string) (string, error) {
	f.fileID = fileID
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

type envelope[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

type profileResponse struct {
	PublicID               string `json:"publicId"`
	Nickname               string `json:"nickname"`
	AvatarFileID           string `json:"avatarFileId,omitempty"`
	AvatarURL              string `json:"avatarUrl,omitempty"`
	Bio                    string `json:"bio,omitempty"`
	StrangerMessageAllowed bool   `json:"strangerMessageAllowed"`
	ProfileVersion         int64  `json:"profileVersion"`
}

func assertErrorEnvelope(t *testing.T, rr *httptest.ResponseRecorder, wantStatus, wantCode int) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var body envelope[json.RawMessage]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if body.Code != wantCode || body.Timestamp <= 0 {
		t.Fatalf("error envelope = %#v, want code=%d timestamp>0", body, wantCode)
	}
}

func decodeJSON(t *testing.T, payload []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; payload=%s", err, string(payload))
	}
}

func newProfile(t *testing.T, seed application.ProfileSeed) application.Profile {
	t.Helper()
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	if seed.CreatedAt.IsZero() {
		seed.CreatedAt = now
	}
	if seed.UpdatedAt.IsZero() {
		seed.UpdatedAt = now
	}
	profile, err := application.NewProfile(seed)
	if err != nil {
		t.Fatalf("application.NewProfile() error = %v", err)
	}
	return profile
}

func withJSONHeader(req *http.Request) *http.Request {
	req.Header.Set("Content-Type", "application/json")
	return req
}

func withUserIDHeader(req *http.Request, userID string) *http.Request {
	req.Header.Set("X-User-Id", userID)
	return req
}

func assertOptionalString(t *testing.T, value *string, want string) {
	t.Helper()
	if value == nil || *value != want {
		t.Fatalf("optional string = %#v, want %q", value, want)
	}
}
