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

func (sc *SiteAdminController) SetUserSiteRole(ctx *gin.Context) {
	req := &struct {
		UserID string `json:"user_id" binding:"required"`
		SiteID string `json:"site_id" binding:"required"`
		RoleID int    `json:"role_id" binding:"required"`
	}{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	err := sc.siteService.SetUserSiteRole(ctx, req.UserID, req.SiteID, req.RoleID)
	handler.HandleResponse(ctx, err, nil)
}

func (sc *SiteAdminController) GetUserSiteRole(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	siteID := ctx.Query("site_id")
	r, err := sc.siteService.GetUserSiteRole(ctx, userID, siteID)
	handler.HandleResponse(ctx, err, r)
}
