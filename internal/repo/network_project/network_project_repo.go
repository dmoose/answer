package network_project

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/segmentfault/pacman/errors"
)

type NetworkProjectRepo struct {
	data *data.Data
}

func NewNetworkProjectRepo(data *data.Data) *NetworkProjectRepo {
	return &NetworkProjectRepo{data: data}
}

func (r *NetworkProjectRepo) Get(ctx context.Context, id string) (*entity.NetworkProject, bool, error) {
	p := &entity.NetworkProject{}
	exist, err := r.data.DB.Context(ctx).ID(id).Get(p)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return p, exist, nil
}

func (r *NetworkProjectRepo) ListByUser(ctx context.Context, userID string) ([]*entity.NetworkProject, error) {
	var projects []*entity.NetworkProject
	err := r.data.DB.Context(ctx).
		Where("user_id = ?", userID).
		OrderBy("updated_at DESC").
		Find(&projects)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return projects, nil
}

// ListRecent returns the most-recently-updated active projects across all
// members, for the directory home "what people are working on" feed.
func (r *NetworkProjectRepo) ListRecent(ctx context.Context, limit int) ([]*entity.NetworkProject, error) {
	var projects []*entity.NetworkProject
	err := r.data.DB.Context(ctx).
		Where("status = ?", entity.NetworkProjectStatusActive).
		OrderBy("updated_at DESC").
		Limit(limit).
		Find(&projects)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return projects, nil
}

func (r *NetworkProjectRepo) Insert(ctx context.Context, p *entity.NetworkProject) error {
	_, err := r.data.DB.Context(ctx).Insert(p)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *NetworkProjectRepo) Update(ctx context.Context, p *entity.NetworkProject) error {
	_, err := r.data.DB.Context(ctx).ID(p.ID).
		Cols("title", "description", "repo_url", "status", "seeking_help").
		Update(p)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *NetworkProjectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.data.DB.Context(ctx).ID(id).Delete(&entity.NetworkProject{})
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}
