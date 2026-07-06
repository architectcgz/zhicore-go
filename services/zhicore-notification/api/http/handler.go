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
	GetNotificationPreferences(ctx context.Context, query application.GetNotificationPreferencesQuery) (application.NotificationPreferencesResult, error)
	UpdateNotificationPreferences(ctx context.Context, cmd application.UpdateNotificationPreferencesCommand) (application.NotificationPreferencesResult, error)
	GetNotificationDND(ctx context.Context, query application.GetNotificationDNDQuery) (application.NotificationDNDResult, error)
	UpdateNotificationDND(ctx context.Context, cmd application.UpdateNotificationDNDCommand) (application.NotificationDNDResult, error)
	GetAuthorSubscription(ctx context.Context, query application.GetAuthorSubscriptionQuery) (application.AuthorSubscriptionResult, error)
	UpdateAuthorSubscription(ctx context.Context, cmd application.UpdateAuthorSubscriptionCommand) (application.AuthorSubscriptionResult, error)
	ListDeliveries(ctx context.Context, query application.ListDeliveriesQuery) (application.DeliveryPage, error)
	RetryDelivery(ctx context.Context, cmd application.RetryDeliveryCommand) (application.DeliveryRetryResult, error)
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
	h.router.GET("/api/v1/notification-preferences", h.getNotificationPreferences)
	h.router.PUT("/api/v1/notification-preferences", h.updateNotificationPreferences)
	h.router.GET("/api/v1/notifications/preferences", h.getNotificationPreferences)
	h.router.PUT("/api/v1/notifications/preferences", h.updateNotificationPreferences)
	h.router.GET("/api/v1/notification-dnd", h.getNotificationDND)
	h.router.PUT("/api/v1/notification-dnd", h.updateNotificationDND)
	h.router.GET("/api/v1/notifications/dnd", h.getNotificationDND)
	h.router.PUT("/api/v1/notifications/dnd", h.updateNotificationDND)
	h.router.GET("/api/v1/author-subscriptions/:authorId", h.getAuthorSubscription)
	h.router.PUT("/api/v1/author-subscriptions/:authorId", h.updateAuthorSubscription)
	h.router.GET("/api/v1/notifications/author-subscriptions/:authorId", h.getAuthorSubscription)
	h.router.PUT("/api/v1/notifications/author-subscriptions/:authorId", h.updateAuthorSubscription)
	h.router.GET("/api/v1/notification-deliveries", h.listDeliveries)
	h.router.POST("/api/v1/notification-deliveries/:deliveryId/retry", h.retryDelivery)
	h.router.GET("/api/v1/notifications/deliveries", h.listDeliveries)
	h.router.POST("/api/v1/notifications/deliveries/:deliveryId/retry", h.retryDelivery)
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

func (h *Handler) getNotificationPreferences(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	result, err := h.service.GetNotificationPreferences(r.Context(), application.GetNotificationPreferencesQuery{Actor: actor})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, notificationPreferencesResponse(result))
}

func (h *Handler) updateNotificationPreferences(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	var req updateNotificationPreferencesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.UpdateNotificationPreferences(r.Context(), application.UpdateNotificationPreferencesCommand{
		Actor:       actor,
		Preferences: notificationPreferenceInputs(req.Preferences),
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, notificationPreferencesResponse(result))
}

func (h *Handler) getNotificationDND(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	result, err := h.service.GetNotificationDND(r.Context(), application.GetNotificationDNDQuery{Actor: actor})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, notificationDNDResponse(result))
}

func (h *Handler) updateNotificationDND(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	var req updateNotificationDNDReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.UpdateNotificationDND(r.Context(), application.UpdateNotificationDNDCommand{
		Actor:      actor,
		Enabled:    req.Enabled,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Timezone:   req.Timezone,
		Categories: req.Categories,
		Channels:   req.Channels,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, notificationDNDResponse(result))
}

func (h *Handler) getAuthorSubscription(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	authorID, ok := parsePositivePathInt(c.Param("authorId"))
	if !ok {
		writeValidationError(w)
		return
	}
	result, err := h.service.GetAuthorSubscription(r.Context(), application.GetAuthorSubscriptionQuery{Actor: actor, AuthorID: authorID})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, authorSubscriptionResponse(result))
}

func (h *Handler) updateAuthorSubscription(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	authorID, ok := parsePositivePathInt(c.Param("authorId"))
	if !ok {
		writeValidationError(w)
		return
	}
	var req updateAuthorSubscriptionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.UpdateAuthorSubscription(r.Context(), application.UpdateAuthorSubscriptionCommand{
		Actor:            actor,
		AuthorID:         authorID,
		Level:            req.Level,
		InAppEnabled:     req.InAppEnabled,
		WebsocketEnabled: req.WebsocketEnabled,
		EmailEnabled:     req.EmailEnabled,
		DigestEnabled:    req.DigestEnabled,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, authorSubscriptionResponse(result))
}

func (h *Handler) listDeliveries(c *gin.Context) {
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
	result, err := h.service.ListDeliveries(r.Context(), application.ListDeliveriesQuery{
		Actor:   actor,
		Channel: strings.TrimSpace(r.URL.Query().Get("channel")),
		Status:  strings.TrimSpace(r.URL.Query().Get("status")),
		Cursor:  strings.TrimSpace(r.URL.Query().Get("cursor")),
		Size:    size,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, deliveryPageResponse(result))
}

func (h *Handler) retryDelivery(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, application.ErrLoginRequired)
		return
	}
	deliveryID := strings.TrimSpace(c.Param("deliveryId"))
	if deliveryID == "" {
		writeValidationError(w)
		return
	}
	result, err := h.service.RetryDelivery(r.Context(), application.RetryDeliveryCommand{Actor: actor, DeliveryID: deliveryID})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, deliveryRetryResp{DeliveryID: result.DeliveryID, Status: result.Status, Retried: result.Retried})
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
	return application.Actor{UserID: userID, Roles: rolesFromRequest(r)}, true
}

func rolesFromRequest(r *http.Request) []string {
	raw := strings.TrimSpace(r.Header.Get("X-User-Roles"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		role := strings.TrimSpace(part)
		if role != "" {
			roles = append(roles, role)
		}
	}
	return roles
}

func parsePositivePathInt(raw string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	return value, err == nil && value > 0
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
