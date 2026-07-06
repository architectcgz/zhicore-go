package application

import (
	"context"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

func (s *Service) BatchGetUserSimple(ctx context.Context, userIDs []UserID) (BatchUserSimpleResult, error) {
	domainIDs := make([]domain.UserID, 0, len(userIDs))
	for _, userID := range userIDs {
		domainIDs = append(domainIDs, domainUserID(userID))
	}
	profiles, err := s.queries.BatchGetByUserIDs(ctx, domainIDs)
	if err != nil {
		return BatchUserSimpleResult{}, err
	}
	byID := make(map[domain.UserID]domain.Profile, len(profiles))
	for _, profile := range profiles {
		byID[profile.UserID] = profile
	}
	result := BatchUserSimpleResult{Items: make([]UserSimple, 0, len(profiles))}
	seenMissing := map[UserID]bool{}
	for _, userID := range userIDs {
		profile, ok := byID[domainUserID(userID)]
		if !ok {
			// provider 响应按调用方请求顺序返回已命中的用户，同时 missing 列表去重，
			// 方便下游用原请求集合做幂等补偿或降级展示。
			if !seenMissing[userID] {
				result.MissingUserIDs = append(result.MissingUserIDs, userID)
				seenMissing[userID] = true
			}
			continue
		}
		result.Items = append(result.Items, userSimpleFromDomain(profile))
	}
	return result, nil
}

func (s *Service) BatchGetUserAvailability(ctx context.Context, userIDs []UserID) ([]UserAvailability, error) {
	simple, err := s.BatchGetUserSimple(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[UserID]UserSimple, len(simple.Items))
	for _, item := range simple.Items {
		byID[item.UserID] = item
	}
	result := make([]UserAvailability, 0, len(userIDs))
	for _, userID := range userIDs {
		item, ok := byID[userID]
		if !ok {
			// 缺失用户视为不可用于评论互动，但保留 userId 让调用方能定位被拒绝对象。
			result = append(result, UserAvailability{UserID: userID})
			continue
		}
		result = append(result, UserAvailability{
			UserID:    userID,
			Available: item.Status == UserStatusActive,
			Status:    item.Status,
		})
	}
	return result, nil
}

func (s *Service) GetMyProfile(ctx context.Context, userID UserID) (Profile, error) {
	profile, err := s.queries.GetByUserID(ctx, domainUserID(userID))
	if err != nil {
		return Profile{}, fmt.Errorf("get my profile: %w", err)
	}
	// DELETED profiles are hidden from owner-facing HTTP just like public profile reads;
	// historical rendering should use dedicated summary APIs, not this editable profile view.
	if profile.Status == domain.UserStatusDeleted {
		return Profile{}, domain.ErrProfileNotFound
	}
	return profileFromDomain(profile), nil
}

func (s *Service) GetUserProfileByPublicID(ctx context.Context, publicID PublicID) (Profile, error) {
	profile, err := s.queries.GetByPublicID(ctx, domainPublicID(publicID))
	if err != nil {
		return Profile{}, fmt.Errorf("get user profile by public id: %w", err)
	}
	if profile.Status == domain.UserStatusDeleted {
		return Profile{}, domain.ErrProfileNotFound
	}
	return profileFromDomain(profile), nil
}
