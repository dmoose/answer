package controller

import (
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/middleware"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/network_directory"
	"github.com/gin-gonic/gin"
)

// MemberDirectoryController hosts the public-read and member-write endpoints
// for the network directory: tag catalog, member search, and own-profile
// updates (profile fields, tag set, projects). The extended-profile read
// itself stays on SiteController.GetNetworkProfile so the existing
// /network/user/profile route returns one assembled payload.
type MemberDirectoryController struct {
	editService      *network_directory.ProfileEditService
	tagService       *network_directory.ProfileTagService
	directoryService *network_directory.MemberDirectoryService
}

func NewMemberDirectoryController(
	editService *network_directory.ProfileEditService,
	tagService *network_directory.ProfileTagService,
	directoryService *network_directory.MemberDirectoryService,
) *MemberDirectoryController {
	return &MemberDirectoryController{
		editService:      editService,
		tagService:       tagService,
		directoryService: directoryService,
	}
}

// ListTags returns the active tag catalog. Optional `kind` query (1=skill,
// 2=interest, 3=both) narrows the list for the picker UIs.
func (mc *MemberDirectoryController) ListTags(ctx *gin.Context) {
	kind := 0
	switch ctx.Query("kind") {
	case "1":
		kind = 1
	case "2":
		kind = 2
	case "3":
		kind = 3
	}
	tags, err := mc.tagService.ListActive(ctx, kind)
	handler.HandleResponse(ctx, err, tags)
}

// ListMembers runs the faceted directory query and returns a page of cards.
func (mc *MemberDirectoryController) ListMembers(ctx *gin.Context) {
	req := &schema.DirectorySearchReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	resp, err := mc.directoryService.Search(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

func (mc *MemberDirectoryController) UpdateProfile(ctx *gin.Context) {
	req := &schema.NetworkProfileUpdateReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.UserID = middleware.GetLoginUserIDFromContext(ctx)
	err := mc.editService.UpdateProfile(ctx, req)
	handler.HandleResponse(ctx, err, nil)
}

func (mc *MemberDirectoryController) SetTags(ctx *gin.Context) {
	req := &schema.NetworkSetProfileTagsReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.UserID = middleware.GetLoginUserIDFromContext(ctx)
	err := mc.editService.SetTags(ctx, req)
	handler.HandleResponse(ctx, err, nil)
}

func (mc *MemberDirectoryController) CreateProject(ctx *gin.Context) {
	req := &schema.NetworkProjectCreateReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.UserID = middleware.GetLoginUserIDFromContext(ctx)
	resp, err := mc.editService.CreateProject(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

func (mc *MemberDirectoryController) UpdateProject(ctx *gin.Context) {
	req := &schema.NetworkProjectUpdateReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.ID = ctx.Param("id")
	req.UserID = middleware.GetLoginUserIDFromContext(ctx)
	resp, err := mc.editService.UpdateProject(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

func (mc *MemberDirectoryController) DeleteProject(ctx *gin.Context) {
	id := ctx.Param("id")
	userID := middleware.GetLoginUserIDFromContext(ctx)
	err := mc.editService.DeleteProject(ctx, id, userID)
	handler.HandleResponse(ctx, err, nil)
}
