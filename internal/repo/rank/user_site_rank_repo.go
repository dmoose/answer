package rank

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/multisite"
	rankservice "github.com/apache/answer/internal/service/rank"
	"xorm.io/xorm"
)

type UserSiteRankRepo struct {
	data *data.Data
}

func NewUserSiteRankRepo(data *data.Data) rankservice.SiteRankRepo {
	return &UserSiteRankRepo{data: data}
}

func (r *UserSiteRankRepo) GetUserSiteRank(ctx context.Context, userID, siteID string) (int, error) {
	usr := &entity.UserSiteRank{}
	exist, err := r.data.DB.Context(ctx).
		Where("user_id = ? AND site_id = ?", userID, siteID).Get(usr)
	if err != nil {
		return 1, err
	}
	if !exist {
		return 1, nil
	}
	return usr.Rank, nil
}

func (r *UserSiteRankRepo) ChangeSiteRank(ctx context.Context, session *xorm.Session,
	userID string, deltaRank int) error {
	siteID := multisite.SiteIDFromContext(ctx)
	if siteID == "" || deltaRank == 0 {
		return nil
	}

	existing := &entity.UserSiteRank{}
	exist, err := session.Where("user_id = ? AND site_id = ?", userID, siteID).Get(existing)
	if err != nil {
		return err
	}

	if exist {
		newRank := existing.Rank + deltaRank
		if newRank < 1 {
			newRank = 1
		}
		_, err = session.Where("user_id = ? AND site_id = ?", userID, siteID).
			Cols("rank").Update(&entity.UserSiteRank{Rank: newRank})
		return err
	}

	rank := 1 + deltaRank
	if rank < 1 {
		rank = 1
	}
	_, err = session.Insert(&entity.UserSiteRank{
		UserID: userID,
		SiteID: siteID,
		Rank:   rank,
	})
	return err
}

func (r *UserSiteRankRepo) GetUserAllSiteRanks(ctx context.Context, userID string) ([]entity.UserSiteRank, error) {
	var ranks []entity.UserSiteRank
	err := r.data.DB.Context(ctx).Where("user_id = ?", userID).Find(&ranks)
	return ranks, err
}
