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

package repo_test

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/repo/config"
	"github.com/apache/answer/internal/repo/plugin_config"
	"github.com/apache/answer/internal/repo/site_info"
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

// Test_configRepo_SiteUpdateDoesNotMutateGlobal verifies that UpdateConfig
// from a site context never touches the global default row. Pre-fix
// behavior fell through to update the global row when no override existed.
func Test_configRepo_SiteUpdateDoesNotMutateGlobal(t *testing.T) {
	repo := config.NewConfigRepo(testDataSource)
	db := testDataSource.DB

	// Seed a global config row directly.
	key := "test_leak_update_global"
	globalValue := "global-original"
	_, err := db.Insert(&entity.Config{Key: key, Value: globalValue, SiteID: ""})
	require.NoError(t, err)

	// Site A updates the key — should create an override, not touch global.
	err = repo.UpdateConfig(siteCtx(siteA), key, "site-a-value")
	require.NoError(t, err)

	// Global row unchanged.
	globalRow := &entity.Config{}
	exist, err := db.Where("`key` = ? AND site_id = ''", key).Get(globalRow)
	require.NoError(t, err)
	require.True(t, exist, "global row must still exist")
	assert.Equal(t, globalValue, globalRow.Value, "site update must not mutate global row")

	// Site A override exists with new value.
	siteRow := &entity.Config{}
	exist, err = db.Where("`key` = ? AND site_id = ?", key, siteA).Get(siteRow)
	require.NoError(t, err)
	require.True(t, exist, "site override row must have been inserted")
	assert.Equal(t, "site-a-value", siteRow.Value)
}

// Test_configRepo_SiteUpdateDoesNotLeakToOtherSite verifies that updating a
// config in Site A does not change what Site B reads (Site B continues to
// see the global default).
func Test_configRepo_SiteUpdateDoesNotLeakToOtherSite(t *testing.T) {
	repo := config.NewConfigRepo(testDataSource)
	db := testDataSource.DB

	key := "test_leak_cross_site"
	_, err := db.Insert(&entity.Config{Key: key, Value: "global-default", SiteID: ""})
	require.NoError(t, err)

	err = repo.UpdateConfig(siteCtx(siteA), key, "site-a-override")
	require.NoError(t, err)

	// Site B should still see the global default — bypass cache by clearing it.
	clearConfigCache(t, key, siteB)
	got, err := repo.GetConfigByKey(siteCtx(siteB), key)
	require.NoError(t, err)
	assert.Equal(t, "global-default", got.Value, "site B must not see site A's override")

	// And Site A sees its own override.
	clearConfigCache(t, key, siteA)
	gotA, err := repo.GetConfigByKey(siteCtx(siteA), key)
	require.NoError(t, err)
	assert.Equal(t, "site-a-override", gotA.Value)
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
	repo := site_info.NewSiteInfo(testDataSource)
	db := testDataSource.DB

	// Seed a global site_info row.
	siteType := "test_leak_siteinfo"
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
	repo := site_info.NewSiteInfo(testDataSource)
	db := testDataSource.DB

	siteType := "test_leak_siteinfo_read"
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

// Test_pluginConfigRepo_SiteScopedWrite verifies that SavePluginConfig from
// a site context writes to a site-specific row and does not overwrite the
// global row that the runtime loads at startup.
func Test_pluginConfigRepo_SiteScopedWrite(t *testing.T) {
	repo := plugin_config.NewPluginConfigRepo(testDataSource)
	db := testDataSource.DB

	slug := "test_leak_plugin"
	_, err := db.Insert(&entity.PluginConfig{PluginSlugName: slug, Value: "global-cfg", SiteID: ""})
	require.NoError(t, err)

	err = repo.SavePluginConfig(siteCtx(siteA), slug, "site-a-cfg")
	require.NoError(t, err)

	// Global row preserved — this is what startup actually loads.
	globalRow := &entity.PluginConfig{}
	exist, err := db.Where("plugin_slug_name = ? AND site_id = ''", slug).Get(globalRow)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "global-cfg", globalRow.Value)

	// Site A row exists.
	siteRow := &entity.PluginConfig{}
	exist, err = db.Where("plugin_slug_name = ? AND site_id = ?", slug, siteA).Get(siteRow)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, "site-a-cfg", siteRow.Value)

	// GetPluginConfigAll (startup path, no site context) returns global only.
	all, err := repo.GetPluginConfigAll(context.Background())
	require.NoError(t, err)
	for _, c := range all {
		assert.Equal(t, "", c.SiteID, "GetPluginConfigAll must return global rows only (got slug=%q site_id=%q)", c.PluginSlugName, c.SiteID)
	}
}

// clearConfigCache wipes the per-key config cache entries for a site so that
// a subsequent read hits the DB. Without this, a stale cache from an earlier
// step inside the same test pollutes the assertion.
func clearConfigCache(t *testing.T, key, siteID string) {
	t.Helper()
	prefix := ""
	if siteID != "" {
		prefix = siteID + ":"
	}
	cacheKey := "answer:config:key:" + prefix + key
	_ = testDataSource.Cache.Del(context.Background(), cacheKey)
}
