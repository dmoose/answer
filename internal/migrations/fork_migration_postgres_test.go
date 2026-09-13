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
	"os"
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// Runs only when ANSWER_TEST_POSTGRES_DSN names a scratch Postgres database
// (e.g. host=127.0.0.1 port=5432 user=answer password=answer dbname=answer
// sslmode=disable). The database is wiped: every table is dropped first.
func newPostgresMigrationEngine(t *testing.T) *xorm.Engine {
	t.Helper()
	dsn := os.Getenv("ANSWER_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ANSWER_TEST_POSTGRES_DSN not set")
	}
	x, err := data.NewDB(false, &data.Database{Driver: string(schemas.POSTGRES), Connection: dsn})
	require.NoError(t, err)
	t.Cleanup(func() { _ = x.Close() })
	_, err = x.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public")
	require.NoError(t, err)
	return x
}

func pgTableExists(t *testing.T, x *xorm.Engine, table string) bool {
	t.Helper()
	var n int64
	_, err := x.SQL("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1", table).Get(&n)
	require.NoError(t, err)
	return n > 0
}

func pgUniqueIndexes(t *testing.T, x *xorm.Engine, table string) map[string]bool {
	t.Helper()
	type row struct {
		Name string `xorm:"'indexname'"`
	}
	var rows []row
	require.NoError(t, x.SQL("SELECT indexname FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1 AND indexdef LIKE 'CREATE UNIQUE%'", table).Find(&rows))
	names := map[string]bool{}
	for _, r := range rows {
		names[r.Name] = true
	}
	return names
}

// TestMultisiteMigrationOnPostgres is the Postgres twin of the SQLite
// upgrade test: pre-multisite schema → all fork migrations → repair again.
func TestMultisiteMigrationOnPostgres(t *testing.T) {
	x := newPostgresMigrationEngine(t)
	seedPreV33(t, x)
	runMultisiteMigrations(t, x)
	ctx := context.Background()

	assert.False(t, pgUniqueIndexes(t, x, "tag")["UQE_tag_slug_name"], "legacy tag unique must be dropped")
	assert.True(t, pgUniqueIndexes(t, x, "tag")["UQE_tag_uq_tag_slug_site"])
	assert.True(t, pgUniqueIndexes(t, x, "config")["UQE_config_uq_config_key_site"])
	assert.True(t, pgUniqueIndexes(t, x, "plugin_config")["UQE_plugin_config_uq_plugin_cfg_site"])

	type siteIDRow struct {
		SiteID string `xorm:"'site_id'"`
	}
	var tagRow siteIDRow
	_, err := x.SQL("SELECT site_id FROM tag WHERE id = '10'").Get(&tagRow)
	require.NoError(t, err)
	assert.Equal(t, constant.DefaultSiteID, tagRow.SiteID, "content backfilled to the default site")

	var roleRows []entity.UserSiteRoleRel
	require.NoError(t, x.Context(ctx).Find(&roleRows))
	got := map[string]int{}
	for _, r := range roleRows {
		got[r.UserID] = r.RoleID
	}
	assert.Equal(t, map[string]int{"101": 2, "102": 3}, got)

	assert.False(t, pgTableExists(t, x, "user_site_rank"))
	assert.True(t, pgTableExists(t, x, "site"))
	assert.True(t, pgTableExists(t, x, "fork_version"))

	require.NoError(t, repairMultisiteSchema(ctx, x), "repair must be idempotent")
	var again []entity.UserSiteRoleRel
	require.NoError(t, x.Context(ctx).Find(&again))
	assert.Len(t, again, len(roleRows))
}

// TestRepairRefusesDuplicatesOnPostgres pins the duplicate check's SQL on
// the dialect that rejects backtick quoting.
func TestRepairRefusesDuplicatesOnPostgres(t *testing.T) {
	x := newPostgresMigrationEngine(t)
	ctx := context.Background()
	for _, stmt := range []string{
		`CREATE TABLE tag (id BIGINT PRIMARY KEY, slug_name VARCHAR(35) NOT NULL DEFAULT '', site_id VARCHAR(36) NOT NULL DEFAULT '')`,
		`INSERT INTO tag (id, slug_name, site_id) VALUES (1, 'go', 'default')`,
		`INSERT INTO tag (id, slug_name, site_id) VALUES (2, 'go', 'default')`,
	} {
		_, err := x.Context(ctx).Exec(stmt)
		require.NoError(t, err, stmt)
	}
	err := repairMultisiteSchema(ctx, x)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicated")
}
