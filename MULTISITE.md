# Multi-Site Support

This fork adds Stack Exchange-style multi-tenancy to Apache Answer: shared user identity across multiple independent Q&A communities, each with its own content, tags, reputation, and moderation roles.

## Quick Start

```bash
go build -o answer ./cmd/answer
```

Or use the provided Docker setup:

```bash
docker compose -f docker-compose.multisite.yaml build
docker compose -f docker-compose.multisite.yaml up -d
```

There is no separate build flavor: site scoping, site resolution and per-site cron passes are always compiled in. (Earlier revisions gated them behind a `multisite` build tag; the untagged variant ran the fork migrations but stamped no site on new content, so it was removed.)

## How It Works

### Site Resolution

Every request is resolved to a site via middleware, checked in order:

1. Subdomain (`golang.example.com` → slug `golang`)
2. Path prefix (`/s/golang/...` → slug `golang`)
3. `X-Site-Slug` header (validated against known sites; used by the SPA since API calls don't carry the basename)
4. `default` slug
5. Lowest-ID active site (safety net so the network keeps serving if the default ever gets deactivated)

`/answer/admin/api/**`, `/static/**`, `/install/**`, and `/healthz` skip resolution entirely so the admin UI can recover the network even when routing is broken.

The resolver records both the site ID and the slug it matched on the request (`multisite.SiteIDFromContext` / `multisite.SiteSlugFromContext`); the slug is what the OIDC login link carries so the callback can return to the originating sub-site.

The frontend uses path-based routing. Navigating to `/s/golang` sets the React Router basename and sends `X-Site-Slug` on every API call.

### Data Model

Three tiers of `site_id`:

| Tier | Value | Purpose |
|------|-------|---------|
| Global defaults | `''` (empty) | Config and site_info inherited by all sites |
| Default site | `'1'` | Content belonging to the default site |
| Per-site | `'{id}'` | Content and config overrides for other sites |

Twelve content tables are site-scoped (`question`, `answer`, `comment`, `tag`, `tag_rel`, `activity`, `revision`, `review`, `report`, `meta`, `question_link`, `file_record`). `config`, `site_info` and `plugin_config` carry `site_id` for the tier model above. The `user`, `role`, `power`, and `badge` tables remain global (shared identity), and so do the per-user surfaces: notifications, badge awards, and collections (bookmarks) belong to the person, not a sub-site; their `site_id` columns exist but are vestigial.

### Query Scoping

- `Data.SiteDB(ctx)` returns a session with `WHERE site_id = ?` from request context
- `Data.SiteInsert(ctx, entity)` sets `SiteID` on entities before insertion (handles structs and slices)
- `multisite.Scope(session, ctx)` adds site filtering to transaction sessions
- `multisite.WithSiteID(ctx, id)` rebuilds site scope on derived/background contexts
- The async queue captures the enqueuing request's site in an envelope and rebuilds it before each handler runs, so queue-routed writes (activity attribution, email rendering) keep their site
- Per-site cron work (`SitemapCron`) iterates active sites, each pass under that site's context
- JOIN queries use table-qualified `WHERE table.site_id = ?` to avoid ambiguity
- Raw SQL builders (search, plugin-search hydration) add site_id INSIDE the builder — xorm drops session-level `Where()` when `Query()` is handed a raw builder

### Reputation — global, with per-site attribution retained

Reputation is **global**: one rank per user, earned anywhere, gating privileges identically on every sub-site (upstream's native model). The daily rank limit is likewise global. The per-site rank layer (`user_site_rank`) is retired — dropped by migration `fork-003`.

**Invariant (do not regress):** every `activity` and vote row keeps its originating `site_id`. Privilege is computed from the global sum, but the per-site attribution stays queryable — `SUM(rank) GROUP BY site_id` per user — so a per-site "reputation earned here" display (Option C) is a later query plus UI, never a re-migration.

### Per-Site Roles

The `user_site_role_rel` table stores per-site role assignments — board authority is an appointment, fully decoupled from reputation. Role resolution cascades: site role → global role → most privileged wins (a site role can raise, never lower). Resolution happens **per request** in the auth middleware — the token cache stores only the global role, because a role baked at login time would carry one site's escalation to every other board. Network-global authority (the admin API, the admin token cache) gates on the global role only. Role read errors fail closed (demote to plain user), never open.

### Migration

Fork migrations live in `internal/migrations/fork*.go` with their own ledger (`fork_version` table, `forkMigrations` list) so upstream's index-tracked list stays byte-for-byte upstream. `answer upgrade` runs upstream's pending migrations first, then the fork's. Fork migrations are named `fork-NNN`, never upstream semver, and must be idempotent because they run after whatever upstream migrations landed in the same upgrade. Adding one means appending to `forkMigrations`; upstream merges never touch the order.

Databases migrated before the split (`version` at 33 upstream + N fork) are converted once on the next upgrade: `version` is reset to 33 and `fork_version` set to N, then any pending fork migrations run. Any other layout with a `site` table present refuses to upgrade until inspected, and `answer upgrade` exits non-zero so the container does not start against it.

`fork-001` (multi-site support) handles existing installs:
- Creates `site` and `user_site_role_rel` via entity Sync
- Adds `site_id` columns/indexes with explicit, idempotent DDL quoted through xorm's dialect quoter, so it runs on MySQL, Postgres and SQLite (xorm Sync is used only to CREATE missing tables — its ALTER path emits MySQL-only syntax). The Postgres path is pinned by `TestMultisiteMigrationOnPostgres`, which runs when `ANSWER_TEST_POSTGRES_DSN` points at a scratch database (CI provides one).
- Inserts a default site (guarded, idempotent)
- Backfills content tables to the default site
- Backfills each user's global role onto the default site in Go (portable, idempotent, most privileged role wins for multi-role users)
- Leaves config/site_info as global defaults
- Drops single-column unique indexes, replaced by composite (column, site_id) with names matching a fresh install

`fork-003` (repair) fixes databases that ran the original multi-site migration, whose dialect-specific steps silently failed on SQLite: it converges the schema (missing composite unique indexes), re-runs the role backfill, and drops the retired `user_site_rank` table. It refuses loudly if duplicate rows would violate the composite uniques rather than deleting content.

Fresh installs get the correct schema from `InitDB` with the default site seeded first and both ledgers set to their expected values. Migration behavior is pinned by `internal/migrations/fork_migration_test.go`, including fresh-vs-upgraded index convergence and the ledger split.

## Admin

### API

```
POST /answer/admin/api/site          — create site
PUT  /answer/admin/api/site          — update site
PUT  /answer/admin/api/site/status   — activate / suspend (the default site cannot be suspended)
GET  /answer/admin/api/site          — get site by id
GET  /answer/admin/api/sites         — list all sites
PUT  /answer/admin/api/site/role     — assign per-site role
GET  /answer/admin/api/site/role     — get user's site role
```

### UI

Admin > Advanced > Sites — table of all sites with Edit and Assign Role actions.

### Public API

```
GET /answer/api/v1/sites                          — list active sites
GET /answer/api/v1/network/user/profile?user_id=  — cross-site reputation
```

## Frontend

- **Site context picker** — card-style block at the top of the desktop left nav (icon + site name + chevron; descriptions only in the open dropdown). Hidden when only one site exists. The legacy header dropdown is preserved for mobile via `.d-lg-none`.
- **Dynamic basename** — React Router uses `/s/{slug}` as basename when on a site path.
- **Request interceptor** — `X-Site-Slug` header sent unconditionally on every API call from the active site context. (Earlier logic skipped it when the SPA URL had `/s/<slug>`, but API requests go to `/answer/api/v1/...` without the basename, so the server saw no slug and fell back to the default site.)
- **Site-aware search placeholder** — search input shows `Search <site name>` so the scope is visible after a switch.
- **Search scope toggle** — when more than one site exists, the search results page renders a "This site / All sites" toggle next to the sort. Network-scope results carry a small site-name badge so cross-site origin is visible at a glance.
- **App switcher** — 3×3 grid dropdown pinned to the header far-right. Admin-managed list under **Admin → Customize → App switcher** (`app-switcher` site_info type, forced global regardless of site context — one list shared across every tenant). Hidden until enabled with at least one link.
- **Site store** — Zustand store resolves the current site from URL path or subdomain.

## Plugin Compatibility

Existing plugins work without changes. They query via `plugin.Data.DB.Context(ctx)` which doesn't filter by site — they see all data. This is correct for storage, cache, notification, and captcha plugins.

For site-aware plugins:
- `SearchContent.SiteID` and `VectorSearchContent.SiteID` fields are populated during index sync
- `plugin.SiteRoleRepo` interface available for role-aware plugins

## Fastgate OIDC Connector

A built-in Connector plugin for [fastgate](https://git.6-p.cc/catapulsion/fastgate) SSO is included at `internal/plugins/fastgate_connector/`. It implements the standard OIDC authorization code flow.

### Setup

1. Register Answer as an OIDC client in fastgate:
   ```bash
   fgctl oidc-client -create -name "Answer" \
     -redirect-uri "https://answer.example.com/answer/api/v1/connector/redirect/fastgate-connector"
   ```

2. In Answer admin, go to Installed Plugins > Fastgate and configure:
   - **Issuer URL** — fastgate base URL (e.g. `https://gate.example.com`)
   - **Client ID** and **Client Secret** — from step 1

3. To make fastgate the only login method, disable under Admin > Login:
   - Allow new registrations — OFF
   - Allow email registration — OFF
   - Password login — OFF

   The connector button remains and auto-creates accounts on first login regardless of these toggles.

### Auth Flow

Browser → Answer login → redirect to fastgate `/authorize` → magic link email → click → fastgate session → redirect back to Answer with auth code → Answer exchanges code for tokens via `/token` → fetches user info via `/userinfo` → creates/logs in user.

A login started on a sub-site returns there: the login link carries `?site=<slug>`, the core keeps it in the OAuth state, and the callback lands on `/s/<slug>/users/auth-landing`. Connector URLs (login links, `redirect_uri`, landing) are always built from the global site URL, never a sub-site's General override. See `FASTGATE_INTEGRATION.md` §1.

Existing fastgate sessions enable true SSO — if already authenticated with fastgate (e.g. from another app), the login is instant with no email prompt. Answer logout clears the Answer session only; the fastgate session persists (standard SSO behavior).

### Handle = Answer username

The fastgate `preferred_username` claim becomes the Answer username verbatim — no transform, no dedup suffix, no random fallback. Username edits are disabled in both UI and API for fastgate-bound users; the IdP owns the handle. See `FASTGATE_INTEGRATION.md` §3 for the full spec, `plugin/connector.go` for the `UsernameAuthoritative` contract, and `internal/service/content/user_service.go`'s `authoritativeUsernameProviders` map for the connector list.

## Scoping Doctrine

Two data axes govern every scoping decision. **Content is site-scoped**: questions, answers, comments, tags, per-object activity, content search, sitemaps belong to a sub-site and must not mingle. **Identity is global**: users, profiles, the member directory, reputation, badges, notifications, and bookmarks are properties of the person and span the whole instance. Cross-site content bleed is a correctness/UX defect, never a confidentiality breach — every authenticated user can see every sub-site by design (single trust domain).

- **Identity is global; content is per-site.** `user`, `role`, `power`, `badge` catalog, `api_key`, `user_notification_config`, `user_external_login`, network directory tables (`network_profile`, `network_project`, `profile_tag`, `member_directory`) are all unscoped. So are the per-user surfaces: **notifications (one inbox), badge awards (one achievement set), and collections (one bookmark list)** — their repos are deliberately un-scoped and their entity `site_id` columns are vestigial.
- **`config` is purely global.** Functional config (rank thresholds, limits, reasons, SMTP) is org-wide: one row set (`site_id=''`), one cache key, no per-site override branch. A config override would also fork the auto-increment ID that activity rows store as `activity_type`, orphaning history.
- **`site_info` is per-site for presentation only.** The per-site override surface is `general` (name), `branding` (logo/favicon), and `css-html` (the injection bundle); everything else resolves the global row. The repo enforces this allowlist on both reads and writes (`presentationTypes` in `internal/repo/site_info`), so a functional type saved from a site context lands on the global row and a stray override row for one is ignored. An override for a site that does not exist is refused. Overrides are wholesale per type: a site that defines `css-html` authors the whole bundle, a site without an override inherits the global row entirely. Admin saves target a site explicitly via `?site_id=` on the general/branding/custom-css-html endpoints (the admin API itself stays site-less); the admin UI exposes this as the "Editing site" picker. Cache entries are keyed by the tier the row belongs to — override rows under site keys, the global row under the global key, plus a short "no override" marker — so a global edit is immediately visible on every non-overridden site.
- **`plugin_config` is purely global.** Plugin runtime is process-wide and applies config once at startup. Earlier site-scoped writes silently hid admin saves from the loader — `SavePluginConfig` now always writes the global row regardless of context.
- **The default site cannot be deactivated.** `SiteService.SetSiteStatus` refuses to disable `DefaultSiteID`. The resolver also falls back to the lowest-ID active site if the default ever goes missing.
- **An explicitly named unknown site is a 404.** `/s/typo` and a bogus `X-Site-Slug` return not-found (JSON for API calls, the SPA shell with 404 status for navigations) instead of silently serving the default site under the wrong URL. Only an ABSENT site signal falls back to the default. The subdomain heuristic stays silent on a miss — the first host label usually names the deployment, not a site (deployment-ish labels like `answer`, `www` are also reserved slugs).
- **Search is site-scoped by default; opt-in network scope via `?scope=network`.** Server validates the param at the DTO layer (`oneof=site network`) and clears the site_id from context before hitting the repo. The plugin search path receives `SiteID` in `SearchBasicCond` (plugins MUST filter at the index level for correct totals) and the hydration query carries the site condition inside the raw builder. Each result carries its origin `site_id` so the UI can render a cross-site badge; cross-site results link absolutely into their owning site.
- **The member directory follows `directory_visibility`.** `public` or `members-only` (default members-only) in service config, alongside `directory_enabled`. Directory avatars resolve through `FormatAvatar` like every other public surface — never the raw stored JSON blob.
- **Admin dashboard counts are network-wide.** The admin API is site-less by design; a network admin sees network totals. Deliberate, not a leak.

## i18n authoring gotcha

`i18n/*.yaml` is loaded by `LinkinStars/go-i18n` via segmentfault/pacman. Bare key names share a global namespace across the whole bundle: if `foo` appears anywhere as a string leaf, `foo` cannot appear elsewhere as a nested map (and vice-versa). The parse error is `expected value for key "<name>" be a string but got map[...]`. **Flatten new keys with underscores instead of nesting** (`enabled_label: "..."`, not `enabled: { label: "..." }`).

The translator loader now fails fast on bundle errors — bad YAML crashes startup loudly instead of leaving a nil localizer that 502s every request. Earlier upstream behavior was to `log.Debugf` and continue.

## Known Limitations

- **Site role only escalates** — a user's per-site role takes effect only when it is more privileged than the global role; a site role cannot demote a global admin. By design (network admin overrides everywhere), but worth noting.
- **Uploads are not site-partitioned** — one physical `/uploads/` tree serves all sub-sites (hashed filenames, public assets). `file_record` rows are site-stamped; the files are not.
- **Badge awards are global** — one achievement set per person; `badge_award.site_id` is vestigial and threshold rules (e.g. "10 accepted answers") count across all sites, like reputation.
- **`Site.base_url` is honored for links, not for resolution** — the UI builds every cross-site URL through `siteURL()` (`ui/src/utils/siteUrl.ts`) and the OIDC landing looks the site up, so a sub-site with an admin-set base URL is linked at that host. Resolving a request to that site still relies on the subdomain heuristic (first host label = slug); a base URL whose host label is not the slug will not resolve.
- **Single-instance state** — site routing reads an in-process slug→id map refreshed only by the instance that handled the admin change, and the fastgate connector keeps its in-flight login records in memory. Run one replica.

## Future: Plugin Page Framework

The Agent plugin interface (`RegisterAuthUserRouter`) can serve templ/htmx content from custom routes. Combined with Answer's `custom_head` injection (for loading htmx) and a generic React Router catch-all, this enables server-rendered plugin pages inside the React shell:

1. Load htmx via Admin > Customize > Custom Head
2. Agent plugin registers `/guild/api/*` routes serving templ partials
3. A single React route at `/guild/*` renders an htmx container that loads from the plugin

This pattern allows plugins to add full pages (member directory, resource library, office hours) without modifying the React codebase. The plugin serves HTML via Go handlers, htmx handles interactivity, and the React shell provides navigation and auth context.

## Build & Test

```bash
go build ./...
go vet ./...
go test ./...

# Postgres migration tests (skipped without the DSN; the database is wiped)
docker run -d --name answer-pg -e POSTGRES_USER=answer -e POSTGRES_PASSWORD=answer -e POSTGRES_DB=answer -p 127.0.0.1:55432:5432 postgres:16-alpine
ANSWER_TEST_POSTGRES_DSN="host=127.0.0.1 port=55432 user=answer password=answer dbname=answer sslmode=disable" go test ./internal/migrations/

# Multisite smoke test against a running instance (see script header)
ANSWER_ADMIN_TOKEN=<token> BASE_URL=http://localhost:9080 script/test-multisite.sh

# Try an upgrade against a copy of a live database
make pull-db HOST=<ssh alias> TENANT=<tenant>
make upgrade-check DB=answer-data/pulled/<tenant>/answer.db

# Docker (includes UI embed guard)
docker compose -f docker-compose.multisite.yaml build --no-cache
docker compose -f docker-compose.multisite.yaml up -d
```

The Dockerfile uses a separate `node:22-alpine` stage for the UI build and validates that `index.html` exists before the Go embed step.
