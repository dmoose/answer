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

package controller_admin

import (
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/middleware"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
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

// AddSite create a sub-site
// @Summary create a sub-site
// @Description create a sub-site
// @Security ApiKeyAuth
// @Tags admin
// @Accept json
// @Produce json
// @Param data body schema.SiteAddReq true "site"
// @Success 200 {object} handler.RespBody{data=entity.Site}
// @Router /answer/admin/api/site [post]
func (sc *SiteAdminController) AddSite(ctx *gin.Context) {
	req := &schema.SiteAddReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	s, err := sc.siteService.AddSite(ctx, req.Name, req.Slug, req.Description, req.BaseURL)
	if err == nil {
		sc.siteMiddleware.RefreshSiteCache()
	}
	handler.HandleResponse(ctx, err, s)
}

// UpdateSite update a sub-site
// @Summary update a sub-site
// @Description update a sub-site; empty status/slug fields leave the stored values untouched
// @Security ApiKeyAuth
// @Tags admin
// @Accept json
// @Produce json
// @Param data body entity.Site true "site"
// @Success 200 {object} handler.RespBody
// @Router /answer/admin/api/site [put]
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

// GetSite get a sub-site by id
// @Summary get a sub-site by id
// @Description get a sub-site by id
// @Security ApiKeyAuth
// @Tags admin
// @Produce json
// @Param id query string true "site id"
// @Success 200 {object} handler.RespBody{data=entity.Site}
// @Router /answer/admin/api/site [get]
func (sc *SiteAdminController) GetSite(ctx *gin.Context) {
	id := ctx.Query("id")
	s, err := sc.siteService.GetSite(ctx, id)
	handler.HandleResponse(ctx, err, s)
}

// GetSiteList list all sub-sites, including suspended ones
// @Summary list all sub-sites
// @Description list all sub-sites, including suspended ones
// @Security ApiKeyAuth
// @Tags admin
// @Produce json
// @Success 200 {object} handler.RespBody{data=[]entity.Site}
// @Router /answer/admin/api/sites [get]
func (sc *SiteAdminController) GetSiteList(ctx *gin.Context) {
	// Admin listing includes suspended sites: a site that disappears from
	// every list the moment it is suspended can never be reactivated.
	sites, err := sc.siteService.GetAllSitesForAdmin(ctx)
	handler.HandleResponse(ctx, err, sites)
}

// SetSiteStatus activate or suspend a sub-site
// @Summary activate or suspend a sub-site
// @Description activate or suspend a sub-site; the default site cannot be suspended
// @Security ApiKeyAuth
// @Tags admin
// @Accept json
// @Produce json
// @Param data body schema.SiteStatusReq true "status"
// @Success 200 {object} handler.RespBody
// @Router /answer/admin/api/site/status [put]
func (sc *SiteAdminController) SetSiteStatus(ctx *gin.Context) {
	req := &schema.SiteStatusReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	err := sc.siteService.SetSiteStatus(ctx, req.SiteID, req.Active)
	if err == nil {
		sc.siteMiddleware.RefreshSiteCache()
	}
	handler.HandleResponse(ctx, err, nil)
}

// SetUserSiteRole assign a user's role on a sub-site
// @Summary assign a user's role on a sub-site
// @Description assign a user's role on a sub-site; a site role can raise but never lower the global role
// @Security ApiKeyAuth
// @Tags admin
// @Accept json
// @Produce json
// @Param data body schema.SiteUserRoleReq true "role assignment"
// @Success 200 {object} handler.RespBody
// @Router /answer/admin/api/site/role [put]
func (sc *SiteAdminController) SetUserSiteRole(ctx *gin.Context) {
	req := &schema.SiteUserRoleReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	err := sc.siteService.SetUserSiteRole(ctx, req.UserID, req.SiteID, req.RoleID)
	handler.HandleResponse(ctx, err, nil)
}

// GetUserSiteRole get a user's effective role on a sub-site
// @Summary get a user's role on a sub-site
// @Description get a user's effective role on a sub-site
// @Security ApiKeyAuth
// @Tags admin
// @Produce json
// @Param user_id query string true "user id"
// @Param site_id query string true "site id"
// @Success 200 {object} handler.RespBody{data=site.SiteUserRole}
// @Router /answer/admin/api/site/role [get]
func (sc *SiteAdminController) GetUserSiteRole(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	siteID := ctx.Query("site_id")
	r, err := sc.siteService.GetUserSiteRole(ctx, userID, siteID)
	handler.HandleResponse(ctx, err, r)
}
