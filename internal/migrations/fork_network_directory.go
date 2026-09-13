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

package migrations

import (
	"context"
	"fmt"

	"github.com/apache/answer/internal/entity"
	"xorm.io/xorm"
)

// addNetworkDirectory creates the network-level member directory tables:
// extended profile, projects-in-progress, and curated skill/interest tags.
// All tables key by user_id (BIGINT) and are network-level — no site_id —
// matching the network profile concept already in place.
func addNetworkDirectory(ctx context.Context, x *xorm.Engine) error {
	if err := x.Context(ctx).Sync(
		new(entity.NetworkProfile),
		new(entity.NetworkProject),
		new(entity.ProfileTag),
		new(entity.UserProfileTag),
	); err != nil {
		return fmt.Errorf("create network directory tables: %w", err)
	}
	return nil
}
