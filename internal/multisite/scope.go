//go:build multisite

package multisite

import (
	"context"
	"reflect"

	"github.com/apache/answer/internal/base/constant"
	"github.com/gin-gonic/gin"
	"xorm.io/xorm"
)

func SiteIDFromContext(ctx context.Context) string {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		if val, exists := ginCtx.Get(constant.SiteIDFlag); exists {
			if siteID, ok := val.(string); ok {
				return siteID
			}
		}
	}
	if val, ok := ctx.Value(constant.SiteIDContextKey).(string); ok {
		return val
	}
	return ""
}

func Scope(session *xorm.Session, ctx context.Context) *xorm.Session {
	if siteID := SiteIDFromContext(ctx); siteID != "" {
		return session.Where("site_id = ?", siteID)
	}
	return session
}

func SetSiteID(ctx context.Context, entities ...any) {
	siteID := SiteIDFromContext(ctx)
	if siteID == "" {
		return
	}
	for _, e := range entities {
		v := reflect.ValueOf(e)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			continue
		}
		f := v.FieldByName("SiteID")
		if f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
			f.SetString(siteID)
		}
	}
}
