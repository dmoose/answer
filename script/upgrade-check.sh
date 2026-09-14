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

# Run `answer upgrade` from the current tree against a COPY of a SQLite
# database and show the migration ledgers before, after, and after a second
# (must be no-op) run. The source file is never touched.
#
#   script/upgrade-check.sh <path-to-answer.db> [binary]
#
# The binary defaults to ./answer-dev, built from this tree if absent.
set -euo pipefail

src=${1:?path to answer.db}
bin=${2:-./answer-dev}
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

[ -f "$src" ] || { echo "no such file: $src" >&2; exit 1; }
if [ ! -x "$bin" ]; then
  echo "building $bin"
  go build -o "$bin" ./cmd/answer
fi

stage=$(mktemp -d "${TMPDIR:-/tmp}/answer-upgrade-check.XXXXXX")
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/conf" "$stage/cache" "$stage/uploads" "$stage/i18n"
cp "$src" "$stage/answer.db"
cat > "$stage/conf/config.yaml" <<EOF
server:
  http:
    addr: 127.0.0.1:0
data:
  database:
    driver: sqlite3
    connection: $stage/answer.db
  cache:
    file_path: $stage/cache/cache.db
i18n:
  bundle_dir: $stage/i18n
swaggerui:
  show: false
service_config:
  upload_path: $stage/uploads
EOF

ledgers() {
  sqlite3 "$stage/answer.db" "select 'version', version_number from version;"
  sqlite3 "$stage/answer.db" "select 'fork_version', version_number from fork_version;" 2>/dev/null \
    || echo "fork_version absent"
}

echo "== before"; ledgers
echo "== upgrade"; "$bin" upgrade -C "$stage/"
echo "== after"; ledgers
echo "== second upgrade (expect no migrations)"; "$bin" upgrade -C "$stage/"
echo "== after second"; ledgers
echo "== tables: $(sqlite3 "$stage/answer.db" "select count(*) from sqlite_master where type='table'")"
