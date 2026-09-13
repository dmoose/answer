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

package migrations

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func newMigrationTestEngine(t *testing.T) *xorm.Engine {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "migrate-test.db")
	engine, err := data.NewDB(false, &data.Database{
		Driver:     string(schemas.SQLITE),
		Connection: dbFile,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = engine.Close() })
	return engine
}

// Legacy (pre-multisite) table shapes: identical to the current entities minus
// site_id, with the original single-column unique indexes. Declared as xorm
// entities so the fixture columns are exactly what xorm itself would have
// created — hand-rolled DDL drifts from xorm's type mapping and makes Sync
// attempt unsupported ALTERs on SQLite.
type legacyTag struct {
	ID              string    `xorm:"not null pk comment('tag_id') BIGINT(20) id"`
	CreatedAt       time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt       time.Time `xorm:"updated TIMESTAMP updated_at"`
	MainTagID       int64     `xorm:"not null default 0 BIGINT(20) main_tag_id"`
	MainTagSlugName string    `xorm:"not null default '' VARCHAR(35) main_tag_slug_name"`
	SlugName        string    `xorm:"not null default '' VARCHAR(35) UNIQUE slug_name"`
	DisplayName     string    `xorm:"not null default '' VARCHAR(35) display_name"`
	OriginalText    string    `xorm:"not null MEDIUMTEXT original_text"`
	ParsedText      string    `xorm:"not null MEDIUMTEXT parsed_text"`
	FollowCount     int       `xorm:"not null default 0 INT(11) follow_count"`
	QuestionCount   int       `xorm:"not null default 0 INT(11) question_count"`
	Status          int       `xorm:"not null default 1 INT(11) status"`
	Recommend       bool      `xorm:"not null default false BOOL recommend"`
	Reserved        bool      `xorm:"not null default false BOOL reserved"`
	RevisionID      string    `xorm:"not null default 0 BIGINT(20) revision_id"`
	UserID          string    `xorm:"not null default 0 BIGINT(20) user_id"`
}

func (legacyTag) TableName() string { return "tag" }

type legacyConfig struct {
	ID    int    `xorm:"not null pk autoincr INT(11) id"`
	Key   string `xorm:"VARCHAR(128) UNIQUE key"`
	Value string `xorm:"TEXT value"`
}

func (legacyConfig) TableName() string { return "config" }

type legacyPluginConfig struct {
	ID             int    `xorm:"not null pk autoincr INT(11) id"`
	PluginSlugName string `xorm:"VARCHAR(128) UNIQUE plugin_slug_name"`
	Value          string `xorm:"TEXT value"`
}

func (legacyPluginConfig) TableName() string { return "plugin_config" }

// seedPreV33 builds the slice of a pre-multisite database that the fork migrations
// exercise: the three tables with legacy single-column unique indexes plus
// user_role_rel content (one multi-role user to prove the dedup).
func seedPreV33(t *testing.T, x *xorm.Engine) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, x.Context(ctx).Sync(
		new(legacyTag), new(legacyConfig), new(legacyPluginConfig)))
	_, err := x.Context(ctx).Insert(&legacyTag{ID: "10", SlugName: "go", DisplayName: "Go"})
	require.NoError(t, err)
	_, err = x.Context(ctx).Insert(&legacyConfig{Key: "daily_rank_limit", Value: "200"})
	require.NoError(t, err)

	require.NoError(t, x.Context(ctx).Sync(new(entity.UserRoleRel)))
	rels := []entity.UserRoleRel{
		{UserID: "u1", RoleID: 1},
		{UserID: "u1", RoleID: 2}, // multi-role user: admin must win the dedup
		{UserID: "u2", RoleID: 3},
	}
	for i := range rels {
		_, err := x.Context(ctx).Insert(&rels[i])
		require.NoError(t, err)
	}
}

// runMultisiteMigrations runs the fork ledger the way the real runner would:
// every fork migration, in order, from a fork version of 0.
func runMultisiteMigrations(t *testing.T, x *xorm.Engine) {
	t.Helper()
	require.NoError(t, migrateFork(context.Background(), x, nil))
	v, err := getForkDBVersion(context.Background(), x)
	require.NoError(t, err)
	require.Equal(t, ForkExpectedVersion(), v)
}

func uniqueIndexNames(t *testing.T, x *xorm.Engine, table string) map[string]bool {
	t.Helper()
	type row struct {
		Name string `xorm:"'name'"`
	}
	var rows []row
	err := x.SQL(
		"SELECT name FROM sqlite_master WHERE type='index' AND tbl_name=? AND sql LIKE 'CREATE UNIQUE%'",
		table).Find(&rows)
	require.NoError(t, err)
	names := map[string]bool{}
	for _, r := range rows {
		names[r.Name] = true
	}
	return names
}

func TestMultisiteMigrationOnSQLite(t *testing.T) {
	x := newMigrationTestEngine(t)
	seedPreV33(t, x)
	runMultisiteMigrations(t, x)
	ctx := context.Background()

	// Legacy single-column uniques replaced by the composite (col, site_id).
	tagIdx := uniqueIndexNames(t, x, "tag")
	assert.False(t, tagIdx["UQE_tag_slug_name"], "legacy tag unique must be dropped")
	assert.False(t, uniqueIndexNames(t, x, "config")["UQE_config_key"])
	assert.False(t, uniqueIndexNames(t, x, "plugin_config")["UQE_plugin_config_plugin_slug_name"])

	// Per-site uniqueness enforced: same slug on the same site rejected,
	// same slug on another site accepted.
	_, err := x.Context(ctx).Exec(
		"INSERT INTO `tag` (`id`, `slug_name`, `display_name`, `original_text`, `parsed_text`, `site_id`) VALUES (11, 'go', 'Go', '', '', ?)",
		constant.DefaultSiteID)
	assert.Error(t, err, "duplicate (slug, site) must violate the composite unique")
	_, err = x.Context(ctx).Exec(
		"INSERT INTO `tag` (`id`, `slug_name`, `display_name`, `original_text`, `parsed_text`, `site_id`) VALUES (12, 'go', 'Go', '', '', 'site-b')")
	assert.NoError(t, err, "same slug on another site must be allowed")

	_, err = x.Context(ctx).Exec(
		"INSERT INTO `config` (`key`, `value`, `site_id`) VALUES ('daily_rank_limit', '300', ?)",
		constant.DefaultSiteID)
	assert.NoError(t, err, "config override on another site id must be allowed")
	_, err = x.Context(ctx).Exec(
		"INSERT INTO `config` (`key`, `value`, `site_id`) VALUES ('daily_rank_limit', '400', ?)",
		constant.DefaultSiteID)
	assert.Error(t, err, "duplicate (key, site) must violate the composite unique")

	// Existing content backfilled onto the default site; config rows stay
	// site_id='' as the global fallback tier.
	type siteIDRow struct {
		SiteID string `xorm:"'site_id'"`
	}
	var tagRow siteIDRow
	_, err = x.SQL("SELECT site_id FROM tag WHERE id = '10'").Get(&tagRow)
	require.NoError(t, err)
	assert.Equal(t, constant.DefaultSiteID, tagRow.SiteID)
	var cfgRow siteIDRow
	_, err = x.SQL("SELECT site_id FROM config WHERE `key` = 'daily_rank_limit' AND site_id = ''").Get(&cfgRow)
	require.NoError(t, err)

	// Role backfill: exactly one row per user on the default site, most
	// privileged role winning for multi-role users.
	var roleRows []entity.UserSiteRoleRel
	require.NoError(t, x.Context(ctx).Find(&roleRows))
	got := map[string]int{}
	for _, r := range roleRows {
		assert.Equal(t, constant.DefaultSiteID, r.SiteID)
		_, dup := got[r.UserID]
		assert.False(t, dup, "one site-role row per user, got extra for %s", r.UserID)
		got[r.UserID] = r.RoleID
	}
	assert.Equal(t, map[string]int{"u1": 2, "u2": 3}, got)

	// user_site_rank is retired.
	var n int64
	_, err = x.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='user_site_rank'").Get(&n)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "user_site_rank must be dropped")

	// The repair migration is idempotent.
	require.NoError(t, repairMultisiteSchema(ctx, x))
	var again []entity.UserSiteRoleRel
	require.NoError(t, x.Context(ctx).Find(&again))
	assert.Len(t, again, len(roleRows), "re-running the repair must not duplicate role rows")
}

// A database whose composite-unique candidates hold duplicate rows must fail
// the repair loudly instead of silently deleting content.
func TestRepairRefusesDuplicates(t *testing.T) {
	x := newMigrationTestEngine(t)
	ctx := context.Background()
	stmts := []string{
		"CREATE TABLE `tag` (`id` INTEGER PRIMARY KEY, `slug_name` VARCHAR(35) NOT NULL DEFAULT '', `site_id` VARCHAR(36) NOT NULL DEFAULT '')",
		"INSERT INTO `tag` (`id`, `slug_name`, `site_id`) VALUES (1, 'go', 'default')",
		"INSERT INTO `tag` (`id`, `slug_name`, `site_id`) VALUES (2, 'go', 'default')",
	}
	for _, stmt := range stmts {
		_, err := x.Context(ctx).Exec(stmt)
		require.NoError(t, err, stmt)
	}

	err := repairMultisiteSchema(ctx, x)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag")
	assert.Contains(t, err.Error(), "duplicated")
}

// Fresh installs (full entity Sync) and migrated installs must converge on
// the same unique-index sets for the three per-site-unique tables.
func TestFreshAndMigratedSchemasConverge(t *testing.T) {
	migrated := newMigrationTestEngine(t)
	seedPreV33(t, migrated)
	runMultisiteMigrations(t, migrated)

	fresh := newMigrationTestEngine(t)
	require.NoError(t, fresh.Sync(tables...))

	for _, table := range []string{"tag", "config", "plugin_config"} {
		assert.Equal(t,
			uniqueIndexNames(t, fresh, table),
			uniqueIndexNames(t, migrated, table),
			"unique indexes on %s must match a fresh install", table)
	}
}

func upstreamVersion(t *testing.T, x *xorm.Engine) int64 {
	t.Helper()
	row := &entity.Version{ID: 1}
	has, err := x.Get(row)
	require.NoError(t, err)
	require.True(t, has, "version row missing")
	return row.VersionNumber
}

func forkLedger(t *testing.T, x *xorm.Engine) (int64, bool) {
	t.Helper()
	row := &forkVersion{ID: forkVersionID}
	has, err := x.Get(row)
	require.NoError(t, err)
	return row.VersionNumber, has
}

func mustForkLedger(t *testing.T, x *xorm.Engine) int64 {
	t.Helper()
	v, has := forkLedger(t, x)
	require.True(t, has, "fork_version row missing")
	return v
}

// legacyLedgerDB builds a database in the pre-split layout: fork migrations
// applied, version row 1 holding the shared count, no fork row.
func legacyLedgerDB(t *testing.T) (*xorm.Engine, string) {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "legacy-ledger.db")
	x, err := data.NewDB(false, &data.Database{Driver: string(schemas.SQLITE), Connection: dbFile})
	require.NoError(t, err)
	t.Cleanup(func() { _ = x.Close() })
	seedPreV33(t, x)
	runMultisiteMigrations(t, x)
	require.NoError(t, x.DropTables(new(forkVersion)))
	require.NoError(t, x.Sync(new(entity.Version)))
	_, err = x.Insert(&entity.Version{ID: 1, VersionNumber: legacySharedVersion})
	require.NoError(t, err)
	return x, dbFile
}

func TestForkLedgerBootstrapSplitsLegacyLayout(t *testing.T) {
	x, _ := legacyLedgerDB(t)
	ctx := context.Background()

	require.NoError(t, bootstrapForkLedger(ctx, x))
	assert.EqualValues(t, legacyUpstreamCount, upstreamVersion(t, x))
	assert.Equal(t, ForkExpectedVersion(), mustForkLedger(t, x))

	// Second run is a no-op: the fork row is the "already split" marker.
	require.NoError(t, bootstrapForkLedger(ctx, x))
	assert.EqualValues(t, legacyUpstreamCount, upstreamVersion(t, x))
	assert.Equal(t, ForkExpectedVersion(), mustForkLedger(t, x))
}

func TestForkLedgerBootstrapVanillaStartsAtZero(t *testing.T) {
	x := newMigrationTestEngine(t)
	ctx := context.Background()
	require.NoError(t, x.Sync(new(entity.Version)))
	_, err := x.Insert(&entity.Version{ID: 1, VersionNumber: legacyUpstreamCount})
	require.NoError(t, err)

	require.NoError(t, bootstrapForkLedger(ctx, x))
	assert.EqualValues(t, legacyUpstreamCount, upstreamVersion(t, x), "upstream ledger untouched")
	assert.EqualValues(t, 0, mustForkLedger(t, x), "vanilla DB owes every fork migration")
}

func TestForkLedgerBootstrapRefusesUnknownLayout(t *testing.T) {
	x, _ := legacyLedgerDB(t)
	_, err := x.ID(1).Cols("version_number").Update(&entity.Version{VersionNumber: legacySharedVersion - 1})
	require.NoError(t, err)

	err = bootstrapForkLedger(context.Background(), x)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inspect the version table")
	_, has := forkLedger(t, x)
	assert.False(t, has, "no fork row written on refusal")
}

// TestMigrateSplitsLedgerEndToEnd drives the real upgrade entry point on a
// legacy-layout database: both ledgers land at their expected values and the
// fork migrations, already applied, no-op through.
func TestMigrateSplitsLedgerEndToEnd(t *testing.T) {
	x, dbFile := legacyLedgerDB(t)
	require.NoError(t, x.Close())
	dbConf := &data.Database{Driver: string(schemas.SQLITE), Connection: dbFile}

	require.NoError(t, Migrate(false, dbConf, &data.CacheConf{}, ""))

	x, err := data.NewDB(false, dbConf)
	require.NoError(t, err)
	defer func() { _ = x.Close() }()
	assert.Equal(t, ExpectedVersion(), upstreamVersion(t, x))
	assert.Equal(t, ForkExpectedVersion(), mustForkLedger(t, x))
	var n int64
	_, err = x.SQL("SELECT COUNT(*) FROM `user_site_role_rel`").Get(&n)
	require.NoError(t, err)
	assert.Greater(t, n, int64(0), "fork data survives the split")

	// Running upgrade again is a pure no-op.
	require.NoError(t, Migrate(false, dbConf, &data.CacheConf{}, ""))
	assert.Equal(t, ExpectedVersion(), upstreamVersion(t, x))
	assert.Equal(t, ForkExpectedVersion(), mustForkLedger(t, x))
}
