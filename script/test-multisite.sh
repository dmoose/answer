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
# Requires: a running multisite Answer instance at BASE_URL (default
# http://localhost:9080) and an admin bearer token in ANSWER_ADMIN_TOKEN.
#
# Sites are selected the way the SPA does it: X-Site-Slug. The test creates
# (or reuses) a second site, posts a question into it, and checks the
# question is visible there and only there. Re-runnable: the site and the
# question are found by slug/title on later runs.

BASE_URL="${BASE_URL:-http://localhost:9080}"
API="${BASE_URL}/answer/api/v1"
ADMIN_API="${BASE_URL}/answer/admin/api"
TOKEN="${ANSWER_ADMIN_TOKEN:?Set ANSWER_ADMIN_TOKEN to an admin bearer token}"
SITE_B_SLUG="${SITE_B_SLUG:-golang}"
MARKER="multisite-smoke-$(date +%s)"

AUTH="Authorization: ${TOKEN}"
CT="Content-Type: application/json"

pass=0
fail=0

check() {
  local desc="$1" expected="$2" actual="$3"
  if [[ "$actual" == *"$expected"* ]]; then
    echo "  PASS: $desc"
    pass=$((pass+1))
  else
    echo "  FAIL: $desc — expected '$expected', got '$actual'"
    fail=$((fail+1))
  fi
}

check_not() {
  local desc="$1" unexpected="$2" actual="$3"
  if [[ "$actual" != *"$unexpected"* ]]; then
    echo "  PASS: $desc"
    pass=$((pass+1))
  else
    echo "  FAIL: $desc — must not contain '$unexpected'"
    fail=$((fail+1))
  fi
}

json() { python3 -c "import sys,json; d=json.load(sys.stdin); print($1)" 2>/dev/null || echo ""; }

echo "=== Multi-site E2E smoke test ==="
echo "Target: $BASE_URL"
echo ""

# 1. Site list
echo "--- Site list ---"
sites=$(curl -sf "$API/sites" -H "$AUTH")
check "site list returns data" '"slug"' "$sites"

# 2. Second site: create, or reuse if the slug already exists
echo "--- Site B ($SITE_B_SLUG) ---"
SITE_B_ID=$(echo "$sites" | json "next((s['id'] for s in d.get('data',[]) if s.get('slug')=='$SITE_B_SLUG'),'')")
if [[ -z "$SITE_B_ID" ]]; then
  create=$(curl -sf -X POST "$ADMIN_API/site" -H "$AUTH" -H "$CT" \
    -d "{\"name\":\"Go Community\",\"slug\":\"$SITE_B_SLUG\",\"description\":\"Go Q&A\",\"base_url\":\"\"}")
  check "create site returns id" '"id"' "$create"
  SITE_B_ID=$(echo "$create" | json "d.get('data',{}).get('id','')")
else
  echo "  reusing existing site $SITE_B_ID"
fi

if [[ -z "$SITE_B_ID" ]]; then
  echo "  FAIL: no site B id, cannot run scoped tests"
  fail=$((fail+1))
else
  sites2=$(curl -sf "$API/sites" -H "$AUTH")
  count=$(echo "$sites2" | json "len(d.get('data',[]))")
  check "at least two sites exist" "ok" "$( [[ ${count:-0} -ge 2 ]] && echo ok || echo "count=$count" )"

  # 3. Unknown slug is a hard 404, never a silent default-site render
  echo "--- Unknown slug ---"
  code=$(curl -s -o /dev/null -w '%{http_code}' "$API/question/page?page=1&page_size=1" \
    -H "$AUTH" -H "X-Site-Slug: no-such-site-$MARKER")
  check "unknown X-Site-Slug returns 404" "404" "$code"

  # 4. Post a question into site B
  echo "--- Seed site B ---"
  post=$(curl -sf -X POST "$API/question" -H "$AUTH" -H "$CT" -H "X-Site-Slug: $SITE_B_SLUG" \
    -d "{\"title\":\"$MARKER\",\"content\":\"Isolation probe for $MARKER. Long enough to pass validation.\",\"tags\":[{\"slug_name\":\"smoke\",\"display_name\":\"smoke\",\"original_text\":\"smoke tag\",\"parsed_text\":\"smoke tag\"}]}")
  check "question created in site B" '"id"' "$post"

  # 5. Isolation: visible in B, invisible in the default site
  echo "--- Isolation ---"
  q_b=$(curl -sf "$API/question/page?page=1&page_size=20" -H "$AUTH" -H "X-Site-Slug: $SITE_B_SLUG")
  q_default=$(curl -sf "$API/question/page?page=1&page_size=20" -H "$AUTH")
  check "site B lists the question" "$MARKER" "$q_b"
  check_not "default site does not list it" "$MARKER" "$q_default"
fi

# 6. Network profile
echo "--- Network profile ---"
profile=$(curl -sf "$API/network/user/profile?user_id=1" -H "$AUTH")
check "network profile returns data" '"user_id"' "$profile"

echo ""
echo "=== Results: $pass passed, $fail failed ==="
[[ $fail -eq 0 ]] && exit 0 || exit 1
