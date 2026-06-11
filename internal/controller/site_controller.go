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
