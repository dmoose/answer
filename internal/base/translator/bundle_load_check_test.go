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

package translator

import (
	"path/filepath"
	"runtime"
	"testing"
)

// The go-i18n bundle has a single global key namespace: a bare key that is a
// string leaf anywhere in i18n/*.yaml cannot also be a nested map elsewhere.
// A violation crashes startup (the loader fails fast by design), so catch it
// here instead of at boot.
func TestBundleLoads(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate source file")
	}
	bundleDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "i18n")
	_, err := NewTranslator(&I18n{BundleDir: bundleDir})
	if err != nil {
		t.Fatalf("i18n bundle failed to load: %v", err)
	}
}
