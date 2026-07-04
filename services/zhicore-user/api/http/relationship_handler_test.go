package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
)

func TestRelationshipCommandsUseTrustedIdentityAndPublicTarget(t *testing.T) {
	service := &fakeProfileService{}
	handler := NewHandler(service, nil)

	for _, tc := range []struct {
		name         string
		method       string
		path         string
		wantBlock    *application.BlockUserCommand
		wantUnblock  *application.UnblockUserCommand
		wantFollow   *application.FollowUserCommand
		wantUnfollow *application.UnfollowUserCommand
	}{
		{name: "block", method: http.MethodPost, path: "/api/v1/users/user_pub_target/block", wantBlock: &application.BlockUserCommand{ActorUserID: 42, TargetPublicID: "user_pub_target"}},
		{name: "unblock", method: http.MethodDelete, path: "/api/v1/users/user_pub_target/block", wantUnblock: &application.UnblockUserCommand{ActorUserID: 42, TargetPublicID: "user_pub_target"}},
		{name: "follow", method: http.MethodPost, path: "/api/v1/users/user_pub_target/follow", wantFollow: &application.FollowUserCommand{ActorUserID: 42, TargetPublicID: "user_pub_target"}},
		{name: "unfollow", method: http.MethodDelete, path: "/api/v1/users/user_pub_target/follow", wantUnfollow: &application.UnfollowUserCommand{ActorUserID: 42, TargetPublicID: "user_pub_target"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := withUserIDHeader(httptest.NewRequest(tc.method, tc.path, nil), "42")

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
			}
			if tc.wantBlock != nil && service.blockCmd != *tc.wantBlock {
				t.Fatalf("blockCmd = %#v, want %#v", service.blockCmd, *tc.wantBlock)
			}
			if tc.wantUnblock != nil && service.unblockCmd != *tc.wantUnblock {
				t.Fatalf("unblockCmd = %#v, want %#v", service.unblockCmd, *tc.wantUnblock)
			}
			if tc.wantFollow != nil && service.followCmd != *tc.wantFollow {
				t.Fatalf("followCmd = %#v, want %#v", service.followCmd, *tc.wantFollow)
			}
			if tc.wantUnfollow != nil && service.unfollowCmd != *tc.wantUnfollow {
				t.Fatalf("unfollowCmd = %#v, want %#v", service.unfollowCmd, *tc.wantUnfollow)
			}
		})
	}
}

func TestRelationshipCommandsRequireTrustedUserID(t *testing.T) {
	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/users/user_pub_target/block"},
		{method: http.MethodDelete, path: "/api/v1/users/user_pub_target/block"},
		{method: http.MethodPost, path: "/api/v1/users/user_pub_target/follow"},
		{method: http.MethodDelete, path: "/api/v1/users/user_pub_target/follow"},
		{method: http.MethodGet, path: "/api/v1/users/me/blocked"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			service := &fakeProfileService{}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)

			NewHandler(service, nil).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
			if service.blockCalls+service.unblockCalls+service.followCalls+service.unfollowCalls+service.listBlockedCalls != 0 {
				t.Fatalf("service should not be called without trusted identity")
			}
		})
	}
}

func TestRelationshipErrorsMapToPublicCodes(t *testing.T) {
	testCases := []struct {
		name       string
		method     string
		path       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "self block", method: http.MethodPost, path: "/api/v1/users/user_pub_self/block", err: application.ErrCannotBlockSelf, wantStatus: http.StatusBadRequest, wantCode: 3011},
		{name: "self follow", method: http.MethodPost, path: "/api/v1/users/user_pub_self/follow", err: application.ErrCannotFollowSelf, wantStatus: http.StatusBadRequest, wantCode: 3007},
		{name: "blocked follow", method: http.MethodPost, path: "/api/v1/users/user_pub_target/follow", err: application.ErrInteractionBlocked, wantStatus: http.StatusForbidden, wantCode: 3010},
		{name: "invalid cursor", method: http.MethodGet, path: "/api/v1/users/user_pub_target/followers?cursor=bad", err: application.ErrCursorInvalid, wantStatus: http.StatusBadRequest, wantCode: 1001},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeProfileService{}
			switch tc.method + " " + routeFamily(tc.path) {
			case http.MethodPost + " block":
				service.blockErr = tc.err
			case http.MethodPost + " follow":
				service.followErr = tc.err
			case http.MethodGet + " followers":
				service.listFollowersErr = tc.err
			}
			rr := httptest.NewRecorder()
			req := withUserIDHeader(httptest.NewRequest(tc.method, tc.path, nil), "42")

			NewHandler(service, nil).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestRelationshipDuplicateCommandsReturnIdempotentSuccess(t *testing.T) {
	for _, tc := range []struct {
		method string
		path   string
		field  string
	}{
		{method: http.MethodPost, path: "/api/v1/users/user_pub_target/block", field: "blocked"},
		{method: http.MethodDelete, path: "/api/v1/users/user_pub_target/block", field: "blocked"},
		{method: http.MethodPost, path: "/api/v1/users/user_pub_target/follow", field: "following"},
		{method: http.MethodDelete, path: "/api/v1/users/user_pub_target/follow", field: "following"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			service := &fakeProfileService{}
			handler := NewHandler(service, nil)
			for i := 0; i < 2; i++ {
				rr := httptest.NewRecorder()
				req := withUserIDHeader(httptest.NewRequest(tc.method, tc.path, nil), "42")

				handler.ServeHTTP(rr, req)

				var body envelope[map[string]bool]
				decodeJSON(t, rr.Body.Bytes(), &body)
				if rr.Code != http.StatusOK || body.Code != 200 || body.Timestamp <= 0 {
					t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
				}
				if _, ok := body.Data[tc.field]; !ok {
					t.Fatalf("response data missing %q: %#v", tc.field, body.Data)
				}
			}
		})
	}
}

func TestRelationshipListReturnsCursorPage(t *testing.T) {
	service := &fakeProfileService{
		relationshipPage: application.RelationshipProfilePage{
			Items: []application.Profile{
				newProfile(t, application.ProfileSeed{UserID: 88, PublicID: "user_pub_88", AccountID: 1088, Nickname: "Bob", StrangerMessageAllowed: true, Status: application.UserStatusActive}),
			},
			NextCursor: "12",
			HasMore:    true,
		},
	}
	rr := httptest.NewRecorder()
	req := withUserIDHeader(httptest.NewRequest(http.MethodGet, "/api/v1/users/me/blocked?cursor=22&limit=1", nil), "42")

	NewHandler(service, nil).ServeHTTP(rr, req)

	if service.listBlockedQuery.ActorUserID != 42 || service.listBlockedQuery.Cursor != "22" || service.listBlockedQuery.Limit != 1 {
		t.Fatalf("listBlockedQuery = %#v", service.listBlockedQuery)
	}
	var body envelope[relationshipPageResponse]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 {
		t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].PublicID != "user_pub_88" || body.Data.NextCursor != "12" || !body.Data.HasMore {
		t.Fatalf("relationship page = %#v", body.Data)
	}
}

func routeFamily(path string) string {
	switch {
	case len(path) >= len("/api/v1/users/x/block") && path[len(path)-len("/block"):] == "/block":
		return "block"
	case len(path) >= len("/api/v1/users/x/follow") && path[len(path)-len("/follow"):] == "/follow":
		return "follow"
	default:
		return "followers"
	}
}

type relationshipPageResponse struct {
	Items      []profileResponse `json:"items"`
	NextCursor string            `json:"nextCursor,omitempty"`
	HasMore    bool              `json:"hasMore"`
}
