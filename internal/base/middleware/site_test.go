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
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/entity"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func newSiteTestMiddleware(t *testing.T) *SiteMiddleware {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "site-mw-test.db")
	engine, err := data.NewDB(false, &data.Database{
		Driver:     string(schemas.SQLITE),
		Connection: dbFile,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = engine.Close() })

	require.NoError(t, engine.Sync(new(entity.Site)))
	seedSites(t, engine)
	return NewSiteMiddleware(engine)
}

func seedSites(t *testing.T, engine *xorm.Engine) {
	t.Helper()
	sites := []entity.Site{
		{ID: constant.DefaultSiteID, Name: "Default", Slug: "default", Status: entity.SiteStatusActive},
		{ID: "site-a", Name: "Alpha", Slug: "alpha", Status: entity.SiteStatusActive},
		{ID: "site-s", Name: "Suspended", Slug: "asleep", Status: entity.SiteStatusSuspended},
	}
	for i := range sites {
		_, err := engine.Insert(&sites[i])
		require.NoError(t, err)
	}
}

// resolveWith runs ResolveSite over a synthetic request and reports the
// resolved site ID (empty when the middleware skipped or aborted) plus the
// response status and whether the chain continued.
func resolveWith(sm *SiteMiddleware, method, target, host, slugHeader string) (siteID string, status int, aborted bool) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, nil)
	if host != "" {
		req.Host = host
	}
	if slugHeader != "" {
		req.Header.Set("X-Site-Slug", slugHeader)
	}
	ctx.Request = req

	sm.ResolveSite()(ctx)

	if v, ok := ctx.Get(constant.SiteIDFlag); ok {
		siteID, _ = v.(string)
	}
	return siteID, w.Code, ctx.IsAborted()
}

// slugWith is resolveWith's companion for the recorded slug.
func slugWith(sm *SiteMiddleware, target, host, slugHeader string) string {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", target, nil)
	if host != "" {
		req.Host = host
	}
	if slugHeader != "" {
		req.Header.Set("X-Site-Slug", slugHeader)
	}
	ctx.Request = req
	sm.ResolveSite()(ctx)
	v, _ := ctx.Get(constant.SiteSlugFlag)
	slug, _ := v.(string)
	return slug
}

func TestResolveSite_RecordsSlug(t *testing.T) {
	sm := newSiteTestMiddleware(t)
	assert.Equal(t, "alpha", slugWith(sm, "/answer/api/v1/question/page", "", "alpha"), "header")
	assert.Equal(t, "alpha", slugWith(sm, "/s/alpha/questions", "", ""), "path prefix")
	assert.Equal(t, "alpha", slugWith(sm, "/questions", "alpha.example.com", ""), "subdomain")
	assert.Equal(t, "default", slugWith(sm, "/questions", "", ""), "fallback names the default site")
	assert.Equal(t, "", slugWith(sm, "/answer/admin/api/sites", "", "alpha"), "admin API skips resolution")
}

func TestResolveSite_AbsentSlugUsesDefault(t *testing.T) {
	sm := newSiteTestMiddleware(t)
	siteID, _, aborted := resolveWith(sm, "GET", "/questions", "example.com", "")
	assert.False(t, aborted)
	assert.Equal(t, constant.DefaultSiteID, siteID)
}

func TestResolveSite_PathPrefixWins(t *testing.T) {
	sm := newSiteTestMiddleware(t)
	siteID, _, aborted := resolveWith(sm, "GET", "/s/alpha/questions", "example.com", "")
	assert.False(t, aborted)
	assert.Equal(t, "site-a", siteID)
}

func TestResolveSite_UnknownExplicitSlugIs404(t *testing.T) {
	sm := newSiteTestMiddleware(t)

	// Page navigation: aborted with a 404 status (SPA shell rendering).
	_, status, aborted := resolveWith(sm, "GET", "/s/typo/questions", "example.com", "")
	assert.True(t, aborted, "unknown explicit slug must abort")
	assert.Equal(t, http.StatusNotFound, status)

	// API call with a bogus header: aborted, never falls back to default.
	siteID, _, aborted := resolveWith(sm, "GET", "/answer/api/v1/question/page", "example.com", "typo")
	assert.True(t, aborted)
	assert.Empty(t, siteID, "bogus X-Site-Slug must not resolve any site")
}

func TestResolveSite_SuspendedSiteNotServed(t *testing.T) {
	sm := newSiteTestMiddleware(t)
	_, _, aborted := resolveWith(sm, "GET", "/s/asleep/questions", "example.com", "")
	assert.True(t, aborted, "suspended site must not resolve")
}

func TestResolveSite_HeaderResolvesForAPI(t *testing.T) {
	sm := newSiteTestMiddleware(t)
	siteID, _, aborted := resolveWith(sm, "GET", "/answer/api/v1/question/page", "example.com", "alpha")
	assert.False(t, aborted)
	assert.Equal(t, "site-a", siteID)
}

func TestResolveSite_PathBeatsHeader(t *testing.T) {
	sm := newSiteTestMiddleware(t)
	siteID, _, aborted := resolveWith(sm, "GET", "/s/alpha/x", "example.com", "default")
	assert.False(t, aborted)
	assert.Equal(t, "site-a", siteID, "explicit path must outrank the header")
}

func TestResolveSite_SubdomainHeuristic(t *testing.T) {
	sm := newSiteTestMiddleware(t)

	// A first host label that IS a site slug resolves it.
	siteID, _, aborted := resolveWith(sm, "GET", "/questions", "alpha.example.com", "")
	assert.False(t, aborted)
	assert.Equal(t, "site-a", siteID)

	// A deployment host label that is NOT a slug falls through to the
	// default silently (it names the deployment, not a site).
	siteID, _, aborted = resolveWith(sm, "GET", "/questions", "answer.example.com", "")
	assert.False(t, aborted)
	assert.Equal(t, constant.DefaultSiteID, siteID)
}

func TestResolveSite_AdminAPISkipsResolution(t *testing.T) {
	sm := newSiteTestMiddleware(t)
	siteID, _, aborted := resolveWith(sm, "GET", "/answer/admin/api/dashboard", "example.com", "")
	assert.False(t, aborted)
	assert.Empty(t, siteID, "admin API must stay site-less")
}

func TestResolveSite_AdminSkipHonorsBasePath(t *testing.T) {
	sm := newSiteTestMiddleware(t)
	sm.SetBasePath("/community")
	siteID, _, aborted := resolveWith(sm, "GET", "/community/answer/admin/api/dashboard", "example.com", "")
	assert.False(t, aborted)
	assert.Empty(t, siteID, "admin API under a base path must also skip resolution")
}
