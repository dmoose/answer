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

package role

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
)

type mockSiteRoleRepo struct {
	roles map[string]map[string]int // userID -> siteID -> roleID
}

func (m *mockSiteRoleRepo) GetUserSiteRole(_ context.Context, userID, siteID string) (int, bool, error) {
	if siteRoles, ok := m.roles[userID]; ok {
		if r, ok := siteRoles[siteID]; ok {
			return r, true, nil
		}
	}
	return 0, false, nil
}

func (m *mockSiteRoleRepo) SaveUserSiteRole(_ context.Context, userID, siteID string, roleID int) error {
	if m.roles[userID] == nil {
		m.roles[userID] = make(map[string]int)
	}
	m.roles[userID][siteID] = roleID
	return nil
}

func TestGetEffectiveRole_SiteModeratorOverridesUser(t *testing.T) {
	repo := &mockSiteRoleRepo{
		roles: map[string]map[string]int{
			"user-1": {"site-a": RoleModeratorID},
		},
	}
	svc := &UserRoleRelService{siteRoleRepo: repo}

	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-a")
	got := svc.getEffectiveRole(ctx, "user-1", RoleUserID)
	if got != RoleModeratorID {
		t.Errorf("got %d, want %d (site moderator)", got, RoleModeratorID)
	}
}

func TestGetEffectiveRole_GlobalAdminWins(t *testing.T) {
	repo := &mockSiteRoleRepo{
		roles: map[string]map[string]int{
			"user-1": {"site-a": RoleModeratorID},
		},
	}
	svc := &UserRoleRelService{siteRoleRepo: repo}

	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-a")
	got := svc.getEffectiveRole(ctx, "user-1", RoleAdminID)
	if got != RoleAdminID {
		t.Errorf("got %d, want %d (global admin wins)", got, RoleAdminID)
	}
}

func TestGetEffectiveRole_NoSiteRole(t *testing.T) {
	repo := &mockSiteRoleRepo{roles: make(map[string]map[string]int)}
	svc := &UserRoleRelService{siteRoleRepo: repo}

	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-a")
	got := svc.getEffectiveRole(ctx, "user-1", RoleUserID)
	if got != RoleUserID {
		t.Errorf("got %d, want %d (fallback to global)", got, RoleUserID)
	}
}

func TestGetEffectiveRole_NoSiteContext(t *testing.T) {
	repo := &mockSiteRoleRepo{}
	svc := &UserRoleRelService{siteRoleRepo: repo}

	got := svc.getEffectiveRole(context.Background(), "user-1", RoleModeratorID)
	if got != RoleModeratorID {
		t.Errorf("got %d, want %d (no site = global)", got, RoleModeratorID)
	}
}

// A moderator of site A holds no authority on site B: board authority is an
// explicit per-site appointment, resolved against the request's site.
func TestGetEffectiveRole_ModeratorOfADoesNotModerateB(t *testing.T) {
	repo := &mockSiteRoleRepo{
		roles: map[string]map[string]int{
			"user-1": {"site-a": RoleModeratorID},
		},
	}
	svc := &UserRoleRelService{siteRoleRepo: repo}

	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-b")
	got := svc.getEffectiveRole(ctx, "user-1", RoleUserID)
	if got != RoleUserID {
		t.Errorf("got %d, want %d (site-a moderator is a plain user on site-b)", got, RoleUserID)
	}
}

type erroringSiteRoleRepo struct{}

func (erroringSiteRoleRepo) GetUserSiteRole(context.Context, string, string) (int, bool, error) {
	return RoleAdminID, true, context.DeadlineExceeded
}
func (erroringSiteRoleRepo) SaveUserSiteRole(context.Context, string, string, int) error {
	return nil
}

// A failed site-role read must never grant the (possibly elevated) site
// role — it falls back to the global baseline.
func TestGetEffectiveRole_ErrorFallsBackToGlobalNeverElevates(t *testing.T) {
	svc := &UserRoleRelService{siteRoleRepo: erroringSiteRoleRepo{}}

	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-a")
	got := svc.getEffectiveRole(ctx, "user-1", RoleUserID)
	if got != RoleUserID {
		t.Errorf("got %d, want %d (error must not elevate)", got, RoleUserID)
	}
}

type stubGlobalRoleRepo struct {
	roleID int
	exist  bool
}

func (s stubGlobalRoleRepo) SaveUserRoleRel(context.Context, string, int) error { return nil }
func (s stubGlobalRoleRepo) GetUserRoleRelList(context.Context, []string) ([]*entity.UserRoleRel, error) {
	return nil, nil
}
func (s stubGlobalRoleRepo) GetUserRoleRelListByRoleID(context.Context, []int) ([]*entity.UserRoleRel, error) {
	return nil, nil
}
func (s stubGlobalRoleRepo) GetUserRoleRel(context.Context, string) (*entity.UserRoleRel, bool, error) {
	return &entity.UserRoleRel{RoleID: s.roleID}, s.exist, nil
}

// GetUserGlobalRole must ignore per-site escalation entirely: the admin API
// and admin token cache are network-wide authority.
func TestGlobalRoleIgnoresSiteEscalation(t *testing.T) {
	siteRepo := &mockSiteRoleRepo{
		roles: map[string]map[string]int{
			"user-1": {"site-a": RoleAdminID},
		},
	}
	svc := &UserRoleRelService{
		userRoleRelRepo: stubGlobalRoleRepo{roleID: RoleUserID, exist: true},
		siteRoleRepo:    siteRepo,
	}

	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-a")

	effective, err := svc.GetUserRole(ctx, "user-1")
	if err != nil || effective != RoleAdminID {
		t.Errorf("GetUserRole = %d, %v; want site-escalated %d", effective, err, RoleAdminID)
	}

	global, err := svc.GetUserGlobalRole(ctx, "user-1")
	if err != nil || global != RoleUserID {
		t.Errorf("GetUserGlobalRole = %d, %v; want un-escalated %d", global, err, RoleUserID)
	}
}
