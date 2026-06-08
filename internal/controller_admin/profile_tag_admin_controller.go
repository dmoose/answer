package controller_admin

import (
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/network_directory"
	"github.com/gin-gonic/gin"
)

type ProfileTagAdminController struct {
	tagService *network_directory.ProfileTagService
}

func NewProfileTagAdminController(tagService *network_directory.ProfileTagService) *ProfileTagAdminController {
	return &ProfileTagAdminController{tagService: tagService}
}

// CreateTag adds a curated tag to the profile tag catalog.
func (pc *ProfileTagAdminController) CreateTag(ctx *gin.Context) {
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
	req := &schema.AdminProfileTagUpsertReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.ID = ctx.Param("id")
	resp, err := pc.tagService.AdminUpsert(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}
