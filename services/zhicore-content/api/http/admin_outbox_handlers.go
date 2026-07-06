package httpapi

import (
	"strings"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listAdminOutboxEvents(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := requireAdminActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	page, err := optionalPositiveIntQuery(c, "page")
	if err != nil {
		writeValidationError(w)
		return
	}
	size, err := optionalPositiveIntQuery(c, "size")
	if err != nil {
		writeValidationError(w)
		return
	}

	result, err := h.service.ListAdminOutboxEvents(r.Context(), application.ListAdminOutboxEventsQuery{
		Actor:     actor,
		Status:    c.Query("status"),
		EventType: c.Query("eventType"),
		Page:      page,
		Size:      size,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAdminOutbox)
		return
	}

	items := make([]adminOutboxEventResp, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, adminOutboxEventResp{
			EventID:          item.EventID,
			EventType:        item.EventType,
			AggregateType:    item.AggregateType,
			AggregateID:      item.AggregateID,
			AggregateVersion: item.AggregateVersion,
			Status:           item.Status,
			RetryCount:       item.RetryCount,
			LastError:        item.LastError,
			OccurredAt:       formatTime(item.OccurredAt),
			CreatedAt:        formatTime(item.CreatedAt),
			UpdatedAt:        formatTime(item.UpdatedAt),
		})
	}
	sharedhttp.WriteSuccess(w, adminOutboxListResp{
		Items: items,
		Page:  result.Page,
		Size:  result.Size,
		Total: result.Total,
	})
}

func (h *Handler) retryAdminOutboxEvent(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := requireAdminActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	eventID := strings.TrimSpace(c.Param("eventId"))
	if eventID == "" {
		writeValidationError(w)
		return
	}

	var req adminOutboxRetryReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		writeValidationError(w)
		return
	}

	result, err := h.service.RetryAdminOutboxEvent(r.Context(), application.RetryAdminOutboxEventCommand{
		Actor:   actor,
		EventID: eventID,
		Reason:  req.Reason,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAdminOutbox)
		return
	}
	sharedhttp.WriteSuccess(w, adminOutboxRetryResp{
		EventID:    result.EventID,
		Status:     result.Status,
		RetryCount: result.RetryCount,
		RetriedAt:  formatTime(result.RetriedAt),
	})
}
