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

package queue

import (
	"context"
	"testing"
	"time"

	"github.com/apache/answer/internal/multisite"
)

// The handler must run under the site scope of the enqueuing request, so
// site-aware writes (activity attribution, per-site email rendering) keep
// the site the work originated on.
func TestQueue_PropagatesSiteScope(t *testing.T) {
	q := New[*testMessage]("test-site", 10)
	defer q.Close()

	gotSite := make(chan string, 2)
	q.RegisterHandler(func(ctx context.Context, msg *testMessage) error {
		gotSite <- multisite.SiteIDFromContext(ctx)
		return nil
	})

	q.Send(multisite.WithSiteID(context.Background(), "site-a"), &testMessage{ID: 1})
	q.Send(context.Background(), &testMessage{ID: 2})

	for i, want := range []string{"site-a", ""} {
		select {
		case got := <-gotSite:
			if got != want {
				t.Errorf("message %d: handler site = %q, want %q", i+1, got, want)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for handler")
		}
	}
}
