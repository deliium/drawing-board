# Implementation Plan: Public Account Onboarding

Branch: main
Created: 2026-09-07

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Summary

Anonymous users are redirected to `/login`, but `LoginPage.vue` only supports login. Registration UI lives on `BoardPage` behind `requiresAuth`, so new users cannot create accounts through the normal flow.

Deliver a public authentication experience (login + registration) on the public auth routes, keep cookie-based sessions (`credentials: 'include'` + `sid`), redirect authenticated users away from auth pages to the board, and remove the dead login/register controls from the board. Logout remains on the board.

This is roadmap item **1. Public onboarding** from `.ai-factory/japanese-learning-aif-plan-prompts.md`. Password KDF hardening, Secure cookies, and CSRF/CORS belong to later plans; do not expand into those here.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Guard | `web/src/router/guards.ts` sends unauthenticated users to `login` when `meta.requiresAuth` |
| Routes | `/` → `BoardPage` (auth required); `/login` → `LoginPage` (public) |
| Login page | Email/password only; placeholders, no labels/autocomplete; `window.location.hash = '#/'` after success |
| Board page | Duplicate login/register when `!user` (unreachable under the guard) + logout |
| API | `POST /api/register`, `/api/login`, `/api/logout`, `GET /api/me` already exist |
| Enumeration | Register returns `409 {"error":"email exists"}` — reveals account existence |
| Validation | Server only checks non-empty email/password; client has none |
| `apiFetch` | Throws `{ status, message: "Request failed: N" }` — ignores JSON `error` body |
| Auth tests | `internal/auth/auth_test.go` only checks `NewService` wiring; no HTTP register/login tests |
| Frontend tests | Route name smoke only; no guard/auth-page coverage; a11y test is a placeholder |
| Sessions | Cookie `sid`, `HttpOnly`, `SameSite=Lax`, `Path=/`; client already sends credentials |

## Approach

1. **Reuse existing auth HTTP endpoints**; tighten validation and error codes without changing cookie session mechanics.
2. **Replace `LoginPage` with a public auth page** that supports login and create-account modes (tabs or segmented control), optionally aliased as `/register` → same component with register mode selected.
3. **Redirect authenticated users** off auth routes to `/`; after logout, navigate to `/login`.
4. **Strip unreachable auth forms from `BoardPage`**; keep logout and practice UI for authenticated users only.
5. **Surface specific, safe errors** to the UI via `apiFetch` + stable error codes; never confirm whether an email is registered on login, and never return `email exists` on register.

## Auth Error Contract

Stable JSON shape for auth failures:

```json
{ "error": "<code>", "message": "<optional human-readable>" }
```

| HTTP | Code | When | User-facing copy (English) |
|------|------|------|----------------------------|
| 400 | `bad_json` | Body not JSON | Something went wrong. Try again. |
| 400 | `missing_fields` | Empty email or password | Enter email and password. |
| 400 | `invalid_email` | Email fails basic format check | Enter a valid email address. |
| 400 | `password_too_short` | Password below minimum (8) | Password must be at least 8 characters. |
| 400 | `registration_failed` | Duplicate email or other create failure | Unable to create account. If you already have one, sign in. |
| 401 | `invalid_credentials` | Bad login | Email or password is incorrect. |
| 401 | `unauthorized` | `/api/me` without session | (client treats as anonymous) |
| 200 | — | Success | Body: `{ "id", "email" }` (register/login) or `{ "ok": "true" }` (logout) |

**Enumeration rules:**
- Login always uses `invalid_credentials` for unknown email and wrong password.
- Register never returns `email exists` or 409 that uniquely signals a taken email; map unique-constraint / existing-user to `registration_failed`.
- Server DEBUG/INFO logs may record the real reason (e.g. `reason=email_taken`); responses and UI must not.

**Out of scope:** password reset, OAuth/OIDC, Argon2/bcrypt migration, `Secure` cookie flags, CSRF tokens, rate limiting (note as follow-up risk only).

## Acceptance Criteria

1. Unauthenticated visit to `/#/` lands on public auth UI that can **both** register and log in.
2. Successful register or login sets session cookie, updates `sessionContext`, and navigates to `/#/` board without a full hard reload requirement beyond existing cookie flow.
3. Authenticated visit to `/#/login` (and `/#/register` if added) redirects to `/#/`.
4. `BoardPage` has **no** email/password/register/login controls; logout remains and clears session + returns user to auth page.
5. Forms have visible `<label>` (or `aria-label`), `autocomplete` (`username`/`email`, `current-password` / `new-password`), and disable submit while pending.
6. Client and server reject empty fields, invalid email shape, and passwords shorter than 8 characters with specific messages.
7. Duplicate registration does not reveal “email exists”; login failures stay generic.
8. Auth layout is usable on a narrow phone viewport (stacked fields, ≥44px tap targets, no horizontal-only control row).
9. Automated Go API tests cover register/login/logout/me success and validation/enumeration-safe failures.
10. Automated Vue/route tests cover guard redirects, auth-page modes, and board no longer exposing register.
11. README “Getting Started” / API notes match the public onboarding flow.

## Security Implications

- Closing the unreachable-registration bug is the primary product fix; anti-enumeration reduces account discovery via `/api/register`.
- Cookie session model is preserved (`credentials: 'include'`); do not introduce bearer tokens.
- Password storage remains unsalted SHA-256 until the Password and Session Security plan — document that limitation in README briefly, do not invent a new hash format here.
- Logging must never include passwords or full cookie values.
- Residual risks deferred: weak password hashing, missing CSRF on cookie-auth mutations, reflective CORS — owned by later roadmap plans.

## Failure Handling

| Failure | Behavior |
|---------|----------|
| Network / 5xx | Show generic “Unable to reach the server. Try again.”; clear loading; leave form values |
| Validation (client) | Inline field errors; do not call API |
| Validation (server) | Map `error` code to copy; keep email; clear password only after successful auth |
| Duplicate register | `registration_failed` copy + affordance to switch to Sign in |
| Invalid login | `invalid_credentials`; keep email; clear or keep password per UX choice (prefer clear password) |
| Session restore fail on boot | Existing `main.ts` → anonymous; guard sends protected routes to login |
| Double-submit | Disable buttons / ignore while `submitting` |

## Commit Plan
- **Commit 1** (after tasks 1–2): `fix(auth): validate register/login and stop account enumeration`
- **Commit 2** (after tasks 3–6): `feat(web): public login and registration onboarding`
- **Commit 3** (after tasks 7–9): `test(docs): cover onboarding routes/API and update README`

## Tasks

### Phase 1: Auth API contract and server validation

- [x] Task 1: Harden register/login validation and anti-enumeration responses
  - Update `internal/auth/auth.go` to:
    - Normalize email (`TrimSpace` + `ToLower`) as today.
    - Reject missing fields, invalid email (lightweight RFC-ish check: `@` + domain label), and passwords shorter than 8 with the codes in the Auth Error Contract.
    - On existing email / unique constraint during register: respond with `registration_failed` (not `email exists` / not a distinct “taken” signal). Prefer HTTP 400 for this code for a single client mapping path, or document 400 consistently.
    - Keep successful register/login calling `startSession` and returning `userView`.
    - Preserve cookie options already set in `startSession` / `cmd/server/main.go` (Path, HttpOnly, SameSite=Lax).
  - LOGGING REQUIREMENTS:
    - DEBUG: validation failures with `code=` (never password).
    - INFO: successful register/login with `userID=` only.
    - WARN: registration conflict mapped to `registration_failed` with `reason=email_taken` server-side only.
    - ERROR: unexpected store errors (no request body dump).
    - Prefix: `[auth.Register]`, `[auth.Login]`, `[auth.Logout]`, `[auth.Me]`.
  - Files: `internal/auth/auth.go`
  - Depends on: none

- [x] Task 2: Add HTTP-level auth API tests
  - Add table-driven / httptest tests covering:
    - Register success → 200 + `Set-Cookie` + subsequent `/api/me` 200.
    - Register missing/invalid/short password → 400 with expected codes.
    - Register duplicate email → `registration_failed` (assert body does not contain `email exists` / email string as “exists” wording).
    - Login success / wrong password / unknown email → 401 `invalid_credentials` only.
    - Logout clears session so `/api/me` becomes 401.
  - Use existing SQLite test patterns from `internal/db` / `internal/httpapi/handlers_test.go` (temp DB + cookie store).
  - LOGGING REQUIREMENTS: tests assert behavior, not log lines; handlers still emit verbose logs when run under test if default logger used.
  - Files: `internal/auth/auth_handlers_test.go` (new) or extend `internal/auth/auth_test.go`
  - Depends on: Task 1

### Phase 2: Public auth UX and board cleanup

- [x] Task 3: Teach `apiFetch` to expose server auth error codes
  - Parse JSON `{ error, message? }` on non-OK responses when present; put `error` on `ApiError` (e.g. `code?: string`) while keeping `status`.
  - Do not break existing callers that only catch and ignore.
  - LOGGING REQUIREMENTS: in DEV only, `console.debug('[apiFetch]', status, code)` — never log request bodies with passwords.
  - Files: `web/src/services/apiClient.ts`
  - Depends on: Task 1 (contract)

- [x] Task 4: Build public login + registration page
  - Replace/expand `web/src/pages/LoginPage.vue` (or rename to `AuthPage.vue` and update imports) to support:
    - Modes: **Sign in** and **Create account** (toggle; deep-link via route name/query or `/register` alias).
    - Accessible labels associated with inputs; `autocomplete="email"` / `username`, `autocomplete="current-password"` (login) and `new-password` (register).
    - Client-side validation mirroring server rules before submit.
    - Loading/disabled submit state; `aria-busy` / live region for errors (`role="alert"`).
    - Map API codes to the user-facing copy table.
    - On success: `setAuthenticatedUser` from login/register response or `/api/me`, then `router.replace({ name: 'board' })` (prefer router over `window.location.hash`).
    - Mobile: stacked column layout, full-width controls, comfortable tap targets; reuse light styles in `web/src/styles/` or scoped CSS (no purple-gradient redesign; keep consistent with current practice app shell).
  - LOGGING REQUIREMENTS: DEV `console.debug('[AuthPage]', mode, 'submit'|'success'|'error', code?)` without credentials.
  - Files: `web/src/pages/LoginPage.vue` and/or `web/src/pages/AuthPage.vue`, optional small CSS module
  - Depends on: Task 3

- [x] Task 5: Router redirects for guests and authenticated users
  - Keep hash history.
  - Add `meta.guestOnly` (or equivalent) on auth routes so authenticated users are redirected to `board`.
  - Optionally add `{ path: '/register', name: 'register', ... }` pointing at the same auth component with register mode.
  - Update `requireAuth` / add `requireGuest` in `web/src/router/guards.ts`; cover both in `beforeEach`.
  - Ensure anonymous → board still goes to login; authenticated → login/register goes to board.
  - LOGGING REQUIREMENTS: DEV debug when a redirect fires: `[router] redirect reason=auth|guest from= to=`.
  - Files: `web/src/router/index.ts`, `web/src/router/guards.ts`
  - Depends on: Task 4

- [x] Task 6: Remove duplicated auth controls from the board; keep logout
  - Delete email/password inputs, `doLogin`, `doRegister`, and the `v-if="!user"` auth strip from `BoardPage.vue`.
  - Keep logout; after logout call `setAuthenticatedUser(null)`, close WS, clear local strokes/candidates, `router.replace({ name: 'login' })`.
  - Board assumes authenticated user (guard); remove dead “Sign in to practice” branch if unreachable, or replace with a safe empty state only if needed for tests.
  - LOGGING REQUIREMENTS: DEBUG logout success `[BoardPage] logout`.
  - Files: `web/src/pages/BoardPage.vue`
  - Depends on: Task 5

### Phase 3: Tests, docs, verification

- [x] Task 7: Frontend route and auth-page tests
  - Extend/replace thin route specs:
    - Routes include `login` (and `register` if added); board still `requiresAuth`.
    - Guard: anonymous + board → login; authenticated + guest route → board.
  - Component/unit tests for auth page: mode switch, client validation messages, submit disabled while loading (mock `apiFetch`).
  - Assert `BoardPage` template no longer contains Register/Login form controls (mount or static import/source assertion acceptable if full mount is heavy; prefer `@vue/test-utils` if added as devDependency — add only if needed).
  - LOGGING REQUIREMENTS: none in tests beyond silenced console if noisy.
  - Files: `web/tests/integration/us1-deeplink.spec.ts`, `web/tests/integration/us3-route-parity.spec.ts`, new `web/tests/unit/auth-page.spec.ts` and/or `web/tests/unit/auth-guards.spec.ts`; `web/package.json` only if test util dependency required
  - Depends on: Tasks 4–6

- [x] Task 8: Documentation updates
  - Update README Getting Started: register/login happen on the public auth page before the board; remove any implication that registration is on the canvas header.
  - Document auth error codes briefly under API Reference (no password-reset section).
  - Note that password hashing hardening is a follow-up (do not claim Argon2/bcrypt yet).
  - Avoid calling the recognizer “AI” in any new copy touched in this task; leave unrelated stale AI wording to later cleanup plan unless editing the same paragraph.
  - LOGGING REQUIREMENTS: N/A for docs; mention that auth handlers emit `[auth.*]` logs for operators.
  - Files: `README.md` (and `specs/001-migrate-frontend-vue/contracts/frontend-parity-contract.md` only if auth entry-flow wording is still referenced)
  - Depends on: Tasks 4–6

- [x] Task 9: End-to-end verification of acceptance criteria
  - Run `make test` (or project Go test script) and `cd web && npm test`.
  - Manually smoke (document checklist in plan completion notes when implementing):
    1. Fresh browser: open app → auth page → create account → land on board with session.
    2. Logout → auth page; login again restores strokes for that user only.
    3. Open `/#/login` while logged in → redirected to board.
    4. Duplicate register attempt → safe message; sign-in still works.
    5. Narrow viewport (~375px): form usable without horizontal scrolling of controls.
  - LOGGING REQUIREMENTS: during smoke, confirm no passwords appear in server logs.
  - Files: none required; record results in commit/PR notes
  - Depends on: Tasks 2, 7, 8

## Out of Scope
- Password reset / email verification
- Third-party identity providers
- Argon2id/bcrypt migration, session rotation, production `Secure` cookies, `SESSION_SECRET`/`COOKIE_KEY` Docker mismatch (Prompt 02)
- CORS / CSRF / WebSocket origin hardening (Prompt 03)
- Full bilingual/responsive product redesign (Prompt 14) — only mobile-usable auth form layout here
- Lesson/curriculum features

## Risks & Notes
- Changing register’s `409 email exists` is an intentional breaking change for any external client relying on that string; acceptable for this private app — document in README API section.
- Minimum password length of 8 is a product policy for onboarding; Prompt 02 may raise policy further with KDF work.
- `AppShell` already wraps all routes; avoid duplicating the product title awkwardly if `BoardPage` still has its own header — prefer one clear auth heading (“Sign in” / “Create account”) without a second marketing hero.
- Hash router: all redirects must use route names/paths understood by `createWebHashHistory`.
