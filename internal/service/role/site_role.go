package role

import (
	"context"

	"github.com/apache/answer/internal/multisite"
	"github.com/segmentfault/pacman/log"
)

type SiteRoleRepo interface {
	GetUserSiteRole(ctx context.Context, userID, siteID string) (roleID int, exist bool, err error)
	SaveUserSiteRole(ctx context.Context, userID, siteID string, roleID int) error
}

func (us *UserRoleRelService) getEffectiveRole(ctx context.Context, userID string, globalRoleID int) int {
	siteID := multisite.SiteIDFromContext(ctx)
	if siteID == "" || us.siteRoleRepo == nil {
		return globalRoleID
	}
	siteRoleID, exist, err := us.siteRoleRepo.GetUserSiteRole(ctx, userID, siteID)
	if err != nil {
		log.Errorf("get site role for user %s site %s: %v", userID, siteID, err)
		return globalRoleID
	}
	if !exist {
		return globalRoleID
	}
	if isMorePrivileged(siteRoleID, globalRoleID) {
		return siteRoleID
	}
	return globalRoleID
}

func isMorePrivileged(a, b int) bool {
	return privilegeLevel(a) > privilegeLevel(b)
}

func privilegeLevel(roleID int) int {
	switch roleID {
	case RoleAdminID:
		return 3
	case RoleModeratorID:
		return 2
	case RoleUserID:
		return 1
	default:
		return 0
	}
}
