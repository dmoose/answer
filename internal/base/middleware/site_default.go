//go:build !multisite

package middleware

import (
	"github.com/gin-gonic/gin"
	"xorm.io/xorm"
)

type SiteMiddleware struct{}

func NewSiteMiddleware(_ *xorm.Engine) *SiteMiddleware {
	return &SiteMiddleware{}
}

func (sm *SiteMiddleware) ResolveSite() gin.HandlerFunc {
	return func(ctx *gin.Context) { ctx.Next() }
}

func (sm *SiteMiddleware) RefreshSiteCache() {}
