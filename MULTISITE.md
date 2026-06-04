# Multi-Site Support

This fork adds Stack Exchange-style multi-tenancy to Apache Answer: shared user identity across multiple independent Q&A communities, each with its own content, tags, reputation, and moderation roles.

## Quick Start

Build with the `multisite` tag:

```bash
go build -tags multisite -o answer ./cmd/answer
```

Or use the provided Docker setup:

```bash
docker compose -f docker-compose.multisite.yaml build
docker compose -f docker-compose.multisite.yaml up -d
```

Without the `multisite` tag, the binary behaves identically to upstream Answer.

## How It Works

### Site Resolution

Every request is resolved to a site via middleware, checked in order:

1. `X-Site-ID` header (validated against known sites)
2. Subdomain (`golang.example.com` → slug `golang`)
3. Path prefix (`/s/golang` → slug `golang`)
4. Falls back to the default site

The frontend uses path-based routing. Navigating to `/s/golang` sets the React Router basename and sends `X-Site-ID` on all API calls.

### Data Model

Three tiers of `site_id`:

| Tier | Value | Purpose |
|------|-------|---------|
| Global defaults | `''` (empty) | Config and site_info inherited by all sites |
| Default site | `'1'` | Content belonging to the default site |
| Per-site | `'{id}'` | Content and config overrides for other sites |

Twenty content tables carry `site_id`. The `user`, `role`, `power`, and `badge` tables remain global (shared identity).

### Query Scoping

- `Data.SiteDB(ctx)` returns a session with `WHERE site_id = ?` from request context
- `Data.SiteInsert(ctx, entity)` sets `SiteID` on entities before insertion (handles structs and slices)
- `multisite.Scope(session, ctx)` adds site filtering to transaction sessions
- JOIN queries use table-qualified `WHERE table.site_id = ?` to avoid ambiguity
- Raw SQL builders (search) add site_id to the query conditions directly
- Config and site_info repos use unscoped queries with fallback to global defaults

### Per-Site Reputation

Privilege checks use site-specific rank, not global rank. When a user earns reputation (votes, accepted answers), both `user.Rank` (global aggregate) and `user_site_rank` (per-site) are updated. A user with high reputation on one site has no special privileges on another.

### Per-Site Roles

The `user_site_role_rel` table stores per-site role assignments. Role resolution cascades: site role → global role → most privileged wins. A network admin has full access everywhere. A site admin only has admin access on their assigned site.

### Migration

The v33 migration handles existing installs:
- Adds `site_id` column to 20 tables
- Creates `site`, `user_site_rank`, `user_site_role_rel` tables
- Inserts a default site
- Backfills content tables to the default site
- Leaves config/site_info as global defaults
- Drops single-column unique indexes, replaced by composite (column, site_id)

Fresh installs get the correct schema from `InitDB` with the default site seeded first.

## Admin

### API

```
POST /answer/admin/api/site          — create site
PUT  /answer/admin/api/site          — update site
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

- **Site switcher** — dropdown in the header showing all sites, navigates via `/s/{slug}` URLs
- **Dynamic basename** — React Router uses `/s/{slug}` as basename when on a site path
- **Request interceptor** — `X-Site-ID` header sent on all API calls from the current site context
- **Site store** — Zustand store resolves current site from URL path or subdomain

## Plugin Compatibility

Existing plugins work without changes. They query via `plugin.Data.DB.Context(ctx)` which doesn't filter by site — they see all data. This is correct for storage, cache, notification, and captcha plugins.

For site-aware plugins:
- `SearchContent.SiteID` and `VectorSearchContent.SiteID` fields are populated during index sync
- `plugin.SiteRoleRepo` interface available for role-aware plugins

## Known Limitations

- **Private mode API leak** — upstream registers content routes in the `MustUnAuth` group which bypasses `login_required`. Needs route group fix for true API-level content protection.
- **Network profile frontend** — API exists at `/network/user/profile`, no React page yet.
- **Migration v33 error handling** — index creation errors are logged but not distinguished from unexpected failures.

## Future: Plugin Page Framework

The Agent plugin interface (`RegisterAuthUserRouter`) can serve templ/htmx content from custom routes. Combined with Answer's `custom_head` injection (for loading htmx) and a generic React Router catch-all, this enables server-rendered plugin pages inside the React shell:

1. Load htmx via Admin > Customize > Custom Head
2. Agent plugin registers `/guild/api/*` routes serving templ partials
3. A single React route at `/guild/*` renders an htmx container that loads from the plugin

This pattern allows plugins to add full pages (member directory, resource library, office hours) without modifying the React codebase. The plugin serves HTML via Go handlers, htmx handles interactivity, and the React shell provides navigation and auth context.

## Build & Test

```bash
# Go builds (both modes)
go build ./...
go build -tags multisite ./...

# Tests
go test -tags multisite ./internal/multisite/ ./internal/service/site/ ./internal/service/rank/ ./internal/service/role/

# Docker (includes UI embed guard)
docker compose -f docker-compose.multisite.yaml build --no-cache
docker compose -f docker-compose.multisite.yaml up -d
```

The Dockerfile uses a separate `node:22-alpine` stage for the UI build and validates that `index.html` exists before the Go embed step.
