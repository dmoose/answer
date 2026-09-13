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

package middleware

import (
	"net/http"
	"strings"
	"sync"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/ui"
	"github.com/gin-gonic/gin"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
	"xorm.io/xorm"
)

type SiteMiddleware struct {
	db       *xorm.Engine
	basePath string // uiConf.APIBaseURL — routes register under it
	mu       sync.RWMutex
	cache    map[string]string // slug → site_id
	fallback string            // lowest-ID active site, used when default is gone
}

// SetBasePath records the deployment's API base URL so the resolution
// skip-list matches the paths routes actually register under. Without this,
// a non-empty api_url would put the admin API through site resolution and
// break its recovery role.
func (sm *SiteMiddleware) SetBasePath(basePath string) {
	sm.basePath = strings.TrimRight(basePath, "/")
}

func NewSiteMiddleware(db *xorm.Engine) *SiteMiddleware {
	sm := &SiteMiddleware{db: db, cache: make(map[string]string)}
	sm.refreshCache()
	return sm
}

func (sm *SiteMiddleware) refreshCache() {
	var sites []entity.Site
	if err := sm.db.OrderBy("id ASC").
		Where("status = ?", entity.SiteStatusActive).Find(&sites); err != nil {
		log.Errorf("load sites: %v", err)
		return
	}
	m := make(map[string]string, len(sites))
	fallback := ""
	for _, s := range sites {
		m[s.Slug] = s.ID
		if fallback == "" {
			fallback = s.ID
		}
	}
	sm.mu.Lock()
	sm.cache = m
	sm.fallback = fallback
	sm.mu.Unlock()
}

func (sm *SiteMiddleware) resolve(slug string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.cache[slug]
}

// fallbackSiteID returns the lowest-ID active site, used when "default"
// is missing or inactive. Keeps the network serving even when an admin
// manages to break the default routing.
func (sm *SiteMiddleware) fallbackSiteID() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.fallback
}

// ResolveSite picks the active site from request identity in priority order:
//
//  1. Subdomain (product-a.example.com)
//  2. /s/<slug>/... path prefix
//  3. X-Site-Slug header — SPA bootstrap before the URL has /s/, validated
//     against the known-sites cache like the other paths
//  4. Fall back to the default site so the network is reachable before any
//     sites are configured
//
// Host and path always win over headers so a browser cannot fetch another
// site's content from the wrong domain.
func (sm *SiteMiddleware) ResolveSite() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := strings.TrimPrefix(ctx.Request.URL.Path, sm.basePath)
		if path == "/healthz" ||
			strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/install/") ||
			strings.HasPrefix(path, "/answer/admin/api/") {
			// Admin API skips site resolution so an admin can recover the
			// network from the UI even if all the site routing is broken
			// (e.g. default site got deactivated, slug got renamed).
			ctx.Next()
			return
		}

		var siteID, siteSlug string

		// 1. Subdomain — a heuristic: the first host label is often the
		// deployment host ("answer.example.com"), not a site slug, so a
		// miss here falls through silently rather than 404ing.
		host := ctx.Request.Host
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			host = host[:idx]
		}
		parts := strings.SplitN(host, ".", 2)
		if len(parts) >= 2 && parts[0] != "www" {
			if siteID = sm.resolve(parts[0]); siteID != "" {
				siteSlug = parts[0]
			}
		}

		// 2. Path prefix — explicit: /s/<slug> names a site, so an unknown
		// slug is a real not-found, never a silent default-site render
		// under the wrong URL.
		if strings.HasPrefix(path, "/s/") {
			rest := path[3:]
			slug := rest
			if idx := strings.Index(rest, "/"); idx > 0 {
				slug = rest[:idx]
			}
			if resolved := sm.resolve(slug); resolved != "" {
				siteID, siteSlug = resolved, slug
			} else {
				siteNotFound(ctx, path)
				return
			}
		}

		// 3. X-Site-Slug header (SPA bootstrap) — explicit like the path.
		if siteID == "" {
			if h := ctx.GetHeader("X-Site-Slug"); h != "" {
				if resolved := sm.resolve(h); resolved != "" {
					siteID, siteSlug = resolved, h
				} else {
					siteNotFound(ctx, path)
					return
				}
			}
		}

		// 4. Default fallback — only when NO explicit site was specified.
		// Try the "default" slug first, then any active site.
		// Belt-and-suspenders so the network keeps serving even if the
		// default site gets deactivated by accident.
		if siteID == "" {
			if siteID = sm.resolve("default"); siteID != "" {
				siteSlug = "default"
			}
		}
		if siteID == "" {
			siteID = sm.fallbackSiteID()
		}

		if siteID == "" {
			handler.HandleResponse(ctx, errors.NotFound(reason.ObjectNotFound), nil)
			ctx.Abort()
			return
		}
		ctx.Set(constant.SiteIDFlag, siteID)
		ctx.Set(constant.SiteSlugFlag, siteSlug)
		ctx.Next()
	}
}

// siteNotFound rejects an explicitly named but unknown site: JSON for API
// calls, the SPA shell with a 404 status for browser navigations (so the
// frontend renders its not-found page instead of raw JSON).
func siteNotFound(ctx *gin.Context, path string) {
	if strings.HasPrefix(path, "/answer/") {
		handler.HandleResponse(ctx, errors.NotFound(reason.SiteNotFound), nil)
		ctx.Abort()
		return
	}
	ctx.Header("content-type", "text/html;charset=utf-8")
	ctx.Header("X-Frame-Options", "DENY")
	if file, err := ui.Build.ReadFile("build/index.html"); err == nil {
		ctx.Data(http.StatusNotFound, "text/html;charset=utf-8", file)
	} else {
		ctx.Status(http.StatusNotFound)
	}
	ctx.Abort()
}

func (sm *SiteMiddleware) RefreshSiteCache() {
	sm.refreshCache()
}
