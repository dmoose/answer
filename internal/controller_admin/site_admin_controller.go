package controller_admin

import (
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/middleware"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/service/site"
	"github.com/gin-gonic/gin"
)

type SiteAdminController struct {
	siteService    *site.SiteService
	siteMiddleware *middleware.SiteMiddleware
}

func NewSiteAdminController(
	siteService *site.SiteService,
	siteMiddleware *middleware.SiteMiddleware,
) *SiteAdminController {
	return &SiteAdminController{
		siteService:    siteService,
		siteMiddleware: siteMiddleware,
	}
}

func (sc *SiteAdminController) AddSite(ctx *gin.Context) {
	req := &struct {
		Name        string `json:"name" binding:"required"`
		Slug        string `json:"slug" binding:"required"`
		Description string `json:"description"`
		BaseURL     string `json:"base_url"`
	}{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	s, err := sc.siteService.AddSite(ctx, req.Name, req.Slug, req.Description, req.BaseURL)
	if err == nil {
		sc.siteMiddleware.RefreshSiteCache()
	}
	handler.HandleResponse(ctx, err, s)
}

func (sc *SiteAdminController) UpdateSite(ctx *gin.Context) {
	req := &entity.Site{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	err := sc.siteService.UpdateSite(ctx, req)
	if err == nil {
		sc.siteMiddleware.RefreshSiteCache()
	}
	handler.HandleResponse(ctx, err, nil)
}

func (sc *SiteAdminController) GetSite(ctx *gin.Context) {
	id := ctx.Query("id")
	s, err := sc.siteService.GetSite(ctx, id)
	handler.HandleResponse(ctx, err, s)
}

func (sc *SiteAdminController) GetSiteList(ctx *gin.Context) {
	sites, err := sc.siteService.GetAllSites(ctx)
	handler.HandleResponse(ctx, err, sites)
}
