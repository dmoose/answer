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
	SiteStatusActive    = 1
	SiteStatusSuspended = 9
)

type Site struct {
	ID          string    `xorm:"not null pk VARCHAR(36) id" json:"id"`
	CreatedAt   time.Time `xorm:"created TIMESTAMP created_at" json:"created_at"`
	UpdatedAt   time.Time `xorm:"updated TIMESTAMP updated_at" json:"updated_at"`
	Name        string    `xorm:"not null VARCHAR(255) name" json:"name"`
	Slug        string    `xorm:"not null unique VARCHAR(64) slug" json:"slug"`
	Description string    `xorm:"TEXT description" json:"description"`
	Status      int       `xorm:"not null default 1 INT(11) status" json:"status"`
	IconURL     string    `xorm:"VARCHAR(512) icon_url" json:"icon_url"`
	BaseURL     string    `xorm:"VARCHAR(512) base_url" json:"base_url"`
}

func (Site) TableName() string {
	return "site"
}
