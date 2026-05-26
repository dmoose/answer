//go:build multisite

package config

import (
	"context"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/multisite"
)

func (cr configRepo) getConfigFallback(ctx context.Context, key string) (*entity.Config, bool, error) {
	siteID := multisite.SiteIDFromContext(ctx)
	if siteID == "" {
		return nil, false, nil
	}
	c := &entity.Config{Key: key}
	exist, err := cr.data.DB.Context(ctx).Where("site_id = ''").Get(c)
	return c, exist, err
}
