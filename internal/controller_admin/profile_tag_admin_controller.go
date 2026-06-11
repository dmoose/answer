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

// ListTags returns the full tag catalog, including inactive entries.
func (pc *ProfileTagAdminController) ListTags(ctx *gin.Context) {
	if pc.disabled(ctx) {
		return
	}
	tags, err := pc.tagService.AdminListAll(ctx)
	handler.HandleResponse(ctx, err, tags)
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
