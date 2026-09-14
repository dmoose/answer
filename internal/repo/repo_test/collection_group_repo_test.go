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
	"testing"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/repo/collection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bookmarks are identity-global: the default group a user gets on site A is
// the same group found from site B, and it carries no site stamp.
func Test_collectionGroupRepo_DefaultGroupIsGlobal(t *testing.T) {
	repo := collection.NewCollectionGroupRepo(testDataSource)
	const userID = "9001"

	created, err := repo.CreateDefaultGroupIfNotExist(siteCtx(siteA), userID)
	require.NoError(t, err)
	assert.Empty(t, created.SiteID, "no site stamp on a global row")

	again, err := repo.CreateDefaultGroupIfNotExist(siteCtx(siteB), userID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, again.ID, "site B finds site A's default group instead of creating another")

	fromB, has, err := repo.GetDefaultID(siteCtx(siteB), userID)
	require.NoError(t, err)
	require.True(t, has)
	assert.Equal(t, created.ID, fromB.ID)

	got, exist, err := repo.GetCollectionGroup(siteCtx(siteB), created.ID)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, userID, got.UserID)

	var n int64
	n, err = testDataSource.DB.Where("user_id = ?", userID).Count(&entity.CollectionGroup{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "exactly one default group per person")
}
