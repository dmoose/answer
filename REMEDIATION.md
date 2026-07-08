# Multisite Fork — Remediation Spec

**Status: IMPLEMENTED 2026-07-05** (all phases; see Appendix A0 for verification amendments and final decisions, and "Implementation notes" below for deviations). Both build flavors compile, vet, and pass the full test suite; frontend passes tsc and eslint. Not yet verified in a running instance — do that before any deploy, especially the v35 repair against a copy of a live DB.

**Implementation notes — deviations from the spec as written:**

- **Config went fully global** (reads and writes), not just "quarantine the override branch": one row set, one cache key, no site cascade. Stray override rows on live DBs are ignored by reads (test-pinned).
- **REP-2 needed more than specced:** the effective role was baked into the token cache at login, so per-site roles now resolve **per request** in the auth middleware; the token/admin caches store only the **global** role (a site-admin of A could otherwise mint a network-admin session by logging in via A).
- **v33 was rewritten AND v35 appended** — live DBs (catapulsion, guild) already ran the broken v33, so the repair ships as a new migration; v33's fix covers only future vanilla→multisite upgrades. Discovered en route: xorm 1.3.2's `Sync` emits MySQL-only `ALTER TABLE MODIFY` on any perceived column drift, so existing tables get explicit idempotent DDL and Sync is used only for table creation.
- **IDENT-2 resolved via the queue envelope:** emails render with the originating site's settings (the recommended option).
- **Collections/bookmarks de-scoped to identity-global** per decision; notifications and badge awards likewise.
- **AUTH-4 reframed:** no email hash was ever exposed (sha256, computed at format time, never stored); fixed as a correctness issue by routing through `FormatAvatar`.
- **`access_token` in the auth-landing redirect URL kept** — upstream's standard mechanism for all connectors; changing it means changing the shared frontend auth-landing flow.
- **Left deliberately undone:** SchemaForm parity for the admin Sites/AppSwitcher/Network forms and the site-switcher dedupe (QUAL-7 tail) — UI rework that needs visual verification; the forms did get i18n. String PKs declared `autoincr` on fork entities left as-is (changing live PK semantics isn't worth the churn; xorm materializes them as INTEGER PRIMARY KEY and string scanning works). Swagger/docs regeneration for the new/changed admin routes (`PUT /site/status`, `site_id` params) not run — `make generate` is currently broken on the local toolchain (wire v0.5.0 panics under modern Go; consider bumping to v0.6.0). CONTENT-1's plugin-path scoping has unit-level coverage only via the native-pattern mirror; verify end-to-end if a search plugin is ever enabled.

**Audit basis:** Fork point `2746bf5b` (still the tip of upstream `main`) vs working tree at `10c74dde`. Findings were verified against actual code and against vendored `xorm.io/xorm@v1.3.2` behavior.

> **This spec was recut after clarifying the real use case.** The first pass audited the fork as a hostile multi-tenant SaaS and over-weighted cross-tenant isolation. That threat model does not apply here. Read the Doctrine section first — it changes the direction of roughly half the original findings, not just their priority.

---

## 0. Doctrine — what this fork actually is

One Answer instance per **organization** (`answer-catapulsion`, `answer-guild`, `answer-aeyrea`; separate containers, separate SQLite volumes — see `../../infra/design.md`). Inside a single instance, "multisite" means **sub-sites** for different products / interest areas within that one org. It is **not** a SaaS hosting mutually-distrusting tenants.

Consequences that govern every decision below:

- **Single trust domain.** There is one admin authority per instance. A network admin is trusted across all sub-sites by design. "A site-admin can act network-wide" is the intended model, not a privilege escalation.
- **Every authenticated user can see every sub-site.** Content partitioning exists so product boards don't *mingle* in the UI, not to hide content from users. Cross-site content bleed is therefore a **correctness / UX** defect, never a confidentiality breach.
- **Two data axes:**
  - **Content is site-scoped.** Questions, answers, comments, tags, per-object activity/timelines, content search, sitemaps — these belong to a sub-site and must not mingle across sub-sites.
  - **Identity is global.** Users, profiles, the member directory, reputation, badges, and the notification inbox are properties of the *person* and span the whole instance. The user pages **are** the org's shared directory; that is the point of the fork.
- **The fork's core mistake was applying site-scoping uniformly across both axes.** On identity surfaces the bug is usually the opposite of what the first audit implied: not a missing site filter, but a site filter that should not be there.
- **Database reality:** SQLite (WAL) today, Postgres a possible future, **MySQL never**. Migrations must be correct on SQLite now and Postgres-ready before that move. MySQL-only DDL is a defect aimed at the one engine that will never run.

### Reputation model — DECIDED: global reputation + per-site roles (Option B), C-ready

Reputation is **global**. Upvotes/accepts count everywhere, one rank per user gates privileges across all sub-sites, and that rank is what the directory surfaces. This is upstream's native rank model; the fork's per-site rank layer is removed as a *privilege* source.

Board authority is a separate concept: **moderator/admin of a sub-site is an appointment, not earned rank.** Keep `user_site_role_rel` purely as role assignment, fully decoupled from reputation.

**Architect for Option C (per-site standing as display) without a future migration.** The rule that makes C free later:

> Privilege is computed from the **global** reputation sum, but activity/vote records **retain their originating `site_id`**.

So: drop per-site rank from *gating*, but do **not** drop `site_id` from `activity`/vote rows. That column is the substrate for a later read-only feature that shows "820 rep — 500 in Security, 320 in Gaming" as a directory/board display, implemented as `SUM(delta) GROUP BY site_id` per user. No schema change, no re-migration — C becomes a query and some UI whenever you want it.

---

## How to use this document

Issues have stable IDs (`AUTH-1`, `REP-1`, …) for commit/PR references. Work the phases in order; within a phase, items are dependency-sorted. Each issue states **Symptom → Root cause (file:line) → Fix → Tests → Done when**.

### Repo conventions you must respect

- **Build tag:** the whole multisite feature is behind `//go:build multisite` (noop `*_default.go` for vanilla builds; `Dockerfile.multisite` builds `-tags multisite`). Everything you touch must compile and pass both ways:
  - `go build ./...` **and** `go build -tags multisite ./...`
  - `go vet ./...` **and** `go vet -tags multisite ./...`
  - `go test ./...` **and** `go test -tags multisite ./...` (the plain form silently tests the noop build)
- `internal/repo/repo_test` falls back to SQLite in `os.TempDir()` when `TEST_DB_DRIVER` is unset — real-DB tests run with no external setup. This is also the harness to reproduce SQLite migration behavior.
- **Scoping primitives:** site-scoped reads/writes go through `Data.SiteDB(ctx)` / `Data.SiteInsert(ctx, ...)` / `multisite.Scope(...)`, keyed off the site on the context. `multisite.SiteIDFromContext` (`internal/multisite/scope.go:33-45`) reads it from a `*gin.Context` (`constant.SiteIDFlag`, `ctx_flag.go:25`) **or** a context explicitly carrying `constant.SiteIDContextKey` (`ctx_flag.go:33`) — the middleware only sets the gin key (`site.go:156`), so any derived/background context loses it (see IDENT-1).
- **Migrations are positional:** `ExpectedVersion() = minDBVersion + len(migrations)`; the runner executes `migrations[currentDBVersion-minDBVersion:]`. Append only; never reorder or insert mid-slice.
- Don't kill running dev servers; hand builds back to the operator.

---

# PHASE 1 — Real security bugs (threat-model-independent)

These are genuine auth defects that stand on their own regardless of tenancy. They are the only true "security" items left after the doctrine recut. Membership is closed (fastgate is your own IdP), which mitigates severity, but these are cheap and correct to fix.

## AUTH-1 — OIDC login is CSRF / session-fixation open
**Severity:** High

**Symptom:** An attacker completes `/authorize` for their own fastgate account, captures the unused `code`, and gets a victim's browser to hit the redirect endpoint — the victim is silently logged into the **attacker's** account.

**Root cause:** `internal/plugins/fastgate_connector/connector.go:83` sets `"state": {receiverURL}` — a constant — and `ConnectorReceiver` never validates `state`. No nonce, no PKCE. The `id_token` (`connector.go:123`) is decoded but never verified.

**Fix:**
1. Per-request random `state`, stored server-side (short-TTL cache) bound to the browser; reject on mismatch/replay.
2. Add `nonce`; validate against the ID token.
3. Add PKCE (S256) — confirm fastgate supports it (it's your IdP).
4. Validate the `id_token`: signature via JWKS, `iss`, `aud`, `exp`/`iat`.

**Tests:** missing/mismatched/replayed `state` rejected; tampered/expired/wrong-`aud` ID token rejected; happy path binds the correct account.

**Done when:** a login without a matching server-issued `state` and a valid ID token cannot establish a session.

## AUTH-2 — OIDC trusts email without `email_verified`; userinfo status unchecked
**Severity:** Medium (mitigated by fastgate controlling emails)

**Symptom:** A fastgate identity with an unverified/editable email set to an existing member's address gets bound to and activates that account (core binds by email alone: `user_external_login_service.go:151-166`).

**Root cause:** `connector.go:136-151` doesn't check `uiResp.StatusCode` before decoding and forwards `claims.Email` unconditionally.

**Fix:** check the userinfo HTTP status; require `email_verified == true` before allowing email-based binding to a pre-existing account; document in `FASTGATE_INTEGRATION.md` that fastgate emails are pre-verified (making the invariant explicit rather than assumed).

**Tests:** non-200 userinfo → login fails; `email_verified:false`/absent → no bind-by-email; verified → normal bind.

**Done when:** an unverified IdP email can never bind to or activate a pre-existing local account.

## AUTH-3 — Reserved-username bypass by case
**Severity:** Low-Medium (mitigated by controlled handles)

**Symptom:** A fastgate handle `Admin` becomes a live username and, on case-sensitive stores (SQLite/Postgres `UNIQUE` on `user.username`), coexists with `admin`.

**Root cause:** `applyAuthoritativeUsername` (`user_external_login_service.go:210-227`) calls `checker.IsReservedUsername` (`pkg/checker/reserved_username.go:55`) — exact-match map lookup, no case folding — and the verbatim-handle contract does no lowercasing.

**Fix:** case-fold the reserved-name comparison in `IsReservedUsername` (normalize to lower before lookup). Leave the legit verbatim-handle behavior for non-reserved names intact.

**Tests:** `Admin`/`ADMIN`/`aDmIn` → reserved; `Alice` still allowed; existing reserved tests green.

**Done when:** no case-variant of a reserved username can be claimed via SSO or local signup.

## AUTH-4 — Member directory leaks `md5(email)`
**Severity:** Medium

**Symptom:** `GET /network/members` (anonymous) returns `user.avatar` as the raw stored JSON blob whose `gravatar` field embeds `md5(email)` — harvestable for offline reversal. Also renders broken avatars on the frontend (JSON string used as `<img src>`).

**Root cause:** `internal/repo/member_directory/directory_repo.go:125` selects `user.avatar` verbatim instead of routing through `FormatAvatar` (`user_common/user.go:95`) like every other public surface.

**Fix:** run directory avatars through `FormatAvatar` so the API returns a plain URL, never the hash blob. (Directory data is intentionally instance-global — no site scoping needed or wanted here; see DIR-1 for the separate visibility question.)

**Tests:** `/network/members` response contains no email hash and avatar is a URL; frontend cards render.

**Done when:** no email-derived hash appears in any directory response.

---

# PHASE 2 — Reputation re-architecture (Option B, C-ready)

This is the centerpiece of the recut. It replaces the fork's per-site rank layer with global reputation + decoupled per-site roles, and it deletes more code than it adds. It subsumes the old RANK-1 findings and the rank half of the old queue bug.

## REP-1 — Make reputation global; remove per-site rank as a privilege source
**Severity:** High (feature-model correctness)

**Symptom / current state:** the fork added per-site rank (`user_site_rank`, commit `e1a65f1d`) and gates privileges on it, with a read-modify-write update (`internal/repo/rank/user_site_rank_repo.go:60-79`), a delta clamped against the *global* score but applied to the *site* score (`user_rank_repo.go:94-104`), a nil-return-after-global-change divergence path (`user_site_rank_repo.go:55-58`), and a **fail-open** fallback to the global value on DB error (`service/rank/site_rank.go:40-45`). Per the decided model, this whole layer is unwanted.

**Fix:**
1. **Privilege/rank source = global.** Route rank reads and privilege checks back through upstream's global rank path (atomic `Incr` at `user_rank_repo.go:98`). Remove the per-site rank gating in `service/rank/site_rank.go` and any permission path that reads site rank.
2. **Retire `user_site_rank` as a gating table.** You may drop it, or (cheaper and reversible) stop reading it for privileges and leave it dormant. Do **not** invest in fixing its race/clamp bugs — they disappear with the global path.
3. **Preserve `site_id` on `activity`/vote rows** (do not remove the column). This is the C-ready substrate — see REP-3.
4. Fix the fail-open: any residual rank/role read that errors must fail **closed**, never fall back to a higher value.

**Tests (`-tags multisite`, real-DB):** a user earning rep on site A has the same global rank visible on site B; privilege thresholds gate identically on every sub-site; concurrent upvotes don't lose increments (inherited from upstream's atomic path); injected DB error on a role read denies rather than grants.

**Done when:** one global rank per user gates all sub-sites, the per-site rank code no longer participates in privilege decisions, and `activity.site_id` is retained.

## REP-2 — Keep per-site roles, fully decoupled from rank
**Severity:** Medium (correctness of delegated moderation)

**Intent:** `user_site_role_rel` stays as the answer to "is user U a moderator/admin of sub-site S," independent of reputation. A global network-admin (global role row) retains cross-site authority; a per-site role grants authority only on its site.

**Fix:**
1. Ensure `getEffectiveRole` (`service/role/site_role.go:34-51`) resolves the role for the **request's** sub-site (moderator of A is a moderator on A, not on B), while a true network-admin outranks everywhere.
2. Confirm the moderator/admin gate used by content controllers (`GetUserIsAdminModerator`, `internal/base/middleware/auth.go:284-313`, consumed at `question_controller.go:90,417,...`) evaluates against the resolved sub-site for per-site roles. (This is the *legitimate* residue of the old SEC-1 — not a tenant-isolation fix, just "a board moderator's powers apply to their board.")
3. Same fail-closed rule as REP-1 for role reads.

**Tests:** moderator of A can moderate content on A, not on B; network-admin can moderate anywhere; role read error → denied.

**Done when:** board authority follows explicit role assignment per sub-site, with a global-admin override, and never leaks from rank.

## REP-3 — Retain per-site attribution so Option C is a later query, not a migration
**Severity:** Low now (architectural guard) — **must be honored while doing REP-1**

**Intent:** the eventual "reputation earned per sub-site" directory display must not require re-instrumenting history.

**Fix / guardrail:**
- Keep `site_id` stamped on every `activity` and vote row at write time. Synchronous writes already do this correctly (`vote_repo.go:351`, `answer_repo.go:203`, `user_active_repo.go:111`, `follow.go:258`) — don't regress them.
- Queue-routed activity (question close/reopen etc.) currently loses `site_id` (see IDENT-1); fixing that keeps attribution complete. Under the global-rank model this is no longer a privilege-correctness bug — it only degrades the future C breakdown for those specific activity types — so it's a *should-fix-for-attribution* item, not a blocker.
- Document the invariant in `MULTISITE.md`: privilege = global sum; `activity.site_id` = attribution for display.

**Tests:** an activity row from site A carries `site_id='A'`; a `SUM(delta) GROUP BY site_id` per user returns a correct per-site breakdown (proves C is reachable).

**Done when:** per-site reputation attribution is queryable from retained data with no schema change required.

---

# PHASE 3 — Wrongly-scoped global-identity surfaces

These are the "site filter that should not be there" bugs. The old audit called them cross-tenant leaks; under the doctrine they're identity surfaces incorrectly forced through a site-scoped path, which is why data silently disappears.

## IDENT-1 — Notifications and badge awards are site-scoped but should be global
**Severity:** High (broken functionality: inbox and achievements silently empty)

**Symptom:** Inbox/achievement notifications never appear; badge awards are orphaned. Written with `site_id=''` (async workers run under a site-less context) and then filtered out by site-scoped reads.

**Root cause:** `internal/base/queue/queue.go:127` invokes `handler(context.TODO(), msg)`; no message struct carries a site, so `SiteInsert` writes `site_id=''` (`notification_repo.go:51`, `badge_award_repo.go:84`) while scoped reads (`notification_repo.go:120,138`; `badge_award_repo.go:113-183`) filter by the request's site → the rows never match.

**Fix (de-scope, don't propagate):** notifications and badges are per-user/global by the doctrine. Remove site scoping from these surfaces: write and read them **without** a site filter (global rows). This is simpler than plumbing site through the queue and matches the model — one inbox, one achievement set per user, feeding the directory.
- Notification read/write paths (`notification_repo.go`) → drop the site predicate.
- Badge award read/write (`badge_award_repo.go`) → drop the site predicate.
- Cross-site follower fan-out (`notification_common/notification.go:343`) is then *correct* rather than a leak — a user follows people, not boards.

**Note on the queue context:** the generic worker's `context.TODO()` (IDENT-1 root) still matters for **activity attribution** (REP-3) and **external-notification email rendering** (IDENT-2). Fix it there by capturing the site at `Queue.Send` and rebuilding a ctx carrying `constant.SiteIDContextKey` before `handler` runs — but notifications/badges themselves stop caring once de-scoped.

**Tests:** enqueue a notification/badge for a user via activity on site A; assert it appears in that user's global inbox/achievements regardless of which sub-site they're viewing; a user gets exactly one inbox across sub-sites.

**Done when:** notifications and badges are global per user and always visible, independent of sub-site.

## IDENT-2 — External-notification emails render with the wrong site's settings
**Severity:** Medium

**Symptom:** Emails triggered by activity on site B render with the default site's interface/language.

**Root cause:** `internal/service/notification/external_notification.go:74-79` runs under the queue's empty context → `GetSiteInterface` resolves the global row.

**Fix:** decide per the doctrine which of these is "identity-global" vs "content-site": email *content about a question* originates on that question's sub-site, so the email should use that sub-site's interface/branding. Capture the originating site at enqueue (the `Queue.Send` fix from IDENT-1) and render with it. If you'd rather keep email uniform org-wide, make that explicit instead — but don't leave it accidental.

**Tests:** an email about a question on site B uses site B's interface settings.

**Done when:** notification email rendering uses a deliberately chosen site's settings, not whatever the empty context resolves to.

---

# PHASE 4 — Content-partition correctness (site-scoping that must actually work)

These *are* site-scoped surfaces where the scoping is broken. Not breaches (all users see all sites), but they mingle boards, which is precisely what the fork exists to prevent.

## CONTENT-1 — Plugin full-text search is not site-filtered
**Severity:** Medium-High (the site/all-sites search toggle you built doesn't actually scope)

**Symptom:** With a search plugin (ES/Meilisearch) enabled, a `scope=site` search on A returns B's content — the "this site / all sites" toggle (`e5013058`) is a no-op for the plugin path.

**Root cause:** `internal/repo/search_common/search_repo.go:504` hydrates plugin hits via `SiteDB(ctx).Query(rawBuilder)`, but xorm 1.3.2 ignores session-level `.Where()` when passed a raw `builder.Builder` (`internal/statements/query.go:19-22`), so the site filter never reaches SQL. The native path proves the fix by embedding the condition *inside* the builder. `Convert2PluginSearchCond` (`search_schema.go:112-136`) also passes no SiteID to the plugin.

**Fix:** add the `site_id` condition inside the `builder.Builder` in `ParseSearchPluginResult` (read site from ctx); plumb SiteID into `Convert2PluginSearchCond` so the plugin scopes at source (also fixes result counts/pagination). Gate on the validated `search.scope` DTO (`search_schema.go:36`) for the deliberate all-sites case.

**Tests:** stub plugin returning IDs from both sites; under site A + `scope=site`, only A's rows and correct totals; `scope=all` returns both.

**Done when:** plugin search scopes identically to native search and the toggle works.

## CONTENT-2 — Sitemap mingles sites
**Severity:** Medium

**Symptom:** Site B's `sitemap.xml` lists site A's question URLs.

**Root cause:** cache key `SiteMapQuestionCacheKeyPrefix = "answer:sitemap:question:%d"` (`constant/cache_key.go:45`) has no site component while the query it caches is site-scoped; the cron runs site-less (`internal/base/cron/cron.go:65,70`, `SitemapCron(context.Background())`); render uses global SEO config (`question_service.go:1735-1742`).

**Fix:** add a site component to the sitemap cache key(s); make `SitemapCron` iterate sub-sites, each under a context carrying its site (depends on the `Queue.Send`/cron context fix from IDENT-1); render per-site base URL/SEO.

**Tests:** two sites with distinct questions → each sitemap contains only its own URLs and base URL; cache keys differ per site.

**Done when:** each sub-site's sitemap contains only its own content.

## CONTENT-3 — Frontend loads the wrong sub-site on deep links and drops the `/s/<slug>` prefix
**Severity:** High (most user-visible items — real navigation breakage)

**Symptoms & fixes (bundle):**
1. **Bootstrap loads default-site data.** `ui/src/utils/guard.ts:450` fires bootstrap calls before the site store is populated (`guard.ts:406-413`); the interceptor reads the slug only from the store (`ui/src/utils/request.ts:69-72`). **Fix (one line):** in the interceptor, when the store is empty, derive the slug from `window.location.pathname` via the existing `^\/s\/([^/]+)` regex (`ui/src/stores/currentSite.ts:37-41`) and set `X-Site-Slug`.
2. **Cross-site search results 404.** `ui/src/pages/Search/components/SearchItem/index.tsx:50-53` builds router-relative URLs even for all-sites results. **Fix:** build an absolute URL from the result's own site slug.
3. **`window.location` redirects drop the prefix.** `ui/src/utils/floppyNavigation.ts:145-162` prepends only build-time `REACT_BASE_PATH`, not the runtime basename (`App.tsx:28-32`). Affects 401/inactive/suspended/expired (`request.ts:166,174-189`, `guard.ts:304`), logout (`Users/Logout/index.tsx:46`), and Share copy-link (`Share/index.tsx:49-50`). **Fix:** prepend the runtime basename in all `window.location` navigations.
4. **`linkedRedirect` double-prefixes** (`guard.ts:427` + router basename → `/s/foo/s/foo/...`). **Fix:** feed a router-relative path or strip the basename first.
5. **`isIgnoredPath` never matches under `/s/<slug>`** (`floppyNavigation.ts:70-78`, positional segment compare) → 403/404/500 suppression breaks on in-site auth pages. **Fix:** strip the site prefix before comparing.

**Tests:** interceptor sends correct `X-Site-Slug` from URL when store empty; cross-site link builder; redirect-with-basename; no double-prefix; `isIgnoredPath` under `/s/slug`. Manual: hard-reload a non-default `/s/<slug>` page; expire a session there; copy a share link.

**Done when:** every deep link, redirect, and shared URL stays on the correct sub-site.

## CONTENT-4 — Unknown slug silently serves the default sub-site
**Severity:** Medium

**Symptom:** `/s/typo` or a bogus `X-Site-Slug` renders default-site content under the wrong URL instead of 404.

**Root cause:** backend `internal/base/middleware/site.go:125-149` falls through to `default` on any unresolved slug/header; frontend `currentSite.ts:53-67` mirrors it while the basename stays `/s/typo`.

**Fix:** distinguish "no site specified" (→ default is fine) from "site specified but not found" (→ 404). Backend: only fall back to default when *no* slug/host/header was given. Frontend: route an explicit unknown slug to not-found.

**Tests:** explicit unknown slug → not-found both ends; absent slug → default.

**Done when:** a nonexistent explicit slug returns a real not-found.

---

# PHASE 5 — Database & migrations (SQLite-correct now, Postgres-ready)

Confirmed deployment: SQLite/WAL per tenant today, Postgres a possible future, MySQL never (`../../infra/design.md`). The v33 migration is MySQL-dialect — wrong for both engines you'll actually run.

## DB-1 — v33 migration is MySQL-only: breaks SQLite backfill now, blocks Postgres later
**Severity:** High (live SQLite defect + hard Postgres blocker)

**Symptom:**
- **SQLite (now):** the `user_site_rank` backfill uses `NOW()` (`v33.go:94,102`), which SQLite lacks → backfill silently no-ops. *(Under REP-1 this table is retired, which removes this specific failure — but the pattern must still be fixed for the content-table steps.)*
- **Postgres (future):** backtick-quoted `ALTER TABLE`/`CREATE INDEX` (`v33.go:58,63`) is a syntax error → `site_id` columns never added, yet errors are demoted to `log.Warnf` (`:60,62,88,97,105`) so the migration reports success and bumps the version. First scoped query then hard-fails with no clean re-run.

**Root cause:** hand-rolled dialect-specific DDL applied to all engines, with errors swallowed. The same file proves dialect-awareness is available — `dropLegacyUniqueIndexes` switches on `x.Dialect().URI().DBType` (`v33.go:124,152`).

**Fix:**
1. Make column-add / index-create / backfill dialect-aware (identifier quoting per dialect; portable current-timestamp or a bound `time.Now()` instead of SQL `NOW()`). Prefer xorm `Sync` on the entity where it can create the column/index (it's already dialect-correct).
2. Stop swallowing errors: a failed structural step returns an error and does **not** advance the version.
3. Fold in REP-1: drop the `user_site_rank` backfill entirely (table retired), removing the `NOW()` hazard.

**Tests (needs a migration harness — see TEST-2):** build a pre-v33 SQLite schema, run `Migrate()`, assert `site_id` columns exist on all content tables and version advances only on full success; inject a failing step → migration errors, version unchanged. Add Postgres to the matrix before any Postgres move.

**Done when:** v33 applies correctly on SQLite and Postgres, or fails loudly without advancing the version.

## DB-2 — Upgraded installs never get the composite unique indexes
**Severity:** Medium-High (uniqueness silently unenforced after upgrade; schema divergence)

**Symptom:** After v33, upgraded installs accept duplicate tag slugs / config keys within a site; fresh vs upgraded schemas diverge permanently.

**Root cause:** v33 drops the old unique indexes (`v33.go:111-118`) claiming xorm `Sync()` recreates the composites — but no such Sync runs on upgrade. v33 syncs only the three new tables (`:34-38`); `Migrate()` never syncs entities; the only full `Sync(tables...)` is fresh-install-only (`init.go:114`). Entities define the intended composites (`tag_entity.go:52`, `config_entity.go:35`, `plugin_config_entity.go:27`); upgraded DBs get none.

**Fix:** in v33 (or a follow-up), explicitly create the composite unique indexes for `tag`/`config`/`plugin_config`, dialect-aware (DB-1), matching the entity names so fresh and upgraded installs converge. Dedup or fail clearly if existing data violates them.

**Tests:** post-migration the composites exist with entity names; duplicate `(slug, site_id)` rejected, same slug under different site accepted; fresh-vs-migrated index sets identical (TEST-2).

**Done when:** upgraded and fresh installs have identical constraints and per-site uniqueness is enforced.

---

# PHASE 6 — Config model & quality batch

## CFG-1 — Per-site presentation (name + CSS/HTML injection) with global fallback
**Severity:** Medium (feature reachability) — **mostly wiring; the read cascade already exists**

**Decided model:** functional config stays global. The *only* per-sub-site surface is **presentation**, and specifically the small set that actually differentiates a product's pages:
- **name** — `site_info` type `general`
- **logo / favicon** — `site_info` type `branding`
- **the CSS/HTML injection bundle** — `site_info` type `css-html` (`SiteCustomCssHTMLReq`, `siteinfo_schema.go:227-233`): `custom_head`, `custom_css`, `custom_header`, `custom_footer`, `custom_sidebar`

Everything else (`login`, `users`, `write`, `privileges`, `interface`, `theme`, `seo`, plugin config, SMTP) is global. `seo`/`theme` can be added to the per-site set later trivially — same mechanism — if a product ever needs them.

**Key finding — the per-site-with-global-fallback machinery already works on reads.** `siteinfo_repo.go:80-102` already does, per type: *if the context carries a site and a `type+site_id` row exists, use it; otherwise fall back to the `site_id=''` global row*, with a site-qualified cache prefix (`:105-110`). This is exactly the "store per site, fall back to global" model — no new storage or read logic needed. It's proven by the existing site_info leak tests. The fork simply never fed it from the two ends:

1. **Write can't target a site.** Because the admin API skips resolution (`site.go:104`), `TierSiteID(ctx)==""` on every save, so `SaveByType` always writes the global row and a site override can never be created. **Fix:** give the admin `general` / `branding` / `css-html` save endpoints an **explicit target `site_id` argument** (body/param), gated by the network-admin. Saving with no target edits the global default; saving with `site_id=A` creates/updates A's override. Admin stays network-global; the admin just picks which sub-site (or "all / global default") they're styling.
2. **Render doesn't consume per-site values.** Once the frontend holds the correct resolved site (depends on **CONTENT-3**), the existing cascade already returns per-site-or-global `general`/`branding`/`custom_css_html` in `SiteInfoResp` with no extra work. Then apply the custom CSS/HTML from **one** consolidated injection path. Today there are two (`ui/src/components/Customize/index.tsx:118-134` and the fork's added banner at `ui/src/components/Header/index.tsx:229-234`), which is also the QUAL-7 double-render bug — collapse them into a single path that reads the resolved site's `custom_css_html` and injects each of head/css/header/footer/sidebar once.

**Fallback granularity — DECIDED: wholesale per-type.** A site's `css-html` (or `branding`) override replaces the global row of that type entirely. If a site defines a `css-html` override at all, it authors the whole bundle (head/css/header/footer/sidebar); it does not partially inherit individual global fields. This matches the existing cascade exactly — no field-level merge, no extra read logic. A site with no override row of a given type falls back to the global row wholesale, which is the common case. Same rule for `branding` and `general`: override or inherit, per type, not per field.

**No `config` override work.** Functional `config` stays global; remove or clearly quarantine the dead per-site override branch (`config_repo.go:163-193`) so it isn't a latent trap. (If it were ever revived, the config-ID forking bug would bite — a per-site override gets a new auto-increment ID at `config_repo.go:188-191` while activity stores config IDs as `activity_type` at `activity_repo.go:71`, resolved by ID and dropped on miss at `rank_service.go:317`. Not relevant to the presentation-only model.)

**Tests:** save `css-html` targeting site A → only A's rendered pages change, other sites and the global default unchanged; a site with no override renders the global default; save with no target edits the global default and cascades to all un-overridden sites; assert the single injection path applies each custom block exactly once (QUAL-7 regression).

**Done when:** an admin can set a sub-site's name, logo, and custom CSS/HTML with global fallback, those values render on that sub-site's pages only, and functional config remains org-wide with no dead override branch.

## QUAL — batch (low-medium, threat-model-independent)

- **QUAL-1 — Global config/site_info cache not invalidated per-site.** Fallback reads cache global values under site-qualified keys (`config_repo.go:87-89`, `siteinfo_repo.go:100`) but global updates rewrite only the empty-prefix key (`config_repo.go:213-223`, `siteinfo_repo.go:67`) → non-default sites serve stale values up to 1h (`cache_key.go:39,42`). Fix: invalidate/rewrite all per-site fallback keys on global update (or version the key).
- **QUAL-2 — Admin tag slug update dropped, echoed as success.** `profile_tag_repo.go:95-102` omits `slug` from update cols but `AdminUpsert` returns the requested slug (`tag_service.go:81-93`), and no existence check (updating a missing ID "succeeds" with 0 rows). Fix: include `slug` (or reject slug changes) and return the persisted row / 404.
- **QUAL-3 — Retired tags stay visible.** `SetTags` filters inactive at write (`profile_edit_service.go:92-110`) but reads don't (`directory_service.go:104`, `network_profile_service.go:163`). Fix: filter inactive at read.
- **QUAL-4 — `GetNetworkProfile` swallows all sub-fetch errors** (`site/network_profile_service.go:119-174`) → empty profile indistinguishable from DB error. Fix: log (and consider a partial-failure signal).
- **QUAL-5 — OIDC login-path HTTP calls have no timeout/context.** `connector.go:110,136` use bare `http.DefaultClient` + `http.NewRequest` on token/userinfo → a hung fastgate pins login goroutines. Fix: timeout client + `http.NewRequestWithContext`. Also `AfterLogin` runs synchronously in the login request (`user_external_login_service.go:186-204`), adding up to 5s per SSO login — consider async. Trim the oversold "multi-provider" comment (`connector.go:153-158`) given the hardcoded paths, or implement discovery.
- **QUAL-6 — Unescaped LIKE wildcards in member search** (`directory_repo.go:101-108`) → `q=%%%` forces full scans on an anonymous endpoint. Fix: escape `%`/`_`/`\`.
- **QUAL-7 — Frontend hygiene:** `custom_header` renders twice (`Header/index.tsx:229-234` duplicates `Customize/index.tsx:129`); admin Sites/AppSwitcher/Network forms lack validation, error paths, and store refresh, diverging from upstream `SchemaForm` (`Admin/Interface/index.tsx:176`); dedupe the two near-identical site switchers (`SideNav/SiteContextPicker`, `Header/components/SiteSwitcher`); remove dead `reset()` (`featuresControl.ts:37`, `appSwitcher.ts:45`); `window.confirm` → `Modal.confirm` (`Settings/Projects:120`); restore `ui/package.json` trailing newline.
- **QUAL-8 — Missing i18n** across new UI (site pickers, app switcher, entire Members / NetworkProfile / Settings-Network/Projects / Admin-Sites/NetworkTags pages, manual "match/matches" pluralization). `ui.header.app_switcher.*` used via `t()` but keys never added to `i18n/en_US.yaml`. Fix: extract strings; add missing keys.
- **QUAL-9 — Backend odds:** dead `SiteTransaction` only in `!multisite` build (`site_db_default.go:34`, no multisite counterpart) — add or remove; `SetSiteStatus` has no route and `GetAllSites` filters `status=active` (`site_repo.go:76`) so a suspended site can't be shown to reactivate — wire the route, include inactive in admin listing; `AddSite` swallows dup-check error (`site_service.go`); middleware skip-list matches absolute `/answer/admin/api/` while routes register under `uiConf.APIBaseURL` (`http.go:100`) — build prefixes from `APIBaseURL`; host-label shadowing (`site.go:115-122` — a sub-site slugged like the deployment host label hijacks all requests) — reserve the deployment host label(s); `siteinfo_repo.go:121,126` ignores marshal errors; `config_repo.go` value-receiver + 3× duplicated cascade.

## DIR-1 — Directory visibility (product decision, not a bug)
The member directory is intentionally instance-global. `/network/members` + `/network/tags` are anonymous (`answer_api_router.go:230-237`); writes and admin tag curation are correctly gated. Given the internal-guild framing, decide: keep anonymous (sanitized — AUTH-4 already required), members-only (move the two GETs into the `MustAuth` group), or configurable (`directory_visibility: public|members-only` alongside `directory_enabled`). Recommendation: members-only or configurable for an internal directory. Fix AUTH-4 regardless.

---

# PHASE 7 — Tests & CI (the "well tested" gate)

## TEST-1 — Content leak/scope tests across content repos
`internal/repo/repo_test/multisite_leak_test.go` covers 3 of ~38 `SiteDB()` repos (`config`, `site_info`, `plugin_config`). Add real-DB scope tests for the **content** repos where mingling would occur: question, answer, comment, tag, tag_common, revision, report, meta, review, collection, search_common (native + plugin per CONTENT-1). Pattern: seed two sub-sites, write to A, assert scoped read on B is empty and on A returns it. Also add **identity-global** assertions (the inverse) for notifications/badges after IDENT-1: written on A, visible to the user everywhere.

## TEST-2 — Migration + fresh-vs-upgrade convergence
No test builds a pre-v33 DB and asserts post-state, and none asserts fresh and migrated installs converge. Add a SQLite migration test (drives DB-1, DB-2) and a schema-convergence test (fresh `Sync` vs migrated index/constraint sets). Add Postgres to the matrix before any Postgres move.

## TEST-3 — Site-resolution middleware
`internal/base/middleware/site.go` (identity resolution) has zero tests. Cover: subdomain/path/`X-Site-Slug`/`X-Site-ID` precedence, absent-slug → default, unknown-slug → not-found (CONTENT-4), inactive-site handling, the admin-API skip, and `WithoutSite`.

## TEST-4 — Fastgate connector + reputation model
No tests on the connector, session-kill, or authoritative-username branches. Cover AUTH-1/2/3 and the deactivate → token-kill path. Add REP-1/REP-2 tests (global rank across sites; per-site role scoping; fail-closed).

## TEST-5 — CI must run tests and lint the multisite build
CI (`.github/workflows/build-fork-image.yml`) only builds/pushes the image; no workflow runs `go test`, and `make lint` only sees the noop build (no build tags in `.golangci.yaml`/`Makefile`). Add: a job running `go test ./...` **and** `go test -tags multisite ./...` (SQLite fallback); vet/lint the multisite build (`go vet -tags multisite ./...`, golangci `build-tags: [multisite]`); optionally boot the image and run `script/test-multisite.sh` (the e2e smoke, currently run nowhere).

## TEST-6 — Working-tree hygiene
`answer-dev` is a 74 MB untracked Mach-O binary — gitignore or remove (accidental-commit hazard). The uncommitted `MULTISITE.md` / `FASTGATE_INTEGRATION.md` edits are docs catching up to shipped code — safe to commit; while there, fold in this spec's doctrine (content-vs-identity axes, global reputation + per-site roles, REP-3 attribution invariant, CFG-1 branding model, AUTH-2 email-trust assumption).

---

# APPENDIX A0 — Verification amendments (2026-07-04)

Every issue was re-verified against the working tree before implementation. All claims held (minor line drift only) except as amended below. Decisions recorded here are final.

**Corrections:**

- **AUTH-4 reframed — no email-hash leak.** The stored `user.avatar` blob never contains an email or hash; gravatar hashing is **sha256** (`pkg/gravatar/gravatar.go:32`), computed at format time from the email, which the directory query does not select. The real defect is returning the raw JSON blob (broken `<img src>`, bypasses deleted-user avatar suppression in `selectedAvatar`). Severity → Low; fix unchanged (route through `FormatAvatar`). `/network/user/profile` is NOT affected (already formats).
- **CONTENT-2 mechanism corrected.** SEO/general config resolves site-scoped correctly at render. The actual bug: the page-only cache key (`question_repo.go:352`) is first-writer-wins across sites, and the site-less cron pre-populates it with a **global** question list every site then serves.

**New findings folded into phases:**

- **REP-2 blocker:** the effective role is computed once at login and baked into the token cache (`CacheLoginUserInfo` → `UserCacheInfo.RoleID`, `user_common/user.go:228-239`); `GetUserIsAdminModerator` reads the cached value and never re-resolves for the request's sub-site. Per-site moderation requires **per-request role resolution**. Also `getEffectiveRole` is escalate-only (site role can raise, never lower).
- **REP-3 wider:** ALL queue-routed activity (`HandleActivity` → `activity_common/activity_repo.go:139-145`) writes `site_id=''`, not just close/reopen. `TriggerUserRank` never calls `ChangeSiteRank` (further site-rank divergence — moot under REP-1).
- **DB-1 worse + already live:** `v33.go:62-63` discards CREATE INDEX errors entirely (`_, _ =`, not even Warnf). The `user_site_role_rel` backfill is a single INSERT…SELECT that violates its own `UNIQUE(ux_user_site)` for multi-role users — the whole statement aborts even where `NOW()` works. **Live DBs (catapulsion, guild — see `../../infra/STATUS.md`) already ran v33**, so their backfills silently no-op'd and composite unique indexes are absent. Consequence: fix v33 for future upgrades **and append a v35 repair migration** for live DBs. Nothing broke visibly only because fail-open + escalate-only masked it.
- **Postgres fresh-install breaker:** `user_site_rank_entity.go` and `user_site_role_rel_entity.go` both name their composite index `ux_user_site`; Postgres index names are schema-global → fresh Postgres `Sync` fails. (Resolved by dropping `user_site_rank` — see decisions.)
- **Fork entities declare string PKs as `autoincr`** (`user_site_rank`, `network_project`, `profile_tag`, `user_profile_tag`) — brittle across dialects; clean up in Phase 5.
- **Daily rank limit is per-site** (`user_rank_repo.go:65-74,180-188`) — under global reputation it must be global; fold into REP-1.
- **Admin dashboard shows global counts** under a global cache key (admin API is site-less by design). Deliberate: acceptable for a network admin; documented, not changed.
- **Vector search is clean:** syncer stamps `SiteID` from the entity row, not ctx; plugin contract already carries site scoping.
- **Minor AUTH extras → QUAL:** `access_token` in the auth-landing redirect URL is upstream's standard mechanism — kept; `connector_controller.go:134` debug-logs the full external identity — drop; `user.e_mail` has no UNIQUE index (email-bind relies on app logic) — noted.

**Enablers confirmed:**

- fastgate supports discovery, JWKS, nonce, and PKCE S256, and emits `email_verified: true` — AUTH-1/2 fully implementable with no fastgate-side work.
- Session-kill on deactivation exists end-to-end (fastgate exec hook → admin status endpoint → `RemoveUserAllTokens`).

**Decisions (final):**

1. **DIR-1:** `directory_visibility: public | members-only` config knob alongside `directory_enabled`, **default members-only**.
2. **Bookmarks/collections are identity-global.** One bookmark list per person, visible from any sub-site; items link into their owning site. De-scope the collection repo (Phase 3).
3. **`user_site_rank` is dropped in v35.** Its data is wrong on live DBs anyway (backfill no-op'd; clamp-bug divergence); `activity.site_id` is the Option-C substrate. Dropping also removes the `ux_user_site` name collision.

---

# APPENDIX A — What changed from the first audit (traceability)

- **SEC-1 (admin escalation) → dissolved.** Single trust domain; network-global admin is the intended model. Its one legitimate residue (a board moderator's powers apply to their board) is now REP-2.
- **SEC-2 (queue site loss) → split.** Notifications/badges → de-scope to global (IDENT-1). Activity attribution + email → keep the `Queue.Send` context fix (REP-3, IDENT-2).
- **SEC-3/4/8 → reclassified** from cross-tenant breach to content-partition correctness (CONTENT-1/2) and a privacy scrub (AUTH-4); the "cross-site" framing dropped because all users see all sites.
- **RANK-1 → absorbed into REP-1** (disappears with global rank).
- **FEAT-1 → CFG-1**, reframed as "functional config global, branding per-sub-site," not a per-site-everything override.
- **Genuine security survivors:** AUTH-1/2/3 (OIDC/username), threat-model-independent.

# APPENDIX B — Deferred upstream-PR mechanics

Not needed now (PR deferred); captured so it isn't re-derived. Upstream `main` unchanged since fork; `dev` has 8 commits. On any future rebase onto `dev`: rename our `v33.go`/`v34.go` (collide with upstream's `addAIConversationReasoningContent`/`addRequireEmailVerification`) to `v35.go`/`v36.go` and append our slice entries after upstream's; handle positional-migration skew for existing DBs (version bump or idempotency); manual-merge `internal/base/server/http.go:52-60` (upstream panic-recovery vs our `ResolveSite()` — Recovery outermost; net win, we currently run without panic recovery); audit `SiteLoginReq.RequireEmailVerification` (now a required `*bool`); regenerate `docs/docs.go`+swagger rather than hand-merge. No upstream commit fixes a bug we worked around differently.

---

## Quick index (handling order)

| Phase | ID | Title | Sev |
|---|----|-------|-----|
| 1 | AUTH-1 | OIDC state/nonce/PKCE + ID-token validation | High |
| 1 | AUTH-2 | OIDC email_verified / userinfo status | Med |
| 1 | AUTH-3 | Reserved-username case folding | Low-Med |
| 1 | AUTH-4 | Directory email-hash scrub | Med |
| 2 | REP-1 | Global reputation; remove per-site rank gating | High |
| 2 | REP-2 | Per-site roles, decoupled from rank | Med |
| 2 | REP-3 | Retain `activity.site_id` for Option C | Guard |
| 3 | IDENT-1 | Notifications/badges → global (de-scope) | High |
| 3 | IDENT-2 | External email uses originating site | Med |
| 4 | CONTENT-1 | Plugin search site filter | Med-High |
| 4 | CONTENT-2 | Sitemap per-site | Med |
| 4 | CONTENT-3 | Frontend site-context / `/s/` prefix | High |
| 4 | CONTENT-4 | Unknown slug → 404 | Med |
| 5 | DB-1 | v33 SQLite/Postgres dialect safety | High |
| 5 | DB-2 | Composite unique indexes on upgrade | Med-High |
| 6 | CFG-1 | Functional config global / branding per-site | Med |
| 6 | QUAL-1..9, DIR-1 | Quality batch + directory visibility | Low-Med |
| 7 | TEST-1..6 | Content-leak, migration, middleware, connector, CI, hygiene | — |
