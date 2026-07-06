package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	refreshTokenCookieName     = "refresh_token"
	csrfTokenCookieName        = "csrf_token"
	csrfHeaderName             = "X-CSRF-Token"
	authCookiePath             = "/api/v1/auth"
	accountIDHeaderName        = "X-Account-Id"
	userIDHeaderName           = "X-User-Id"
	userRolesHeaderName        = "X-User-Roles"
	sessionIDHeaderName        = "X-Session-Id"
	sessionVersionHeaderName   = "X-Session-Version"
	principalVersionHeaderName = "X-Principal-Version"

	defaultPageSize = 20
	maxPageSize     = 50
	fixedExpiresIn  = 7200
)

var (
	ErrEmailInvalid             = errors.New("email is invalid")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrAccountDisabled          = errors.New("account is disabled")
	ErrAccountBanned            = errors.New("account is banned")
	ErrAccountLocked            = errors.New("account is locked")
	ErrRateLimited              = errors.New("rate limit exceeded")
	ErrServiceDegraded          = errors.New("service degraded")
	ErrPasswordInvalid          = errors.New("password is invalid")
	ErrEmailExists              = errors.New("email already exists")
	ErrRegisterPendingRetryable = errors.New("register pending retryable")
	ErrTokenInvalid             = errors.New("token is invalid")
	ErrTokenExpired             = errors.New("token is expired")
	ErrTokenReplayed            = errors.New("token is replayed")
	ErrSessionRevoked           = errors.New("session is revoked")
	ErrCSRFInvalid              = errors.New("csrf is invalid")
	ErrLoginRequired            = errors.New("login required")
	ErrPermissionDenied         = errors.New("permission denied")
	ErrRoleRequired             = errors.New("role required")
	ErrResourceAccessDenied     = errors.New("resource access denied")
	ErrPrincipalUnavailable     = errors.New("principal unavailable")
	ErrDataNotFound             = errors.New("data not found")
)

type Service interface {
	Register(ctx context.Context, cmd RegisterCommand) (RegisterResult, error)
	Login(ctx context.Context, cmd LoginCommand) (LoginResult, error)
	Refresh(ctx context.Context, cmd RefreshCommand) (RefreshResult, error)
	Logout(ctx context.Context, cmd LogoutCommand) (LogoutResult, error)
	GetCurrentPrincipal(ctx context.Context, query CurrentPrincipalQuery) (Principal, error)
	IssueCSRFToken(ctx context.Context) (IssueCSRFTokenResult, error)
	ListSessions(ctx context.Context, query ListSessionsQuery) (ListSessionsResult, error)
	RevokeSession(ctx context.Context, cmd RevokeSessionCommand) (RevokeSessionResult, error)
	GetSecurityOperation(ctx context.Context, query GetSecurityOperationQuery) (SecurityOperationResult, error)
}

type RegisterCommand struct {
	Email                  string
	Nickname               string
	Password               string
	EmailVerificationToken string
}

type RegisterResult struct {
	Registered            bool
	Authenticated         bool
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	Principal             Principal
	LoginDeferredReason   string
}

type LoginCommand struct {
	Email    string
	Password string
}

type LoginResult struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	Principal             Principal
}

type RefreshCommand struct {
	RefreshToken string
	CSRFToken    string
}

type RefreshResult struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	Principal             Principal
	Processing            *AcceptedSecurityOperation
}

type LogoutCommand struct {
	Identity     TrustedIdentity
	HasIdentity  bool
	RefreshToken string
	CSRFToken    string
}

type LogoutResult struct {
	LoggedOut     bool
	ServerRevoked bool
	Processing    *AcceptedSecurityOperation
}

type CurrentPrincipalQuery struct {
	Identity TrustedIdentity
}

type IssueCSRFTokenResult struct {
	CSRFToken string
}

type ListSessionsQuery struct {
	Identity TrustedIdentity
	Page     int
	Size     int
}

type SessionSummary struct {
	SessionID   string
	CreatedAt   time.Time
	LastSeenAt  *time.Time
	ExpiresAt   time.Time
	DeviceLabel string
	Current     bool
}

type ListSessionsResult struct {
	Items []SessionSummary
	Page  int
	Size  int
	Total int
}

type RevokeSessionCommand struct {
	Identity    TrustedIdentity
	SessionID   string
	CurrentOnly bool
	CSRFToken   string
}

type RevokeSessionResult struct {
	SessionID  string
	Current    bool
	Processing *AcceptedSecurityOperation
}

type GetSecurityOperationQuery struct {
	Identity    TrustedIdentity
	OperationID string
}

type SecurityOperationResult struct {
	OperationID       string
	Type              string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CompletedAt       *time.Time
	RetryAfterSeconds *int
	ErrorCode         *string
}

type AcceptedSecurityOperation struct {
	OperationID       string
	RetryAfterSeconds int
}

type Principal struct {
	AccountID        string
	UserID           string
	Email            string
	Roles            []string
	AccountStatus    string
	SessionID        string
	SessionVersion   int64
	PrincipalVersion int64
}

type TrustedIdentity struct {
	AccountID        string
	UserID           string
	SessionID        string
	SessionVersion   int64
	PrincipalVersion int64
	Roles            []string
}

type Handler struct {
	service Service
	router  *gin.Engine
}

func NewHandler(service Service) *gin.Engine {
	h := &Handler{
		service: service,
		router:  gin.New(),
	}
	h.routes()
	return h.router
}

func (h *Handler) routes() {
	h.router.POST("/api/v1/auth/register", h.register)
	h.router.POST("/api/v1/auth/login", h.login)
	h.router.POST("/api/v1/auth/refresh", h.refresh)
	h.router.POST("/api/v1/auth/logout", h.logout)
	h.router.GET("/api/v1/auth/me", h.me)
	h.router.GET("/api/v1/auth/csrf", h.csrf)
	h.router.GET("/api/v1/auth/sessions", h.listSessions)
	h.router.DELETE("/api/v1/auth/sessions/current", h.revokeCurrentSession)
	h.router.DELETE("/api/v1/auth/sessions/:sessionId", h.revokeSession)
	h.router.GET("/api/v1/auth/security-operations/:operationId", h.getSecurityOperation)
}
