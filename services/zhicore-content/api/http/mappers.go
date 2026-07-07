package httpapi

import (
	"encoding/json"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

func extractCanonicalBlocks(canonicalJSON []byte) (json.RawMessage, bool) {
	if len(canonicalJSON) == 0 {
		return nil, false
	}
	var body struct {
		Blocks json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal(canonicalJSON, &body); err != nil || len(body.Blocks) == 0 {
		return nil, false
	}
	return body.Blocks, true
}

func mapPostSummaryResponses(items []application.PostSummary) []postSummaryResp {
	resp := make([]postSummaryResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, mapPostSummaryResponse(item))
	}
	return resp
}

func mapPostSummaryResponse(item application.PostSummary) postSummaryResp {
	return postSummaryResp{
		PostID:             item.PostID,
		AuthorID:           item.AuthorID,
		AuthorName:         item.AuthorName,
		AuthorAvatarFileID: item.AuthorAvatarFileID,
		Title:              item.Title,
		Summary:            item.Summary,
		CoverFileID:        item.CoverFileID,
		Status:             item.Status,
		PostVersion:        item.PostVersion,
		PublishedAt:        formatTime(item.PublishedAt),
		CreatedAt:          formatTime(item.CreatedAt),
		UpdatedAt:          formatTime(item.UpdatedAt),
		Stats: postStatsResp{
			ViewCount:     item.Stats.ViewCount,
			LikeCount:     item.Stats.LikeCount,
			FavoriteCount: item.Stats.FavoriteCount,
			CommentCount:  item.Stats.CommentCount,
		},
	}
}

func mapAuthorDraftResponse(item application.AuthorDraftResult) authorDraftResp {
	resp := authorDraftResp{
		PostID:        item.PostID,
		PostVersion:   item.PostVersion,
		Title:         item.Title,
		Summary:       item.Summary,
		CoverFileID:   item.CoverFileID,
		Status:        item.Status,
		DraftBodyID:   item.DraftBodyID,
		DraftBodyHash: item.DraftBodyHash,
		CreatedAt:     formatTime(item.CreatedAt),
		UpdatedAt:     formatTime(item.UpdatedAt),
	}
	if item.Body != nil {
		body, ok := mapPostBodyResponse(*item.Body)
		if ok {
			resp.Body = &body
		}
	}
	return resp
}

func mapPostLifecycleResponse(result application.PostLifecycleResult) postLifecycleResp {
	return postLifecycleResp{
		PostID:      result.PostID,
		PostVersion: result.PostVersion,
		Status:      result.Status,
		UpdatedAt:   formatTime(result.UpdatedAt),
	}
}

func mapDraftMutationResponse(item application.DraftMutationResult) draftMutationResp {
	return draftMutationResp{
		PostID:      item.PostID,
		PostVersion: item.PostVersion,
		Title:       item.Title,
		Summary:     item.Summary,
		CoverFileID: item.CoverFileID,
		UpdatedAt:   formatTime(item.UpdatedAt),
	}
}

func mapPostBodyResponse(body application.PostBodyResult) (postBodyResp, bool) {
	blocks, ok := extractCanonicalBlocks(body.CanonicalJSON)
	if !ok {
		return postBodyResp{}, false
	}
	return postBodyResp{
		BodyID:        body.BodyID,
		SchemaVersion: body.SchemaVersion,
		Format:        "blocks",
		Blocks:        blocks,
		PlainText:     body.PlainText,
		ContentHash:   body.ContentHash,
		SizeBytes:     body.SizeBytes,
		CreatedAt:     formatTime(body.CreatedAt),
	}, true
}

func mapTagResponses(tags []application.Tag) []tagResp {
	resp := make([]tagResp, 0, len(tags))
	for _, tag := range tags {
		resp = append(resp, mapTagResponse(tag))
	}
	return resp
}

func mapTagResponse(tag application.Tag) tagResp {
	return tagResp{TagID: tag.TagID, Name: tag.Name, Slug: tag.Slug, PostCount: tag.PostCount}
}

func mapPostTagsMutationResponse(result application.PostTagsMutationResult) postTagsMutationResp {
	return postTagsMutationResp{
		PostID:      result.PostID,
		PostVersion: result.PostVersion,
		Tags:        mapTagResponses(result.Tags),
		UpdatedAt:   formatTime(result.UpdatedAt),
	}
}

func mapEngagementMutationResponse(result application.EngagementResult) engagementMutationResp {
	return engagementMutationResp{
		PostID:    result.PostID,
		Liked:     result.Liked,
		Favorited: result.Favorited,
		Stats:     mapPostStatsResponse(result.Stats),
	}
}

func mapPostEngagementResponse(result application.PostEngagementResult) postEngagementResp {
	resp := postEngagementResp{
		PostID: result.PostID,
		Stats:  mapPostStatsResponse(result.Stats),
	}
	if result.Viewer != nil {
		resp.Viewer = &engagementViewerResp{
			Liked:     result.Viewer.Liked.Ptr(),
			Favorited: result.Viewer.Favorited.Ptr(),
			Degraded:  result.Viewer.Degraded,
		}
	}
	return resp
}

func mapBatchEngagementStatusResponse(result application.BatchEngagementStatusResult) batchEngagementStatusResp {
	items := make([]engagementStatusItemResp, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, engagementStatusItemResp{
			PostID:    item.PostID,
			Liked:     item.Liked.Ptr(),
			Favorited: item.Favorited.Ptr(),
			Degraded:  item.Degraded,
		})
	}
	return batchEngagementStatusResp{Items: items}
}

func mapPostStatsResponse(stats application.PostStats) postStatsResp {
	return postStatsResp{
		ViewCount:     stats.ViewCount,
		LikeCount:     stats.LikeCount,
		FavoriteCount: stats.FavoriteCount,
		CommentCount:  stats.CommentCount,
	}
}

func mapReaderPresenceResponse(result application.ReaderPresenceResult) readerPresenceResp {
	return readerPresenceResp{
		PostID:      result.PostID,
		OnlineCount: result.OnlineCount,
		Degraded:    result.Degraded,
		TTLSeconds:  result.TTLSeconds,
	}
}

func formatTime(value time.Time) string {
	return sharedhttp.FormatRFC3339UTC(value)
}
