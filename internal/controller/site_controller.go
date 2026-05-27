package controller

import (
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/service/site"
	"github.com/gin-gonic/gin"
)

type SiteController struct {
	siteService           *site.SiteService
	networkProfileService *site.NetworkProfileService
}

func NewSiteController(
	siteService *site.SiteService,
	networkProfileService *site.NetworkProfileService,
) *SiteController {
	return &SiteController{
		siteService:           siteService,
		networkProfileService: networkProfileService,
	}
}

func (sc *SiteController) GetSiteList(ctx *gin.Context) {
	sites, err := sc.siteService.GetAllSites(ctx)
	handler.HandleResponse(ctx, err, sites)
}

func (sc *SiteController) GetNetworkProfile(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	if userID == "" {
		handler.HandleResponse(ctx, nil, nil)
		return
	}
	profile, err := sc.networkProfileService.GetNetworkProfile(ctx, userID)
	handler.HandleResponse(ctx, err, profile)
}
