package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/application"
	"github.com/gin-gonic/gin"
)

const userIDHeaderName = "X-User-Id"

var errLoginRequired = errors.New("login required")

type Service interface {
	CreateComment(ctx context.Context, cmd application.CreateCommentCommand) (application.CreateCommentResult, error)
	ListTopLevelCommentsByPage(ctx context.Context, query application.ListTopLevelCommentsQuery) (application.TopLevelCommentPage, error)
}

type Handler struct {
	service Service
	router  *gin.Engine
}

func NewHandler(service Service) http.Handler {
	h := &Handler{service: service, router: gin.New()}
	h.routes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	h.router.POST("/api/v1/posts/:postId/comments", ginHTTPHandler(h.createComment))
	h.router.GET("/api/v1/posts/:postId/comments/page", ginHTTPHandler(h.listCommentsPage))
}

func ginHTTPHandler(next http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Route params are copied back to net/http so Gin remains confined to
		// the transport adapter instead of becoming an application dependency.
		for _, param := range c.Params {
			c.Request.SetPathValue(param.Key, param.Value)
		}
		next(c.Writer, c.Request)
	}
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	actorID, ok := trustedUserIDFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, r)
	if !ok {
		return
	}
	var req createCommentReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	result, err := h.service.CreateComment(r.Context(), application.CreateCommentCommand{
		ActorUserID:     actorID,
		PostID:          postID,
		ParentCommentID: application.PublicCommentID(strings.TrimSpace(req.ParentCommentID)),
		Content:         req.Content,
		ImageFileIDs:    req.ImageFileIDs,
		VoiceFileID:     req.VoiceFileID,
		VoiceDuration:   req.VoiceDuration,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, createCommentResp{
		PostID:          string(result.PostID),
		CommentID:       string(result.CommentID),
		RootCommentID:   string(result.RootCommentID),
		ParentCommentID: string(result.ParentCommentID),
		CreatedAt:       formatTime(result.CreatedAt),
	})
}

func (h *Handler) listCommentsPage(w http.ResponseWriter, r *http.Request) {
	postID, ok := postIDFromPath(w, r)
	if !ok {
		return
	}
	page, size, sort, ok := decodeListCommentsPageQuery(w, r)
	if !ok {
		return
	}
	viewerID, _ := trustedUserIDFromRequest(r)
	result, err := h.service.ListTopLevelCommentsByPage(r.Context(), application.ListTopLevelCommentsQuery{
		PostID:       postID,
		ViewerUserID: viewerID,
		Page:         page,
		Size:         size,
		Sort:         sort,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, topLevelCommentPageResponse(result))
}

type createCommentReq struct {
	Content         string   `json:"content"`
	ParentCommentID string   `json:"parentCommentId"`
	ImageFileIDs    []string `json:"imageFileIds"`
	VoiceFileID     string   `json:"voiceFileId"`
	VoiceDuration   int      `json:"voiceDuration"`
}

type createCommentResp struct {
	PostID          string `json:"postId"`
	CommentID       string `json:"commentId"`
	RootCommentID   string `json:"rootCommentId,omitempty"`
	ParentCommentID string `json:"parentCommentId,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type topLevelCommentPageResp struct {
	Items                 []commentItemResp `json:"items"`
	Page                  int               `json:"page"`
	Size                  int               `json:"size"`
	TotalComments         int64             `json:"totalComments"`
	TotalTopLevelComments int64             `json:"totalTopLevelComments"`
	Pages                 int               `json:"pages"`
}

type commentItemResp struct {
	PostID          string            `json:"postId"`
	CommentID       string            `json:"commentId"`
	RootCommentID   string            `json:"rootCommentId,omitempty"`
	ParentCommentID string            `json:"parentCommentId,omitempty"`
	Author          authorSummaryResp `json:"author"`
	Content         string            `json:"content,omitempty"`
	ImageFileIDs    []string          `json:"imageFileIds,omitempty"`
	VoiceFileID     string            `json:"voiceFileId,omitempty"`
	VoiceDuration   int               `json:"voiceDuration,omitempty"`
	Status          string            `json:"status"`
	Stats           commentStatsResp  `json:"stats"`
	Viewer          *viewerStateResp  `json:"viewer,omitempty"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
}

type authorSummaryResp struct {
	PublicID     string `json:"publicId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	AvatarFileID string `json:"avatarFileId,omitempty"`
	AvatarURL    string `json:"avatarUrl,omitempty"`
	Unavailable  bool   `json:"unavailable,omitempty"`
}

type commentStatsResp struct {
	LikeCount  int64 `json:"likeCount"`
	ReplyCount int64 `json:"replyCount"`
}

type viewerStateResp struct {
	Liked bool `json:"liked"`
}

func topLevelCommentPageResponse(page application.TopLevelCommentPage) topLevelCommentPageResp {
	items := make([]commentItemResp, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, commentItemResponse(item))
	}
	return topLevelCommentPageResp{
		Items:                 items,
		Page:                  page.Page,
		Size:                  page.Size,
		TotalComments:         page.TotalComments,
		TotalTopLevelComments: page.TotalTopLevelComments,
		Pages:                 page.Pages,
	}
}

func commentItemResponse(item application.CommentItem) commentItemResp {
	var viewer *viewerStateResp
	if item.Viewer != nil {
		viewer = &viewerStateResp{Liked: item.Viewer.Liked}
	}
	return commentItemResp{
		PostID:          string(item.PostID),
		CommentID:       string(item.CommentID),
		RootCommentID:   string(item.RootCommentID),
		ParentCommentID: string(item.ParentCommentID),
		Author: authorSummaryResp{
			PublicID:     item.Author.PublicID,
			DisplayName:  item.Author.DisplayName,
			AvatarFileID: item.Author.AvatarFileID,
			AvatarURL:    item.Author.AvatarURL,
			Unavailable:  item.Author.Unavailable,
		},
		Content:       item.Content,
		ImageFileIDs:  item.ImageFileIDs,
		VoiceFileID:   item.VoiceFileID,
		VoiceDuration: item.VoiceDuration,
		Status:        string(item.Status),
		Stats:         commentStatsResp{LikeCount: item.Stats.LikeCount, ReplyCount: item.Stats.ReplyCount},
		Viewer:        viewer,
		CreatedAt:     formatTime(item.CreatedAt),
		UpdatedAt:     formatTime(item.UpdatedAt),
	}
}

func trustedUserIDFromRequest(r *http.Request) (application.UserID, bool) {
	raw := strings.TrimSpace(r.Header.Get(userIDHeaderName))
	if raw == "" {
		return 0, false
	}
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}
	return application.UserID(userID), true
}

func postIDFromPath(w http.ResponseWriter, r *http.Request) (application.PostID, bool) {
	postID := strings.TrimSpace(r.PathValue("postId"))
	if postID == "" {
		writeValidationError(w)
		return "", false
	}
	return application.PostID(postID), true
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(out); err != nil {
		writeValidationError(w)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeValidationError(w)
		return false
	}
	return true
}

func decodeListCommentsPageQuery(w http.ResponseWriter, r *http.Request) (int, int, application.CommentSort, bool) {
	values := r.URL.Query()
	page, ok := decodePositiveIntQuery(w, values.Get("page"), 0, 1000000)
	if !ok {
		return 0, 0, "", false
	}
	size, ok := decodePositiveIntQuery(w, values.Get("size"), 0, 100)
	if !ok {
		return 0, 0, "", false
	}
	sort := application.CommentSort(strings.TrimSpace(values.Get("sort")))
	if sort != "" {
		switch sort {
		case application.CommentSortRecommended, application.CommentSortHot, application.CommentSortTime:
		default:
			writeValidationError(w)
			return 0, 0, "", false
		}
	}
	return page, size, sort, true
}

func decodePositiveIntQuery(w http.ResponseWriter, raw string, defaultValue int, max int) (int, bool) {
	if strings.TrimSpace(raw) == "" {
		return defaultValue, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 || value > max {
		writeValidationError(w)
		return 0, false
	}
	return value, true
}

func writeValidationError(w http.ResponseWriter) {
	sharedhttp.WriteErrorCode(w, http.StatusBadRequest, 1001, "Invalid request")
}

func writeMappedError(w http.ResponseWriter, err error) {
	status, code, message := errorMapping(err)
	sharedhttp.WriteErrorCode(w, status, code, message)
}

func errorMapping(err error) (int, int, string) {
	switch {
	case errors.Is(err, errLoginRequired):
		return http.StatusUnauthorized, 2006, "Authentication required"
	case errors.Is(err, application.ErrInvalidRequest), errors.Is(err, application.ErrCommentIDInvalid):
		return http.StatusBadRequest, 1001, "Invalid request"
	case errors.Is(err, application.ErrDependencyUnavailable):
		return http.StatusServiceUnavailable, 1004, "Service unavailable"
	case errors.Is(err, application.ErrPostNotFound):
		return http.StatusNotFound, 4001, "Post not found"
	case errors.Is(err, application.ErrInteractionBlocked):
		return http.StatusForbidden, 2008, "Forbidden"
	case errors.Is(err, application.ErrCommentContentRequired):
		return http.StatusBadRequest, 5003, "Comment content is required"
	case errors.Is(err, application.ErrCommentContentTooLong):
		return http.StatusBadRequest, 5004, "Comment content is too long"
	case errors.Is(err, application.ErrRootCommentNotFound):
		return http.StatusNotFound, 5005, "Root comment not found"
	case errors.Is(err, application.ErrParentCommentNotFound):
		return http.StatusNotFound, 5006, "Parent comment not found"
	default:
		return http.StatusInternalServerError, 1000, "Internal server error"
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
