package config

import (
	"context"
	"fmt"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/multisite"
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

func (cr configRepo) cachePrefix(ctx context.Context) string {
	if siteID := multisite.SiteIDFromContext(ctx); siteID != "" {
		return siteID + ":"
	}
	return ""
}

func (cr configRepo) GetConfigByID(ctx context.Context, id int) (c *entity.Config, err error) {
	prefix := cr.cachePrefix(ctx)
	cacheKey := fmt.Sprintf("%s%s%d", constant.ConfigID2KEYCacheKeyPrefix, prefix, id)
	cacheData, exist, err := cr.data.Cache.GetString(ctx, cacheKey)
	if err == nil && exist && len(cacheData) > 0 {
		c = &entity.Config{}
		c.BuildByJSON([]byte(cacheData))
		if c.ID > 0 {
			return c, nil
		}
	}

	c = &entity.Config{}
	siteID := multisite.SiteIDFromContext(ctx)

	// Try site-specific config first
	if siteID != "" {
		exist, err = cr.data.DB.Context(ctx).Where("id = ? AND site_id = ?", id, siteID).Get(c)
		if err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
	}

	// Fall back to global default
	if !exist {
		c = &entity.Config{}
		exist, err = cr.data.DB.Context(ctx).Where("id = ? AND site_id = ''", id).Get(c)
		if err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
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
	prefix := cr.cachePrefix(ctx)
	cacheKey := constant.ConfigKEY2ContentCacheKeyPrefix + prefix + key
	cacheData, exist, err := cr.data.Cache.GetString(ctx, cacheKey)
	if err == nil && exist && len(cacheData) > 0 {
		c = &entity.Config{}
		c.BuildByJSON([]byte(cacheData))
		if c.ID > 0 {
			return c, nil
		}
	}

	c = &entity.Config{}
	siteID := multisite.SiteIDFromContext(ctx)

	// Try site-specific config first
	if siteID != "" {
		exist, err = cr.data.DB.Context(ctx).Where("`key` = ? AND site_id = ?", key, siteID).Get(c)
		if err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
	}

	// Fall back to global default
	if !exist {
		c = &entity.Config{}
		exist, err = cr.data.DB.Context(ctx).Where("`key` = ? AND site_id = ''", key).Get(c)
		if err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
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
	siteID := multisite.SiteIDFromContext(ctx)

	if siteID != "" {
		exist, err := cr.data.DB.Context(ctx).Where("`key` = ? AND site_id = ?", key, siteID).Get(c)
		if err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
		if exist {
			return c, nil
		}
	}

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
	siteID := multisite.SiteIDFromContext(ctx)

	// Find the config row to update: site-specific first, then global
	oldConfig := &entity.Config{}
	var exist bool
	if siteID != "" {
		exist, err = cr.data.DB.Context(ctx).Where("`key` = ? AND site_id = ?", key, siteID).Get(oldConfig)
		if err != nil {
			return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
	}
	if !exist {
		oldConfig = &entity.Config{}
		exist, err = cr.data.DB.Context(ctx).Where("`key` = ? AND site_id = ''", key).Get(oldConfig)
		if err != nil {
			return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
	}
	if !exist {
		return errors.BadRequest(reason.ObjectNotFound)
	}

	_, err = cr.data.DB.Context(ctx).ID(oldConfig.ID).Update(&entity.Config{Value: value})
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}

	oldConfig.Value = value
	cacheVal := oldConfig.JsonString()
	prefix := cr.cachePrefix(ctx)
	if err := cr.data.Cache.SetString(ctx,
		constant.ConfigKEY2ContentCacheKeyPrefix+prefix+key, cacheVal, constant.ConfigCacheTime); err != nil {
		log.Error(err)
	}
	if err := cr.data.Cache.SetString(ctx,
		fmt.Sprintf("%s%s%d", constant.ConfigID2KEYCacheKeyPrefix, prefix, oldConfig.ID), cacheVal, constant.ConfigCacheTime); err != nil {
		log.Error(err)
	}
	return
}
