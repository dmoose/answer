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

// Content is site-scoped: a row written on sub-site A must be invisible to
// scoped reads on sub-site B and visible on A. Identity is global: a user's
// notifications, badges and bookmarks span all sub-sites. These tests pin
// both axes at the repo layer.

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/repo/answer"
	"github.com/apache/answer/internal/repo/badge_award"
	"github.com/apache/answer/internal/repo/collection"
	"github.com/apache/answer/internal/repo/comment"
	"github.com/apache/answer/internal/repo/notification"
	"github.com/apache/answer/internal/repo/question"
	"github.com/apache/answer/internal/repo/unique"
	"github.com/apache/answer/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_questionRepo_SiteScoped(t *testing.T) {
	uniqueIDRepo := unique.NewUniqueIDRepo(testDataSource)
	repo := question.NewQuestionRepo(testDataSource, uniqueIDRepo)

	qa := &entity.Question{
		UserID: "1", Title: "scope-a", OriginalText: "t", ParsedText: "t",
		Status: entity.QuestionStatusAvailable, RevisionID: "0",
	}
	require.NoError(t, repo.AddQuestion(siteCtx(siteA), qa))
	t.Cleanup(func() { _ = repo.RemoveQuestion(context.TODO(), qa.ID) })

	// Write stamped with the site from ctx.
	row := &entity.Question{}
	exist, err := testDataSource.DB.ID(qa.ID).Get(row)
	require.NoError(t, err)
	require.True(t, exist)
	assert.Equal(t, siteA, row.SiteID, "insert must stamp the writing site")

	// Scoped read on the owning site finds it; on another site it does not.
	_, existA, err := repo.GetQuestion(siteCtx(siteA), qa.ID)
	require.NoError(t, err)
	assert.True(t, existA, "own site must see its question")
	_, existB, err := repo.GetQuestion(siteCtx(siteB), qa.ID)
	require.NoError(t, err)
	assert.False(t, existB, "another site must not see it")
}

func Test_answerRepo_SiteScoped(t *testing.T) {
	uniqueIDRepo := unique.NewUniqueIDRepo(testDataSource)
	qRepo := question.NewQuestionRepo(testDataSource, uniqueIDRepo)
	// rank/activity deps are only used by vote-related methods, not
	// AddAnswer/GetAnswer.
	aRepo := answer.NewAnswerRepo(testDataSource, uniqueIDRepo, nil, nil)

	qa := &entity.Question{
		UserID: "1", Title: "scope-ans", OriginalText: "t", ParsedText: "t",
		Status: entity.QuestionStatusAvailable, RevisionID: "0",
	}
	require.NoError(t, qRepo.AddQuestion(siteCtx(siteA), qa))
	t.Cleanup(func() { _ = qRepo.RemoveQuestion(context.TODO(), qa.ID) })

	ans := &entity.Answer{
		QuestionID: qa.ID, UserID: "1", OriginalText: "a", ParsedText: "a",
		Status: entity.AnswerStatusAvailable, RevisionID: "0",
	}
	require.NoError(t, aRepo.AddAnswer(siteCtx(siteA), ans))

	_, existA, err := aRepo.GetAnswer(siteCtx(siteA), ans.ID)
	require.NoError(t, err)
	assert.True(t, existA)
	_, existB, err := aRepo.GetAnswer(siteCtx(siteB), ans.ID)
	require.NoError(t, err)
	assert.False(t, existB, "answers must not cross sites")
}

func Test_commentRepo_SiteScoped(t *testing.T) {
	uniqueIDRepo := unique.NewUniqueIDRepo(testDataSource)
	repo := comment.NewCommentRepo(testDataSource, uniqueIDRepo)

	c := &entity.Comment{
		UserID: "1", ObjectID: "10010000000000001", QuestionID: "10010000000000001",
		OriginalText: "c", ParsedText: "c", Status: entity.CommentStatusAvailable,
	}
	require.NoError(t, repo.AddComment(siteCtx(siteA), c))

	_, existA, err := repo.GetComment(siteCtx(siteA), c.ID)
	require.NoError(t, err)
	assert.True(t, existA)
	_, existB, err := repo.GetComment(siteCtx(siteB), c.ID)
	require.NoError(t, err)
	assert.False(t, existB, "comments must not cross sites")
}

// Notifications are identity-global: one inbox per person, regardless of
// which sub-site produced the notification or which one the user is viewing.
func Test_notificationRepo_IdentityGlobal(t *testing.T) {
	repo := notification.NewNotificationRepo(testDataSource)

	userID := "notif-global-user"
	n := &entity.Notification{
		UserID: userID, ObjectID: "10010000000000002", Content: "{}",
		Type: schema.NotificationTypeInbox, MsgType: 0,
		IsRead: schema.NotificationNotRead, Status: schema.NotificationStatusNormal,
	}
	require.NoError(t, repo.AddNotification(siteCtx(siteA), n))

	// Visible from another sub-site.
	got, _, err := repo.GetNotificationPage(siteCtx(siteB), &schema.NotificationSearch{
		UserID: userID, Type: schema.NotificationTypeInbox, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, got, 1, "inbox must be global across sub-sites")
	assert.Equal(t, n.ID, got[0].ID)

	// And counted globally.
	count, err := repo.CountNotificationByUser(siteCtx(siteB), &entity.Notification{
		UserID: userID, Type: schema.NotificationTypeInbox,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
}

// Badge awards are identity-global: earned once, visible everywhere.
func Test_badgeAwardRepo_IdentityGlobal(t *testing.T) {
	uniqueIDRepo := unique.NewUniqueIDRepo(testDataSource)
	repo := badge_award.NewBadgeAwardRepo(testDataSource, uniqueIDRepo)

	userID := "badge-global-user"
	// AwardBadgeForUser bumps the badge's award_count, so the badge row
	// must exist first.
	_, err := testDataSource.DB.Insert(&entity.Badge{
		ID: "520000000000001", Name: "test-badge", Icon: "star", AwardCount: 0,
		Description: "t", Status: entity.BadgeStatusActive, Level: 1,
	})
	require.NoError(t, err)
	award := &entity.BadgeAward{
		UserID: userID, BadgeID: "520000000000001", AwardKey: "key-1",
		IsBadgeDeleted: entity.IsBadgeNotDeleted,
	}
	require.NoError(t, repo.AwardBadgeForUser(siteCtx(siteA), award))

	count := repo.CountByUserIdAndBadgeId(siteCtx(siteB), userID, "520000000000001")
	assert.EqualValues(t, 1, count, "badge awards must be visible from any sub-site")
}

// Collections (bookmarks) are identity-global: one list per person.
func Test_collectionRepo_IdentityGlobal(t *testing.T) {
	uniqueIDRepo := unique.NewUniqueIDRepo(testDataSource)
	repo := collection.NewCollectionRepo(testDataSource, uniqueIDRepo)

	userID := "collect-global-user"
	col := &entity.Collection{
		UserID: userID, ObjectID: "10010000000000003", UserCollectionGroupID: "0",
	}
	require.NoError(t, repo.AddCollection(siteCtx(siteA), col))

	_, exist, err := repo.GetOneByObjectIDAndUser(siteCtx(siteB), userID, "10010000000000003")
	require.NoError(t, err)
	assert.True(t, exist, "bookmarks must be visible from any sub-site")
}

// REP-3 guard: activity rows retain their originating site_id, so per-site
// reputation attribution (Option C) stays a query away — SUM(rank) GROUP BY
// site_id — with no schema change.
func Test_activityAttribution_PerSiteBreakdownQueryable(t *testing.T) {
	db := testDataSource.DB
	userID := "rep-attrib-user"
	rows := []entity.Activity{
		{UserID: userID, ObjectID: "1", OriginalObjectID: "1", ActivityType: 1, Rank: 10, HasRank: 1, SiteID: siteA},
		{UserID: userID, ObjectID: "2", OriginalObjectID: "2", ActivityType: 1, Rank: 5, HasRank: 1, SiteID: siteA},
		{UserID: userID, ObjectID: "3", OriginalObjectID: "3", ActivityType: 1, Rank: 7, HasRank: 1, SiteID: siteB},
	}
	for i := range rows {
		_, err := db.Insert(&rows[i])
		require.NoError(t, err)
	}

	type breakdown struct {
		SiteID string `xorm:"'site_id'"`
		Sum    int64  `xorm:"'sum_rank'"`
	}
	var got []breakdown
	err := db.SQL(
		"SELECT site_id, SUM(`rank`) AS sum_rank FROM activity WHERE user_id = ? AND cancelled = 0 GROUP BY site_id ORDER BY site_id",
		userID).Find(&got)
	require.NoError(t, err)
	require.Len(t, got, 2)
	sums := map[string]int64{}
	for _, g := range got {
		sums[g.SiteID] = g.Sum
	}
	assert.EqualValues(t, 15, sums[siteA])
	assert.EqualValues(t, 7, sums[siteB])
}
