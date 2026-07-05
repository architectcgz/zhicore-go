package runtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authhttp "github.com/architectcgz/zhicore-go/services/zhicore-auth/api/http"
)

func TestBuildRejectsMissingServiceDependency(t *testing.T) {
	_, err := Build(Deps{})
	if err == nil || !strings.Contains(err.Error(), "Service") {
		t.Fatalf("Build() error = %v, want mention Service", err)
	}
}

func TestBuildReturnsAuthAndHealthHandlers(t *testing.T) {
	module, err := Build(Deps{Service: stubService{csrfToken: "csrf-token"}})
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/csrf", nil)
	rec := httptest.NewRecorder()
	module.HTTPHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("csrf status = %d, want 200", rec.Code)
	}
}

type stubService struct {
	csrfToken string
}

func (s stubService) Register(context.Context, authhttp.RegisterCommand) (authhttp.RegisterResult, error) {
	return authhttp.RegisterResult{}, errors.New("not implemented")
}

func (s stubService) Login(context.Context, authhttp.LoginCommand) (authhttp.LoginResult, error) {
	return authhttp.LoginResult{}, errors.New("not implemented")
}

func (s stubService) Refresh(context.Context, authhttp.RefreshCommand) (authhttp.RefreshResult, error) {
	return authhttp.RefreshResult{}, errors.New("not implemented")
}

func (s stubService) Logout(context.Context, authhttp.LogoutCommand) (authhttp.LogoutResult, error) {
	return authhttp.LogoutResult{}, errors.New("not implemented")
}

func (s stubService) GetCurrentPrincipal(context.Context, authhttp.CurrentPrincipalQuery) (authhttp.Principal, error) {
	return authhttp.Principal{}, errors.New("not implemented")
}

func (s stubService) IssueCSRFToken(context.Context) (authhttp.IssueCSRFTokenResult, error) {
	return authhttp.IssueCSRFTokenResult{CSRFToken: s.csrfToken}, nil
}

func (s stubService) ListSessions(context.Context, authhttp.ListSessionsQuery) (authhttp.ListSessionsResult, error) {
	return authhttp.ListSessionsResult{}, errors.New("not implemented")
}

func (s stubService) RevokeSession(context.Context, authhttp.RevokeSessionCommand) (authhttp.RevokeSessionResult, error) {
	return authhttp.RevokeSessionResult{}, errors.New("not implemented")
}

func (s stubService) GetSecurityOperation(context.Context, authhttp.GetSecurityOperationQuery) (authhttp.SecurityOperationResult, error) {
	return authhttp.SecurityOperationResult{
		OperationID: "op-1",
		Type:        "LOGOUT_CURRENT",
		Status:      "SUCCEEDED",
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}, nil
}
