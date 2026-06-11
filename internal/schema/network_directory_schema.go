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

package schema

// ProfileExternalLink is one entry in a user's self-attested external links list.
// Not verified; rendered as a labeled link.
type ProfileExternalLink struct {
	Label string `json:"label" validate:"required,max=64"`
	URL   string `json:"url" validate:"required,url,max=512"`
}

// ProfileTagInfo is the tag shape returned in profile and directory responses.
type ProfileTagInfo struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Kind        int    `json:"kind"`
	Description string `json:"description,omitempty"`
}

// ProfileProjectInfo is a project entry in a member's profile.
type ProfileProjectInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	RepoURL     string `json:"repo_url"`
	Status      int    `json:"status"`
	SeekingHelp bool   `json:"seeking_help"`
	UpdatedAt   int64  `json:"updated_at"`
}

// NetworkProfileUpdateReq is the body for PUT /network/profile (own profile only).
type NetworkProfileUpdateReq struct {
	UserID              string                `json:"-"`
	Headline            string                `json:"headline" validate:"max=255"`
	Pronouns            string                `json:"pronouns" validate:"max=64"`
	Timezone            string                `json:"timezone" validate:"max=64"`
	OpenToMentoring     bool                  `json:"open_to_mentoring"`
	OpenToCollaboration bool                  `json:"open_to_collaboration"`
	OpenToHire          bool                  `json:"open_to_hire"`
	ExternalLinks       []ProfileExternalLink `json:"external_links" validate:"max=10,dive"`
}

// NetworkSetProfileTagsReq is the body for PUT /network/me/tags.
type NetworkSetProfileTagsReq struct {
	UserID string   `json:"-"`
	TagIDs []string `json:"tag_ids" validate:"max=30,dive,required"`
}

// NetworkProjectCreateReq is the body for POST /network/projects.
type NetworkProjectCreateReq struct {
	UserID      string `json:"-"`
	Title       string `json:"title" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"max=4000"`
	RepoURL     string `json:"repo_url" validate:"omitempty,url,max=512"`
	Status      int    `json:"status" validate:"oneof=1 2 9"`
	SeekingHelp bool   `json:"seeking_help"`
}

// NetworkProjectUpdateReq is the body for PUT /network/projects/{id}. UserID is
// the requesting user; the service enforces ownership.
type NetworkProjectUpdateReq struct {
	ID          string `json:"-"`
	UserID      string `json:"-"`
	Title       string `json:"title" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"max=4000"`
	RepoURL     string `json:"repo_url" validate:"omitempty,url,max=512"`
	Status      int    `json:"status" validate:"oneof=1 2 9"`
	SeekingHelp bool   `json:"seeking_help"`
}

// DirectorySearchReq is the query for GET /network/members. All filters are
// optional; with no filters the directory returns active members sorted by
// reputation, descending.
type DirectorySearchReq struct {
	Q                   string   `form:"q" validate:"max=128"`
	TagIDs              []string `form:"tag_ids" validate:"max=10,dive"`
	OpenToMentoring     bool     `form:"open_to_mentoring"`
	OpenToCollaboration bool     `form:"open_to_collaboration"`
	OpenToHire          bool     `form:"open_to_hire"`
	Page                int      `form:"page" validate:"omitempty,min=1"`
	PageSize            int      `form:"page_size" validate:"omitempty,min=1,max=100"`
	Sort                string   `form:"sort" validate:"omitempty,oneof=rep_desc newest active"`
}

// DirectoryMember is one card in the directory list.
type DirectoryMember struct {
	UserID              string            `json:"user_id"`
	Username            string            `json:"username"`
	DisplayName         string            `json:"display_name"`
	Avatar              string            `json:"avatar"`
	Reputation          int               `json:"reputation"`
	Headline            string            `json:"headline"`
	Pronouns            string            `json:"pronouns"`
	Timezone            string            `json:"timezone"`
	OpenToMentoring     bool              `json:"open_to_mentoring"`
	OpenToCollaboration bool              `json:"open_to_collaboration"`
	OpenToHire          bool              `json:"open_to_hire"`
	Tags                []*ProfileTagInfo `json:"tags"`
}

// AdminProfileTagInfo is the admin-view tag shape — includes Status so the
// curator can see and toggle inactive tags.
type AdminProfileTagInfo struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Kind        int    `json:"kind"`
	Description string `json:"description,omitempty"`
	Status      int    `json:"status"`
}

// AdminProfileTagUpsertReq is the body for POST/PUT /admin/network/tags.
type AdminProfileTagUpsertReq struct {
	ID          string `json:"-"`
	Slug        string `json:"slug" validate:"required,min=2,max=64"`
	Name        string `json:"name" validate:"required,min=1,max=128"`
	Kind        int    `json:"kind" validate:"oneof=1 2 3"`
	Description string `json:"description" validate:"max=512"`
	Status      int    `json:"status" validate:"oneof=1 9"`
}
