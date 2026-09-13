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
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	assert.Equal(t, "", validSiteSlug(""))
	assert.Equal(t, "", validSiteSlug("default"))
	assert.Equal(t, "go-lang_2", validSiteSlug("go-lang_2"))
	assert.Equal(t, "", validSiteSlug("-leading"))
	assert.Equal(t, "", validSiteSlug("x/y"))
}
