# Fastgate Integration

Reference for the Answer ↔ fastgate integration. Updated as we wire pieces in;
read alongside MULTISITE.md (multisite + network-directory architecture) and
fastgate's `docs/DESIGN.md` §21–22 (identity directory, off-board hook).

## What's wired on the Answer side

### 1. OIDC connector

`internal/plugins/fastgate_connector/connector.go`

Standard OIDC authorization code flow against fastgate. Three config fields
(set in **Admin → Plugins → Fastgate**):

| Field | Value |
|---|---|
| Issuer URL | fastgate base, e.g. `https://gate.example.com` (no trailing slash) |
| Client ID | from `fgctl oidc-client -create` |
| Client Secret | from `fgctl oidc-client -create` |

Redirect URI registered on the fastgate side:
`{ANSWER_BASE}/answer/api/v1/connector/redirect/fastgate-connector`

### 2. Identity report on every login

After Answer's external-login service binds or creates the local user, the
connector POSTs the mapping back to fastgate via the optional plugin hook
`plugin.ConnectorAfterLogin`:

```
POST {issuer}/directory/users/{sub}/apps/{client_id}/identity
Authorization: Basic base64(client_id:client_secret)
Content-Type: application/json

{"app_user_id": "<answer user.id>"}
```

- Fires on **every** successful login (existing user OR new registration).
- 5-second client timeout; failures log a warning, never block login.
- Idempotent: re-posts on every login, fastgate refreshes `last_reported_at`.

### 3. Atomic session invalidation on deactivation

`internal/service/user_admin/user_backyard.go`:`UpdateUserStatus` calls
`RemoveUserAllTokens(userID)` when the new status is `deleted`, `suspended`,
or `inactive`. Without this, an admin status change waits up to the 7-day
`UserTokenCacheTime` for sessions to expire. With this, sessions die at the
DB write.

### 4. Off-boarding endpoint (consumed by fastgate's exec hook)

```
PUT  {ANSWER_BASE}/answer/api/v1/admin/user/status
Authorization: Bearer <answer admin token>
Content-Type: application/json

{"user_id": "<answer user.id>", "status": "deleted"}
```

Status options: `normal | suspended | deleted | inactive`. For off-boarding
`deleted` is the cleanest. Optional `remove_all_content: true` for hard
purge.

Example fastgate hook (drop into the file fastgate execs per
deactivation/per-app pair):

```bash
case "$FASTGATE_CLIENT_NAME" in
  answer)
    curl -sf -X PUT \
      -H "Authorization: Bearer $ANSWER_ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"user_id\":\"$FASTGATE_APP_USER_ID\",\"status\":\"deleted\"}" \
      "$ANSWER_BASE/answer/api/v1/admin/user/status"
    ;;
esac
```

`FASTGATE_APP_USER_ID` is the value Answer reported in step 2.

## Open items — needed from fastgate

These are integration questions that affect what we build next on the Answer
side. Convey upstream when convenient:

### A. Long-lived admin auth for the hook script

The exec hook needs `ANSWER_ADMIN_TOKEN` in its environment. Answer's
"admin token" is a normal user JWT cached for 7 days — manually copying it
out of a browser session means re-pasting weekly.

**Cleaner:** point fastgate's hook at an Answer **API key** (Answer has
`internal/repo/api_key/`, admin-issuable). API keys don't expire on the
7-day session schedule. Procedure once we wire it on our side:

1. In Answer's Admin → API Keys, create a key scoped to admin user-status
   operations.
2. Drop the key into the env file fastgate's hook sources.

Not blocking — current Bearer-token flow works — but worth doing before
production. The Answer API key system already supports this; we just haven't
created a scope label for "off-boarding" yet.

### B. Reactivation flow

Fastgate's spec covers `user.deactivated`; we don't see a `user.reactivated`
analogue. If an admin un-disables a fastgate user, should Answer flip the
local user back from `deleted` to `normal`?

Two paths:

- **No reactivation hook**: the user simply logs in again and Answer's
  external-login service either finds the (soft-deleted) row and refuses, or
  creates a new local user. Currently the soft-delete path mangles the email
  (`UpdateUserStatus` appends `.<unix>` to `email` on delete), so the next
  fastgate login would create a fresh local user with no history. Bad.
- **Reactivation hook**: fastgate fires the same exec script with an event
  like `user.reactivated`, Answer's endpoint accepts
  `{"user_id":"X","status":"normal"}` and the existing soft-deleted row is
  restored. Answer's `UpdateUserStatus` already supports this when called.

Recommendation: add `user.reactivated` to fastgate's exec-hook event set.
Same script can branch on `$FASTGATE_EVENT`. Answer side is already ready.

### C. Email change propagation

The connector's `AfterLogin` re-runs on every login and posts the current
sub → answer-user mapping. fastgate's email-change webhook (`user.email_changed`
in the original spec) was dropped in favor of exec hooks. As long as the
user logs back in after changing their email, the OIDC `userinfo` claim
brings the new email and Answer's `activeUser` updates it. So we don't need
a separate hook — but worth confirming this matches fastgate's intent.

### D. Directory feature gate on corp deployments

Answer can be deployed with `service_config.directory_enabled: false` (corp
Q&A use case). In that mode:

- `/network/profile`, `/network/projects/*`, `/network/tags`, `/network/members`,
  and admin tag CRUD all 404.
- `/network/user/profile` still works but returns only the basic identity
  (display_name + global_rank + site_ranks); extended fields (headline /
  projects / tags / links) are omitted.
- Fastgate `AfterLogin` identity-report still fires (connector-level
  behavior, not gated by `directory_enabled`).

This is intentional: corp Answer reports its identity to fastgate's
directory so cross-app routing works, but doesn't expose a member
directory UI. Confirm this matches fastgate's expectation.

## Issues we've encountered that fastgate should know about

(Empty for now — connector flow works end-to-end against fastgate's
in-progress directory endpoint. Add findings as we hit them.)

## What Answer needs back from fastgate (status checklist)

- [x] `/authorize` + `/token` + `/userinfo` OIDC endpoints (working since
      initial fastgate connector)
- [x] `/directory/users/{sub}/apps/{client_id}/identity` POST endpoint
      (fastgate commit `b2abf28`)
- [x] Off-board hook execution model (per-app exec script with
      `FASTGATE_APP_USER_ID` etc. envs)
- [ ] Reactivation event in the exec-hook envelope (see §B above)
- [ ] Acknowledgment that we're not implementing `user.email_changed` /
      `user.deleted` webhooks (see §C)
- [ ] Long-term: confirm whether Answer admin API key vs. bearer token
      is the preferred auth shape for the hook (see §A)

## Deployment notes

### Guild deployment (full directory)

```yaml
# conf/config.yaml
service_config:
  directory_enabled: true
```

Plus admin curates Profile tags under **Admin → Community → Profile tags**.

### Corp deployment (no directory)

```yaml
# conf/config.yaml
service_config:
  directory_enabled: false   # or omit; default is false
```

Network directory tables (`network_profile`, `network_project`,
`profile_tag`, `user_profile_tag`) still get created by migration v2.2.0
but stay empty. No frontend nav entries, all `/network/*` endpoints 404.
