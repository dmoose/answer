/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package controller

import (
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/middleware"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/network_directory"
	"github.com/apache/answer/internal/service/service_config"
	"github.com/gin-gonic/gin"
	"github.com/segmentfault/pacman/errors"
)

// MemberDirectoryController hosts the public-read and member-write endpoints
// for the network directory: tag catalog, member search, and own-profile
// updates (profile fields, tag set, projects). The extended-profile read
// itself stays on SiteController.GetNetworkProfile so the existing
// /network/user/profile route returns one assembled payload.
//
// All handlers short-circuit with 404 when DirectoryEnabled is false in
// service config, so a deployment that doesn't want the directory feature
// presents as if these endpoints don't exist at all.
type MemberDirectoryController struct {
	editService      *network_directory.ProfileEditService
	tagService       *network_directory.ProfileTagService
	directoryService *network_directory.MemberDirectoryService
	serviceConfig    *service_config.ServiceConfig
}

func NewMemberDirectoryController(
	editService *network_directory.ProfileEditService,
	tagService *network_directory.ProfileTagService,
	directoryService *network_directory.MemberDirectoryService,
	serviceConfig *service_config.ServiceConfig,
) *MemberDirectoryController {
	return &MemberDirectoryController{
		editService:      editService,
		tagService:       tagService,
		directoryService: directoryService,
		serviceConfig:    serviceConfig,
	}
}

// disabled returns true and writes a 404 response when the directory feature
// is off. Each handler calls this first.
func (mc *MemberDirectoryController) disabled(ctx *gin.Context) bool {
	if mc.serviceConfig != nil && mc.serviceConfig.DirectoryEnabled {
		return false
	}
	handler.HandleResponse(ctx, errors.NotFound(reason.ObjectNotFound), nil)
	return true
}

// blockedByVisibility returns true and writes a 401 when the directory is
// members-only (the default) and the request is anonymous. Read handlers on
// the public router group call this after the disabled check.
func (mc *MemberDirectoryController) blockedByVisibility(ctx *gin.Context) bool {
	if mc.serviceConfig.DirectoryPublic() {
		return false
	}
	if middleware.GetLoginUserIDFromContext(ctx) != "" {
		return false
	}
	handler.HandleResponse(ctx, errors.Unauthorized(reason.UnauthorizedError), nil)
	return true
}

// ListTags returns the active tag catalog. Optional `kind` query (1=skill,
// 2=interest, 3=both) narrows the list for the picker UIs.
// @Summary list active profile tags
// @Description list the curated skill/interest tag catalog
// @Tags NetworkDirectory
// @Produce json
// @Param kind query int false "1=skill, 2=interest, 3=both"
// @Success 200 {object} handler.RespBody{data=[]schema.ProfileTagInfo}
// @Router /answer/api/v1/network/tags [get]
func (mc *MemberDirectoryController) ListTags(ctx *gin.Context) {
	if mc.disabled(ctx) || mc.blockedByVisibility(ctx) {
		return
	}
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
// @Summary search the member directory
// @Description faceted member search; members-only unless directory_visibility is public
// @Tags NetworkDirectory
// @Produce json
// @Param q query string false "text query"
// @Param page query int false "page"
// @Param page_size query int false "page size"
// @Success 200 {object} handler.RespBody{data=pager.PageModel{list=[]schema.DirectoryMember}}
// @Router /answer/api/v1/network/members [get]
func (mc *MemberDirectoryController) ListMembers(ctx *gin.Context) {
	if mc.disabled(ctx) || mc.blockedByVisibility(ctx) {
		return
	}
	req := &schema.DirectorySearchReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	resp, err := mc.directoryService.Search(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

// UpdateProfile update the caller's network profile
// @Summary update the caller's network profile
// @Description update the extended directory profile of the logged-in user
// @Security ApiKeyAuth
// @Tags NetworkDirectory
// @Accept json
// @Produce json
// @Param data body schema.NetworkProfileUpdateReq true "profile"
// @Success 200 {object} handler.RespBody
// @Router /answer/api/v1/network/profile [put]
func (mc *MemberDirectoryController) UpdateProfile(ctx *gin.Context) {
	if mc.disabled(ctx) {
		return
	}
	req := &schema.NetworkProfileUpdateReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.UserID = middleware.GetLoginUserIDFromContext(ctx)
	err := mc.editService.UpdateProfile(ctx, req)
	handler.HandleResponse(ctx, err, nil)
}

// SetTags replace the caller's profile tags
// @Summary replace the caller's profile tags
// @Description replace the logged-in user's skill/interest tags
// @Security ApiKeyAuth
// @Tags NetworkDirectory
// @Accept json
// @Produce json
// @Param data body schema.NetworkSetProfileTagsReq true "tags"
// @Success 200 {object} handler.RespBody
// @Router /answer/api/v1/network/me/tags [put]
func (mc *MemberDirectoryController) SetTags(ctx *gin.Context) {
	if mc.disabled(ctx) {
		return
	}
	req := &schema.NetworkSetProfileTagsReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.UserID = middleware.GetLoginUserIDFromContext(ctx)
	err := mc.editService.SetTags(ctx, req)
	handler.HandleResponse(ctx, err, nil)
}

// CreateProject add a project to the caller's profile
// @Summary add a project to the caller's profile
// @Description add a project-in-progress to the logged-in user's profile
// @Security ApiKeyAuth
// @Tags NetworkDirectory
// @Accept json
// @Produce json
// @Param data body schema.NetworkProjectCreateReq true "project"
// @Success 200 {object} handler.RespBody{data=schema.ProfileProjectInfo}
// @Router /answer/api/v1/network/projects [post]
func (mc *MemberDirectoryController) CreateProject(ctx *gin.Context) {
	if mc.disabled(ctx) {
		return
	}
	req := &schema.NetworkProjectCreateReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.UserID = middleware.GetLoginUserIDFromContext(ctx)
	resp, err := mc.editService.CreateProject(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

// UpdateProject update one of the caller's projects
// @Summary update one of the caller's projects
// @Description update a project on the logged-in user's profile
// @Security ApiKeyAuth
// @Tags NetworkDirectory
// @Accept json
// @Produce json
// @Param id path string true "project id"
// @Param data body schema.NetworkProjectUpdateReq true "project"
// @Success 200 {object} handler.RespBody{data=schema.ProfileProjectInfo}
// @Router /answer/api/v1/network/projects/{id} [put]
func (mc *MemberDirectoryController) UpdateProject(ctx *gin.Context) {
	if mc.disabled(ctx) {
		return
	}
	req := &schema.NetworkProjectUpdateReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.ID = ctx.Param("id")
	req.UserID = middleware.GetLoginUserIDFromContext(ctx)
	resp, err := mc.editService.UpdateProject(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

// DeleteProject remove one of the caller's projects
// @Summary remove one of the caller's projects
// @Description remove a project from the logged-in user's profile
// @Security ApiKeyAuth
// @Tags NetworkDirectory
// @Produce json
// @Param id path string true "project id"
// @Success 200 {object} handler.RespBody
// @Router /answer/api/v1/network/projects/{id} [delete]
func (mc *MemberDirectoryController) DeleteProject(ctx *gin.Context) {
	if mc.disabled(ctx) {
		return
	}
	id := ctx.Param("id")
	userID := middleware.GetLoginUserIDFromContext(ctx)
	err := mc.editService.DeleteProject(ctx, id, userID)
	handler.HandleResponse(ctx, err, nil)
}
