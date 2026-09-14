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
	"context"
	"errors"
	"testing"

	"github.com/apache/answer/internal/entity"
	"github.com/stretchr/testify/assert"
)

type stubSiteRepo struct {
	sites map[string]*entity.Site
	err   error
}

func (r stubSiteRepo) AddSite(context.Context, *entity.Site) error         { return nil }
func (r stubSiteRepo) UpdateSite(context.Context, *entity.Site) error      { return nil }
func (r stubSiteRepo) UpdateSiteStatus(context.Context, string, int) error { return nil }
func (r stubSiteRepo) GetSite(context.Context, string) (*entity.Site, bool, error) {
	return nil, false, nil
}
func (r stubSiteRepo) GetSiteBySlug(_ context.Context, slug string) (*entity.Site, bool, error) {
	if r.err != nil {
		return nil, false, r.err
	}
	s, ok := r.sites[slug]
	return s, ok, nil
}
func (r stubSiteRepo) GetAllSites(context.Context) ([]*entity.Site, error) { return nil, nil }
func (r stubSiteRepo) GetAllSitesIncludingInactive(context.Context) ([]*entity.Site, error) {
	return nil, nil
}

func TestLandingURLHonorsBaseURL(t *testing.T) {
	const site = "https://answer.example.com"
	repo := stubSiteRepo{sites: map[string]*entity.Site{
		"hosted": {Slug: "hosted", BaseURL: "https://hosted.example.com/ "},
		"pathed": {Slug: "pathed"},
	}}
	cc := &ConnectorController{siteRepo: repo}
	ctx := context.Background()
	assert.Equal(t, "https://hosted.example.com", cc.landingURL(ctx, site, "hosted"), "base_url wins, trailing slash trimmed")
	assert.Equal(t, site+"/s/pathed", cc.landingURL(ctx, site, "pathed"), "no base_url: path form")
	assert.Equal(t, site+"/s/unknown", cc.landingURL(ctx, site, "unknown"), "unknown slug: path form")
	assert.Equal(t, site, cc.landingURL(ctx, site, ""), "default site")

	failing := &ConnectorController{siteRepo: stubSiteRepo{err: errors.New("db down")}}
	assert.Equal(t, site+"/s/hosted", failing.landingURL(ctx, site, "hosted"), "lookup error falls back, never blocks login")
}

func TestSiteLandingURL(t *testing.T) {
	const site = "https://answer.example.com"
	assert.Equal(t, site, siteLandingURL(site, ""), "no slug: default site")
	assert.Equal(t, site, siteLandingURL(site, "default"), "default slug: no prefix")
	assert.Equal(t, site+"/s/mssgs", siteLandingURL(site, "mssgs"))
	assert.Equal(t, site+"/s/mssgs", siteLandingURL(site+"/", "mssgs"), "trailing slash folded")
	assert.Equal(t, site, siteLandingURL(site, "../evil"), "malformed slug dropped")
	assert.Equal(t, site, siteLandingURL(site, "Evil"), "slugs are lowercase")
	assert.Equal(t, site, siteLandingURL(site, "a b"), "no whitespace")
}

func TestValidSiteSlug(t *testing.T) {
	assert.Empty(t, validSiteSlug(""))
	assert.Empty(t, validSiteSlug("default"))
	assert.Equal(t, "go-lang_2", validSiteSlug("go-lang_2"))
	assert.Empty(t, validSiteSlug("-leading"))
	assert.Empty(t, validSiteSlug("x/y"))
}
