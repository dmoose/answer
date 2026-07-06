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

package site_info

import (
	"context"
	"encoding/json"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/multisite"
	"github.com/apache/answer/internal/service/siteinfo_common"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
	"xorm.io/builder"
)

type siteInfoRepo struct {
	data *data.Data
}

func NewSiteInfo(data *data.Data) siteinfo_common.SiteInfoRepo {
	return &siteInfoRepo{
		data: data,
	}
}

// SaveByType save site setting by type. From a site context, writes a per-site
// override; from no site context, writes the global default. Never overwrites
// the global row from inside a site request.
func (sr *siteInfoRepo) SaveByType(ctx context.Context, siteType string, data *entity.SiteInfo) (err error) {
	siteID := multisite.TierSiteID(ctx)
	data.SiteID = siteID

	old := &entity.SiteInfo{}
	exist, err := sr.data.DB.Context(ctx).Where("type = ? AND site_id = ?", siteType, siteID).Get(old)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if exist {
		_, err = sr.data.DB.Context(ctx).ID(old.ID).Update(data)
	} else {
		_, err = sr.data.DB.Context(ctx).Insert(data)
	}
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	sr.setCache(ctx, siteID, siteType, data)
	return
}

// GetByType returns the per-site override when present, falling back to the
// global default row.
//
// Cache entries are keyed by the tier the ROW belongs to (the override's
// site, or the global key), never by the requesting site: caching a global
// row under a site-qualified key would leave every non-overridden site
// serving stale values for up to the TTL after a global edit, because the
// edit only rewrites the global key. A site with no override caches a short
// "absent" marker so the fallback doesn't hit the DB on every request.
func (sr *siteInfoRepo) GetByType(ctx context.Context, siteType string, withoutCache ...bool) (siteInfo *entity.SiteInfo, exist bool, err error) {
	useCache := len(withoutCache) == 0
	siteID := multisite.TierSiteID(ctx)

	if siteID != "" {
		overrideKnownAbsent := false
		if useCache {
			if info, state := sr.getCache(ctx, siteID, siteType); state == siteInfoCacheHit {
				return info, true, nil
			} else if state == siteInfoCacheAbsent {
				overrideKnownAbsent = true
			}
		}
		if !overrideKnownAbsent {
			siteInfo = &entity.SiteInfo{}
			exist, err = sr.data.DB.Context(ctx).Where("type = ? AND site_id = ?", siteType, siteID).Get(siteInfo)
			if err != nil {
				return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
			}
			if exist {
				sr.setCache(ctx, siteID, siteType, siteInfo)
				return
			}
			sr.markOverrideAbsent(ctx, siteID, siteType)
		}
	}

	if useCache {
		if info, state := sr.getCache(ctx, "", siteType); state == siteInfoCacheHit {
			return info, true, nil
		}
	}
	siteInfo = &entity.SiteInfo{}
	exist, err = sr.data.DB.Context(ctx).Where("type = ? AND site_id = ''", siteType).Get(siteInfo)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if exist {
		sr.setCache(ctx, "", siteType, siteInfo)
	}
	return
}

type siteInfoCacheState int

const (
	siteInfoCacheMiss siteInfoCacheState = iota
	siteInfoCacheHit
	// siteInfoCacheAbsent means "this site has no override row" — fall to
	// the global tier without re-querying the site row.
	siteInfoCacheAbsent
)

// siteInfoOverrideAbsentMarker is stored under a site's cache key when the
// site has no override row; it is not valid SiteInfo JSON.
const siteInfoOverrideAbsentMarker = "__absent__"

func siteInfoCacheKey(siteID, siteType string) string {
	if siteID == "" {
		return constant.SiteInfoCacheKey + siteType
	}
	return constant.SiteInfoCacheKey + siteID + ":" + siteType
}

func (sr *siteInfoRepo) getCache(ctx context.Context, siteID, siteType string) (*entity.SiteInfo, siteInfoCacheState) {
	siteInfoCache, exist, err := sr.data.Cache.GetString(ctx, siteInfoCacheKey(siteID, siteType))
	if err != nil || !exist {
		return nil, siteInfoCacheMiss
	}
	if siteInfoCache == siteInfoOverrideAbsentMarker {
		return nil, siteInfoCacheAbsent
	}
	siteInfo := &entity.SiteInfo{}
	if err := json.Unmarshal([]byte(siteInfoCache), siteInfo); err != nil {
		log.Errorf("unmarshal cached site info %s: %v", siteType, err)
		return nil, siteInfoCacheMiss
	}
	return siteInfo, siteInfoCacheHit
}

func (sr *siteInfoRepo) setCache(ctx context.Context, siteID, siteType string, siteInfo *entity.SiteInfo) {
	siteInfoCache, err := json.Marshal(siteInfo)
	if err != nil {
		log.Errorf("marshal site info %s for cache: %v", siteType, err)
		return
	}
	if err := sr.data.Cache.SetString(ctx,
		siteInfoCacheKey(siteID, siteType), string(siteInfoCache), constant.SiteInfoCacheTime); err != nil {
		log.Error(err)
	}
}

func (sr *siteInfoRepo) markOverrideAbsent(ctx context.Context, siteID, siteType string) {
	if err := sr.data.Cache.SetString(ctx,
		siteInfoCacheKey(siteID, siteType), siteInfoOverrideAbsentMarker, constant.SiteInfoCacheTime); err != nil {
		log.Error(err)
	}
}

func (sr *siteInfoRepo) IsBrandingFileUsed(ctx context.Context, filePath string) (bool, error) {
	siteInfo := &entity.SiteInfo{}
	count, err := sr.data.DB.Context(ctx).
		Table("site_info").
		Where(builder.Eq{"type": "branding"}).
		And(builder.Like{"content", "%" + filePath + "%"}).
		Count(&siteInfo)

	if err != nil {
		return false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}

	return count > 0, nil
}
