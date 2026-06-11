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
	"encoding/json"

	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/repo/network_profile"
	"github.com/apache/answer/internal/repo/network_project"
	"github.com/apache/answer/internal/repo/profile_tag"
	"github.com/apache/answer/internal/schema"
	"github.com/segmentfault/pacman/errors"
)

// ProfileEditService owns the write side of a member's own directory entry:
// the profile fields, the tag set, and their project list. Read assembly
// lives on NetworkProfileService; admin tag curation lives on ProfileTagService.
type ProfileEditService struct {
	profileRepo *network_profile.NetworkProfileRepo
	projectRepo *network_project.NetworkProjectRepo
	tagRepo     *profile_tag.ProfileTagRepo
}

func NewProfileEditService(
	profileRepo *network_profile.NetworkProfileRepo,
	projectRepo *network_project.NetworkProjectRepo,
	tagRepo *profile_tag.ProfileTagRepo,
) *ProfileEditService {
	return &ProfileEditService{
		profileRepo: profileRepo,
		projectRepo: projectRepo,
		tagRepo:     tagRepo,
	}
}

// UpdateProfile writes a member's own profile fields. Caller has already been
// authenticated; UserID on req is the logged-in user's ID.
func (s *ProfileEditService) UpdateProfile(ctx context.Context, req *schema.NetworkProfileUpdateReq) error {
	linksJSON := "[]"
	if len(req.ExternalLinks) > 0 {
		b, err := json.Marshal(req.ExternalLinks)
		if err != nil {
			return errors.BadRequest(reason.RequestFormatError).WithError(err).WithStack()
		}
		linksJSON = string(b)
	}
	return s.profileRepo.Upsert(ctx, &entity.NetworkProfile{
		UserID:              req.UserID,
		Headline:            req.Headline,
		Pronouns:            req.Pronouns,
		Timezone:            req.Timezone,
		OpenToMentoring:     req.OpenToMentoring,
		OpenToCollaboration: req.OpenToCollaboration,
		OpenToHire:          req.OpenToHire,
		ExternalLinks:       linksJSON,
	})
}

func (s *ProfileEditService) SetTags(ctx context.Context, req *schema.NetworkSetProfileTagsReq) error {
	// Dedupe input and drop inactive/unknown tag IDs so a transient mistake
	// on the client doesn't produce a constraint error from the composite
	// UNIQUE on user_profile_tag.
	seen := make(map[string]struct{}, len(req.TagIDs))
	clean := make([]string, 0, len(req.TagIDs))
	for _, id := range req.TagIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		clean = append(clean, id)
	}
	if len(clean) > 0 {
		tags, err := s.tagRepo.GetByIDs(ctx, clean)
		if err != nil {
			return err
		}
		valid := make(map[string]struct{}, len(tags))
		for _, t := range tags {
			if t.Status == entity.ProfileTagStatusActive {
				valid[t.ID] = struct{}{}
			}
		}
		filtered := clean[:0]
		for _, id := range clean {
			if _, ok := valid[id]; ok {
				filtered = append(filtered, id)
			}
		}
		clean = filtered
	}
	return s.tagRepo.SetUserTags(ctx, req.UserID, clean)
}

func (s *ProfileEditService) CreateProject(ctx context.Context, req *schema.NetworkProjectCreateReq) (*schema.ProfileProjectInfo, error) {
	p := &entity.NetworkProject{
		UserID:      req.UserID,
		Title:       req.Title,
		Description: req.Description,
		RepoURL:     req.RepoURL,
		Status:      req.Status,
		SeekingHelp: req.SeekingHelp,
	}
	if err := s.projectRepo.Insert(ctx, p); err != nil {
		return nil, err
	}
	return ProjectInfo(p), nil
}

func (s *ProfileEditService) UpdateProject(ctx context.Context, req *schema.NetworkProjectUpdateReq) (*schema.ProfileProjectInfo, error) {
	existing, exist, err := s.projectRepo.Get(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.BadRequest(reason.ObjectNotFound)
	}
	if existing.UserID != req.UserID {
		return nil, errors.Forbidden(reason.UnauthorizedError)
	}
	existing.Title = req.Title
	existing.Description = req.Description
	existing.RepoURL = req.RepoURL
	existing.Status = req.Status
	existing.SeekingHelp = req.SeekingHelp
	if err := s.projectRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return ProjectInfo(existing), nil
}

func (s *ProfileEditService) DeleteProject(ctx context.Context, projectID, userID string) error {
	existing, exist, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if !exist {
		return errors.BadRequest(reason.ObjectNotFound)
	}
	if existing.UserID != userID {
		return errors.Forbidden(reason.UnauthorizedError)
	}
	return s.projectRepo.Delete(ctx, projectID)
}

// TagInfo converts an entity ProfileTag to the API shape. Exported so the
// NetworkProfileService (in a sibling package) can reuse it when assembling
// the extended profile read.
func TagInfo(t *entity.ProfileTag) *schema.ProfileTagInfo {
	return &schema.ProfileTagInfo{
		ID:          t.ID,
		Slug:        t.Slug,
		Name:        t.Name,
		Kind:        t.Kind,
		Description: t.Description,
	}
}

// ProjectInfo converts an entity NetworkProject to the API shape.
func ProjectInfo(p *entity.NetworkProject) *schema.ProfileProjectInfo {
	return &schema.ProfileProjectInfo{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		RepoURL:     p.RepoURL,
		Status:      p.Status,
		SeekingHelp: p.SeekingHelp,
		UpdatedAt:   p.UpdatedAt.Unix(),
	}
}
