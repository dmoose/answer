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

package plugin_config

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/multisite"
	"github.com/apache/answer/internal/service/plugin_common"
	"github.com/segmentfault/pacman/errors"
)

type pluginConfigRepo struct {
	data *data.Data
}

// NewPluginConfigRepo new repository
func NewPluginConfigRepo(data *data.Data) plugin_common.PluginConfigRepo {
	return &pluginConfigRepo{
		data: data,
	}
}

// SavePluginConfig writes to the row scoped by the current site context, never
// the global row from inside a site request. Caller in a site context with no
// existing override inserts a new row.
func (ur *pluginConfigRepo) SavePluginConfig(ctx context.Context, pluginSlugName, configValue string) (err error) {
	siteID := multisite.SiteIDFromContext(ctx)
	old := &entity.PluginConfig{}
	exist, err := ur.data.DB.Context(ctx).
		Where("plugin_slug_name = ? AND site_id = ?", pluginSlugName, siteID).Get(old)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if exist {
		old.Value = configValue
		_, err = ur.data.DB.Context(ctx).ID(old.ID).Update(old)
	} else {
		_, err = ur.data.DB.Context(ctx).Insert(&entity.PluginConfig{
			PluginSlugName: pluginSlugName,
			Value:          configValue,
			SiteID:         siteID,
		})
	}
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

// GetPluginConfigAll is called at process startup with no site context to seed
// the in-memory plugin config. It returns global rows only; per-site plugin
// overrides exist in the schema but are not yet applied at the plugin runtime
// layer (see MULTISITE.md).
func (ur *pluginConfigRepo) GetPluginConfigAll(ctx context.Context) (pluginConfigs []*entity.PluginConfig, err error) {
	pluginConfigs = make([]*entity.PluginConfig, 0)
	err = ur.data.DB.Context(ctx).Where("site_id = ''").Find(&pluginConfigs)
	if err != nil {
		err = errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return pluginConfigs, err
}
