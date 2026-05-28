//go:build multisite

package role

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/base/constant"
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
