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

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/service/site"
	"github.com/segmentfault/pacman/errors"
)

type siteRepo struct {
	data *data.Data
}

func NewSiteRepo(data *data.Data) site.SiteRepo {
	return &siteRepo{data: data}
}

func (r *siteRepo) AddSite(ctx context.Context, s *entity.Site) error {
	_, err := r.data.DB.Context(ctx).Insert(s)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *siteRepo) UpdateSite(ctx context.Context, s *entity.Site) error {
	_, err := r.data.DB.Context(ctx).ID(s.ID).
		Cols("name", "slug", "description", "status", "icon_url", "base_url").Update(s)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *siteRepo) GetSite(ctx context.Context, id string) (*entity.Site, bool, error) {
	s := &entity.Site{}
	exist, err := r.data.DB.Context(ctx).ID(id).Get(s)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return s, exist, nil
}

func (r *siteRepo) GetSiteBySlug(ctx context.Context, slug string) (*entity.Site, bool, error) {
	s := &entity.Site{Slug: slug}
	exist, err := r.data.DB.Context(ctx).Get(s)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return s, exist, nil
}

func (r *siteRepo) GetAllSites(ctx context.Context) ([]*entity.Site, error) {
	var sites []*entity.Site
	err := r.data.DB.Context(ctx).Where("status = ?", entity.SiteStatusActive).Find(&sites)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return sites, nil
}
