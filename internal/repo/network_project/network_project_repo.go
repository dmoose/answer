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

package network_project

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/segmentfault/pacman/errors"
)

type NetworkProjectRepo struct {
	data *data.Data
}

func NewNetworkProjectRepo(data *data.Data) *NetworkProjectRepo {
	return &NetworkProjectRepo{data: data}
}

func (r *NetworkProjectRepo) Get(ctx context.Context, id string) (*entity.NetworkProject, bool, error) {
	p := &entity.NetworkProject{}
	exist, err := r.data.DB.Context(ctx).ID(id).Get(p)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return p, exist, nil
}

func (r *NetworkProjectRepo) ListByUser(ctx context.Context, userID string) ([]*entity.NetworkProject, error) {
	var projects []*entity.NetworkProject
	err := r.data.DB.Context(ctx).
		Where("user_id = ?", userID).
		OrderBy("updated_at DESC").
		Find(&projects)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return projects, nil
}

// ListRecent returns the most-recently-updated active projects across all
// members, for the directory home "what people are working on" feed.
func (r *NetworkProjectRepo) ListRecent(ctx context.Context, limit int) ([]*entity.NetworkProject, error) {
	var projects []*entity.NetworkProject
	err := r.data.DB.Context(ctx).
		Where("status = ?", entity.NetworkProjectStatusActive).
		OrderBy("updated_at DESC").
		Limit(limit).
		Find(&projects)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return projects, nil
}

func (r *NetworkProjectRepo) Insert(ctx context.Context, p *entity.NetworkProject) error {
	_, err := r.data.DB.Context(ctx).Insert(p)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *NetworkProjectRepo) Update(ctx context.Context, p *entity.NetworkProject) error {
	_, err := r.data.DB.Context(ctx).ID(p.ID).
		Cols("title", "description", "repo_url", "status", "seeking_help").
		Update(p)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *NetworkProjectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.data.DB.Context(ctx).ID(id).Delete(&entity.NetworkProject{})
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}
