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

package entity

import "time"

// NetworkProfile is the guild-level identity record for a user, separate from
// the per-site Answer profile. One row per user; absence means the user has
// not filled in any guild fields yet.
//
// ExternalLinks is a JSON array of {label, url} objects, self-attested and
// presentation-only. Verified cross-app identity (Zulip, GitHub, etc.) is
// resolved through fastgate's directory, not from this field.
type NetworkProfile struct {
	UserID              string    `xorm:"not null pk BIGINT(20) user_id"`
	CreatedAt           time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt           time.Time `xorm:"updated TIMESTAMP updated_at"`
	Headline            string    `xorm:"VARCHAR(255) headline"`
	Pronouns            string    `xorm:"VARCHAR(64) pronouns"`
	Timezone            string    `xorm:"VARCHAR(64) timezone"`
	OpenToMentoring     bool      `xorm:"not null default false BOOL open_to_mentoring"`
	OpenToCollaboration bool      `xorm:"not null default false BOOL open_to_collaboration"`
	OpenToHire          bool      `xorm:"not null default false BOOL open_to_hire"`
	ExternalLinks       string    `xorm:"TEXT external_links"`
}

func (NetworkProfile) TableName() string {
	return "network_profile"
}
