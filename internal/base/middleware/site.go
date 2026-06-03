//go:build multisite

package middleware

import (
	"strings"
	"sync"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/gin-gonic/gin"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
	"xorm.io/xorm"
)

type SiteMiddleware struct {
	db    *xorm.Engine
	mu    sync.RWMutex
	cache map[string]string // slug → site_id
}

func NewSiteMiddleware(db *xorm.Engine) *SiteMiddleware {
	sm := &SiteMiddleware{db: db, cache: make(map[string]string)}
	sm.refreshCache()
	return sm
}

func (sm *SiteMiddleware) refreshCache() {
	var sites []entity.Site
	if err := sm.db.Where("status = ?", entity.SiteStatusActive).Find(&sites); err != nil {
		log.Errorf("load sites: %v", err)
		return
	}
	m := make(map[string]string, len(sites))
	for _, s := range sites {
		m[s.Slug] = s.ID
	}
	sm.mu.Lock()
	sm.cache = m
	sm.mu.Unlock()
}

func (sm *SiteMiddleware) resolve(slug string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.cache[slug]
}

func (sm *SiteMiddleware) validSiteID(id string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, v := range sm.cache {
		if v == id {
			return true
		}
	}
	return false
}

func (sm *SiteMiddleware) ResolveSite() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		if path == "/healthz" ||
			strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/install/") {
			ctx.Next()
			return
		}

		var siteID string

		// 1. Explicit ID header (API clients) — validate against known sites
		if h := ctx.GetHeader("X-Site-ID"); h != "" {
			if sm.validSiteID(h) {
				siteID = h
			}
		}

		// 1b. Slug header (frontend before site ID is known)
		if siteID == "" {
			if h := ctx.GetHeader("X-Site-Slug"); h != "" {
				siteID = sm.resolve(h)
			}
		}

		// 2. Subdomain: product-a.example.com
		if siteID == "" {
			host := ctx.Request.Host
			if idx := strings.LastIndex(host, ":"); idx > 0 {
				host = host[:idx]
			}
			parts := strings.SplitN(host, ".", 2)
			if len(parts) >= 2 && parts[0] != "www" {
				siteID = sm.resolve(parts[0])
			}
		}

		// 3. Path prefix: /s/{slug} or /s/{slug}/...
		if siteID == "" {
			path := ctx.Request.URL.Path
			if strings.HasPrefix(path, "/s/") {
				rest := path[3:]
				slug := rest
				if idx := strings.Index(rest, "/"); idx > 0 {
					slug = rest[:idx]
				}
				if id := sm.resolve(slug); id != "" {
					siteID = id
				}
			}
		}

		// 4. Fallback to the default site
		if siteID == "" {
			siteID = sm.resolve("default")
		}

		if siteID == "" {
			handler.HandleResponse(ctx, errors.NotFound(reason.ObjectNotFound), nil)
			ctx.Abort()
			return
		}
		ctx.Set(constant.SiteIDFlag, siteID)
		ctx.Next()
	}
}

func (sm *SiteMiddleware) RefreshSiteCache() {
	sm.refreshCache()
}
