//go:build multisite

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
		path := ctx.Request.URL.Path
		if path == "/healthz" ||
			strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/install/") {
			ctx.Next()
			return
		}

		var siteID string

		// 1. Subdomain
		host := ctx.Request.Host
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			host = host[:idx]
		}
		parts := strings.SplitN(host, ".", 2)
		if len(parts) >= 2 && parts[0] != "www" {
			siteID = sm.resolve(parts[0])
		}

		// 2. Path prefix
		if siteID == "" && strings.HasPrefix(path, "/s/") {
			rest := path[3:]
			slug := rest
			if idx := strings.Index(rest, "/"); idx > 0 {
				slug = rest[:idx]
			}
			siteID = sm.resolve(slug)
		}

		// 3. X-Site-Slug header (SPA bootstrap)
		if siteID == "" {
			if h := ctx.GetHeader("X-Site-Slug"); h != "" {
				siteID = sm.resolve(h)
			}
		}

		// 4. Default fallback
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
