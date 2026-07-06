package application

import (
	"context"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

func (s *Service) invalidateProfileCache(ctx context.Context, profile domain.Profile) {
	keys := []string{
		fmt.Sprintf("user:%d:simple", profile.UserID),
		fmt.Sprintf("user:%d:profile", profile.UserID),
		fmt.Sprintf("user:%d:availability", profile.UserID),
		fmt.Sprintf("user:public:%s:id", profile.PublicID),
	}
	if err := s.cache.Delete(ctx, keys...); err != nil {
		s.cacheFailures.RecordCacheDeleteFailure(ctx, "user.profile.invalidate", keys, err)
	}
}
