# Implementation Plan: Password and Session Security

Branch: main
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Summary

The Go backend stores unsalted SHA-256 password hashes in `internal/auth/auth.go`, permits a default cookie signing key, and never sets cookie `Secure`. Docker Compose documents and injects `SESSION_SECRET`, but the server only reads `COOKIE_KEY`, so container deployments silently fall back to the insecure default.

Migrate password storage to **bcrypt** (`golang.org/x/crypto/bcrypt`) with transparent legacy-hash upgrade on successful login so existing users are not locked out. Harden session lifecycle (rotation after authentication, production `Secure` cookies, required secret validation, explicit logout invalidation) and align deployment configuration with the real env var name.

This is roadmap item **2. Password and session security** from `.ai-factory/japanese-learning-aif-plan-prompts.md`. Public onboarding (item 1) already delivered min-length 8 validation and anti-enumeration; this plan builds on that contract. CORS/CSRF/WebSocket origin hardening belongs to Prompt 03 — do not expand into those here.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Hash | `hashPassword` = `sha256.Sum256` → hex; no salt (`internal/auth/auth.go`) |
| Verify | Direct string equality `u.PasswordHash != hashPassword(c.Password)` — not constant-time |
| Policy | Min length 8 (server + `LoginPage.vue`); no max length |
| Schema | `users.password_hash TEXT NOT NULL` — opaque string; no version column (`internal/db/db.go`) |
| Update hash | No `UpdateUserPasswordHash` (or equivalent) store method |
| Cookie key | `COOKIE_KEY` / `-cookie` flag; default `"change-me-please-32-bytes-min"` (`cmd/server/main.go`) |
| Cookie opts | Global + per-session: `Path=/`, `HttpOnly=true`, `SameSite=Lax`; **`Secure` unset** (false) |
| Session after auth | `startSession` uses `Sessions.Get` then sets `user_id` — no explicit session ID regeneration |
| Logout | `MaxAge = -1` + `Save`; does not clear `Values`; CookieStore has no server-side revoke list |
| Docker | `SESSION_SECRET=...` in `docker-compose.yml` / `docker-compose.dev.yml`; README Docker section documents `SESSION_SECRET`; app never reads it |
| Compose PORT | Compose sets `PORT=8080`; server listens on `ADDR` (default `:8080`) — PORT is currently unused |
| Deps | `github.com/gorilla/sessions`; no `golang.org/x/crypto` yet (`go.mod`) |
| Auth tests | `internal/auth/auth_handlers_test.go` covers register/login/logout/me + validation; no hash-format or Secure-cookie tests |

## Approach

1. **Select bcrypt** via `golang.org/x/crypto/bcrypt` (cost 12). Self-describing `$2a$`/`$2b$` strings — no custom cryptographic format. Prefer bcrypt over Argon2id here because Go’s Argon2 helpers do not ship a standard encoded string API; rolling a PHC encoder would risk inventing a format. Document the choice and cost in README.
2. **Legacy dual-verify**: On login, if stored hash looks like bcrypt (`HasPrefix "$2"`), use `bcrypt.CompareHashAndPassword`. Otherwise treat as legacy SHA-256 hex and verify with `subtle.ConstantTimeCompare` on digests. Never invent a second encoding for new hashes.
3. **Transparent upgrade**: After successful legacy verification, rehash with bcrypt and persist via a new store update. Login still succeeds if the upgrade write fails (log ERROR); next login retries upgrade.
4. **New registrations** always store bcrypt hashes.
5. **Password policy**: Keep min 8; add **max 72 bytes** (bcrypt input limit) with a stable error code; no complexity rules for this private educational app.
6. **Session rotation**: On register/login, invalidate any existing `sid` then create a **new** session with `Sessions.New` (not `Get`) before setting `user_id`.
7. **Production cookie security**: Env-driven `Secure` (default on when `APP_ENV=production` or `COOKIE_SECURE=true`); reject default/short `COOKIE_KEY` when production-secure mode is active.
8. **Fix Docker/docs**: Replace `SESSION_SECRET` with `COOKIE_KEY` everywhere; set a non-default placeholder; document that changing the key invalidates existing cookies.

## Decision: bcrypt (not Argon2id)

| Criterion | bcrypt | Argon2id |
|-----------|--------|----------|
| Established Go API | `GenerateFromPassword` / `CompareHashAndPassword` | Hashing only in `x/crypto/argon2`; encoding is DIY or third-party |
| Format | Standard `$2a$`/`$2b$` | PHC string if encoded carefully |
| Constant-time verify | Library-provided | Depends on compare implementation |
| Fits “no custom crypto format” | Yes | Higher risk of ad-hoc encoding |

**Decision:** bcrypt cost **12**. Revisit Argon2id only with a maintained PHC helper in a later hardening pass if threat model requires it.

## Password Hash Contract

| Kind | Storage in `password_hash` | Verify | After success |
|------|----------------------------|--------|---------------|
| New (register / upgrade) | bcrypt string from `GenerateFromPassword` | `bcrypt.CompareHashAndPassword` | — |
| Legacy | 64-char hex SHA-256 of raw password (current) | SHA-256 then `subtle.ConstantTimeCompare` | Rewrite to bcrypt |

Detection rule: `strings.HasPrefix(hash, "$2")` → bcrypt path; else → legacy path. Do not add a `hash_version` column in this plan (TEXT column already holds either format). Versioned SQL migrations for learning domain remain Prompt 09.

## Password Policy

| Rule | Value | Error code | Notes |
|------|-------|------------|-------|
| Minimum length | 8 (unchanged) | `password_too_short` | Register + login validation |
| Maximum length | 72 **bytes** (UTF-8) | `password_too_long` | bcrypt truncation hazard; apply on register and login |
| Complexity | None | — | Out of scope |

Frontend (`LoginPage.vue`) must mirror max-72 with the same user-facing copy. Keep anti-enumeration behavior from onboarding unchanged.

## Session and Cookie Contract

| Concern | Behavior |
|---------|----------|
| Cookie name | `sid` (unchanged) |
| HttpOnly | `true` |
| SameSite | `Lax` |
| Path | `/` |
| Secure | `true` when production-secure mode; `false` for local HTTP/dev |
| MaxAge | Leave session cookie as browser-session default unless already set; logout uses `-1` |
| Rotation | Register/login: expire prior session cookie, then `Sessions.New` + save authenticated session |
| Logout | Clear `Values`, set `MaxAge=-1`, preserve Path/HttpOnly/SameSite/Secure on delete cookie, `Save` |
| Secret | Required env/flag `COOKIE_KEY`; production-secure mode refuses empty, default sentinel, or length &lt; 32 bytes |

**Production-secure mode** is true when any of:
- `APP_ENV=production` (case-insensitive), or
- `COOKIE_SECURE=true` / `1` / `yes`

Local `make run` without those vars keeps `Secure=false` so Vite/HTTP continues to work. Docker production compose should set `APP_ENV=production` and a strong `COOKIE_KEY`.

**Trusted HTTPS note:** `Secure` cookies require HTTPS at the browser. If TLS terminates at Nginx in front of the Go process, document that the public site must be HTTPS; do not invent proxy header parsing here (Prompt 03 owns perimeter/proxy assumptions). Cookie flags are still set correctly on the Set-Cookie response from Go.

## Acceptance Criteria

1. New registrations store bcrypt hashes only (prefix `$2`); no new SHA-256 hex hashes are written.
2. Existing users with legacy SHA-256 hashes can still log in with the same password.
3. After a successful legacy login, `password_hash` is bcrypt (or a logged ERROR explains upgrade failure and a later login upgrades).
4. Password verification for both paths uses library/constant-time compare — no `==` on password material.
5. Passwords longer than 72 bytes are rejected with `password_too_long` on register and login (client + server).
6. Register and login issue a rotated session (`Sessions.New` after invalidating any prior `sid`); a pre-login session cookie cannot be silently reused as the post-auth session identity.
7. When production-secure mode is on, cookies are `Secure`; startup fails fast if `COOKIE_KEY` is missing, is the documented default sentinel, or is shorter than 32 bytes.
8. Logout clears the cookie such that subsequent `/api/me` returns 401; delete-cookie options include the same Path/Secure/SameSite as live cookies so browsers actually drop them.
9. `docker-compose.yml`, `docker-compose.dev.yml`, and README Docker env docs use `COOKIE_KEY` (not `SESSION_SECRET`); compose no longer relies on the insecure default key by omission.
10. Automated tests cover bcrypt register, legacy login + upgrade, wrong password, max length, session rotation, Secure flag under production-secure mode, secret validation, and logout invalidation.
11. README documents bcrypt, legacy upgrade, `COOKIE_KEY`, `APP_ENV`/`COOKIE_SECURE`, and removes the “unsalted SHA-256” production caveat in favor of accurate current behavior.

## Security Implications

- Replaces unsalted SHA-256 (fast offline crackable) with bcrypt cost 12.
- Legacy hashes remain weak until each user logs in once; acceptable for this private app; optional operator note: force password reset is out of scope.
- Session fixation resistance improves via post-auth session regeneration.
- `Secure` cookies reduce theft over cleartext HTTP in production deployments.
- Fail-fast secret validation prevents Docker’s current silent default-key footgun.
- CookieStore sessions are client-side signed blobs: logout cannot revoke a stolen cookie copy until expiry/key rotation; document this limit. Server-side session store is out of scope.
- Logging must never include passwords, raw cookie values, or full hashes (log `hash_kind=bcrypt|legacy` and `userID=` only).
- Residual risks deferred to Prompt 03: reflective CORS, CSRF on cookie-auth mutations, WebSocket origin checks.

## Failure Handling

| Failure | Behavior |
|---------|----------|
| bcrypt generate error | Register → 500 `registration_failed`; log ERROR without password |
| Legacy verify OK, upgrade write fails | Login succeeds; log ERROR `[auth.Login] upgrade_hash_failed userID=`; retry next login |
| Unknown / corrupt hash string | Treat as invalid credentials (401); DEBUG log `hash_kind=unknown` |
| Production-secure + bad `COOKIE_KEY` | Process exits at startup with clear message (do not listen) |
| Logout Save error | Still return 200 `{ok:true}` only if cookie clear likely succeeded; if Save fails, log ERROR and return 500 so client can retry |
| Password &gt; 72 bytes | 400 `password_too_long` (same message client/server) |

## Commit Plan
- **Commit 1** (after tasks 1–3): `feat(auth): bcrypt passwords with legacy SHA-256 upgrade`
- **Commit 2** (after tasks 4–6): `feat(auth): rotate sessions and enforce secure cookie secrets`
- **Commit 3** (after tasks 7–9): `fix(deploy): align COOKIE_KEY and document password/session hardening`

## Tasks

### Phase 1: Password KDF and legacy migration

- [x] Task 1: Introduce bcrypt hash helpers and constant-time verify/upgrade API
  - Add `golang.org/x/crypto/bcrypt` dependency (`go get`).
  - Replace `hashPassword` usage with a small internal API in `internal/auth/` (same package or `password.go`):
    - `HashPassword(pw string) (string, error)` → `bcrypt.GenerateFromPassword([]byte(pw), 12)`.
    - `VerifyPassword(stored, pw string) (ok bool, needsUpgrade bool, err error)`:
      - bcrypt prefix → `CompareHashAndPassword`; `needsUpgrade=false`.
      - else legacy SHA-256 hex → constant-time compare; `needsUpgrade=true` on match.
    - Do not retain a public unsalted SHA-256 helper for new writes.
  - Register: hash with bcrypt only; on hash error return `registration_failed`.
  - Login: verify via helper; on success + `needsUpgrade`, call store update (Task 2).
  - Keep existing auth error codes; add `password_too_long` for `len([]byte(password)) > 72`.
  - LOGGING REQUIREMENTS:
    - DEBUG: `hash_kind=bcrypt|legacy|unknown`, validation `code=` (never password/hash material).
    - INFO: successful login/register with `userID=` and `upgraded=true|false`.
    - ERROR: bcrypt generate/compare unexpected errors; upgrade persistence failures.
    - Prefix: `[auth.Register]`, `[auth.Login]`, `[auth.hash]`.
  - Files: `internal/auth/auth.go`, optional `internal/auth/password.go`, `go.mod`, `go.sum`
  - Depends on: none

- [x] Task 2: Persist upgraded password hashes
  - Add `Store.UpdateUserPasswordHash(userID int64, passwordHash string) error` updating `users.password_hash` by id.
  - Call from Login only after successful legacy verify; do not change email uniqueness or other columns.
  - No DDL migration required for this change.
  - LOGGING REQUIREMENTS: ERROR on update failure with `userID=` only; DEBUG on successful upgrade write.
  - Files: `internal/db/db.go`, `internal/db/db_test.go` (basic update test), `internal/auth/auth.go`
  - Depends on: Task 1

- [x] Task 3: Password policy max length + frontend parity
  - Extend `validateCredentials` / messages for `password_too_long`.
  - Update `LoginPage.vue` client validation and `ERROR_COPY` to match (byte length: document that client uses UTF-16 `length` vs server byte length carefully — prefer counting UTF-8 bytes in TS via `new TextEncoder().encode(pw).length` for parity).
  - Extend auth handler tests for too-long rejection on register and login.
  - LOGGING REQUIREMENTS: DEBUG validation failures with `code=password_too_long`.
  - Files: `internal/auth/auth.go`, `web/src/pages/LoginPage.vue`, `web/tests/unit/auth-page.spec.ts`, `internal/auth/auth_handlers_test.go`
  - Depends on: Task 1

### Phase 2: Session rotation, Secure cookies, secret validation

- [x] Task 4: Rotate session on authentication; harden logout invalidation
  - Refactor `startSession` to:
    1. Load existing session (if any), clear `Values`, set `MaxAge=-1`, apply canonical Options (Path/HttpOnly/SameSite/Secure), `Save`.
    2. `sess, err := s.Sessions.New(r, sessionName)` — fail closed on error.
    3. Set `user_id`, apply canonical Options (`MaxAge` back to store default / `0`), `Save`.
  - Extract `applySessionOptions(sess *sessions.Session)` used by startSession and Logout so Secure/Path stay consistent.
  - Logout: delete all `Values`, `MaxAge=-1`, apply same Options, `Save`; handle Save errors.
  - LOGGING REQUIREMENTS:
    - DEBUG: `[auth.startSession] rotated userID=`
    - DEBUG: `[auth.Logout] invalidated`
    - ERROR: session New/Save failures without cookie bytes.
  - Files: `internal/auth/auth.go`
  - Depends on: none (can parallelize with Phase 1; integrate before Task 6 tests)

- [x] Task 5: Production-secure cookie mode and required `COOKIE_KEY` validation
  - In `cmd/server/main.go` (and shared helpers if cleaner in `internal/auth`):
    - Resolve `secureCookies` from `APP_ENV` / `COOKIE_SECURE` as defined above.
    - Set `sessionStore.Options.Secure` accordingly; ensure `startSession`/Logout copy the store Options (or Service field `Secure bool`).
    - Validate cookie key: if `secureCookies`, reject empty, length &lt; 32, and exact default sentinel `change-me-please-32-bytes-min` (and README’s `please-change-this-32-bytes-min` if still documented as example — treat both as forbidden sentinels in production-secure mode).
    - Non-secure local mode: WARN once at startup if key is default/short, but allow run for developer convenience.
  - Pass Secure into `auth.Service` if needed so handlers do not re-read env inconsistently.
  - LOGGING REQUIREMENTS:
    - INFO: `[main] cookie_secure=true|false`
    - WARN: `[main] weak COOKIE_KEY in non-production mode`
    - FATAL: validation failure in production-secure mode (existing `log.Fatalf`).
  - Files: `cmd/server/main.go`, `internal/auth/auth.go` (Options helper)
  - Depends on: Task 4

- [x] Task 6: Auth/session automated tests for KDF, upgrade, rotation, Secure, logout
  - Extend `internal/auth/auth_handlers_test.go` (and/or new `password_test.go`):
    - Register → stored hash bcrypt-prefix; login success.
    - Seed user with legacy SHA-256 hex → login success → DB hash now bcrypt; second login still works.
    - Wrong password legacy + bcrypt → 401 `invalid_credentials`.
    - `password_too_long` → 400.
    - Session rotation: capture pre-auth cookie (if any), after login ensure `/api/me` works; after logout `/api/me` is 401.
    - With Service/store Options `Secure=true`, `Set-Cookie` for `sid` includes `Secure`.
    - Unit-test key validation helper for forbidden sentinels when secure mode on.
  - Keep tests using explicit test keys (≥32 bytes), never production defaults.
  - LOGGING REQUIREMENTS: tests assert behavior; do not assert log text.
  - Files: `internal/auth/auth_handlers_test.go`, `internal/auth/password_test.go` (new), optional `cmd/server` validation test if helper extracted
  - Depends on: Tasks 1–5

### Phase 3: Deployment config and documentation

- [x] Task 7: Fix Docker Compose `SESSION_SECRET` → `COOKIE_KEY` mismatch
  - In `docker-compose.yml`: set `COOKIE_KEY` to a clearly placeholder secret (≥32 chars), add `APP_ENV=production` (or `COOKIE_SECURE=true`). Remove `SESSION_SECRET`.
  - In `docker-compose.dev.yml`: set `COOKIE_KEY=dev-secret-key-change-me-32b!!` (or similar ≥32), omit production-secure flags so HTTP works; remove `SESSION_SECRET`.
  - Optionally align `ADDR=:8080` instead of unused `PORT=8080` while touching compose (small related fix; do not expand into full container networking redesign).
  - LOGGING REQUIREMENTS: N/A; verify container boot log shows `cookie_secure=` expected for each compose file.
  - Files: `docker-compose.yml`, `docker-compose.dev.yml`
  - Depends on: Task 5

- [x] Task 8: Documentation updates
  - README Advanced Setup / Docker / API sections:
    - Document bcrypt + transparent legacy upgrade.
    - Document `COOKIE_KEY`, `APP_ENV`, `COOKIE_SECURE`, startup validation, Secure cookie behavior.
    - Remove `SESSION_SECRET` references; fix Docker env example.
    - Note bcrypt 72-byte limit / `password_too_long`.
    - Note CookieStore logout limitation (no server-side revoke list).
    - Do not claim Argon2id; do not call the recognizer “AI” in newly edited paragraphs.
  - LOGGING REQUIREMENTS: document `[auth.*]` / `[main] cookie_secure` operator signals and `LOG_LEVEL`.
  - Files: `README.md`
  - Depends on: Tasks 5, 7

- [x] Task 9: Verification against acceptance criteria
  - Run Go auth/db tests and `cd web && npm test`.
  - Smoke checklist (record in PR/commit notes when implementing):
    1. Fresh register → DB hash starts with `$2`; session cookie present; `/api/me` OK.
    2. Manually insert legacy SHA-256 user (or use test helper) → login → hash upgraded → strokes still owned by same user id.
    3. Logout → `/api/me` 401; login again works.
    4. `APP_ENV=production COOKIE_KEY=change-me-please-32-bytes-min` → process refuses to start.
    5. Production-secure mode → `Set-Cookie` contains `Secure`.
    6. `docker compose config` shows `COOKIE_KEY`, not `SESSION_SECRET`.
  - Confirm logs never print passwords.
  - Files: none required
  - Depends on: Tasks 6–8

## Out of Scope
- Password reset / email verification / forced mass re-hash without login
- Argon2id (deferred unless bcrypt proves insufficient)
- Server-side session store / refresh tokens / JWT
- CSRF, CORS allowlists, WebSocket origin checks (Prompt 03)
- Rate limiting / account lockout
- Versioned SQL migration framework (Prompt 09) — not required for hash-string upgrade
- OAuth/OIDC
- Changing cookie name `sid` or moving off gorilla/sessions

## Risks & Notes
- **Residual legacy hashes:** Users who never log in keep SHA-256 until they do; acceptable for private MVP; operators can delete stale accounts manually if needed.
- **bcrypt cost 12:** ~100–300ms per hash on typical CPUs — fine for low-traffic educational app; do not lower cost without documenting why.
- **Cookie key rotation:** Changing `COOKIE_KEY` logs everyone out (signed cookies unverifiable) — document as expected.
- **Secure + HTTP:** Enabling `APP_ENV=production` behind plain HTTP will make browsers drop/ignore cookies — compose/docs must pair Secure with HTTPS termination.
- **Onboarding dependency:** Keep error JSON shape and anti-enumeration rules from `.ai-factory/plans/public-onboarding.md`; only add `password_too_long`.
- **Parallel Prompt 03:** Secure cookies assume later perimeter work will make production same-origin/HTTPS coherent; this plan only sets the cookie flag and secret validation.
