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

	"github.com/apache/answer/internal/entity"
	"github.com/segmentfault/pacman/cache"
	"xorm.io/xorm"
)

// Fork migrations keep their own ledger, separate from upstream's.
//
// Upstream tracks migrations by list index: the `version` row holds how many
// entries of `migrations` have run. Fork entries appended to that list only
// stay correct while they remain the last entries, so every upstream merge
// that adds a migration would silently reorder or skip them. The fork list
// below is tracked in its own table instead and runs after the upstream
// list on every upgrade. Upstream's `migrations` slice stays byte-for-byte
// upstream; the only touch points are the two calls in Migrate and the row
// insert in init.
//
// The ledger is a separate table rather than a second row in `version`:
// upstream advances its row with an unconditioned Update that rewrites
// every row in the table.
//
// Fork migrations must be idempotent: on upgrade they run after whatever
// upstream migrations were pending, against a schema upstream may have
// touched in between.
type forkVersion struct {
	ID            int   `xorm:"not null pk autoincr INT(11) id"`
	VersionNumber int64 `xorm:"not null default 0 INT(11) version_number"`
}

func (forkVersion) TableName() string { return "fork_version" }

const forkVersionID = 1

// legacyUpstreamCount and legacySharedVersion describe the ledger layout
// before the split: upstream had 33 migrations at the fork point and the
// three fork migrations sat at indexes 33..35, so a fully migrated database
// read 36 in `version`.
const (
	legacyUpstreamCount   = 33
	legacySharedVersion   = legacyUpstreamCount + 3
	legacyForkMarkerTable = "site"
)

var forkMigrations = []Migration{
	NewMigration("fork-001", "add multi-site support", addMultiSiteSupport, true),
	NewMigration("fork-002", "add network directory (profile, projects, tags)", addNetworkDirectory, false),
	NewMigration("fork-003", "repair multisite schema (composite uniques, role backfill, retire user_site_rank)", repairMultisiteSchema, false),
}

// ForkExpectedVersion returns the fork ledger value of a fully migrated database.
func ForkExpectedVersion() int64 {
	return int64(len(forkMigrations))
}

// getForkDBVersion reads the fork ledger, creating it at 0 when absent.
func getForkDBVersion(ctx context.Context, x *xorm.Engine) (int64, error) {
	if err := x.Sync(new(forkVersion)); err != nil {
		return -1, fmt.Errorf("sync fork_version failed: %v", err)
	}
	row := &forkVersion{ID: forkVersionID}
	has, err := x.Context(ctx).Get(row)
	if err != nil {
		return -1, fmt.Errorf("get fork version failed: %v", err)
	}
	if !has {
		if _, err := x.Context(ctx).InsertOne(&forkVersion{ID: forkVersionID, VersionNumber: 0}); err != nil {
			return -1, fmt.Errorf("insert fork version failed: %v", err)
		}
		return 0, nil
	}
	return row.VersionNumber, nil
}

func setForkDBVersion(ctx context.Context, x *xorm.Engine, v int64) error {
	_, err := x.Context(ctx).ID(forkVersionID).Cols("version_number").Update(&forkVersion{VersionNumber: v})
	return err
}

// bootstrapForkLedger converts a database migrated under the shared ledger
// to the split layout. Runs before the upstream loop so `version` is back
// to counting upstream entries only. It fires exactly once: a database with
// a fork ledger row is left alone.
//
// Detection: no fork row, the marker table from the first fork migration
// exists, and `version` reads the legacy shared value. A vanilla database
// (no marker table) simply gets a fork row at 0 and takes the normal
// upgrade path through all fork migrations.
func bootstrapForkLedger(ctx context.Context, x *xorm.Engine) error {
	current, err := GetCurrentDBVersion(x)
	if err != nil {
		return err
	}
	if err := x.Sync(new(forkVersion)); err != nil {
		return fmt.Errorf("sync fork_version failed: %v", err)
	}
	has, err := x.Context(ctx).Get(&forkVersion{ID: forkVersionID})
	if err != nil {
		return fmt.Errorf("get fork version failed: %v", err)
	}
	if has {
		return nil
	}
	markerExists, err := x.Context(ctx).IsTableExist(legacyForkMarkerTable)
	if err != nil {
		return fmt.Errorf("check %s table failed: %v", legacyForkMarkerTable, err)
	}
	forkVersionValue := int64(0)
	if markerExists && current == legacySharedVersion {
		forkVersionValue = ForkExpectedVersion()
		fmt.Printf("[migrate] fork ledger split: upstream version %d -> %d, fork version -> %d\n",
			current, legacyUpstreamCount, forkVersionValue)
		if _, err := x.Context(ctx).ID(1).Cols("version_number").
			Update(&entity.Version{VersionNumber: legacyUpstreamCount}); err != nil {
			return fmt.Errorf("reset upstream version failed: %v", err)
		}
	} else if markerExists {
		return fmt.Errorf("fork ledger split: %s table exists but version reads %d, expected %d; "+
			"inspect the version table before upgrading", legacyForkMarkerTable, current, legacySharedVersion)
	}
	if _, err := x.Context(ctx).InsertOne(&forkVersion{ID: forkVersionID, VersionNumber: forkVersionValue}); err != nil {
		return fmt.Errorf("insert fork version failed: %v", err)
	}
	return nil
}

// migrateFork runs pending fork migrations. Called after the upstream loop.
func migrateFork(ctx context.Context, x *xorm.Engine, c cache.Cache) error {
	current, err := getForkDBVersion(ctx, x)
	if err != nil {
		return err
	}
	expected := ForkExpectedVersion()
	for current < expected {
		m := forkMigrations[current]
		fmt.Printf("[migrate] fork db version is %d, try to migrate %s: %s\n", current, m.Version(), m.Description())
		if err := m.Migrate(ctx, x); err != nil {
			fmt.Printf("[migrate] fork migration %s failed: %s\n", m.Version(), err.Error())
			return err
		}
		if m.ShouldCleanCache() && c != nil {
			if err := c.Flush(ctx); err != nil {
				fmt.Printf("[migrate] flush cache failed: %s\n", err.Error())
			}
		}
		if err := setForkDBVersion(ctx, x, current+1); err != nil {
			fmt.Printf("[migrate] fork migration %s, update failed: %s\n", m.Version(), err.Error())
			return err
		}
		fmt.Printf("[migrate] fork migration %s success\n", m.Version())
		current++
	}
	return nil
}
