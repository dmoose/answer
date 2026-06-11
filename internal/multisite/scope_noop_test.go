//go:build !multisite

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

package multisite

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/base/constant"
)

func TestSiteIDFromContext_NoopBuild(t *testing.T) {
	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-xyz")
	got := SiteIDFromContext(ctx)
	if got != "" {
		t.Errorf("non-multisite SiteIDFromContext should return empty, got %q", got)
	}
}

func TestSetSiteID_NoopBuild(t *testing.T) {
	type e struct{ SiteID string }
	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-123")
	entity := &e{SiteID: "original"}
	SetSiteID(ctx, entity)
	if entity.SiteID != "original" {
		t.Errorf("non-multisite SetSiteID should be no-op, got %q", entity.SiteID)
	}
}
