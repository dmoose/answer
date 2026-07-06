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
	"github.com/apache/answer/internal/service/service_config"
	"github.com/apache/answer/internal/service/site"
	"github.com/gin-gonic/gin"
	"github.com/segmentfault/pacman/errors"
)

type SiteController struct {
	siteService           *site.SiteService
	networkProfileService *site.NetworkProfileService
	serviceConfig         *service_config.ServiceConfig
}

func NewSiteController(
	siteService *site.SiteService,
	networkProfileService *site.NetworkProfileService,
	serviceConfig *service_config.ServiceConfig,
) *SiteController {
	return &SiteController{
		siteService:           siteService,
		networkProfileService: networkProfileService,
		serviceConfig:         serviceConfig,
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
	// The extended directory profile follows directory visibility: when the
	// directory is enabled and members-only (the default), anonymous
	// requests are rejected like the member list itself.
	if sc.serviceConfig != nil && sc.serviceConfig.DirectoryEnabled &&
		!sc.serviceConfig.DirectoryPublic() &&
		middleware.GetLoginUserIDFromContext(ctx) == "" {
		handler.HandleResponse(ctx, errors.Unauthorized(reason.UnauthorizedError), nil)
		return
	}
	profile, err := sc.networkProfileService.GetNetworkProfile(ctx, userID)
	handler.HandleResponse(ctx, err, profile)
}
