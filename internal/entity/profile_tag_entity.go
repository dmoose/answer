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

const (
	ProfileTagKindSkill    = 1
	ProfileTagKindInterest = 2
	ProfileTagKindBoth     = 3

	ProfileTagStatusActive   = 1
	ProfileTagStatusInactive = 9
)

// ProfileTag is an admin-curated tag attached to members for the directory
// faceting (skill: "Rust", interest: "homelab", etc.). Separate from Answer's
// Q&A tag system; the lifecycles and meaning are different.
type ProfileTag struct {
	ID          string    `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt   time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt   time.Time `xorm:"updated TIMESTAMP updated_at"`
	Slug        string    `xorm:"not null unique VARCHAR(64) slug"`
	Name        string    `xorm:"not null VARCHAR(128) name"`
	Kind        int       `xorm:"not null default 1 INT(11) kind"`
	Description string    `xorm:"VARCHAR(512) description"`
	Status      int       `xorm:"not null default 1 INT(11) status"`
}

func (ProfileTag) TableName() string {
	return "profile_tag"
}
