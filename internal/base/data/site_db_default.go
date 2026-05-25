//go:build !multisite

package data

import (
	"context"

	"xorm.io/xorm"
)

func (d *Data) SiteDB(ctx context.Context) *xorm.Session {
	return d.DB.Context(ctx)
}

func (d *Data) SiteTransaction(ctx context.Context, f func(*xorm.Session) (any, error)) (any, error) {
	return d.DB.Transaction(func(session *xorm.Session) (any, error) {
		session = session.Context(ctx)
		return f(session)
	})
}

func (d *Data) SiteInsert(ctx context.Context, beans ...any) (int64, error) {
	return d.DB.Context(ctx).Insert(beans...)
}
