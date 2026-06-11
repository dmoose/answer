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

package role

import (
	"context"

	"github.com/apache/answer/internal/multisite"
	"github.com/segmentfault/pacman/log"
)

type SiteRoleRepo interface {
	GetUserSiteRole(ctx context.Context, userID, siteID string) (roleID int, exist bool, err error)
	SaveUserSiteRole(ctx context.Context, userID, siteID string, roleID int) error
}

func (us *UserRoleRelService) getEffectiveRole(ctx context.Context, userID string, globalRoleID int) int {
	siteID := multisite.SiteIDFromContext(ctx)
	if siteID == "" || us.siteRoleRepo == nil {
		return globalRoleID
	}
	siteRoleID, exist, err := us.siteRoleRepo.GetUserSiteRole(ctx, userID, siteID)
	if err != nil {
		log.Errorf("get site role for user %s site %s: %v", userID, siteID, err)
		return globalRoleID
	}
	if !exist {
		return globalRoleID
	}
	if isMorePrivileged(siteRoleID, globalRoleID) {
		return siteRoleID
	}
	return globalRoleID
}

func isMorePrivileged(a, b int) bool {
	return privilegeLevel(a) > privilegeLevel(b)
}

func privilegeLevel(roleID int) int {
	switch roleID {
	case RoleAdminID:
		return 3
	case RoleModeratorID:
		return 2
	case RoleUserID:
		return 1
	default:
		return 0
	}
}
