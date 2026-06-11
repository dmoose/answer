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

package network_directory

import (
	"context"

	"github.com/apache/answer/internal/base/pager"
	"github.com/apache/answer/internal/repo/member_directory"
	"github.com/apache/answer/internal/repo/profile_tag"
	"github.com/apache/answer/internal/schema"
)

// MemberDirectoryService assembles the directory page: faceted query against
// the user × network_profile join, plus a per-row tag fetch so each card
// renders with its skill / interest chips.
type MemberDirectoryService struct {
	directoryRepo *member_directory.MemberDirectoryRepo
	tagRepo       *profile_tag.ProfileTagRepo
}

func NewMemberDirectoryService(
	directoryRepo *member_directory.MemberDirectoryRepo,
	tagRepo *profile_tag.ProfileTagRepo,
) *MemberDirectoryService {
	return &MemberDirectoryService{
		directoryRepo: directoryRepo,
		tagRepo:       tagRepo,
	}
}

// Search returns a page of member cards with tag chips populated.
func (s *MemberDirectoryService) Search(ctx context.Context, req *schema.DirectorySearchReq) (*pager.PageModel, error) {
	q := &member_directory.DirectoryQuery{
		Q:                   req.Q,
		TagIDs:              req.TagIDs,
		OpenToMentoring:     req.OpenToMentoring,
		OpenToCollaboration: req.OpenToCollaboration,
		OpenToHire:          req.OpenToHire,
		Page:                req.Page,
		PageSize:            req.PageSize,
		Sort:                req.Sort,
	}
	rows, total, err := s.directoryRepo.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	cards := make([]*schema.DirectoryMember, 0, len(rows))
	userIDs := make([]string, 0, len(rows))
	for _, r := range rows {
		cards = append(cards, &schema.DirectoryMember{
			UserID:              r.UserID,
			Username:            r.Username,
			DisplayName:         r.DisplayName,
			Avatar:              r.Avatar,
			Reputation:          r.Rank,
			Headline:            r.Headline,
			Pronouns:            r.Pronouns,
			Timezone:            r.Timezone,
			OpenToMentoring:     r.OpenToMentoring,
			OpenToCollaboration: r.OpenToCollaboration,
			OpenToHire:          r.OpenToHire,
			Tags:                []*schema.ProfileTagInfo{},
		})
		userIDs = append(userIDs, r.UserID)
	}

	// Batch-fetch tag IDs per user, then resolve to ProfileTagInfo in one go.
	tagIDsByUser, err := s.tagRepo.GetTagsForUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	allTagIDs := make([]string, 0)
	seen := make(map[string]struct{})
	for _, ids := range tagIDsByUser {
		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			allTagIDs = append(allTagIDs, id)
		}
	}
	tagByID := map[string]*schema.ProfileTagInfo{}
	if len(allTagIDs) > 0 {
		tags, err := s.tagRepo.GetByIDs(ctx, allTagIDs)
		if err != nil {
			return nil, err
		}
		for _, t := range tags {
			tagByID[t.ID] = TagInfo(t)
		}
	}
	for _, c := range cards {
		for _, tid := range tagIDsByUser[c.UserID] {
			if info, ok := tagByID[tid]; ok {
				c.Tags = append(c.Tags, info)
			}
		}
	}

	return pager.NewPageModel(total, cards), nil
}
