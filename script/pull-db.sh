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

# Pull a consistent copy of a deployed tenant's SQLite database to
# answer-data/pulled/<tenant>/answer.db for local testing (make upgrade-check).
#
#   script/pull-db.sh <ssh-host> <tenant> [remote-tenant-dir]
#
# <ssh-host> is an ssh config alias; nothing about the box lives here.
# The remote tenant dir defaults to the infra repo's checkout layout and
# must contain volumes/answer-data/answer.db. The copy is taken with
# sqlite's online .backup through an ephemeral container (the volume is
# root-owned; the container runs as the docker-group user), so the live
# site is never paused and WAL contents are included.
set -euo pipefail

host=${1:?ssh host alias}
tenant=${2:?tenant name}
remote_dir=${3:-"~/infra/appServer1/tenants/${tenant}"}

out_dir="$(cd "$(dirname "$0")/.." && pwd)/answer-data/pulled/${tenant}"
mkdir -p "$out_dir"

remote_stage=$(ssh "$host" 'mktemp -d /tmp/answer-pull.XXXXXX')
trap 'ssh "$host" "rm -rf $remote_stage"' EXIT

ssh "$host" "docker run --rm \
  -v ${remote_dir}/volumes/answer-data:/ans:ro \
  -v ${remote_stage}:/out \
  alpine:3.22 sh -c 'apk add -q --no-cache sqlite \
    && sqlite3 /ans/answer.db \".backup /out/answer.db\" \
    && chmod 644 /out/answer.db'"

scp -q "$host:$remote_stage/answer.db" "$out_dir/answer.db"
echo "pulled $tenant -> $out_dir/answer.db"
sqlite3 "$out_dir/answer.db" "select 'version', version_number from version;"
sqlite3 "$out_dir/answer.db" "select 'fork_version', version_number from fork_version;" 2>/dev/null \
  || echo "fork_version: absent (legacy ledger layout)"
