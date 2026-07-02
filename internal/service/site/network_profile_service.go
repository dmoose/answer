/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package site

import (
	"context"
	"encoding/json"

	"github.com/apache/answer/internal/repo/network_profile"
	"github.com/apache/answer/internal/repo/network_project"
	"github.com/apache/answer/internal/repo/profile_tag"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/network_directory"
	"github.com/apache/answer/internal/service/service_config"
	usercommon "github.com/apache/answer/internal/service/user_common"
	"github.com/segmentfault/pacman/log"
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
	networkProfileRepo *network_profile.NetworkProfileRepo
	networkProjectRepo *network_project.NetworkProjectRepo
	profileTagRepo     *profile_tag.ProfileTagRepo
	serviceConfig      *service_config.ServiceConfig
}

func NewNetworkProfileService(
	userCommon *usercommon.UserCommon,
	siteRepo SiteRepo,
	networkProfileRepo *network_profile.NetworkProfileRepo,
	networkProjectRepo *network_project.NetworkProjectRepo,
	profileTagRepo *profile_tag.ProfileTagRepo,
	serviceConfig *service_config.ServiceConfig,
) *NetworkProfileService {
	return &NetworkProfileService{
		userCommon:         userCommon,
		siteRepo:           siteRepo,
		networkProfileRepo: networkProfileRepo,
		networkProjectRepo: networkProjectRepo,
		profileTagRepo:     profileTagRepo,
		serviceConfig:      serviceConfig,
	}
}

func (s *NetworkProfileService) directoryEnabled() bool {
	return s.serviceConfig != nil && s.serviceConfig.DirectoryEnabled
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

	// SiteRanks intentionally stays empty: reputation is global (one rank
	// per person). Per-site attribution for display (Option C) can be
	// derived later from activity.site_id — SUM(rank) GROUP BY site_id.

	// Extended directory fields skipped when the feature is disabled — the
	// frontend likewise hides UI in that mode, so leaving these empty keeps
	// the response shape consistent without exposing guild-only concepts on
	// a plain Q&A deployment.
	if !s.directoryEnabled() {
		return profile, nil
	}

	if s.networkProfileRepo != nil {
		np, _, perr := s.networkProfileRepo.Get(ctx, userID)
		if perr != nil {
			log.Errorf("network profile fetch for user %s: %v", userID, perr)
		}
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
		if terr != nil {
			log.Errorf("profile tags fetch for user %s: %v", userID, terr)
		}
		if terr == nil && len(tagIDs) > 0 {
			tags, gerr := s.profileTagRepo.GetByIDs(ctx, tagIDs)
			if gerr != nil {
				log.Errorf("profile tag resolve for user %s: %v", userID, gerr)
			}
			if gerr == nil {
				for _, t := range tags {
					profile.Tags = append(profile.Tags, network_directory.TagInfo(t))
				}
			}
		}
	}

	if s.networkProjectRepo != nil {
		projects, perr := s.networkProjectRepo.ListByUser(ctx, userID)
		if perr != nil {
			log.Errorf("network projects fetch for user %s: %v", userID, perr)
		}
		if perr == nil {
			for _, p := range projects {
				profile.Projects = append(profile.Projects, network_directory.ProjectInfo(p))
			}
		}
	}

	return profile, nil
}
