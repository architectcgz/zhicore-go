package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listPublishedPosts(c *gin.Context) {
	w, r := c.Writer, c.Request
	limit, err := optionalPositiveIntQuery(c, "limit")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.ListPublishedPosts(r.Context(), application.ListPublishedPostsQuery{
		AuthorID:         c.Query("authorId"),
		Tag:              c.Query("tag"),
		CategoryID:       c.Query("categoryId"),
		Cursor:           c.Query("cursor"),
		Limit:            limit,
		Sort:             c.Query("sort"),
		RateLimitSubject: publicReadRateLimitSubject(r),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, cursorPageResp[postSummaryResp]{
		Items:      mapPostSummaryResponses(result.Items),
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
		Limit:      result.Limit,
	})
}

func (h *Handler) getPostDetail(c *gin.Context) {
	w, r := c.Writer, c.Request
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.GetPostDetail(r.Context(), application.GetPostDetailQuery{
		PostID:           postID,
		RateLimitSubject: publicReadRateLimitSubject(r),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	resp := postDetailResp{Post: mapPostSummaryResponse(result.Post)}
	if result.Body != nil {
		body, ok := mapPostBodyResponse(*result.Body)
		if !ok {
			writeMappedError(w, application.ErrBodySchemaUnsupported, errorOperationPublicPostQuery)
			return
		}
		resp.Body = &body
	}
	sharedhttp.WriteSuccess(w, resp)
}

func (h *Handler) batchGetPublishedPosts(c *gin.Context) {
	w, r := c.Writer, c.Request
	var req batchGetPostsReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if len(req.PostIDs) == 0 || len(req.PostIDs) > 100 {
		writeValidationError(w)
		return
	}
	result, err := h.service.BatchGetPublishedPosts(r.Context(), application.BatchGetPublishedPostsQuery{
		PostIDs: append([]string(nil), req.PostIDs...),
		// includeDeleted is intentionally ignored for anonymous public reads:
		// invisible, deleted and missing posts must collapse into missingPostIds.
		IncludeDeleted:   false,
		RateLimitSubject: publicReadRateLimitSubject(r),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, batchGetPostsResp{
		Items:          mapPostSummaryResponses(result.Items),
		MissingPostIDs: append([]string(nil), result.MissingPostIDs...),
	})
}

func (h *Handler) getPostBody(c *gin.Context) {
	w, r := c.Writer, c.Request
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}

	result, err := h.service.GetPublishedPostBody(r.Context(), application.GetPublishedPostBodyQuery{
		PostID:           postID,
		RateLimitSubject: publicReadRateLimitSubject(r),
		CallerService:    strings.TrimSpace(r.Header.Get(callerServiceHeaderName)),
		CallerOperation:  strings.TrimSpace(r.Header.Get(callerOperationHeaderName)),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationGetPostBody)
		return
	}

	blocks, ok := extractCanonicalBlocks(result.CanonicalJSON)
	if !ok {
		// Application owns body validation and repair registration; this guard
		// prevents a corrupted application result from being exposed as a
		// successful empty published body at the HTTP contract boundary.
		writeMappedError(w, application.ErrBodySchemaUnsupported, errorOperationGetPostBody)
		return
	}
	sharedhttp.WriteSuccess(w, postBodyResp{
		BodyID:        result.BodyID,
		SchemaVersion: result.SchemaVersion,
		Format:        "blocks",
		Blocks:        blocks,
		PlainText:     result.PlainText,
		ContentHash:   result.ContentHash,
		SizeBytes:     result.SizeBytes,
		CreatedAt:     formatTime(result.CreatedAt),
	})
}

func publicReadRateLimitSubject(r *http.Request) string {
	// Public reads use only Gateway-injected identity or the direct peer address
	// for rate-limit bucketing; forwarded IP and Authorization are not trusted here.
	if subject := actorRateLimitSubjectFromHeader(r.Header.Get(userIDHeaderName)); subject != "" {
		return subject
	}
	if host := remoteAddrHost(r.RemoteAddr); host != "" {
		return "ip:" + host
	}
	return "anonymous"
}

func actorRateLimitSubjectFromHeader(value string) string {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return ""
	}
	return "actor:" + strconv.FormatInt(id, 10)
}

func remoteAddrHost(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return validRemoteHost(host)
	}
	// RemoteAddr normally has host:port. If a direct test or non-standard server
	// provides only an address literal, use it; malformed host:port is ignored.
	if !strings.Contains(remoteAddr, ":") {
		return validRemoteHost(remoteAddr)
	}
	return ""
}

func validRemoteHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return host
	}
	if isASCIIDomainName(host) {
		return host
	}
	return ""
}

func isASCIIDomainName(host string) bool {
	host = strings.TrimSuffix(host, ".")
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			ch := label[i]
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			return false
		}
	}
	return true
}
