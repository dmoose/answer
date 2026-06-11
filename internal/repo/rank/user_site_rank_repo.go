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

package rank

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/multisite"
	rankservice "github.com/apache/answer/internal/service/rank"
	"xorm.io/xorm"
)

type UserSiteRankRepo struct {
	data *data.Data
}

func NewUserSiteRankRepo(data *data.Data) rankservice.SiteRankRepo {
	return &UserSiteRankRepo{data: data}
}

func (r *UserSiteRankRepo) GetUserSiteRank(ctx context.Context, userID, siteID string) (int, error) {
	usr := &entity.UserSiteRank{}
	exist, err := r.data.DB.Context(ctx).
		Where("user_id = ? AND site_id = ?", userID, siteID).Get(usr)
	if err != nil {
		return 1, err
	}
	if !exist {
		return 1, nil
	}
	return usr.Rank, nil
}

func (r *UserSiteRankRepo) ChangeSiteRank(ctx context.Context, session *xorm.Session,
	userID string, deltaRank int) error {
	siteID := multisite.SiteIDFromContext(ctx)
	if siteID == "" || deltaRank == 0 {
		return nil
	}

	existing := &entity.UserSiteRank{}
	exist, err := session.Where("user_id = ? AND site_id = ?", userID, siteID).Get(existing)
	if err != nil {
		return err
	}

	if exist {
		newRank := max(existing.Rank+deltaRank, 1)
		_, err = session.Where("user_id = ? AND site_id = ?", userID, siteID).
			Cols("rank").Update(&entity.UserSiteRank{Rank: newRank})
		return err
	}

	rank := max(1+deltaRank, 1)
	_, err = session.Insert(&entity.UserSiteRank{
		UserID: userID,
		SiteID: siteID,
		Rank:   rank,
	})
	return err
}

func (r *UserSiteRankRepo) GetUserAllSiteRanks(ctx context.Context, userID string) ([]entity.UserSiteRank, error) {
	var ranks []entity.UserSiteRank
	err := r.data.DB.Context(ctx).Where("user_id = ?", userID).Find(&ranks)
	return ranks, err
}
