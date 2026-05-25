//go:build multisite

package data

import (
	"context"

	"github.com/apache/answer/internal/multisite"
	"xorm.io/xorm"
)

func (d *Data) SiteDB(ctx context.Context) *xorm.Session {
	session := d.DB.Context(ctx)
	if siteID := multisite.SiteIDFromContext(ctx); siteID != "" {
		session = session.Where("site_id = ?", siteID)
	}
	return session
}

func (d *Data) SiteInsert(ctx context.Context, beans ...any) (int64, error) {
	multisite.SetSiteID(ctx, beans...)
	return d.DB.Context(ctx).Insert(beans...)
}
