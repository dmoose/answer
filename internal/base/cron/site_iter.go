//go:build multisite

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

package cron

import (
	"context"

	"github.com/apache/answer/internal/multisite"
	"github.com/segmentfault/pacman/log"
)

// forEachSite runs fn once per active sub-site, each under a context carrying
// that site, so site-scoped cron work (sitemap generation) is partitioned per
// site instead of running site-less across all of them.
func (s *ScheduledTaskManager) forEachSite(ctx context.Context, fn func(ctx context.Context)) {
	sites, err := s.siteService.GetAllSites(ctx)
	if err != nil {
		log.Errorf("cron: list sites: %v", err)
		return
	}
	for _, site := range sites {
		fn(multisite.WithSiteID(ctx, site.ID))
	}
}
