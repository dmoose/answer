#!/usr/bin/env bash
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

set -euo pipefail

# E2E smoke test for multisite support.
# Requires: a running Answer instance at BASE_URL (default http://localhost:9080)
# and an admin API token.

BASE_URL="${BASE_URL:-http://localhost:9080}"
API="${BASE_URL}/answer/api/v1"
ADMIN_API="${BASE_URL}/answer/admin/api"
TOKEN="${ANSWER_ADMIN_TOKEN:?Set ANSWER_ADMIN_TOKEN to an admin bearer token}"

AUTH="Authorization: ${TOKEN}"
CT="Content-Type: application/json"

pass=0
fail=0

check() {
  local desc="$1" expected="$2" actual="$3"
  if [[ "$actual" == *"$expected"* ]]; then
    echo "  PASS: $desc"
    ((pass++))
  else
    echo "  FAIL: $desc — expected '$expected', got '$actual'"
    ((fail++))
  fi
}

echo "=== Multi-site E2E smoke test ==="
echo "Target: $BASE_URL"
echo ""

# 1. List sites (should have at least the default)
echo "--- Site list ---"
sites=$(curl -sf "$API/sites" -H "$AUTH")
check "site list returns data" '"slug"' "$sites"

# 2. Create a second site
echo "--- Create site ---"
create=$(curl -sf -X POST "$ADMIN_API/site" \
  -H "$AUTH" -H "$CT" \
  -d '{"name":"Go Community","slug":"golang","description":"Go Q&A","base_url":""}')
check "create site returns id" '"id"' "$create"
SITE_B_ID=$(echo "$create" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null || echo "")

if [[ -z "$SITE_B_ID" ]]; then
  echo "  WARN: could not extract site ID, skipping site-scoped tests"
else
  echo "  Site B ID: $SITE_B_ID"

  # 3. List sites again — should have 2
  echo "--- Verify two sites ---"
  sites2=$(curl -sf "$API/sites" -H "$AUTH")
  count=$(echo "$sites2" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('data',[])))" 2>/dev/null || echo "0")
  check "two sites exist" "2" "$count"

  # 4. Query with X-Site-ID header
  echo "--- Site-scoped query ---"
  q_default=$(curl -sf "$API/question/page?page=1&page_size=1" -H "$AUTH")
  q_golang=$(curl -sf "$API/question/page?page=1&page_size=1" -H "$AUTH" -H "X-Site-ID: $SITE_B_ID")
  check "default site returns questions" '"list"' "$q_default"
  check "new site returns empty list" '"list":[]' "$q_golang"
fi

# 5. Network profile
echo "--- Network profile ---"
profile=$(curl -sf "$API/network/user/profile?user_id=1" -H "$AUTH")
check "network profile returns data" '"user_id"' "$profile"

echo ""
echo "=== Results: $pass passed, $fail failed ==="
[[ $fail -eq 0 ]] && exit 0 || exit 1
