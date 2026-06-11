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

package data

import (
	"context"

	"xorm.io/xorm"
)

func (d *Data) SiteDB(ctx context.Context) *xorm.Session {
	return d.DB.Context(ctx)
}

func (d *Data) SiteTransaction(ctx context.Context, f func(*xorm.Session) (any, error)) (any, error) {
	return d.DB.Transaction(func(session *xorm.Session) (any, error) {
		session = session.Context(ctx)
		return f(session)
	})
}

func (d *Data) SiteInsert(ctx context.Context, beans ...any) (int64, error) {
	return d.DB.Context(ctx).Insert(beans...)
}
