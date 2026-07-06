package httpapi

import (
	"context"
	"strings"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
)

func (h *Handler) profileResponse(ctx context.Context, profile application.Profile) userProfileResp {
	resp := userProfileResp{
		PublicID:               string(profile.PublicID),
		Nickname:               profile.Nickname,
		AvatarFileID:           profile.AvatarFileID,
		Bio:                    profile.Bio,
		StrangerMessageAllowed: profile.StrangerMessageAllowed,
		ProfileVersion:         profile.ProfileVersion,
	}
	if h.resolver == nil || strings.TrimSpace(profile.AvatarFileID) == "" {
		return resp
	}

	// avatarUrl 是读取时派生的展示字段；File 解析失败时仍返回 profile 事实，
	// 只省略 avatarUrl，避免把非事实字段的故障升级成整个资料查询失败。
	url, err := h.resolver.ResolveAvatarURL(ctx, profile.AvatarFileID)
	if err == nil && strings.TrimSpace(url) != "" {
		resp.AvatarURL = url
	}
	return resp
}

func (h *Handler) relationshipPageResponse(ctx context.Context, page application.RelationshipProfilePage) relationshipPageResp {
	items := make([]userProfileResp, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, h.profileResponse(ctx, item))
	}
	return relationshipPageResp{
		Items:      items,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}
}

func (h *Handler) simpleUserResponse(ctx context.Context, item application.UserSimple) usercontract.SimpleUser {
	resp := usercontract.SimpleUser{
		UserID:         int64(item.UserID),
		PublicID:       string(item.PublicID),
		Nickname:       item.Nickname,
		AvatarFileID:   item.AvatarFileID,
		ProfileVersion: item.ProfileVersion,
		Status:         string(item.Status),
	}
	if h.resolver == nil || strings.TrimSpace(item.AvatarFileID) == "" {
		return resp
	}
	url, err := h.resolver.ResolveAvatarURL(ctx, item.AvatarFileID)
	if err == nil && strings.TrimSpace(url) != "" {
		resp.AvatarURL = url
	}
	return resp
}

func applicationUserIDs(ids []int64) []application.UserID {
	result := make([]application.UserID, 0, len(ids))
	for _, id := range ids {
		result = append(result, application.UserID(id))
	}
	return result
}
