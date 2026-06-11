package controller_admin

import (
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/network_directory"
	"github.com/apache/answer/internal/service/service_config"
	"github.com/gin-gonic/gin"
	"github.com/segmentfault/pacman/errors"
)

type ProfileTagAdminController struct {
	tagService    *network_directory.ProfileTagService
	serviceConfig *service_config.ServiceConfig
}

func NewProfileTagAdminController(
	tagService *network_directory.ProfileTagService,
	serviceConfig *service_config.ServiceConfig,
) *ProfileTagAdminController {
	return &ProfileTagAdminController{
		tagService:    tagService,
		serviceConfig: serviceConfig,
	}
}

func (pc *ProfileTagAdminController) disabled(ctx *gin.Context) bool {
	if pc.serviceConfig != nil && pc.serviceConfig.DirectoryEnabled {
		return false
	}
	handler.HandleResponse(ctx, errors.NotFound(reason.ObjectNotFound), nil)
	return true
}

// CreateTag adds a curated tag to the profile tag catalog.
func (pc *ProfileTagAdminController) CreateTag(ctx *gin.Context) {
	if pc.disabled(ctx) {
		return
	}
	req := &schema.AdminProfileTagUpsertReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.ID = ""
	resp, err := pc.tagService.AdminUpsert(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

// UpdateTag updates an existing tag (name, kind, description, status).
func (pc *ProfileTagAdminController) UpdateTag(ctx *gin.Context) {
	if pc.disabled(ctx) {
		return
	}
	req := &schema.AdminProfileTagUpsertReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.ID = ctx.Param("id")
	resp, err := pc.tagService.AdminUpsert(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}
