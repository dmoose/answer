package rank

import (
	"context"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/multisite"
	"github.com/segmentfault/pacman/log"
)

type SiteRankRepo interface {
	GetUserSiteRank(ctx context.Context, userID, siteID string) (int, error)
	GetUserAllSiteRanks(ctx context.Context, userID string) ([]entity.UserSiteRank, error)
}

func (rs *RankService) getUserRankForPermission(ctx context.Context, userID string, globalRank int) int {
	siteID := multisite.SiteIDFromContext(ctx)
	if siteID == "" || rs.siteRankRepo == nil {
		return globalRank
	}
	siteRank, err := rs.siteRankRepo.GetUserSiteRank(ctx, userID, siteID)
	if err != nil {
		log.Errorf("get site rank for user %s site %s: %v", userID, siteID, err)
		return globalRank
	}
	return siteRank
}
