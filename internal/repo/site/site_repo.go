package site

import (
	"context"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/service/site"
	"github.com/segmentfault/pacman/errors"
)

type siteRepo struct {
	data *data.Data
}

func NewSiteRepo(data *data.Data) site.SiteRepo {
	return &siteRepo{data: data}
}

func (r *siteRepo) AddSite(ctx context.Context, s *entity.Site) error {
	_, err := r.data.DB.Context(ctx).Insert(s)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *siteRepo) UpdateSite(ctx context.Context, s *entity.Site) error {
	_, err := r.data.DB.Context(ctx).ID(s.ID).
		Cols("name", "slug", "description", "status", "icon_url", "base_url").Update(s)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (r *siteRepo) GetSite(ctx context.Context, id string) (*entity.Site, bool, error) {
	s := &entity.Site{}
	exist, err := r.data.DB.Context(ctx).ID(id).Get(s)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return s, exist, nil
}

func (r *siteRepo) GetSiteBySlug(ctx context.Context, slug string) (*entity.Site, bool, error) {
	s := &entity.Site{Slug: slug}
	exist, err := r.data.DB.Context(ctx).Get(s)
	if err != nil {
		return nil, false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return s, exist, nil
}

func (r *siteRepo) GetAllSites(ctx context.Context) ([]*entity.Site, error) {
	var sites []*entity.Site
	err := r.data.DB.Context(ctx).Where("status = ?", entity.SiteStatusActive).Find(&sites)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return sites, nil
}
