package site

import (
	"context"

	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/pkg/uid"
	"github.com/segmentfault/pacman/errors"
)

type SiteRepo interface {
	AddSite(ctx context.Context, site *entity.Site) error
	UpdateSite(ctx context.Context, site *entity.Site) error
	GetSite(ctx context.Context, id string) (*entity.Site, bool, error)
	GetSiteBySlug(ctx context.Context, slug string) (*entity.Site, bool, error)
	GetAllSites(ctx context.Context) ([]*entity.Site, error)
}

type SiteService struct {
	siteRepo SiteRepo
}

func NewSiteService(siteRepo SiteRepo) *SiteService {
	return &SiteService{siteRepo: siteRepo}
}

func (s *SiteService) AddSite(ctx context.Context, name, slug, description, baseURL string) (*entity.Site, error) {
	_, exist, _ := s.siteRepo.GetSiteBySlug(ctx, slug)
	if exist {
		return nil, errors.BadRequest(reason.ObjectNotFound).WithMsg("site slug already exists")
	}
	site := &entity.Site{
		ID:          uid.IDStr12(),
		Name:        name,
		Slug:        slug,
		Description: description,
		BaseURL:     baseURL,
		Status:      entity.SiteStatusActive,
	}
	if err := s.siteRepo.AddSite(ctx, site); err != nil {
		return nil, err
	}
	return site, nil
}

func (s *SiteService) GetSite(ctx context.Context, id string) (*entity.Site, error) {
	site, exist, err := s.siteRepo.GetSite(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.NotFound(reason.ObjectNotFound)
	}
	return site, nil
}

func (s *SiteService) GetAllSites(ctx context.Context) ([]*entity.Site, error) {
	return s.siteRepo.GetAllSites(ctx)
}

func (s *SiteService) UpdateSite(ctx context.Context, site *entity.Site) error {
	return s.siteRepo.UpdateSite(ctx, site)
}
