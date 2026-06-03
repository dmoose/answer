package site

import (
	"context"

	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/service/role"
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
	siteRepo    SiteRepo
	siteRoleMgr role.SiteRoleRepo
}

func NewSiteService(siteRepo SiteRepo, siteRoleMgr role.SiteRoleRepo) *SiteService {
	return &SiteService{siteRepo: siteRepo, siteRoleMgr: siteRoleMgr}
}

var reservedSlugs = map[string]bool{
	"default": true, "admin": true, "answer": true, "api": true,
	"static": true, "install": true, "www": true, "s": true,
	"healthz": true, "users": true, "questions": true, "tags": true,
}

func validateSlug(slug string) error {
	if len(slug) < 2 || len(slug) > 50 {
		return errors.BadRequest(reason.ObjectNotFound).WithMsg("slug must be 2-50 characters")
	}
	for _, c := range slug {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return errors.BadRequest(reason.ObjectNotFound).WithMsg("slug must be lowercase alphanumeric with hyphens")
		}
	}
	if reservedSlugs[slug] {
		return errors.BadRequest(reason.ObjectNotFound).WithMsg("slug is reserved")
	}
	return nil
}

func (s *SiteService) AddSite(ctx context.Context, name, slug, description, baseURL string) (*entity.Site, error) {
	if err := validateSlug(slug); err != nil {
		return nil, err
	}
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

func (s *SiteService) SetUserSiteRole(ctx context.Context, userID, siteID string, roleID int) error {
	if roleID < 1 || roleID > 3 {
		return errors.BadRequest(reason.ObjectNotFound).WithMsg("invalid role")
	}
	_, exist, err := s.siteRepo.GetSite(ctx, siteID)
	if err != nil {
		return err
	}
	if !exist {
		return errors.NotFound(reason.ObjectNotFound)
	}
	return s.siteRoleMgr.SaveUserSiteRole(ctx, userID, siteID, roleID)
}

type SiteUserRole struct {
	UserID string `json:"user_id"`
	SiteID string `json:"site_id"`
	RoleID int    `json:"role_id"`
}

func (s *SiteService) GetUserSiteRole(ctx context.Context, userID, siteID string) (*SiteUserRole, error) {
	roleID, exist, err := s.siteRoleMgr.GetUserSiteRole(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return &SiteUserRole{UserID: userID, SiteID: siteID, RoleID: 1}, nil
	}
	return &SiteUserRole{UserID: userID, SiteID: siteID, RoleID: roleID}, nil
}
