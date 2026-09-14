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

package repo_test

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/repo/config"
	"github.com/apache/answer/internal/repo/plugin_config"
	"github.com/apache/answer/internal/repo/site_info"
	"github.com/segmentfault/pacman/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	siteA = "site-aaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	siteB = "site-bbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func siteCtx(siteID string) context.Context {
	return context.WithValue(context.Background(), constant.SiteIDContextKey, siteID)
}

// ensureSites seeds the site rows the leak tests address; site_info
// overrides are refused for sites that do not exist.
func ensureSites(t *testing.T) {
	t.Helper()
	for _, id := range []string{siteA, siteB} {
		exist, err := testDataSource.DB.ID(id).Get(&entity.Site{})
		require.NoError(t, err)
		if !exist {
			_, err = testDataSource.DB.Insert(&entity.Site{ID: id, Name: id, Slug: id, Status: entity.SiteStatusActive})
			require.NoError(t, err)
		}
	}
}

// Test_configRepo_ConfigIsGlobal verifies the decided model: functional
// config is org-global. An update from ANY context (site or not) edits the
// one global row, and every site reads the same value. Per-site overrides
// are presentation-only and live in site_info, never in config — a config
// override would fork the auto-increment ID that activity rows store as
// activity_type, orphaning history.
func Test_configRepo_ConfigIsGlobal(t *testing.T) {
	repo := config.NewConfigRepo(testDataSource)
	db := testDataSource.DB

	key := "test_config_is_global"
	_, err := db.Insert(&entity.Config{Key: key, Value: "before", SiteID: ""})
	require.NoError(t, err)

	// An update under a site context edits the global row (config writes
	// are admin-only and the admin is network-global).
	err = repo.UpdateConfig(siteCtx(siteA), key, "after")
	require.NoError(t, err)

	globalRow := &entity.Config{}
	exist, err := db.Where("`key` = ? AND site_id = ''", key).Get(globalRow)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "after", globalRow.Value, "update must edit the global row")

	// No per-site override row is ever created.
	var n int64
	n, err = db.Where("`key` = ? AND site_id != ''", key).Count(&entity.Config{})
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "config must never grow per-site override rows")

	// Every site reads the same global value.
	clearConfigCache(t, key, siteB)
	got, err := repo.GetConfigByKey(siteCtx(siteB), key)
	require.NoError(t, err)
	assert.Equal(t, "after", got.Value)
}

// Test_configRepo_StrayOverrideRowIgnored verifies that a leftover per-site
// config row (from the retired override model on live databases) is never
// served: reads resolve the global row only.
func Test_configRepo_StrayOverrideRowIgnored(t *testing.T) {
	repo := config.NewConfigRepo(testDataSource)
	db := testDataSource.DB

	key := "test_config_stray_override"
	_, err := db.Insert(&entity.Config{Key: key, Value: "global", SiteID: ""})
	require.NoError(t, err)
	_, err = db.Insert(&entity.Config{Key: key, Value: "stray", SiteID: siteA})
	require.NoError(t, err)

	clearConfigCache(t, key, siteA)
	got, err := repo.GetConfigByKey(siteCtx(siteA), key)
	require.NoError(t, err)
	assert.Equal(t, "global", got.Value, "stray override rows must be ignored")
}

// Test_configRepo_GlobalUpdateInNoSiteContext verifies updates made without
// a site context still hit the global row (the no-multisite case).
func Test_configRepo_GlobalUpdateInNoSiteContext(t *testing.T) {
	repo := config.NewConfigRepo(testDataSource)
	db := testDataSource.DB

	key := "test_leak_global_update"
	_, err := db.Insert(&entity.Config{Key: key, Value: "before", SiteID: ""})
	require.NoError(t, err)

	err = repo.UpdateConfig(context.Background(), key, "after")
	require.NoError(t, err)

	row := &entity.Config{}
	exist, err := db.Where("`key` = ? AND site_id = ''", key).Get(row)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "after", row.Value)
}

// Test_siteInfoRepo_SiteSaveDoesNotMutateGlobal verifies that SaveByType
// from a site context writes an override row instead of overwriting the
// global default. Pre-fix the global row was overwritten regardless of
// site context.
func Test_siteInfoRepo_SiteSaveDoesNotMutateGlobal(t *testing.T) {
	ensureSites(t)
	repo := site_info.NewSiteInfo(testDataSource)
	db := testDataSource.DB

	// Seed a global site_info row (a presentation type: only those can be
	// overridden per site).
	siteType := constant.SiteTypeBranding
	_, err := db.Insert(&entity.SiteInfo{Type: siteType, Content: "global-content", Status: 1, SiteID: ""})
	require.NoError(t, err)

	// Site A saves — must write to its own row.
	siteAData := &entity.SiteInfo{Type: siteType, Content: "site-a-content", Status: 1}
	err = repo.SaveByType(siteCtx(siteA), siteType, siteAData)
	require.NoError(t, err)

	// Global row unchanged.
	globalRow := &entity.SiteInfo{}
	exist, err := db.Where("type = ? AND site_id = ''", siteType).Get(globalRow)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "global-content", globalRow.Content)

	// Site A row present with its content.
	siteARow := &entity.SiteInfo{}
	exist, err = db.Where("type = ? AND site_id = ?", siteType, siteA).Get(siteARow)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "site-a-content", siteARow.Content)
}

// Test_siteInfoRepo_SiteReadDoesNotLeak verifies that what Site B reads
// from GetByType is the global default, not Site A's override (cache and
// DB both checked).
func Test_siteInfoRepo_SiteReadDoesNotLeak(t *testing.T) {
	ensureSites(t)
	repo := site_info.NewSiteInfo(testDataSource)
	db := testDataSource.DB

	siteType := constant.SiteTypeCustomCssHTML
	_, err := db.Insert(&entity.SiteInfo{Type: siteType, Content: "global-banner", Status: 1, SiteID: ""})
	require.NoError(t, err)

	// Site A creates an override (using the repo so cache state lines up).
	err = repo.SaveByType(siteCtx(siteA), siteType, &entity.SiteInfo{Type: siteType, Content: "site-a-banner", Status: 1})
	require.NoError(t, err)

	// Site B reads — should get the global default.
	gotB, exist, err := repo.GetByType(siteCtx(siteB), siteType)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "global-banner", gotB.Content, "site B must not see site A's siteinfo override")

	// Site A reads — should get its own override.
	gotA, exist, err := repo.GetByType(siteCtx(siteA), siteType)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "site-a-banner", gotA.Content)
}

// Test_pluginConfigRepo_AlwaysWritesGlobal verifies that SavePluginConfig
// updates the global row regardless of site context. Plugin runtime is
// process-wide; earlier site-scoping silently hid configs from the loader.
func Test_pluginConfigRepo_AlwaysWritesGlobal(t *testing.T) {
	repo := plugin_config.NewPluginConfigRepo(testDataSource)
	db := testDataSource.DB

	slug := "test_leak_plugin"
	_, err := db.Insert(&entity.PluginConfig{PluginSlugName: slug, Value: "global-cfg", SiteID: ""})
	require.NoError(t, err)

	err = repo.SavePluginConfig(siteCtx(siteA), slug, "updated-from-site-a")
	require.NoError(t, err)

	globalRow := &entity.PluginConfig{}
	exist, err := db.Where("plugin_slug_name = ? AND site_id = ''", slug).Get(globalRow)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "updated-from-site-a", globalRow.Value)

	// No site-scoped row was created.
	var siteRows []*entity.PluginConfig
	err = db.Where("plugin_slug_name = ? AND site_id != ''", slug).Find(&siteRows)
	require.NoError(t, err)
	assert.Empty(t, siteRows)

	// GetPluginConfigAll (startup path, no site context) returns global only.
	all, err := repo.GetPluginConfigAll(context.Background())
	require.NoError(t, err)
	for _, c := range all {
		assert.Empty(t, c.SiteID, "GetPluginConfigAll must return global rows only (got slug=%q site_id=%q)", c.PluginSlugName, c.SiteID)
	}
}

// clearConfigCache wipes the config cache entry for a key so a subsequent
// read hits the DB. Config is global, so there is one cache key per config
// key regardless of site; the siteID parameter is kept for call-site clarity.
func clearConfigCache(t *testing.T, key, _ string) {
	t.Helper()
	cacheKey := "answer:config:key:" + key
	_ = testDataSource.Cache.Del(context.Background(), cacheKey)
}

// Test_siteInfoRepo_FunctionalTypesAreGlobal verifies the presentation
// allowlist: a non-presentation type saved from a site context lands on the
// global row and is read back from every site, and a stray override row for
// such a type is ignored on read.
func Test_siteInfoRepo_FunctionalTypesAreGlobal(t *testing.T) {
	ensureSites(t)
	repo := site_info.NewSiteInfo(testDataSource)
	db := testDataSource.DB
	siteType := constant.SiteTypeLogin

	err := repo.SaveByType(siteCtx(siteA), siteType, &entity.SiteInfo{Type: siteType, Content: "login-from-site-a", Status: 1})
	require.NoError(t, err)

	n, err := db.Where("type = ? AND site_id = ?", siteType, siteA).Count(&entity.SiteInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "functional type must never get a site row")
	global := &entity.SiteInfo{}
	exist, err := db.Where("type = ? AND site_id = ''", siteType).Get(global)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "login-from-site-a", global.Content)

	// A stray override row (legacy data) is invisible to reads.
	_, err = db.Insert(&entity.SiteInfo{Type: siteType, Content: "stray-override", Status: 1, SiteID: siteB})
	require.NoError(t, err)
	got, exist, err := repo.GetByType(siteCtx(siteB), siteType, true)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "login-from-site-a", got.Content)
}

// Test_siteInfoRepo_OverrideRequiresExistingSite verifies a presentation
// override cannot be written for a site that does not exist.
func Test_siteInfoRepo_OverrideRequiresExistingSite(t *testing.T) {
	repo := site_info.NewSiteInfo(testDataSource)
	err := repo.SaveByType(siteCtx("no-such-site"), constant.SiteTypeGeneral,
		&entity.SiteInfo{Type: constant.SiteTypeGeneral, Content: "{}", Status: 1})
	require.Error(t, err)
	perr, ok := err.(*errors.Error)
	require.True(t, ok, "expected a pacman error, got %#v", err)
	assert.True(t, errors.IsNotFound(perr), "expected not-found, got reason %q", perr.Reason)
}
