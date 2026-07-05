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

	"xorm.io/xorm"
)

// repairMultisiteSchema fixes databases that ran the original v33, whose
// dialect-specific steps silently failed on SQLite (errors were demoted to
// warnings while the schema version still advanced):
//
//   - The composite unique indexes (tag/config/plugin_config + site_id) were
//     never created on upgraded installs — the old single-column uniques were
//     dropped and nothing replaced them, so per-site uniqueness was
//     unenforced and upgraded schemas diverged from fresh ones.
//   - The user_site_role_rel backfill used SQL NOW() (absent on SQLite), so
//     no per-site role rows were created for the default site.
//   - user_site_rank is retired: reputation is global (one rank per person
//     gates privileges on every sub-site); per-site attribution for display
//     stays derivable from activity.site_id. The table's data was never
//     correct on SQLite installs anyway (same NOW() failure).
//
// Everything here is idempotent: databases migrated by the repaired v33
// no-op straight through.
func repairMultisiteSchema(ctx context.Context, x *xorm.Engine) error {
	// The composite uniques cannot be created over duplicate rows. The old
	// single-column uniques were dropped by v33, so duplicates could have
	// crept in since. Refuse loudly rather than silently deleting content:
	// a duplicate tag row may be referenced by tag_rel and needs a human.
	dupChecks := []struct{ table, cols string }{
		{"tag", "slug_name, site_id"},
		{"config", "`key`, site_id"},
		{"plugin_config", "plugin_slug_name, site_id"},
	}
	for _, d := range dupChecks {
		type dupRow struct {
			N int64 `xorm:"'n'"`
		}
		var dups []dupRow
		err := x.Context(ctx).SQL(fmt.Sprintf(
			"SELECT COUNT(*) AS n FROM `%s` GROUP BY %s HAVING COUNT(*) > 1", d.table, d.cols)).
			Find(&dups)
		if err != nil {
			return fmt.Errorf("check duplicates on %s: %w", d.table, err)
		}
		if len(dups) > 0 {
			return fmt.Errorf(
				"table %s has %d duplicated (%s) groups; resolve them manually before migrating "+
					"(the composite unique index cannot be created over duplicates)",
				d.table, len(dups), strings.ReplaceAll(d.cols, "`", ""))
		}
	}

	// Converge with fresh installs: creates any missing site_id columns,
	// their indexes, and the composite unique indexes. No-ops where the
	// schema is already correct.
	if err := ensureSiteScopedSchema(ctx, x); err != nil {
		return err
	}

	if err := ensureDefaultSite(ctx, x); err != nil {
		return err
	}

	// Re-run the role backfill that silently no-op'd on SQLite.
	if err := backfillDefaultSiteRoles(ctx, x); err != nil {
		return fmt.Errorf("backfill user_site_role_rel: %w", err)
	}

	// Retire the per-site rank table (reputation is global).
	if _, err := x.Context(ctx).Exec("DROP TABLE IF EXISTS `user_site_rank`"); err != nil {
		return fmt.Errorf("drop user_site_rank: %w", err)
	}

	return nil
}
