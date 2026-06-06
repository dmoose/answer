package network_profile

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/segmentfault/pacman/errors"
)

type NetworkProfileRepo struct {
	data *data.Data
}

func NewNetworkProfileRepo(data *data.Data) *NetworkProfileRepo {
	return &NetworkProfileRepo{data: data}
}

// Get returns the guild profile for a user, or (nil, false, nil) if not set yet.
func (r *NetworkProfileRepo) Get(ctx context.Context, userID string) (*entity.NetworkProfile, bool, error) {
	p := &entity.NetworkProfile{}
	exist, err := r.data.DB.Context(ctx).Where("user_id = ?", userID).Get(p)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if !exist {
		return nil, false, nil
	}
	return p, true, nil
}

// GetMany fetches profiles for a batch of user IDs. Useful for directory pages
// where N member cards each need a profile.
func (r *NetworkProfileRepo) GetMany(ctx context.Context, userIDs []string) (map[string]*entity.NetworkProfile, error) {
	out := make(map[string]*entity.NetworkProfile, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	var profiles []*entity.NetworkProfile
	err := r.data.DB.Context(ctx).In("user_id", userIDs).Find(&profiles)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	for _, p := range profiles {
		out[p.UserID] = p
	}
	return out, nil
}

// Upsert writes the profile, inserting if the row doesn't exist yet.
func (r *NetworkProfileRepo) Upsert(ctx context.Context, p *entity.NetworkProfile) error {
	existing := &entity.NetworkProfile{}
	exist, err := r.data.DB.Context(ctx).Where("user_id = ?", p.UserID).Get(existing)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if exist {
		_, err = r.data.DB.Context(ctx).Where("user_id = ?", p.UserID).
			Cols("headline", "pronouns", "timezone",
				"open_to_mentoring", "open_to_collaboration", "open_to_hire",
				"external_links").
			Update(p)
	} else {
		_, err = r.data.DB.Context(ctx).Insert(p)
	}
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}
