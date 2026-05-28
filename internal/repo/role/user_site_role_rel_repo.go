package role

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/entity"
	roleservice "github.com/apache/answer/internal/service/role"
)

type UserSiteRoleRelRepo struct {
	data *data.Data
}

func NewUserSiteRoleRelRepo(data *data.Data) roleservice.SiteRoleRepo {
	return &UserSiteRoleRelRepo{data: data}
}

func (r *UserSiteRoleRelRepo) GetUserSiteRole(ctx context.Context, userID, siteID string) (int, bool, error) {
	rel := &entity.UserSiteRoleRel{}
	exist, err := r.data.DB.Context(ctx).
		Where("user_id = ? AND site_id = ?", userID, siteID).Get(rel)
	if err != nil {
		return 0, false, err
	}
	if !exist {
		return 0, false, nil
	}
	return rel.RoleID, true, nil
}

func (r *UserSiteRoleRelRepo) SaveUserSiteRole(ctx context.Context, userID, siteID string, roleID int) error {
	existing := &entity.UserSiteRoleRel{}
	exist, err := r.data.DB.Context(ctx).
		Where("user_id = ? AND site_id = ?", userID, siteID).Get(existing)
	if err != nil {
		return err
	}
	if exist {
		_, err = r.data.DB.Context(ctx).
			Where("user_id = ? AND site_id = ?", userID, siteID).
			Cols("role_id").Update(&entity.UserSiteRoleRel{RoleID: roleID})
		return err
	}
	_, err = r.data.DB.Context(ctx).Insert(&entity.UserSiteRoleRel{
		UserID: userID,
		SiteID: siteID,
		RoleID: roleID,
	})
	return err
}
