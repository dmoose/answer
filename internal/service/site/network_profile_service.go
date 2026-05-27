package site

import (
	"context"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/service/rank"
	usercommon "github.com/apache/answer/internal/service/user_common"
)

type NetworkProfileSiteRank struct {
	SiteID   string `json:"site_id"`
	SiteName string `json:"site_name"`
	SiteSlug string `json:"site_slug"`
	Rank     int    `json:"rank"`
}

type NetworkProfile struct {
	UserID      string                    `json:"user_id"`
	DisplayName string                    `json:"display_name"`
	Avatar      string                    `json:"avatar"`
	GlobalRank  int                       `json:"global_rank"`
	SiteRanks   []*NetworkProfileSiteRank `json:"site_ranks"`
}

type NetworkProfileService struct {
	userCommon   *usercommon.UserCommon
	siteRepo     SiteRepo
	siteRankRepo rank.SiteRankRepo
}

func NewNetworkProfileService(
	userCommon *usercommon.UserCommon,
	siteRepo SiteRepo,
	siteRankRepo rank.SiteRankRepo,
) *NetworkProfileService {
	return &NetworkProfileService{
		userCommon:   userCommon,
		siteRepo:     siteRepo,
		siteRankRepo: siteRankRepo,
	}
}

func (s *NetworkProfileService) GetNetworkProfile(ctx context.Context, userID string) (*NetworkProfile, error) {
	userInfo, exist, err := s.userCommon.GetUserBasicInfoByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}

	profile := &NetworkProfile{
		UserID:      userInfo.ID,
		DisplayName: userInfo.DisplayName,
		Avatar:      userInfo.Avatar,
		GlobalRank:  userInfo.Rank,
	}

	if s.siteRankRepo == nil {
		return profile, nil
	}
	ranks, err := s.siteRankRepo.GetUserAllSiteRanks(ctx, userID)
	if err != nil {
		return profile, nil
	}

	sites, _ := s.siteRepo.GetAllSites(ctx)
	siteMap := make(map[string]*entity.Site, len(sites))
	for _, st := range sites {
		siteMap[st.ID] = st
	}

	for _, r := range ranks {
		sr := &NetworkProfileSiteRank{
			SiteID: r.SiteID,
			Rank:   r.Rank,
		}
		if st, ok := siteMap[r.SiteID]; ok {
			sr.SiteName = st.Name
			sr.SiteSlug = st.Slug
		}
		profile.SiteRanks = append(profile.SiteRanks, sr)
	}
	return profile, nil
}
