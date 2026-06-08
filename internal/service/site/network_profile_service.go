package site

import (
	"context"
	"encoding/json"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/repo/network_profile"
	"github.com/apache/answer/internal/repo/network_project"
	"github.com/apache/answer/internal/repo/profile_tag"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/network_directory"
	"github.com/apache/answer/internal/service/rank"
	usercommon "github.com/apache/answer/internal/service/user_common"
)

type NetworkProfileSiteRank struct {
	SiteID   string `json:"site_id"`
	SiteName string `json:"site_name"`
	SiteSlug string `json:"site_slug"`
	Rank     int    `json:"rank"`
}

// NetworkProfile is the assembled cross-site identity response for the
// /network/user/profile endpoint. Combines the basic user fields, per-site
// reputation rollup, and the network directory extension (headline, projects,
// tags, external links) when present.
type NetworkProfile struct {
	UserID              string                       `json:"user_id"`
	DisplayName         string                       `json:"display_name"`
	Avatar              string                       `json:"avatar"`
	GlobalRank          int                          `json:"global_rank"`
	SiteRanks           []*NetworkProfileSiteRank    `json:"site_ranks"`
	Headline            string                       `json:"headline"`
	Pronouns            string                       `json:"pronouns"`
	Timezone            string                       `json:"timezone"`
	OpenToMentoring     bool                         `json:"open_to_mentoring"`
	OpenToCollaboration bool                         `json:"open_to_collaboration"`
	OpenToHire          bool                         `json:"open_to_hire"`
	ExternalLinks       []schema.ProfileExternalLink `json:"external_links"`
	Tags                []*schema.ProfileTagInfo     `json:"tags"`
	Projects            []*schema.ProfileProjectInfo `json:"projects"`
}

type NetworkProfileService struct {
	userCommon         *usercommon.UserCommon
	siteRepo           SiteRepo
	siteRankRepo       rank.SiteRankRepo
	networkProfileRepo *network_profile.NetworkProfileRepo
	networkProjectRepo *network_project.NetworkProjectRepo
	profileTagRepo     *profile_tag.ProfileTagRepo
}

func NewNetworkProfileService(
	userCommon *usercommon.UserCommon,
	siteRepo SiteRepo,
	siteRankRepo rank.SiteRankRepo,
	networkProfileRepo *network_profile.NetworkProfileRepo,
	networkProjectRepo *network_project.NetworkProjectRepo,
	profileTagRepo *profile_tag.ProfileTagRepo,
) *NetworkProfileService {
	return &NetworkProfileService{
		userCommon:         userCommon,
		siteRepo:           siteRepo,
		siteRankRepo:       siteRankRepo,
		networkProfileRepo: networkProfileRepo,
		networkProjectRepo: networkProjectRepo,
		profileTagRepo:     profileTagRepo,
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
		UserID:        userInfo.ID,
		DisplayName:   userInfo.DisplayName,
		Avatar:        userInfo.Avatar,
		GlobalRank:    userInfo.Rank,
		ExternalLinks: []schema.ProfileExternalLink{},
		Tags:          []*schema.ProfileTagInfo{},
		Projects:      []*schema.ProfileProjectInfo{},
	}

	if s.siteRankRepo != nil {
		ranks, rerr := s.siteRankRepo.GetUserAllSiteRanks(ctx, userID)
		if rerr == nil {
			sites, _ := s.siteRepo.GetAllSites(ctx)
			siteMap := make(map[string]*entity.Site, len(sites))
			for _, st := range sites {
				siteMap[st.ID] = st
			}
			for _, r := range ranks {
				sr := &NetworkProfileSiteRank{SiteID: r.SiteID, Rank: r.Rank}
				if st, ok := siteMap[r.SiteID]; ok {
					sr.SiteName = st.Name
					sr.SiteSlug = st.Slug
				}
				profile.SiteRanks = append(profile.SiteRanks, sr)
			}
		}
	}

	if s.networkProfileRepo != nil {
		np, _, perr := s.networkProfileRepo.Get(ctx, userID)
		if perr == nil && np != nil {
			profile.Headline = np.Headline
			profile.Pronouns = np.Pronouns
			profile.Timezone = np.Timezone
			profile.OpenToMentoring = np.OpenToMentoring
			profile.OpenToCollaboration = np.OpenToCollaboration
			profile.OpenToHire = np.OpenToHire
			if np.ExternalLinks != "" {
				_ = json.Unmarshal([]byte(np.ExternalLinks), &profile.ExternalLinks)
			}
		}
	}

	if s.profileTagRepo != nil {
		tagIDs, terr := s.profileTagRepo.GetUserTags(ctx, userID)
		if terr == nil && len(tagIDs) > 0 {
			tags, gerr := s.profileTagRepo.GetByIDs(ctx, tagIDs)
			if gerr == nil {
				for _, t := range tags {
					profile.Tags = append(profile.Tags, network_directory.TagInfo(t))
				}
			}
		}
	}

	if s.networkProjectRepo != nil {
		projects, perr := s.networkProjectRepo.ListByUser(ctx, userID)
		if perr == nil {
			for _, p := range projects {
				profile.Projects = append(profile.Projects, network_directory.ProjectInfo(p))
			}
		}
	}

	return profile, nil
}
