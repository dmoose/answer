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

package role

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/entity"
	roleservice "github.com/apache/answer/internal/service/role"
)

type UserSiteRoleRelRepo struct {
	data *data.Data
}

func NewUserSiteRoleRelRepo(data *data.Data) roleservice.SiteRoleRepo {
	return &UserSiteRoleRelRepo{data: data}
}

func (r *UserSiteRoleRelRepo) GetUserSiteRole(ctx context.Context, userID, siteID string) (int, bool, error) {
	rel := &entity.UserSiteRoleRel{}
	exist, err := r.data.DB.Context(ctx).
		Where("user_id = ? AND site_id = ?", userID, siteID).Get(rel)
	if err != nil {
		return 0, false, err
	}
	if !exist {
		return 0, false, nil
	}
	return rel.RoleID, true, nil
}

func (r *UserSiteRoleRelRepo) SaveUserSiteRole(ctx context.Context, userID, siteID string, roleID int) error {
	existing := &entity.UserSiteRoleRel{}
	exist, err := r.data.DB.Context(ctx).
		Where("user_id = ? AND site_id = ?", userID, siteID).Get(existing)
	if err != nil {
		return err
	}
	if exist {
		_, err = r.data.DB.Context(ctx).
			Where("user_id = ? AND site_id = ?", userID, siteID).
			Cols("role_id").Update(&entity.UserSiteRoleRel{RoleID: roleID})
		return err
	}
	_, err = r.data.DB.Context(ctx).Insert(&entity.UserSiteRoleRel{
		UserID: userID,
		SiteID: siteID,
		RoleID: roleID,
	})
	return err
}
