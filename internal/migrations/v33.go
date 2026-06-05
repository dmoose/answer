package migrations

import (
	"context"
	"fmt"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
	"github.com/segmentfault/pacman/log"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func addMultiSiteSupport(ctx context.Context, x *xorm.Engine) error {
	if err := x.Context(ctx).Sync(
		new(entity.Site),
		new(entity.UserSiteRank),
		new(entity.UserSiteRoleRel),
	); err != nil {
		return fmt.Errorf("create multi-site tables: %w", err)
	}

	type siteIDColumn struct {
		table string
	}
	tables := []siteIDColumn{
		{"question"}, {"answer"}, {"comment"},
		{"tag"}, {"tag_rel"},
		{"revision"}, {"activity"}, {"report"}, {"meta"}, {"review"},
		{"notification"}, {"collection"}, {"collection_group"},
		{"config"}, {"site_info"},
		{"badge_award"}, {"file_record"},
		{"plugin_config"}, {"plugin_kv_storage"},
		{"question_link"},
	}

	for _, t := range tables {
		_, err := x.Context(ctx).Exec(
			fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `site_id` VARCHAR(36) NOT NULL DEFAULT ''", t.table))
		if err != nil {
			log.Warnf("add site_id to %s (may already exist): %v", t.table, err)
		}
		_, _ = x.Context(ctx).Exec(
			fmt.Sprintf("CREATE INDEX `idx_%s_site_id` ON `%s` (`site_id`)", t.table, t.table))
	}

	defaultSite := &entity.Site{
		ID:     constant.DefaultSiteID,
		Name:   "Default",
		Slug:   "default",
		Status: entity.SiteStatusActive,
	}
	_, err := x.Context(ctx).Insert(defaultSite)
	if err != nil {
		return fmt.Errorf("insert default site: %w", err)
	}

	// config and site_info rows stay as global defaults (site_id = '')
	// so sites without overrides inherit them
	noBackfill := map[string]bool{"config": true, "site_info": true}
	for _, t := range tables {
		if noBackfill[t.table] {
			continue
		}
		_, err := x.Context(ctx).Exec(
			fmt.Sprintf("UPDATE `%s` SET `site_id` = ? WHERE `site_id` = ''", t.table),
			defaultSite.ID)
		if err != nil {
			log.Warnf("backfill site_id on %s: %v", t.table, err)
		}
	}

	_, err = x.Context(ctx).Exec(`
		INSERT INTO user_site_rank (user_id, site_id, ` + "`rank`" + `, created_at, updated_at)
		SELECT id, ?, ` + "`rank`" + `, NOW(), NOW() FROM ` + "`user`" + ` WHERE ` + "`rank`" + ` > 0`,
		defaultSite.ID)
	if err != nil {
		log.Warnf("backfill user_site_rank: %v", err)
	}

	_, err = x.Context(ctx).Exec(`
		INSERT INTO user_site_role_rel (user_id, site_id, role_id, created_at, updated_at)
		SELECT user_id, ?, role_id, NOW(), NOW() FROM user_role_rel`,
		defaultSite.ID)
	if err != nil {
		log.Warnf("backfill user_site_role_rel: %v", err)
	}

	// Replace single-column unique indexes with composite (column, site_id).
	// The composite indexes are created by xorm Sync() from the entity
	// definitions; here we just drop the leftover single-column ones.
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
