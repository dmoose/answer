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
	"fmt"
	"strings"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// siteScopedTables are the content tables that carry a site_id column,
// paired with their entities so a missing table can be created via Sync
// (dialect-correct) while an existing table only gets additive, explicit
// DDL — xorm 1.3.2's Sync emits MySQL-syntax MODIFY COLUMN on any perceived
// column drift, which is a hard error on SQLite/Postgres, so Sync must never
// run against a table that already exists.
type siteScopedTable struct {
	name string
	bean any
}

func siteScopedTables() []siteScopedTable {
	return []siteScopedTable{
		{"question", new(entity.Question)}, {"answer", new(entity.Answer)},
		{"comment", new(entity.Comment)},
		{"tag", new(entity.Tag)}, {"tag_rel", new(entity.TagRel)},
		{"revision", new(entity.Revision)}, {"activity", new(entity.Activity)},
		{"report", new(entity.Report)}, {"meta", new(entity.Meta)},
		{"review", new(entity.Review)},
		{"notification", new(entity.Notification)},
		{"collection", new(entity.Collection)}, {"collection_group", new(entity.CollectionGroup)},
		{"config", new(entity.Config)}, {"site_info", new(entity.SiteInfo)},
		{"badge_award", new(entity.BadgeAward)}, {"file_record", new(entity.FileRecord)},
		{"plugin_config", new(entity.PluginConfig)}, {"plugin_kv_storage", new(entity.PluginKVStorage)},
		{"question_link", new(entity.QuestionLink)},
	}
}

// compositeUniques replace the legacy single-column unique indexes with
// per-site ones. Index names match what xorm Sync creates on a fresh install
// (UQE_<table>_<group>) so fresh and upgraded schemas converge.
var compositeUniques = []struct {
	table, index string
	cols         []string
}{
	{"tag", "UQE_tag_uq_tag_slug_site", []string{"slug_name", "site_id"}},
	{"config", "UQE_config_uq_config_key_site", []string{"key", "site_id"}},
	{"plugin_config", "UQE_plugin_config_uq_plugin_cfg_site", []string{"plugin_slug_name", "site_id"}},
}

// ensureSiteScopedSchema brings every site-scoped table to the multisite
// shape: site_id column, its index, and the composite unique indexes.
// Idempotent — every step checks before it changes.
func ensureSiteScopedSchema(ctx context.Context, x *xorm.Engine) error {
	quote := func(name string) string {
		return x.Dialect().Quoter().Quote(name)
	}
	for _, tbl := range siteScopedTables() {
		exists, err := x.Context(ctx).IsTableExist(tbl.name)
		if err != nil {
			return fmt.Errorf("check table %s: %w", tbl.name, err)
		}
		if !exists {
			// Sync is safe (and dialect-correct) only for table creation.
			if err := x.Context(ctx).Sync(tbl.bean); err != nil {
				return fmt.Errorf("create table %s: %w", tbl.name, err)
			}
			continue
		}
		hasCol, err := columnExists(ctx, x, tbl.name, "site_id")
		if err != nil {
			return fmt.Errorf("check site_id on %s: %w", tbl.name, err)
		}
		if !hasCol {
			_, err = x.Context(ctx).Exec(fmt.Sprintf(
				"ALTER TABLE %s ADD COLUMN %s VARCHAR(36) NOT NULL DEFAULT ''",
				quote(tbl.name), quote("site_id")))
			if err != nil {
				return fmt.Errorf("add site_id to %s: %w", tbl.name, err)
			}
		}
		idxName := "IDX_" + tbl.name + "_site_id"
		hasIdx, err := indexExists(ctx, x, tbl.name, idxName)
		if err != nil {
			return fmt.Errorf("check index %s: %w", idxName, err)
		}
		if !hasIdx {
			_, err = x.Context(ctx).Exec(fmt.Sprintf(
				"CREATE INDEX %s ON %s (%s)",
				quote(idxName), quote(tbl.name), quote("site_id")))
			if err != nil {
				return fmt.Errorf("create index %s: %w", idxName, err)
			}
		}
	}

	for _, cu := range compositeUniques {
		hasIdx, err := indexExists(ctx, x, cu.table, cu.index)
		if err != nil {
			return fmt.Errorf("check index %s: %w", cu.index, err)
		}
		if hasIdx {
			continue
		}
		cols := make([]string, len(cu.cols))
		for i, c := range cu.cols {
			cols[i] = quote(c)
		}
		_, err = x.Context(ctx).Exec(fmt.Sprintf(
			"CREATE UNIQUE INDEX %s ON %s (%s)",
			quote(cu.index), quote(cu.table), strings.Join(cols, ",")))
		if err != nil {
			return fmt.Errorf("create unique index %s: %w", cu.index, err)
		}
	}
	return nil
}

// columnExists reports whether a column is present, per dialect.
func columnExists(ctx context.Context, x *xorm.Engine, table, column string) (bool, error) {
	var n int64
	switch x.Dialect().URI().DBType {
	case schemas.MYSQL:
		_, err := x.Context(ctx).SQL(
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
			table, column).Get(&n)
		return n > 0, err
	case schemas.POSTGRES:
		_, err := x.Context(ctx).SQL(
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`,
			table, column).Get(&n)
		return n > 0, err
	case schemas.SQLITE:
		_, err := x.Context(ctx).SQL(
			`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`,
			table, column).Get(&n)
		return n > 0, err
	}
	return false, fmt.Errorf("unsupported dialect")
}

// backfillTables are the site-scoped tables whose existing rows belong to the
// default site after the upgrade. config and site_info stay site_id=” — the
// empty value is the global-default row the per-site cascade falls back to.
var backfillTables = []string{
	"question", "answer", "comment", "tag", "tag_rel",
	"revision", "activity", "report", "meta", "review",
	"notification", "collection", "collection_group",
	"badge_award", "file_record",
	"plugin_kv_storage", "question_link", "plugin_config",
}

func addMultiSiteSupport(ctx context.Context, x *xorm.Engine) error {
	if err := x.Context(ctx).Sync(
		new(entity.Site),
		new(entity.UserSiteRoleRel),
	); err != nil {
		return fmt.Errorf("create multi-site tables: %w", err)
	}

	// site_id columns, their indexes, and the composite unique indexes. At
	// this point the legacy single-column uniques still guarantee the
	// composites hold.
	if err := ensureSiteScopedSchema(ctx, x); err != nil {
		return err
	}

	if err := ensureDefaultSite(ctx, x); err != nil {
		return err
	}

	for _, table := range backfillTables {
		_, err := x.Context(ctx).Exec(
			fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ''", x.Quote(table), x.Quote("site_id"), x.Quote("site_id")),
			constant.DefaultSiteID)
		if err != nil {
			return fmt.Errorf("backfill site_id on %s: %w", table, err)
		}
	}

	if err := backfillDefaultSiteRoles(ctx, x); err != nil {
		return fmt.Errorf("backfill user_site_role_rel: %w", err)
	}

	// Drop the now-redundant single-column unique indexes; the composite
	// (column, site_id) uniques created above replace them.
	uniqueFixups := []struct{ table, oldIdx string }{
		{"tag", "UQE_tag_slug_name"},
		{"config", "UQE_config_key"},
		{"plugin_config", "UQE_plugin_config_plugin_slug_name"},
	}
	if err := dropLegacyUniqueIndexes(ctx, x, uniqueFixups); err != nil {
		return fmt.Errorf("drop legacy unique indexes: %w", err)
	}

	return nil
}

// ensureDefaultSite inserts the default site row if it is not present yet.
func ensureDefaultSite(ctx context.Context, x *xorm.Engine) error {
	exist, err := x.Context(ctx).ID(constant.DefaultSiteID).Exist(&entity.Site{})
	if err != nil {
		return fmt.Errorf("check default site: %w", err)
	}
	if exist {
		return nil
	}
	_, err = x.Context(ctx).Insert(&entity.Site{
		ID:     constant.DefaultSiteID,
		Name:   "Default",
		Slug:   "default",
		Status: entity.SiteStatusActive,
	})
	if err != nil {
		return fmt.Errorf("insert default site: %w", err)
	}
	return nil
}

// backfillDefaultSiteRoles copies each user's global role onto the default
// site. user_role_rel may hold several roles per user while
// user_site_role_rel is unique on (user_id, site_id), so the most privileged
// role wins. Runs in Go rather than INSERT…SELECT: portable across dialects
// (no NOW()) and idempotent (skips users that already have a row).
func backfillDefaultSiteRoles(ctx context.Context, x *xorm.Engine) error {
	var rels []entity.UserRoleRel
	if err := x.Context(ctx).Find(&rels); err != nil {
		return fmt.Errorf("read user_role_rel: %w", err)
	}
	// Role IDs per init_data: 1 = User, 2 = Admin, 3 = Moderator.
	privilege := map[int]int{1: 1, 3: 2, 2: 3}
	best := make(map[string]int, len(rels))
	for _, rel := range rels {
		if cur, ok := best[rel.UserID]; !ok || privilege[rel.RoleID] > privilege[cur] {
			best[rel.UserID] = rel.RoleID
		}
	}
	for userID, roleID := range best {
		exist, err := x.Context(ctx).
			Where("user_id = ? AND site_id = ?", userID, constant.DefaultSiteID).
			Exist(&entity.UserSiteRoleRel{})
		if err != nil {
			return fmt.Errorf("check site role for user %s: %w", userID, err)
		}
		if exist {
			continue
		}
		_, err = x.Context(ctx).Insert(&entity.UserSiteRoleRel{
			UserID: userID,
			SiteID: constant.DefaultSiteID,
			RoleID: roleID,
		})
		if err != nil {
			return fmt.Errorf("insert site role for user %s: %w", userID, err)
		}
	}
	return nil
}

func dropLegacyUniqueIndexes(ctx context.Context, x *xorm.Engine, fixups []struct{ table, oldIdx string }) error {
	dbType := x.Dialect().URI().DBType
	for _, f := range fixups {
		exists, err := indexExists(ctx, x, f.table, f.oldIdx)
		if err != nil {
			return fmt.Errorf("check index %s on %s: %w", f.oldIdx, f.table, err)
		}
		if !exists {
			continue
		}
		var stmt string
		switch dbType {
		case schemas.MYSQL:
			stmt = fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", f.table, f.oldIdx)
		case schemas.POSTGRES:
			stmt = fmt.Sprintf(`DROP INDEX IF EXISTS "%s"`, f.oldIdx)
		case schemas.SQLITE:
			stmt = fmt.Sprintf("DROP INDEX IF EXISTS `%s`", f.oldIdx)
		default:
			return fmt.Errorf("unsupported dialect %q", dbType)
		}
		if _, err := x.Context(ctx).Exec(stmt); err != nil {
			return fmt.Errorf("drop %s on %s: %w", f.oldIdx, f.table, err)
		}
	}
	return nil
}

func indexExists(ctx context.Context, x *xorm.Engine, table, idx string) (bool, error) {
	switch x.Dialect().URI().DBType {
	case schemas.MYSQL:
		var n int64
		_, err := x.Context(ctx).SQL(
			`SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`,
			table, idx).Get(&n)
		return n > 0, err
	case schemas.POSTGRES:
		var n int64
		_, err := x.Context(ctx).SQL(
			`SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1 AND indexname = $2`,
			table, idx).Get(&n)
		return n > 0, err
	case schemas.SQLITE:
		var n int64
		_, err := x.Context(ctx).SQL(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND name = ?`,
			table, idx).Get(&n)
		return n > 0, err
	}
	return false, fmt.Errorf("unsupported dialect")
}
