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

package multisite

import (
	"context"
	"reflect"

	"github.com/apache/answer/internal/base/constant"
	"github.com/gin-gonic/gin"
	"xorm.io/xorm"
)

func SiteIDFromContext(ctx context.Context) string {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		if val, exists := ginCtx.Get(constant.SiteIDFlag); exists {
			if siteID, ok := val.(string); ok {
				return siteID
			}
		}
	}
	if val, ok := ctx.Value(constant.SiteIDContextKey).(string); ok {
		return val
	}
	return ""
}

// WithoutSite returns a child context with the site ID cleared. Used for
// opt-in cross-site reads (e.g. network-wide search).
func WithoutSite(ctx context.Context) context.Context {
	return context.WithValue(ctx, constant.SiteIDContextKey, "")
}

func Scope(session *xorm.Session, ctx context.Context) *xorm.Session {
	if siteID := SiteIDFromContext(ctx); siteID != "" {
		return session.Where("site_id = ?", siteID)
	}
	return session
}

func SetSiteID(ctx context.Context, entities ...any) {
	siteID := SiteIDFromContext(ctx)
	if siteID == "" {
		return
	}
	for _, e := range entities {
		setSiteIDValue(reflect.ValueOf(e), siteID)
	}
}

func setSiteIDValue(v reflect.Value, siteID string) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		f := v.FieldByName("SiteID")
		if f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
			f.SetString(siteID)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			setSiteIDValue(v.Index(i), siteID)
		}
	}
}
