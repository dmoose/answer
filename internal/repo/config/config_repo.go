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

package config

import (
	"context"
	"fmt"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/service/config"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
)

type configRepo struct {
	data *data.Data
}

func NewConfigRepo(data *data.Data) config.ConfigRepo {
	return &configRepo{data: data}
}

func (cr configRepo) GetConfigByID(ctx context.Context, id int) (c *entity.Config, err error) {
	cacheKey := fmt.Sprintf("%s%d", constant.ConfigID2KEYCacheKeyPrefix, id)
	cacheData, exist, err := cr.data.Cache.GetString(ctx, cacheKey)
	if err == nil && exist && len(cacheData) > 0 {
		c = &entity.Config{}
		c.BuildByJSON([]byte(cacheData))
		if c.ID > 0 {
			return c, nil
		}
	}

	// Functional config is org-global by design: one row set (site_id=''),
	// one cache key. Per-site presentation lives in site_info, never here.
	// (A per-site override would also fork the auto-increment ID that
	// activity rows store as activity_type, orphaning history.)
	c = &entity.Config{}
	exist, err = cr.data.DB.Context(ctx).Where("id = ? AND site_id = ''", id).Get(c)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if !exist {
		return nil, fmt.Errorf("config not found by id: %d", id)
	}

	if err := cr.data.Cache.SetString(ctx, cacheKey, c.JsonString(), constant.ConfigCacheTime); err != nil {
		log.Error(err)
	}
	return c, nil
}

func (cr configRepo) GetConfigByKey(ctx context.Context, key string) (c *entity.Config, err error) {
	cacheKey := constant.ConfigKEY2ContentCacheKeyPrefix + key
	cacheData, exist, err := cr.data.Cache.GetString(ctx, cacheKey)
	if err == nil && exist && len(cacheData) > 0 {
		c = &entity.Config{}
		c.BuildByJSON([]byte(cacheData))
		if c.ID > 0 {
			return c, nil
		}
	}

	c = &entity.Config{}
	exist, err = cr.data.DB.Context(ctx).Where("`key` = ? AND site_id = ''", key).Get(c)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if !exist {
		return nil, fmt.Errorf("config not found by key: %s", key)
	}

	if err := cr.data.Cache.SetString(ctx, cacheKey, c.JsonString(), constant.ConfigCacheTime); err != nil {
		log.Error(err)
	}
	return c, nil
}

func (cr configRepo) GetConfigByKeyFromDB(ctx context.Context, key string) (c *entity.Config, err error) {
	c = &entity.Config{}
	exist, err := cr.data.DB.Context(ctx).Where("`key` = ? AND site_id = ''", key).Get(c)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if !exist {
		return nil, fmt.Errorf("config not found by key: %s", key)
	}
	return c, nil
}

func (cr configRepo) UpdateConfig(ctx context.Context, key string, value string) (err error) {
	// Config writes always target the global row: functional config is
	// org-global by design (see GetConfigByID).
	row := &entity.Config{}
	exist, err := cr.data.DB.Context(ctx).Where("`key` = ? AND site_id = ''", key).Get(row)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if !exist {
		return errors.BadRequest(reason.ObjectNotFound)
	}
	if _, err := cr.data.DB.Context(ctx).ID(row.ID).Update(&entity.Config{Value: value}); err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	row.Value = value
	cr.cacheConfig(ctx, key, row)
	return nil
}

func (cr configRepo) cacheConfig(ctx context.Context, key string, c *entity.Config) {
	cacheVal := c.JsonString()
	if err := cr.data.Cache.SetString(ctx,
		constant.ConfigKEY2ContentCacheKeyPrefix+key, cacheVal, constant.ConfigCacheTime); err != nil {
		log.Error(err)
	}
	if err := cr.data.Cache.SetString(ctx,
		fmt.Sprintf("%s%d", constant.ConfigID2KEYCacheKeyPrefix, c.ID), cacheVal, constant.ConfigCacheTime); err != nil {
		log.Error(err)
	}
}
