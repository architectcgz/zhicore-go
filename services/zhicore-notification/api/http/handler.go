package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
	"github.com/gin-gonic/gin"
)

const userIDHeaderName = "X-User-Id"

type Service interface {
	MarkNotificationRead(ctx context.Context, cmd application.MarkNotificationReadCommand) (application.MarkNotificationReadResult, error)
	MarkAllNotificationsRead(ctx context.Context, cmd application.MarkAllNotificationsReadCommand) (application.MarkAllNotificationsReadResult, error)
	GetUnreadCount(ctx context.Context, query application.GetUnreadCountQuery) (application.UnreadCountResult, error)
	GetUnreadBreakdown(ctx context.Context, query application.GetUnreadBreakdownQuery) (application.UnreadBreakdownResult, error)
	ListAggregatedNotifications(ctx context.Context, query application.ListNotificationsQuery) (application.NotificationPage, error)
}

type Handler struct {
	service Service
	router  *gin.Engine
}

func NewHandler(service Service) *gin.Engine {
	h := &Handler{service: service, router: gin.New()}
	h.routes()
	return h.router
}

func (h *Handler) routes() {
	h.router.GET("/api/v1/notifications", h.listNotifications)
	h.router.POST("/api/v1/notifications/:notificationId/read", h.markNotificationRead)
	h.router.POST("/api/v1/notifications/read-all", h.markAllNotificationsRead)
	h.router.POST("/api/v1/notifications/mark-all-read", h.markAllNotificationsRead)
	h.router.GET("/api/v1/notifications/unread-count", h.getUnreadCount)
	h.router.GET("/api/v1/notifications/unread/count", h.getUnreadCount)
	h.router.GET("/api/v1/notifications/unread/breakdown", h.getUnreadBreakdown)
}

func (h *Handler) markNotificationRead(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	notificationID := strings.TrimSpace(c.Param("notificationId"))
	if notificationID == "" {
		writeValidationError(w)
		return
	}
	result, err := h.service.MarkNotificationRead(r.Context(), application.MarkNotificationReadCommand{Actor: actor, NotificationID: notificationID})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, markNotificationReadResp{
		NotificationID: result.NotificationID,
		Read:           result.Read,
		Changed:        result.Changed,
		ReadAt:         sharedhttp.FormatRFC3339UTC(result.ReadAt),
	})
}

func (h *Handler) markAllNotificationsRead(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	result, err := h.service.MarkAllNotificationsRead(r.Context(), application.MarkAllNotificationsReadCommand{Actor: actor})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, markAllNotificationsReadResp{
		ReadAll:       result.ReadAll,
		ReadAt:        sharedhttp.FormatRFC3339UTC(result.ReadAt),
		AffectedCount: result.AffectedCount,
	})
}

func (h *Handler) getUnreadCount(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	result, err := h.service.GetUnreadCount(r.Context(), application.GetUnreadCountQuery{Actor: actor})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, unreadCountResp{UnreadCount: result.UnreadCount})
}

func (h *Handler) getUnreadBreakdown(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	result, err := h.service.GetUnreadBreakdown(r.Context(), application.GetUnreadBreakdownQuery{Actor: actor})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, unreadBreakdownResp{
		Total:       result.Total,
		Interaction: result.Interaction,
		Content:     result.Content,
		Social:      result.Social,
		System:      result.System,
		Security:    result.Security,
	})
}

func (h *Handler) listNotifications(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	size, err := sharedhttp.ParsePositiveInt(r.URL.Query().Get("size"), 20, 50)
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.ListAggregatedNotifications(r.Context(), application.ListNotificationsQuery{
		Actor:      actor,
		Cursor:     strings.TrimSpace(r.URL.Query().Get("cursor")),
		Size:       size,
		Category:   strings.TrimSpace(r.URL.Query().Get("category")),
		UnreadOnly: strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("unreadOnly")), "true"),
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, notificationPageResponse(result))
}

func notificationPageResponse(page application.NotificationPage) notificationPageResp {
	items := make([]aggregatedNotificationResp, 0, len(page.Items))
	for _, item := range page.Items {
		content := json.RawMessage(item.AggregatedContent)
		if len(content) == 0 {
			content = json.RawMessage(`{}`)
		}
		items = append(items, aggregatedNotificationResp{
			Type:              item.Type,
			Category:          item.Category,
			TargetType:        item.TargetType,
			TargetID:          item.TargetID,
			TotalCount:        item.TotalCount,
			UnreadCount:       item.UnreadCount,
			LatestTime:        sharedhttp.FormatRFC3339UTC(item.LatestTime),
			LatestContent:     item.LatestContent,
			ActorIDs:          item.ActorIDs,
			AggregatedContent: content,
		})
	}
	return notificationPageResp{Items: items, NextCursor: page.NextCursor, HasMore: page.HasMore}
}

func actorFromRequest(r *http.Request) (application.Actor, bool) {
	raw := strings.TrimSpace(r.Header.Get(userIDHeaderName))
	if raw == "" {
		return application.Actor{}, false
	}
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return application.Actor{}, false
	}
	return application.Actor{UserID: userID}, true
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
	case errors.Is(err, application.ErrLoginRequired):
		return http.StatusUnauthorized, 2006, "Authentication required"
	case errors.Is(err, application.ErrInvalidRequest):
		return http.StatusBadRequest, 1001, "Invalid request"
	case errors.Is(err, application.ErrNotificationNotFound):
		return http.StatusNotFound, 1005, "Notification not found"
	case errors.Is(err, application.ErrDependencyUnavailable):
		return http.StatusServiceUnavailable, 1004, "Service unavailable"
	default:
		return http.StatusInternalServerError, 1000, "Internal server error"
	}
}
