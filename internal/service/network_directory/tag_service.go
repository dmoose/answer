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

	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/repo/profile_tag"
	"github.com/apache/answer/internal/schema"
	"github.com/segmentfault/pacman/errors"
)

// ProfileTagService handles tag-catalog operations: public listing and admin
// curation. Per-user tag assignment lives on ProfileEditService.SetTags.
type ProfileTagService struct {
	tagRepo *profile_tag.ProfileTagRepo
}

func NewProfileTagService(tagRepo *profile_tag.ProfileTagRepo) *ProfileTagService {
	return &ProfileTagService{tagRepo: tagRepo}
}

// AdminListAll returns every tag in the catalog including inactive ones, for
// the admin curation page.
func (s *ProfileTagService) AdminListAll(ctx context.Context) ([]*schema.AdminProfileTagInfo, error) {
	tags, err := s.tagRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*schema.AdminProfileTagInfo, 0, len(tags))
	for _, t := range tags {
		out = append(out, &schema.AdminProfileTagInfo{
			ID:          t.ID,
			Slug:        t.Slug,
			Name:        t.Name,
			Kind:        t.Kind,
			Description: t.Description,
			Status:      t.Status,
		})
	}
	return out, nil
}

// ListActive returns the catalog filtered by kind (0 = any). Used by the
// member tag picker and the directory facet list.
func (s *ProfileTagService) ListActive(ctx context.Context, kind int) ([]*schema.ProfileTagInfo, error) {
	tags, err := s.tagRepo.ListActive(ctx, kind)
	if err != nil {
		return nil, err
	}
	out := make([]*schema.ProfileTagInfo, 0, len(tags))
	for _, t := range tags {
		out = append(out, TagInfo(t))
	}
	return out, nil
}

// AdminUpsert creates a new tag or updates an existing one by ID. Slug
// uniqueness is enforced by the table's UNIQUE index; we additionally
// pre-check so the client gets a clean BadRequest instead of a 500.
func (s *ProfileTagService) AdminUpsert(ctx context.Context, req *schema.AdminProfileTagUpsertReq) (*schema.ProfileTagInfo, error) {
	if req.ID != "" {
		t := &entity.ProfileTag{
			ID:          req.ID,
			Slug:        req.Slug,
			Name:        req.Name,
			Kind:        req.Kind,
			Description: req.Description,
			Status:      req.Status,
		}
		if err := s.tagRepo.Update(ctx, t); err != nil {
			return nil, err
		}
		return TagInfo(t), nil
	}

	if _, exists, err := s.tagRepo.GetBySlug(ctx, req.Slug); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest(reason.UnknownError).WithMsg("tag slug already in use")
	}

	t := &entity.ProfileTag{
		Slug:        req.Slug,
		Name:        req.Name,
		Kind:        req.Kind,
		Description: req.Description,
		Status:      req.Status,
	}
	if err := s.tagRepo.Insert(ctx, t); err != nil {
		return nil, err
	}
	return TagInfo(t), nil
}
