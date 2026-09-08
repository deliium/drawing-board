# Implementation Plan: HTTP and WebSocket Perimeter Security

Branch: main
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Summary

The Go server reflects any `Origin` into `Access-Control-Allow-Origin` while setting `Access-Control-Allow-Credentials: true`, WebSocket `CheckOrigin` always returns `true`, and cookie-authenticated mutating REST endpoints have no explicit CSRF defense. Combined with Prompt 02’s `Secure` cookies, production still lacks a coherent browser-to-server trust boundary.

Deliver an **environment-aware same-origin / allowlist policy**: production requires an explicit origin allowlist (no wildcards); development defaults to a Vite allowlist. Apply the same policy to CORS (including preflight) and WebSocket upgrades. Add CSRF protection for REST mutations. Document trusted-proxy and HTTPS assumptions, harden Docker/Nginx defaults, validate config at startup, and add negative security tests. Preserve local Vite (`5173` proxy) and authenticated WebSocket operation.

This is roadmap item **3. HTTP and WebSocket perimeter security** from `.ai-factory/japanese-learning-aif-plan-prompts.md`. Password/session cookie flags remain owned by Prompt 02; do not rework bcrypt or session rotation here.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| CORS | `withCORS` in `cmd/server/main.go` sets `ACAO` to raw `Origin`, `ACAC=true`, allows `Content-Type`, methods `GET,POST,OPTIONS`; OPTIONS → 204 unconditionally |
| WebSocket origin | `internal/ws/handler.go` `CheckOrigin: func(...) bool { return true }` |
| CSRF | None on REST; mutations rely only on session cookie `sid` + `SameSite=Lax` |
| Mutating POSTs | `/api/register`, `/api/login`, `/api/logout`, `/api/strokes/clear`, `/api/strokes/delete`, `/api/recognize` (auth required except register/login) |
| Safe GETs | `/api/me`, `/api/strokes`, `/healthz` |
| WS auth | `/ws` wrapped in `RequireAuth`; cookie session required before upgrade |
| Frontend API | `apiFetch` uses relative paths + `credentials: 'include'`; Vite proxies `/api` and `/ws` to `:8080` |
| WS URL | Vite port `5173` → `ws://hostname:5173/ws` (proxied); else `ws(s)://location.host/ws` |
| Cookie flags | Prompt 02: `HttpOnly`, `SameSite=Lax`, `Secure` when `APP_ENV=production` / `COOKIE_SECURE` |
| APP_ENV | Already used for production-secure cookies (`internal/auth/cookie.go`) |
| ALLOWED_ORIGINS | Does not exist |
| Nginx | `docker/nginx.conf`: HTTP `:80` only; proxies `/api/` and `/ws` to `backend:8080`; sets `X-Forwarded-*`; weak CSP with `'unsafe-inline'` |
| Compose prod | Publishes `backend:8080:8080` and `frontend:80:80`; `APP_ENV=production` + `COOKIE_KEY` |
| Compose dev | Backend `:8080`, frontend `:3000:80`; no `APP_ENV` |
| Perimeter tests | None for CORS / WS origin / CSRF (auth tests cover session/Secure only) |

## Approach

1. **Centralize origin policy** in a small package or helpers (e.g. `internal/security` or `internal/httpapi/origin.go`) shared by CORS middleware and WebSocket `CheckOrigin`.
2. **Config via `ALLOWED_ORIGINS`**: comma-separated exact origins (`scheme://host[:port]`). No `*`, no path suffixes, no regex wildcards.
3. **Environment defaults**:
   - **Development** (`APP_ENV` unset / not `production`): if `ALLOWED_ORIGINS` empty → default `http://localhost:5173,http://127.0.0.1:5173`.
   - **Production** (`APP_ENV=production`): `ALLOWED_ORIGINS` **required**; refuse empty list and any entry containing `*`.
4. **CORS**: echo `ACAO` only for allowlisted `Origin`; omit CORS allow headers (or respond 403 on preflight) for others; keep `Vary: Origin`; never set `ACAO=*` with credentials.
5. **CSRF**: double-submit cookie for all `POST /api/*` mutations (including login/register/logout). Issue/refresh non-HttpOnly `csrf` cookie on safe bootstrap; require matching `X-CSRF-Token` header.
6. **WebSocket**: same allowlist in `CheckOrigin`; reject missing Origin in production; in development allow missing Origin only if explicitly documented as unsafe-local opt-in is **not** used — prefer requiring Origin always when present from browsers; reject empty Origin in production-secure / production mode.
7. **Proxy/HTTPS**: document TLS at the edge; Nest Go behind Nginx; stop publishing backend port in production compose; do not trust `X-Forwarded-*` for auth decisions in this plan.
8. **Frontend**: `apiFetch` sends `X-CSRF-Token` from the `csrf` cookie; ensure login page bootstraps CSRF before first POST.

## Decision: Origin allowlist + double-submit CSRF

| Option | Pros | Cons |
|--------|------|------|
| CORS/WS allowlist only | Fixes reflective CORS; small change | Prompt requires explicit CSRF for mutations; Lax alone is incomplete if misconfig reopens CORS |
| Synchronizer token in session store | Strong | CookieStore has no server-side session DB; heavier |
| Custom header only (`X-Requested-With`) | Simple | Weaker than token match; still needs CORS locked |
| **Allowlist + double-submit CSRF** | Fits CookieStore SPA; works with Vite proxy; no wildcard temptation | Needs cookie+header wiring on client |

**Decision:** Exact-origin allowlist for CORS and WebSocket, plus double-submit CSRF (`csrf` cookie + `X-CSRF-Token`) on all `POST /api/*`. No `Access-Control-Allow-Origin: *`. No `ALLOWED_ORIGINS=*` or substring wildcards.

## Origin Policy Contract

| Mode | `ALLOWED_ORIGINS` | Behavior |
|------|-------------------|----------|
| Development | unset/empty | Effective list = `http://localhost:5173`, `http://127.0.0.1:5173` |
| Development | set | Use parsed list only (replace defaults entirely) |
| Production | unset/empty | **Startup fatal** |
| Production | set | Exact match only; fatal if any entry is `*`, empty, or invalid |

**Exact match rules:**
- Compare scheme + host + port as URLs (`url.Parse`); reject entries with path/query/fragment (non-empty `Path` other than `""` or `"/"`).
- Allow only `http` and `https` schemes.
- Matching is case-sensitive for scheme/host as normalized by `url.Parse` (lowercase host).
- Request with **no** `Origin` header:
  - **CORS**: do not set `ACAO` (same-origin navigations/non-CORS clients).
  - **Mutating CSRF middleware**: still enforce CSRF header/cookie (not Origin-only).
  - **WebSocket `CheckOrigin`**: return `false` in production; in development return `false` as well unless the request is same-host loopback **and** Origin is present on the allowlist — prefer **reject missing Origin always** for WS to avoid accidental open upgrades from non-browser clients pretending to be local. Document that browsers always send Origin on WS.

**CORS response when Origin allowlisted:**
- `Access-Control-Allow-Origin: <exact origin>`
- `Access-Control-Allow-Credentials: true`
- `Access-Control-Allow-Headers: Content-Type, X-CSRF-Token`
- `Access-Control-Allow-Methods: GET, POST, OPTIONS`
- `Vary: Origin`
- OPTIONS → `204` without hitting handlers

**CORS when Origin present but not allowlisted:**
- Do **not** set `ACAO` / `ACAC`
- OPTIONS → `403` with no body (or empty); non-OPTIONS continue without CORS headers (browser will hide response from foreign JS)

## CSRF Contract

| Item | Value |
|------|-------|
| Cookie name | `csrf` |
| Cookie flags | `Path=/`, `SameSite=Lax`, `Secure` iff production-secure mode (same as `sid`), **not** HttpOnly |
| Header | `X-CSRF-Token` |
| Compare | Constant-time equality of cookie value and header |
| Token entropy | ≥ 32 bytes random, hex or base64url encoded |
| Issue | Ensure cookie on `GET /api/me` (even 401) and on successful register/login; also ensure-on-demand inside CSRF middleware for safe methods if a dedicated bootstrap is added |
| Enforce | All `POST` under `/api/` (register, login, logout, strokes/clear, strokes/delete, recognize) |
| Fail | `403` JSON `{ "error": "csrf_rejected", "message": "..." }` |
| Exempt | `GET /*`, `OPTIONS /*`, `/healthz`, `/ws` (WS uses Origin check) |

**Bootstrap for public auth:** `LoginPage` (or `main.ts`) must obtain a CSRF cookie before `POST /api/login` or `/api/register` — call `GET /api/me` (existing) or add `GET /api/csrf` that only sets/returns token. Prefer reusing `GET /api/me` to avoid new surface if 401 still sets the cookie; if cleaner, add `GET /api/csrf` returning `{ "csrf": "<token>" }` and setting the cookie.

**Frontend:** `apiFetch` merges `X-CSRF-Token` from `document.cookie` for non-GET requests. Do not log token values.

## Trusted Proxy and HTTPS Assumptions

| Assumption | Operator requirement |
|------------|----------------------|
| Browser origin | Public site origin is exactly one of `ALLOWED_ORIGINS` (e.g. `https://learn.example.com`) |
| TLS | HTTPS terminates at Nginx or an upstream load balancer; browsers must see HTTPS when `Secure` cookies are on |
| Go listen | Backend listens on internal network HTTP (`:8080`); not a public TLS terminator in this plan |
| Forwarded headers | Nginx may set `X-Forwarded-For` / `X-Forwarded-Proto` for logs; **this plan does not** use them for origin allowlisting or auth |
| Host header | Do not derive allowlist from `Host` alone; configure `ALLOWED_ORIGINS` explicitly |
| Dev Vite | Browser origin is `http://localhost:5173` (or `127.0.0.1`); proxy preserves path; WS Origin is the Vite origin |

**Out of this plan’s crypto scope:** ACME/Let’s Encrypt automation. Document a TLS-ready Nginx example (listen 443 + cert paths) or “terminate TLS upstream” runbook; production compose may keep HTTP for local Docker demos but must **warn** that `APP_ENV=production` + Secure cookies require HTTPS at the browser.

## Secure Configuration Defaults

| Variable | Dev default | Production |
|----------|-------------|------------|
| `APP_ENV` | unset | `production` |
| `ALLOWED_ORIGINS` | Vite loopback pair | **Required** explicit list |
| `COOKIE_KEY` | Prompt 02 rules | Prompt 02 rules |
| `COOKIE_SECURE` / Secure cookies | false unless set | true via `APP_ENV` |
| Backend publish | `:8080` OK for local | Prefer **no** host port publish; frontend Nginx only |
| CORS | Allowlist only | Allowlist only |
| WS CheckOrigin | Allowlist | Allowlist; reject missing Origin |

Startup must log (INFO) the **count** and **redacted** policy mode (`mode=development|production`, `allowed_origins_count=N`), not necessarily every origin if noisy — logging the effective origins at INFO is acceptable for this private app (they are not secrets).

## Acceptance Criteria

1. Production startup with empty/missing `ALLOWED_ORIGINS` fails before listen; startup with `*` or invalid entries fails with a clear message.
2. Development without `ALLOWED_ORIGINS` allows only `http://localhost:5173` and `http://127.0.0.1:5173` for CORS and WS.
3. Credentialed CORS from a non-allowlisted Origin does not receive `Access-Control-Allow-Origin` reflecting that Origin; preflight returns 403.
4. WebSocket upgrade from a non-allowlisted Origin is rejected (no successful upgrade).
5. `POST` to mutating `/api/*` without valid CSRF cookie+header returns 403 `csrf_rejected`; with valid pair succeeds (subject to auth).
6. Local Vite workflow works: register/login, `/api/me`, stroke clear/delete/recognize, authenticated WS echo — without wildcard CORS.
7. Production Docker path: frontend Nginx proxies `/api` and `/ws`; `ALLOWED_ORIGINS` set to the browser-facing origin; backend not required to be publicly published.
8. Negative automated tests cover disallowed CORS, disallowed WS origin, CSRF missing/mismatch, and startup validation failures.
9. README documents `ALLOWED_ORIGINS`, CSRF header/cookie, Vite defaults, HTTPS/proxy assumptions, and explicitly states wildcards are unsupported.
10. No `Access-Control-Allow-Origin: *` and no `CheckOrigin: return true` remain in code.

## Security Implications

- Stops cross-origin credentialed API access from arbitrary websites (fixes reflective CORS + credentials).
- WS origin checks block browser cross-site WebSocket abuse using the victim’s cookies.
- CSRF tokens block cross-site form/fetch mutation even if cookies are present under edge SameSite cases or future cookie policy changes.
- Publishing only Nginx in production reduces direct backend exposure.
- Residual risks: non-browser clients that can present Origin + steal CSRF cookie via XSS (XSS remains out of scope beyond existing Nginx headers); TLS still operator-owned; recognition payload limits remain Prompt 04.

## Failure Handling

| Failure | Behavior |
|---------|----------|
| Production missing/invalid `ALLOWED_ORIGINS` | `log.Fatalf` / exit before listen |
| CORS preflight disallowed Origin | 403; DEBUG log `origin=` and `reason=not_allowlisted` |
| WS disallowed / missing Origin | Upgrade fails; WARN log without cookie bytes |
| CSRF missing/mismatch | 403 `csrf_rejected`; DEBUG log `reason=missing_cookie|missing_header|mismatch` (no token values) |
| CSRF cookie issue failure (RNG) | 500 on bootstrap paths; ERROR log |
| Frontend missing CSRF before login | apiFetch gets 403; UI shows generic retry; LoginPage must bootstrap first |

## Commit Plan
- **Commit 1** (after tasks 1–3): `feat(security): origin allowlist for CORS and WebSocket`
- **Commit 2** (after tasks 4–6): `feat(security): CSRF double-submit for API mutations`
- **Commit 3** (after tasks 7–9): `chore(deploy): tighten Nginx/Compose origins and document perimeter`

## Tasks

### Phase 1: Origin policy, CORS, WebSocket

- [x] Task 1: Origin allowlist config + startup validation
  - Add helpers (prefer `internal/security/origins.go` or `internal/auth` sibling package `internal/security`):
    - `ParseAllowedOrigins(raw string) ([]string, error)`
    - `DefaultDevOrigins() []string`
    - `ResolveAllowedOrigins(appEnv, raw string) (origins []string, err error)` implementing the Origin Policy Contract
    - `OriginAllowed(origins []string, origin string) bool`
  - Reject `*`, blank entries, non-http(s), paths/queries/fragments.
  - Wire in `cmd/server/main.go`: resolve list after cookie validation; FATAL in production on error; INFO log mode + count (+ list).
  - LOGGING REQUIREMENTS:
    - INFO: `[main] origin_policy mode=… count=…`
    - FATAL: invalid/missing production allowlist
    - DEBUG: optional parse details without request context
  - Files: `internal/security/origins.go` (new), `internal/security/origins_test.go` (new), `cmd/server/main.go`
  - Depends on: none

- [x] Task 2: Replace reflective CORS with allowlist middleware
  - Refactor `withCORS` to accept the resolved allowlist (closure or middleware constructor).
  - Set CORS headers only when `Origin` is allowlisted; include `X-CSRF-Token` in `Allow-Headers` (needed before CSRF task lands — include now).
  - Disallowed Origin + OPTIONS → 403; do not reflect Origin.
  - Never use `*`.
  - Extract middleware to a testable function if that simplifies tests.
  - LOGGING REQUIREMENTS:
    - DEBUG: `[cors] allow origin=…` / `[cors] deny origin=… reason=…`
    - Do not log full cookie headers
  - Files: `cmd/server/main.go` and/or `internal/security/cors.go`
  - Depends on: Task 1

- [x] Task 3: WebSocket `CheckOrigin` uses the same allowlist
  - Replace always-true `CheckOrigin` with policy-backed function; inject allowlist via `ws.Init` / `Hub` / package-level setter initialized from `main` (avoid init-order races; prefer passing config into `Init`).
  - Reject non-allowlisted and (per contract) missing Origin.
  - Preserve `RequireAuth` ordering: auth still required; failed origin check must not upgrade.
  - LOGGING REQUIREMENTS:
    - WARN: `[ws.CheckOrigin] rejected origin=…` (empty string if missing)
    - DEBUG: accepted origin
  - Files: `internal/ws/handler.go`, `cmd/server/main.go`, `cmd/server/wire.go` if needed
  - Depends on: Task 1

### Phase 2: CSRF protection and frontend wiring

- [x] Task 4: CSRF token issue + middleware for POST `/api/*`
  - Implement token create/ensure and constant-time verify helpers in `internal/security` (or `internal/auth`).
  - Middleware order (outer → inner): logging → CORS → **CSRF** → mux.
  - CSRF applies to `POST` paths with prefix `/api/`; skip `/healthz`, static, `/ws`.
  - Ensure CSRF cookie on `GET /api/me` (401 and 200) and after successful login/register `Set-Cookie`.
  - Optional: `GET /api/csrf` if `/api/me` bootstrap is awkward for anonymous users — choose one approach and stick to it in docs/tests.
  - Error code `csrf_rejected` (403).
  - LOGGING REQUIREMENTS:
    - DEBUG: `[csrf] ok` / `[csrf] reject reason=…` without token material
    - ERROR: RNG / encode failures
  - Files: `internal/security/csrf.go` (new), `cmd/server/main.go`, `internal/auth/auth.go` (issue on me/login/register)
  - Depends on: Task 2 (Allow-Headers already lists CSRF header)

- [x] Task 5: Frontend CSRF bootstrap and `apiFetch` header
  - Update `web/src/services/apiClient.ts` to read `csrf` cookie and set `X-CSRF-Token` on mutating requests; preserve `credentials: 'include'`.
  - Ensure `LoginPage` / `main.ts` bootstraps CSRF (call `GET /api/me` or `/api/csrf`) before register/login POST.
  - Handle 403 `csrf_rejected` with a single retry after re-bootstrap where safe (optional); at minimum surface a clear message.
  - Do not change WS client protocol beyond continuing to use Vite/same-host URLs.
  - LOGGING REQUIREMENTS: client `console.debug` in DEV only; never print token values
  - Files: `web/src/services/apiClient.ts`, `web/src/pages/LoginPage.vue`, `web/src/main.ts` (as needed), unit tests under `web/tests/`
  - Depends on: Task 4

- [x] Task 6: Negative and positive automated security tests
  - Go tests (table-driven):
    - Origin parse/validation / production FATAL cases (helper-level).
    - CORS: allowlisted Origin gets `ACAO` + credentials; foreign Origin preflight 403 and no reflecting `ACAO`.
    - CSRF: POST without token → 403; mismatch → 403; valid cookie+header → reaches handler (use logout or clear with test session).
    - WS: `CheckOrigin` false for evil Origin and missing Origin; true for allowlisted Vite origin.
  - Keep tests free of wildcard expectations.
  - LOGGING REQUIREMENTS: assert HTTP behavior, not log text
  - Files: `internal/security/*_test.go`, `cmd/server/cors_csrf_test.go` or `internal/security/middleware_test.go`, `internal/ws/origin_test.go`
  - Depends on: Tasks 2–5

### Phase 3: Docker/Nginx, defaults, documentation

- [x] Task 7: Docker Compose + Nginx perimeter defaults
  - Production `docker-compose.yml`:
    - Set `ALLOWED_ORIGINS` to the browser-facing origin used in docs (e.g. `http://localhost` for local compose demos, with a comment that real deploys must use `https://…`).
    - Stop publishing `8080:8080` on backend (internal network only) **or** bind to `127.0.0.1:8080` if local debugging must remain — prefer internal-only + document `docker compose exec`/network for health checks; keep Nginx→backend health via `depends_on`.
    - Frontend remains the public entrypoint.
  - Dev compose: set `ALLOWED_ORIGINS` explicitly to Vite origins **or** rely on code defaults; if the “frontend” service is Nginx on 3000 and developers use Vite on host, document host `make run` + Vite as primary DX.
  - `docker/nginx.conf`:
    - Forward `Origin` (default proxy behavior; do not overwrite with empty).
    - Keep `/api/` and `/ws` proxy; consider `proxy_set_header Origin $http_origin;`.
    - Add brief comment block on TLS termination (443 example as comments or `nginx-tls.conf.example`).
    - Do not add broad CSP `*` host exceptions; optional minor CSP tighten only if it does not break the Vue app (no large CSP rewrite required).
  - LOGGING REQUIREMENTS: N/A; verify container boot shows `origin_policy` INFO
  - Files: `docker-compose.yml`, `docker-compose.dev.yml`, `docker/nginx.conf`, optional `docker/nginx-tls.conf.example`
  - Depends on: Task 1

- [x] Task 8: Documentation updates
  - README: environment table for `ALLOWED_ORIGINS`, CSRF cookie/header, Vite default allowlist, production requirement, HTTPS/proxy assumptions, “no wildcards”, troubleshooting (CORS errors, WS rejected, csrf_rejected).
  - Cross-link Prompt 02 Secure cookie + HTTPS note; state that perimeter allowlist must match the browser origin.
  - Do not describe the recognizer as AI. (Follow-up `/aif-fix`: stripped leftover AI/confidence marketing from README overview + Recognition System.)
  - LOGGING REQUIREMENTS: document `[cors]`, `[csrf]`, `[ws.CheckOrigin]`, `[main] origin_policy` operator signals
  - Files: `README.md`
  - Depends on: Tasks 1–7

- [x] Task 9: Verification against acceptance criteria
  - Run Go tests + `cd web && npm test`.
  - Manual smoke:
    1. Vite: login, draw (WS), clear, recognize, logout.
    2. `curl` from disallowed Origin preflight → 403 / no ACAO reflect.
    3. POST mutation without CSRF → 403 `csrf_rejected`.
    4. `APP_ENV=production` without `ALLOWED_ORIGINS` → process exits.
    5. `ALLOWED_ORIGINS=*` → process exits.
    6. Compose config: backend not publicly required; `ALLOWED_ORIGINS` present.
  - Files: none required
  - Depends on: Tasks 6–8

## Out of Scope
- Password KDF / session rotation / `COOKIE_KEY` semantics (Prompt 02 — already planned/implemented)
- Rate limiting, WAF, bot management
- Full ACME/certificate automation
- CSP redesign or XSS hardening beyond not weakening Nginx headers
- Recognition input bounds / log redaction of strokes (Prompt 04)
- Changing SameSite to `Strict` (would break common flows; not required if CSRF+allowlist land)
- Bearer tokens / moving off cookie sessions
- Broad wildcard or regex origin matching

## Risks & Notes
- **Vite vs Docker frontend:** Primary local DX is Vite `:5173` + Go `:8080` with default allowlist. Docker “frontend” on `:3000` is a different browser origin — if used, set `ALLOWED_ORIGINS=http://localhost:3000` (and host variants) explicitly.
- **Secure cookies + HTTP compose:** Local `APP_ENV=production` over plain `http://localhost` will drop cookies in browsers; document demo vs real HTTPS.
- **CSRF cookie readable by JS:** Required for double-submit; XSS can steal it — do not treat CSRF as XSS mitigation.
- **Recognize POST:** Included in CSRF even though it is read-mostly against stored strokes; prevents cross-site recognition abuse and keeps one rule for all POSTs.
- **Parallel plans:** Prompt 04+ may add routes; any new `POST /api/*` must inherit CSRF middleware automatically via prefix rule.
- **No wildcards:** Implementers must not “temporarily” set `CheckOrigin: true` or `ACAO=*` for debugging; use explicit localhost entries instead.
