//go:build !multisite

package multisite

import (
	"context"

	"xorm.io/xorm"
)

func SiteIDFromContext(_ context.Context) string { return "" }

func Scope(session *xorm.Session, _ context.Context) *xorm.Session { return session }

func SetSiteID(_ context.Context, _ ...any) {}
