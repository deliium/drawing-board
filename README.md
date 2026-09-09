![drawing-board](http://counter.seku.su/cmoe?name=drawing-board&theme=rule34-big)

# Japanese Handwriting Practice (Vue + Go)

A **personal** Japanese handwriting training app: practice on a private canvas, persist your strokes, and get recognition feedback. Strokes are never shared with other users.

- **Vue 3 (TypeScript, Vite)** practice canvas with authenticated WebSocket persist/echo
- **Go backend** with Gorilla mux, WebSocket, SQLite persistence
- **User authentication** with session-based login/register/logout
- **Drawing tools**: Pencil and Eraser with hit-testing
- **Undo functionality**: Ctrl+Z to undo last stroke
- **Handwriting recognition**: deterministic target comparison for five hiragana (`あいうえお`) via guided practice UI + attempt APIs, plus free-board heuristic ranking with match scores (not a trained AI model)
- **Guided practice journey**: `/#/practice` lesson hub and per-character stages (intro → stroke order → trace → free-write → assess → corrections)
- **Attempt history, mastery & review**: `/#/practice/history` for personal assessed attempts; hub shows explainable mastery labels, a schedule-aware next suggestion, and quiet due hints (Leitner-style personal review — not SM-2/FSRS or streak gamification)
- **Responsive bilingual UI**: mobile-first practice-notebook layout, EN/JA locale preference (`localStorage`), shared fluid canvas sizing, and keyboard/SR-oriented result summaries
- **Private stroke persistence**: drawings are scoped per user and restored only for that account

## Features

### Drawing Tools
- **Pencil**: Draw with customizable color and width
- **Eraser**: Remove individual strokes by clicking on them
- **Undo**: Press `Ctrl+Z` (or `Cmd+Z` on Mac) to undo the last stroke
- **Clear**: Remove all your drawings from the canvas and database

### Handwriting Recognition
- **Target comparison (MVP)**: Deterministic scoring against five hiragana templates (`hiragana5`: あ, い, う, え, お) via `POST /api/attempts/…/assess`
- **Free-board heuristic**: Pattern-based ranking on the board Recognize button (playground only; not the practice assessment path)
- **Match scores**: Ranking / criterion scores (`scoreKind: "match"`), not calibrated confidence
- **Non-goals**: No unrestricted kanji OCR; no ML/ONNX upgrade path in this release

### User Management
- **Registration**: Create new accounts with email and password
- **Login/Logout**: Secure session-based authentication
- **Private practice**: Each user's strokes stay private (REST + WebSocket are per-user)
- **Auto-restore**: Your practice strokes load automatically when you log in

## Requirements

### For Local Development
- **Go 1.22+**
- **Node 18+**
- **Modern web browser** with Canvas and WebSocket support

### For Docker Deployment
- **Docker 20.10+**
- **Docker Compose 2.0+**

## Quick Start

### Option 1: Docker (Recommended)

#### Production Deployment
```bash
# Clone the repository
git clone <repository-url>
cd drawing-board

# Build and run with Docker Compose
make docker-build
make docker-run

# Or use docker-compose directly
docker-compose up -d
```

The application will be available at:
- **Frontend (public entrypoint)**: http://localhost
- Backend listens on the internal Docker network only (`:8080`); use Nginx `/api` and `/ws` proxies

#### Development with Docker
```bash
# Run in development mode with hot reload
make docker-run-dev

# Or use docker-compose directly
docker-compose -f docker-compose.dev.yml up -d
```

#### Docker Management Commands
```bash
# View logs
make docker-logs
make docker-logs-backend
make docker-logs-frontend

# Stop containers
make docker-stop

# Clean up (removes containers, volumes, and images)
make docker-clean

# Access container shells
make docker-shell-backend
make docker-shell-frontend
```

### Option 2: Local Development

#### 1. Clone and Setup
```bash
git clone <repository-url>
cd drawing-board
go mod tidy
```

#### 2. Install Frontend Dependencies
```bash
cd web
npm install
cd ..
```

#### 3. Run Development Servers

**Terminal 1 - Backend:**
```bash
# Target-compare recognizer (hiragana5) + heuristic free-board ranking
go run ./cmd/server
```

**Terminal 2 - Frontend:**
```bash
cd web
npm run dev
```

### 4. Open the Application
- **Frontend**: http://localhost:5173
- **Backend API**: http://localhost:8080

## Usage Guide

### Getting Started
1. **Open the app** — unauthenticated visits land on the public auth page (`/#/login`, or `/#/register` to create an account).
2. **Create account or sign in** — email and password (8–72 bytes). A session cookie (`sid`) is set; the client sends `credentials: 'include'`.
3. **Practice** — after auth you can open **Practice hiragana** (`/#/practice`) for the guided single-character journey, or stay on the free board for scratchpad drawing + heuristic Recognize.
4. **Save (free board)** — strokes are queued and persisted over WebSocket with per-operation `opId` acknowledgements and monotonic `boardRev` / `baseRev` ordering (header shows Connecting / Saving / Saved / Offline / Sync error).
5. **Logout** — use Logout on the board to clear the session and return to the auth page.

### Guided practice journey
Routes: `/#/practice` (hub), `/#/practice/history` (paginated assessed attempts), and `/#/practice/:characterId` (e.g. `hira:%E3%81%82`). Hub shows mastery labels, due/ready review hints, a suggested-next banner (including calm caught-up + optional next-due text), history link, and a confirm dialog to clear personal practice data. Stages: loading → intro → animate → trace → freewrite → submitting → result → complete (plus error/empty). Attempt ink is local-only until Submit; refresh resume uses `sessionStorage` (`practice:v1:{characterId}`) plus attempt status. Leaving mid-draft best-effort `abandon`s. Overlay shows Match score and ≤2 corrections — not candidates as primary UI.

Registration and login are **not** on the board header; they live only on the public auth routes.

### Drawing Tools
- **Color Picker**: Choose any color for your pencil
- **Width Slider**: Adjust line thickness from 1-20 pixels
- **Pencil Tool**: Default drawing tool
- **Eraser Tool**: Click on any stroke to remove it
- **Undo Button**: Click to undo the last stroke (or use Ctrl+Z)
- **Clear Button**: Remove all your drawings

### Handwriting Recognition
1. **Guided practice** — open `/#/practice`, pick a character, follow stroke-order → trace → free-write, then **Submit**. The UI shows a comparison overlay, **Match** score (`scoreKind=match`), and at most two corrections — not confidence / AI language, and not the `candidates` list as primary coaching.
2. **Free board** — draw on `/#/` and click **Recognize** for heuristic match-score candidates (playground only).
3. Scores are **match scores**, not confidence.

### Keyboard Shortcuts
- **Ctrl+Z** (Windows/Linux) or **Cmd+Z** (Mac): Undo last stroke
- **Escape**: Cancel an in-progress pencil stroke (does not enqueue a create); same abort as `pointercancel`

### Practice canvas coordinates
Stroke points are stored in **CSS logical pixels** relative to the canvas layout box. The backing store uses `devicePixelRatio` (`backing = round(css × dpr)`) with a matching 2d transform so drawing stays sharp on high-DPI displays without rewriting historical coordinates. Practice and free-board canvases share CSS token `--canvas-size` (`min(92vw, min(70dvh, 420px))`); submit/recognize `width`/`height` are the **live logical CSS** size from the layout box (never a stale 300 constant). One-point taps are persisted and rendered as dots and can be erased.

### Locale (EN/JA) and romanization
The SPA keeps a client-only preference in `localStorage` key `locale:v1` (`en` | `ja`; default EN, or JA when `navigator.language` starts with `ja`). Toggle lives in the app shell. `document.documentElement.lang` tracks the preference. Japanese glyphs/examples keep `lang="ja"`; assessment API `feedback[].message` stays English for persistence — the UI displays corrections via a client map on `feedback[].code` (+ glyph context) with fallback to the API message. Curriculum pedagogy includes `descriptionJa` / `guidanceJa` / `meaningJa` / `titleJa` from the reviewed pack.

Romanization visibility is a separate client preference (`localStorage` key `romanizationVisible:v1`, default `true`). The shell toggle hides Hepburn romanization on the practice hub and character intro (glyph, IPA/`jaHint`, audio, and meanings stay). When hidden, romanization is omitted from the accessibility tree as well.

### Pronunciation audio
Each hiragana5 character has a reviewed `pronunciation.audioRef` pointing at a short MP3 under `/audio/hiragana5/` (pack canonical: `content/hiragana5/v1/audio/`; SPA mirror: `web/public/audio/hiragana5/`; sync with `make sync-audio`). Intro play is on-demand only (no autoplay). Missing/`null` refs omit the control; load/decode errors soft-fail with a polite live-region message and leave text pedagogy intact. Nginx caches audio with a short `max-age` (not `immutable` — filenames are stable). Licensing and provenance: `content/hiragana5/LICENSES.md`. Do **not** market clips as “AI pronunciation” or “perfect native TTS.”

Pack JSON may include optional `kanjiExtensions` (readings/meanings/radicals/exampleSentences) as a future schema hook; hiragana5 leaves them empty and the validator rejects non-empty payloads on `hira:*` ids.

### Accessibility & responsive smoke
Automated: `web/tests/integration/accessibility-parity.spec.ts` runs axe-core on Login, Practice hub, and result summary (zero serious/critical). Manual checklist (run on ≥1 real phone):

| Check | Pass bar |
|-------|----------|
| Keyboard | Tab through shell → practice → submit; visible focus; Escape cancels in-progress stroke |
| Zoom 200% | Auth + practice actions usable; no clipped primary buttons |
| Screen reader | Stage banner and textual result summary announced; board tool radio state spoken |
| Real device | Draw/trace works; login virtual keyboard does not permanently hide errors; canvas stays square |
| Locale | Toggle EN↔JA; chrome + corrections + hub status switch; glyphs remain `lang=ja` |

## Advanced Setup

### Environment Variables
```bash
# Database configuration
DB_PATH=file:data.db?_fk=1

# Server configuration  
ADDR=:8080

# Session cookie signing key (≥32 bytes). Changing this invalidates existing cookies.
COOKIE_KEY=your-secure-random-cookie-key-here

# Production-secure cookies (Secure flag) + fail-fast COOKIE_KEY validation:
# APP_ENV=production
# or COOKIE_SECURE=true|1|yes

# Exact browser origins allowed for CORS and WebSocket upgrades (comma-separated).
# No wildcards (*). Production (APP_ENV=production) requires this variable.
# Development default when unset: http://localhost:5173,http://127.0.0.1:5173
# ALLOWED_ORIGINS=https://learn.example.com

# Log level for auth/session/perimeter lines (DEBUG|INFO|WARN|ERROR). Default shows DEBUG.
# LOG_LEVEL=info

# Handwriting diagnostic dumps (coordinates / ASCII canvas). Off by default.
# Ignored when APP_ENV=production. Startup logs INFO [recognize] recognize_debug=true|false.
# RECOGNIZE_DEBUG=1

# Deprecated: ONNX_MODEL is ignored if set (WARN [main] once). No model path is loaded.
```

Local `make run` without `APP_ENV=production` / `COOKIE_SECURE` keeps `Secure=false` so HTTP/Vite works. Startup logs `INFO [main] cookie_secure=true|false`, `INFO [main] origin_policy mode=… count=… origins=…`, and `INFO [main] recognizer=target_compare set=hiragana5`. In non-production mode a weak/default `COOKIE_KEY` only warns; in production-secure mode the process exits if `COOKIE_KEY` is missing, shorter than 32 bytes, or equal to a documented sentinel (`change-me-please-32-bytes-min` / `please-change-this-32-bytes-min`). With `APP_ENV=production`, missing/empty/`*`/`invalid` `ALLOWED_ORIGINS` also exits before listen.

If TLS terminates at Nginx in front of Go, the public site must still be HTTPS for browsers to send `Secure` cookies, and `ALLOWED_ORIGINS` must match the browser-facing origin exactly (e.g. `https://learn.example.com`). See `docker/nginx-tls.conf.example`. Forwarded headers may be set for logs; they are **not** used for origin allowlisting or auth.

### Production Build
```bash
# Build frontend
make build-web

# Run production server
ADDR=:8080 STATIC_DIR=web/dist DB_PATH=file:data.db?_fk=1 COOKIE_KEY=your-secure-key make run
```

## API Reference

### Authentication Endpoints
Cookie session name: `sid` (`HttpOnly`, `SameSite=Lax`, `Path=/`; `Secure` when production-secure mode is on). Register/login **rotate** the session (`Sessions.New` after invalidating any prior `sid`). Auth handlers log with prefixes `[auth.Register]`, `[auth.Login]`, `[auth.Logout]`, `[auth.Me]`, `[auth.hash]`, `[auth.startSession]` (level filtered via `LOG_LEVEL`). Operator signals: `INFO [main] cookie_secure=…`, `INFO [main] origin_policy …`, `[cors]`, `[csrf]`, `[ws.CheckOrigin]`.

**CSRF (double-submit):** mutating `/api/*` methods (`POST`, `PUT`, `PATCH`, `DELETE`) require cookie `csrf` (readable by JS, `SameSite=Lax`, `Secure` in production-secure mode) plus matching header `X-CSRF-Token`. `GET /api/me` and `GET /api/csrf` ensure the cookie (including anonymous `401` on `/api/me`). The Vue client sends the header automatically after bootstrap. Failure: `403` `{ "error": "csrf_rejected", "message": "…" }`. WebSocket upgrades are not CSRF-token gated; they require a valid session cookie and an allowlisted `Origin`.

**CORS / WebSocket origins:** credentialed CORS echoes `Access-Control-Allow-Origin` only for exact allowlisted origins (never `*`). Disallowed CORS preflight returns `403`. WebSocket `CheckOrigin` uses the same allowlist and rejects missing Origin.

- `POST /api/register` — Create account `{ email, password }` (password min 8, max 72 UTF-8 bytes). Success `200` `{ id, email }` + rotated session cookie. New passwords are stored with **bcrypt** (cost 12).
- `POST /api/login` — Sign in `{ email, password }`. Success `200` `{ id, email }` + rotated session cookie. Legacy unsalted SHA-256 hashes are verified with constant-time compare and transparently upgraded to bcrypt on successful login (login still succeeds if the upgrade write fails; it retries next login).
- `POST /api/logout` — Clear session values and expire the cookie (same Path/HttpOnly/SameSite/Secure). Success `200` `{ "ok": "true" }`. CookieStore sessions are client-side signed blobs: logout cannot revoke a stolen cookie copy until expiry or `COOKIE_KEY` rotation.
- `GET /api/me` — Current user or `401` `{ "error": "unauthorized" }` (always ensures CSRF cookie).
- `GET /api/csrf` — Ensures CSRF cookie and returns `{ "csrf": "<token>" }`.

Auth error JSON shape: `{ "error": "<code>", "message": "<optional>" }`.

| HTTP | Code | Meaning |
|------|------|---------|
| 400 | `bad_json` | Body not JSON |
| 400 | `missing_fields` | Empty email or password |
| 400 | `invalid_email` | Email fails basic format check |
| 400 | `password_too_short` | Password shorter than 8 characters |
| 400 | `password_too_long` | Password longer than 72 bytes (bcrypt input limit) |
| 400 | `registration_failed` | Unable to create account (includes duplicate email; does **not** return `email exists`) |
| 401 | `invalid_credentials` | Login failed (unknown email or wrong password — same response) |
| 401 | `unauthorized` | `/api/me` without a valid session |
| 403 | `csrf_rejected` | Missing/mismatched CSRF cookie + `X-CSRF-Token` on mutating `/api/*` |

Passwords are hashed with bcrypt. Existing accounts that still have legacy SHA-256 hashes can log in and are upgraded automatically. Logs never include passwords, raw cookies, CSRF token values, or full hashes (`hash_kind=bcrypt|legacy` and `userID=` only).

#### Perimeter troubleshooting
| Symptom | Likely cause |
|---------|----------------|
| Browser CORS error / no `Access-Control-Allow-Origin` | Request `Origin` not in `ALLOWED_ORIGINS` (or Vite defaults) |
| WebSocket fails to connect | Disallowed/missing Origin; check `WARN [ws.CheckOrigin]` |
| `403 csrf_rejected` | Refresh so `GET /api/me` sets `csrf`, then retry; ensure `apiFetch` sends `X-CSRF-Token` |
| Process exits on start in production | Set explicit `ALLOWED_ORIGINS` without `*`; set a strong `COOKIE_KEY` |
| Secure cookies missing on `http://localhost` compose | Use HTTPS at the browser, or avoid `APP_ENV=production` for plain-HTTP demos |

### Drawing Endpoints
- `GET /api/strokes` — `{ "boardRev": <n>, "strokes": [...] }` for the authenticated user (reload source of truth)
- `POST /api/strokes/clear` — revision-gated clear via the same store helper as WS (`opId`/`baseRev` optional in body; server may generate). Returns `{ "ok": true, "boardRev": <n> }`. **Vue board uses WS `clear`** so other tabs receive a live echo; REST clear is for scripts/tests and does not fan out over the hub
- `POST /api/strokes/delete?id={id}` — thin REST delete wrapper (UI uses WS delete)

### Practice Attempt Endpoints

Canonical **single-character practice** assessment. Attempt routes do **not** use `boardRev` / WS queue gating. Draft rows persist metadata only — stroke geometry is sent once on submit and then frozen. Retry = new `POST /api/attempts` (new `clientAttemptId`). Board clear/undo does not mutate attempts. The Vue journey at `/#/practice/:characterId` draws locally (no WS for attempt ink).

State machine: `draft` → `submitted` → `assessed`, or `draft` → `abandoned`.

| Method | Path | Notes |
|--------|------|-------|
| `GET` | `/api/attempts` | Paginated personal history (default `status=assessed`; optional `abandoned`); query `lessonId`, `characterId`, `limit` (1–50, default 20), `cursor`; **no stroke points** |
| `POST` | `/api/attempts` | Create draft `{ characterId, lessonId?, clientAttemptId? }` → `201` (or `200` on idempotent `clientAttemptId` replay) |
| `GET` | `/api/attempts/{id}` | Metadata; includes `canvasWidth`/`canvasHeight`/`strokeCount` after submit (no full points) |
| `POST` | `/api/attempts/{id}/submit` | Body max **64 KiB**: `{ width, height, strokes:[{color,width,points}] }` → freezes strokes |
| `POST` | `/api/attempts/{id}/assess` | Empty body; multi-criterion assess on **attempt** strokes only; returns ≤2 `feedback` messages; idempotent if already assessed |
| `GET` | `/api/attempts/{id}/assessment` | Persisted result (`scoreKind=match`, `feedback[]`; no live `candidates`) |
| `POST` | `/api/attempts/{id}/abandon` | Only from `draft` |

History list rows include glyph, terminal status, pass/fail, match score + `scoreKind`, and ≤2 feedback items when assessed. Abandoned drafts never change mastery, progress counters, or the review schedule. Only **assessed** attempts feed mastery / next-character suggestion / Leitner box updates.

### Curriculum & Progress Read Endpoints

Authenticated GETs for the practice hub / journey (CSRF not required on GET). Pedagogy comes from the seeded pack; stroke **geometry** stays in client fixtures (`hiragana5Traces`).

| Method | Path | Notes |
|--------|------|-------|
| `GET` | `/api/lessons/{id}` | Published lesson only (`lesson:hiragana5`); characters include glyph, romanization, strokeCount, pronunciation JSON (`audioRef`), `descriptionEn`/`descriptionJa`, `guidanceEn`/`guidanceJa`, example word + `meaningEn`/`meaningJa`, lesson `title`/`titleJa` |
| `GET` | `/api/progress` | `{ items:[{ characterId, status, attemptCount, passCount, mastery, review, … }] }`; optional `?lessonId=` / `?setId=` |
| `GET` | `/api/progress/next` | Schedule-aware next suggestion for a published `lessonId` (default `lesson:hiragana5`); prefers due reviews, then introduction/learning; `{ characterId, glyph, reasonCode, masteryState, dueAt?, reviewBox? }` or `characterId: null` + `all_caught_up` (+ optional `nextDueAt` / `nextDueCharacterId`) |
| `DELETE` | `/api/practice-data` | CSRF; deletes **this user’s** practice attempts (CASCADE assessments/strokes) and `user_character_progress` rows (including review box/`due_at`); does **not** delete free-board strokes or the account |

**Mastery** on progress items is a compute-on-read practice summary from assessed attempts only (`not_started` / `learning` / `passed_once` / `steady` + `reasonCode`). It is an engineering heuristic for personal coaching chrome — not a validated proficiency score, belt, grade, or SM-2 stage. Operational `status` (`practicing` / sticky `passed`) remains for journey resume; the hub prefers mastery labels for coaching.

**Review schedule** (`review` on progress items; updated on assess): a lightweight **Leitner-style** personal queue with boxes `0–3` and fixed wall-clock intervals after each assessed pass/fail (`0` / `1 day` / `3 days` / `7 days`). Pass promotes one box; fail demotes one box (forgiving — not a reset to zero). `dueAt` is UTC; overdue means ready when you are (no penalty, no notifications, no streaks). Ratings are automatic from assess pass/fail only — no Again/Hard/Good/Easy UI. Product copy may say “simple review schedule” / “practice reminder timing”; do **not** market it as scientifically optimized SRS, Anki-grade, or SM-2/FSRS.

**Next priority:** due review → first not started → continue learning → encourage steadiness → `all_caught_up` (with optional next future due). Manual practice of a non-due character is always allowed; assess still reschedules from assess `now`.

**Retention:** practice attempt handwriting and assessments are kept until the learner clears them via `DELETE /api/practice-data` or the account is deleted. There is no timed auto-purge and no cross-user export. Free-board strokes are separate.

`clientAttemptId` (optional, ≤36, same rules as WS `opId`): same user + same character/lesson → return existing attempt; mismatched reuse → `409 conflict`. Progress counters update on submit/assess as in the learning store.

| HTTP | Code | Meaning |
|------|------|---------|
| 400 | `invalid_input` / `character_not_active` / limit codes / `body_too_large` | Validation |
| 401 | `unauthorized` | No session |
| 403 | `csrf_rejected` | Missing/mismatched CSRF on mutating methods |
| 404 | `not_found` | Unknown/foreign attempt, character, or lesson |
| 409 | `conflict` / `invalid_status` | Idempotency mismatch or wrong lifecycle state |
| 429 | `rate_limited` | Assess rate limit (same defaults as recognize) |
| 503 | `recognizer_unavailable` | Assessor not configured |

### Recognition Endpoint (free-board playground)
- `POST /api/recognize` — Params only (strokes come from the user store at `boardRev`). Max body **4 KiB**.
  - Free-board: `{ topN: 10, width: 300, height: 300, boardRev: <n> }` → `{ "boardRev": <n>, "candidates": [...], "scoreKind": "match" }`
  - Non-empty `target` is **rejected** (`400` `use_attempt_api`) — use `/api/attempts` for single-character practice
  - Mismatch: `409` `{ "error": "stale_revision", "message": "…", "boardRev": <current> }`

| HTTP | Code | Meaning |
|------|------|---------|
| 400 | `bad_json` | Body not JSON (includes missing `boardRev`) |
| 400 | `payload_too_large` | Body exceeds 4 KiB |
| 400 | `invalid_dimensions` | `width`/`height` outside 1…2048 or pixel product too large |
| 400 | `invalid_top_n` | Present `topN` outside 1…32 (omitted → default 10) |
| 400 | `use_attempt_api` | Non-empty `target` — practice assessment moved to `/api/attempts` |
| 400 | `too_many_strokes` | More than 64 stored strokes |
| 400 | `too_many_points` | Per-stroke or total point caps exceeded |
| 400 | `invalid_stroke_data` | NaN/Inf/out-of-range coords in stored strokes |
| 401 | `unauthorized` | No session |
| 409 | `stale_revision` | Request `boardRev` ≠ store revision |
| 429 | `rate_limited` | Per-user recognize rate exceeded (30/min, burst 5; **per process replica**) |
| 503 | `recognizer_unavailable` | No recognizer configured |
| 500 | `internal_error` | Store/recognizer failure (no raw error text) |

Canvas bounds: width/height **1…2048**, max pixels **2048²**. Legitimate UI (`topN: 10`, ~300px canvas, width 1–20) is unchanged. Board Recognize is enabled in the UI only when sync status is **Saved**, the WS queue is empty, and there is at least one stroke. `score` values are **match scores** (`scoreKind: "match"`), not calibrated confidence.

### WebSocket
- `WS /ws` - Authenticated **private persist + echo** channel (cookie session required)

The hub delivers messages only to connections belonging to the same `user_id` (multi-tab same account receives echoes; other users never see your strokes). This is **not** a collaborative/shared board.

Text frames are capped at **64 KiB**. Stroke ingest is rate-limited per user (**60/min**, burst 20; per process replica). Points per stroke ≤ **2048**; coords in **-512…4096**; line width **1…20**; color `#RGB` / `#RRGGBB` / `#RRGGBBAA`. Each mutating message requires a client `opId` (≤ **36** chars) and `baseRev` (strict equality with the server’s monotonic per-user `boardRev`). Creates are idempotent on active `(user_id, op_id)`; after clear/tombstone the same create `opId` returns `op_cancelled`.

**WebSocket Messages:**
```json
// Create (idempotent while active; ack + echo include boardRev)
{"type":"stroke","opId":"550e8400-e29b-41d4-a716-446655440000","baseRev":12,"stroke":{"points":[{"x":10,"y":20}],"color":"#1d4ed8","width":4,"clientId":"abc","startedAtUnixMs":1690000000000}}

// Delete by stroke id
{"type":"delete","opId":"550e8400-e29b-41d4-a716-446655440001","baseRev":12,"delete":123}

// Delete / cancel by create opId (pending or persisted)
{"type":"delete","opId":"550e8400-e29b-41d4-a716-446655440002","baseRev":12,"deleteOpId":"550e8400-e29b-41d4-a716-446655440000"}

// Clear board (tombstones prior create opIds; Vue primary path)
{"type":"clear","opId":"550e8400-e29b-41d4-a716-446655440003","baseRev":12}

// Acknowledgements
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440000","ok":true,"boardRev":13,"strokeId":456}
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440001","ok":true,"boardRev":13,"delete":123}
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440003","ok":true,"boardRev":13,"clear":true}
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440000","ok":false,"error":"stale_board","message":"board revision mismatch","boardRev":14}
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440000","ok":false,"error":"op_cancelled","message":"create cancelled","boardRev":13}
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440000","ok":false,"error":"rate_limited","message":"too many strokes"}

// Clear echo to the user's other tabs
{"type":"clear","opId":"550e8400-e29b-41d4-a716-446655440003","boardRev":13,"clear":true}

// Frame-level rejection without a bound opId (not persisted)
{"type":"error","error":"bad_json","message":"invalid JSON message"}
```

WS error / nack codes include `bad_json`, `payload_too_large`, `invalid_stroke`, `invalid_op_id`, `too_many_points`, `invalid_coordinates`, `rate_limited`, `stale_board`, `op_cancelled`, `internal_error`. Soft validation prefers an `ack` nack (when `opId` is known) or `error` frame over disconnect; oversize frames may close the connection after the read-limit error.

The Vue client keeps a **bounded in-memory queue** (32 ops), reconnects with exponential backoff, and retries until ack / nack / attempt budget. Each outbound mutate stamps `baseRev` from the latest known `boardRev`. Header status: **Connecting… / Saving… / Saved / Offline — retrying… / Sync error**. Recognize stays disabled until **Saved** with an empty queue. Undo/erase work for pending (`deleteOpId` / drop) and acknowledged strokes. Unmatched inbound stroke creates remain non-authoritative. Full page reload uses `GET /api/strokes` (`boardRev` + strokes) as source of truth and drops the session queue (no durable offline storage in this iteration). On `stale_board`, the client reloads from REST. DEV builds `console.debug` WS frames without toasts.

**Operator extras (optional Nginx):** `client_max_body_size` on `/api/recognize`, `limit_req` for multi-instance deployments. In-process limits are per replica only.
## Recognition System

Startup always wires **target comparison** for the fixed MVP set `hiragana5` (あ, い, う, え, お) and logs `INFO [main] recognizer=target_compare set=hiragana5 contentVersion=…`. There is **no** loaded ML/ONNX model path; a deprecated `ONNX_MODEL` env var is ignored with a single `WARN [main]`.

### Target comparison (`hiragana5`)

Assessment paths come from the reviewed content pack at `content/hiragana5/v1/` (same source as seed). Scores are **match scores**, not calibrated confidence. Practice assessment is via **attempt APIs** (`ListStrokes` on the attempt → `Assessor.Assess` → `SaveResult`); free-board `/api/recognize` does not accept `target`.

**Normalization (shared for learner + templates):** drop empty-point strokes; whole-ink AABB → origin; scale by `1/max(w,h)` (aspect preserved). Degenerate bbox axes use `ε_bounds=1e-9` (`EpsilonBounds`). Classify **short/dot** strokes when path length ≤ `L_short=0.04` (unit space) or ≤1 distinct point — shorts skip direction scoring and compare by centroid. Normal strokes resample to `N_resample=16` (`ResampleCount`) for shape and direction.

**Engineering criteria (named constants in `internal/recognize/tolerances.go`):** `T_pass=0.70`, `ε_bounds=1e-9`, `L_short=0.04`, `T_place=0.15`, `T_prop=0.35`, `N_resample=16`, soft criterion threshold `0.75`. These are match-score thresholds — not “% accurate handwriting”.

**Criteria (bounded match components, weights sum to 1):** stroke count `0.15`, stroke order `0.15`, start/end direction `0.15`, relative placement `0.15`, proportions `0.10`, overall shape `0.30`. Overall score = weighted sum in `[0,1]`.

**Pass rule (engineering):** `pass` when overall ≥ `T_pass=0.70`, stroke count matches for templates with ≤3 strokes (hard-fail on mismatch), and the target is the top-ranked glyph in the five-character set. Within-set `candidates` on live assess are secondary diagnostics — practice UI should prefer score + `feedback`.

**Learner feedback:** at most **two** items `{rank, code, message}` with non-empty English messages (correction catalog in `internal/recognize`). Diagnostics/criterion floats stay on the assessor (`diagnostics`) / DEBUG logs — not as user copy. Assessed attempts are immutable; **retry = new attempt** focusing on the listed corrections. Correctness claims are limited to **fixture-tested** behaviors.

Tolerances and copy review: `content/hiragana5/v1/assessment_review.json`. Fixture eval: `go test ./internal/recognize -run 'Eval|Fixture' -v` (gold must pass; incorrect suites must not pass and must emit the expected correction code in the top-2 feedback).

### Free-board heuristic ranking
- Used by board **Recognize** (no `target`)
- Pattern-based candidate ranking with `scoreKind: "match"` — useful for playground feedback, **not** calibrated confidence
- Not unrestricted kanji OCR; do not treat heuristic suggestions as ground truth

### Logging prefixes
| Prefix | Meaning |
|--------|---------|
| `INFO [main] recognizer=…` | Honest recognizer identity + set id at startup |
| `[recognize.Recognize]` / `[recognize.Assess]` | DEBUG entry/result; Assess logs criterion breakdown + feedback codes (no coordinates unless `RECOGNIZE_DEBUG`) |
| `[recognize.normalize]` | DEBUG stroke counts / short vs normal / degenerate axis (no coordinates) |
| `[recognize.corrections]` | DEBUG selected feedback codes |
| `[recognize.templates]` | Template load (`loaded count=5 version=…`) |
| `[httpapi.Recognize]` | Free-board heuristic result codes — no stroke dumps |
| `[httpapi.Attempt.*]` | create/submit/assess/abandon (`attemptID`, status, pass/scoreKind/feedbackCodes; no coordinates) |

## Learning schema & migrations

SQLite schema changes are **versioned** and applied fail-closed on `db.Open`. Applied versions live in `schema_migrations`. Startup logs `INFO [main] schema_version=N learn_seed=hiragana5 contentVersion=…`.

| Version | Name | Contents |
|---------|------|----------|
| 1 | `baseline_board` | Free-board tables (`users`, `strokes`, `stroke_points`, `boardRev` / tombstones / `board_ops`) |
| 2 | `learning_domain` | Curriculum + attempts (`characters`, `lessons`, `practice_attempts`, `attempt_strokes`, assessments, progress) |
| 3 | `curriculum_pedagogy` | Pedagogy columns on `characters` (description, pronunciation JSON, examples, `content_version`, `trace_ref`) |
| 4 | `curriculum_ja_pedagogy` | `description_ja`, `example_meaning_ja`, lesson `title_ja` |
| 5 | `review_schedule` | Leitner-style `review_box` / `due_at` / `last_reviewed_at` on progress |
| 6 | `curriculum_guidance` | `guidance_en` / `guidance_ja` on `characters` |

**Free-board vs attempts:** Board strokes (`strokes` / `stroke_points`) are a scratchpad with `boardRev` / `opId`. Practice attempts use separate `attempt_strokes` tables — board clear/undo/erase does **not** delete attempt history. Match scores stored on assessments remain `score_kind=match` (not calibrated confidence).

### Hiragana5 content pack

Trusted curriculum lives under `content/hiragana5/v1/` (あ行 vowels). The vowel row is the first gojuon column — a coherent first lesson before denser consonant rows — and keeps stable ids `hira:あ`…`hira:お` / `set_id=hiragana5`. Seed and recognize both load this pack via `internal/curriculum` — draft/WIP files under `content/hiragana5/drafts/` are never imported.

| File | Role |
|------|------|
| `manifest.json` | `contentVersion` (e.g. `hiragana5-content-v2`), `schemaVersion` (≥2), `reviewStatus=published`, `contentHash`, lesson title |
| `characters.json` | Glyph, romanization, pronunciation (`audioRef`), description, concise guidance, example word; optional empty `kanjiExtensions` |
| `audio/*.mp3` | Reviewed mora clips; `audioRef` = `/audio/hiragana5/<romaji>.mp3` |
| `strokes.json` | Canonical assessment polylines (normalized 0–1) |
| `traces.json` | UI trace templates (v1 matches strokes) |
| `review.json` | Pedagogy pack human-review checklist (includes audio + guidance) |
| `assessment_review.json` | Scoring tolerances + correction-copy review (Prompt 12) |

Validate with `make validate-content` (syncs/mirrors audio, runs `go test ./internal/curriculum` and `go run ./cmd/contentvalidate`). Authoring/review checklist: `content/hiragana5/README.md`. Licensing notes: `content/hiragana5/LICENSES.md`.

**Seed:** Every Open upserts five characters + published lesson `lesson:hiragana5` from the pack (deterministic; fails closed if pack is not `published` or hash mismatches).

### Operator recovery (rollback)

Migrations are **forward-only** in production (no automatic `Down` on startup).

1. Before upgrading the binary: `cp data.db data.db.bak`
2. Start new binary — pending versions apply on Open
3. On migrate failure: restore `cp data.db.bak data.db`, pin the previous binary, investigate logs (`ERROR [db.migrate] failed version=…`)
4. Forward fixes: add a new numbered migration; never edit already-applied migration bodies

### Logging prefixes (learning / migrate / curriculum)
| Prefix | Meaning |
|--------|---------|
| `[db.migrate]` | Apply / up-to-date / failed version |
| `[db.seed]` | Idempotent hiragana5 seed (`contentVersion`, `contentHash`) |
| `[curriculum.load]` / `[curriculum.validate]` | Pack load + invariant checks (audioRef path/bytes; no coordinates; no audio payloads) |
| `INFO [main] schema_version=` | Final version + `learn_seed=hiragana5` + `contentVersion=` |
| `[learn.*]` | Attempt/assessment repo DEBUG/INFO (ids/status/counts — no coordinates) |
| `[httpapi.Lesson.Get]` / `[httpapi.Progress.List]` / `[httpapi.Progress.Next]` | Curriculum/progress/next reads (counts/ids — no stroke geometry) |
| `[httpapi.Attempts.List]` / `[httpapi.PracticeData.Clear]` | Attempt history list filters/counts; practice-data clear deleted counts |
| `[learn.mastery]` / `[learn.review]` / `[learn.AttemptRepo.List]` | Mastery derive DEBUG; review box/dueAt DEBUG; history list DEBUG |
| `[learn.review.Suggest]` | Schedule-aware next DEBUG (reasonCode, dueCount) |
| `[practiceJourney]` / `[strokeOrder]` / `[compareOverlay]` | Vue DEV-only journey / animation / overlay debug (no coordinates) |

## Troubleshooting

### Common Issues
1. **"Address already in use"**: Stop existing server processes with `pkill -f "go run"`
2. **Recognition not working**: Check `[httpapi.Recognize]` logs; enable `RECOGNIZE_DEBUG=1` locally only if you need feature dumps
3. **WebSocket connection failed**: Ensure backend is running on port 8080 and you are signed in
4. **Frontend not loading**: Check if `npm run dev` is running on port 5173
5. **Seeing another user's strokes**: Should not happen; verify you are on a build with per-user `sendToUser` (not global broadcast) and check logs below
6. **Blank canvas after resize/zoom but strokes still listed**: Backing-store resize clears the bitmap; the practice canvas should redraw from `strokes` immediately. In DEV builds check `[practiceCanvas] resize … wiped=true` then a redraw; if the canvas stays blank, hard-reload so `GET /api/strokes` repaints.

### Debug Mode

Verbose server log prefixes for privacy and persistence:

| Prefix | Meaning |
|--------|---------|
| `[ws.Handle]` | Connect/disconnect (`userID`, remote), inbound stroke/delete/clear + `baseRev`, save/delete/clear INFO with `boardRev`, upgrade/read errors, reject codes |
| `[ws.sendToUser]` | Delivery to one user's connections; `recipients=` should stay within that account (e.g. 1–N tabs) |
| `[httpapi.ListStrokes]` / `[httpapi.ClearStrokes]` / `[httpapi.DeleteStroke]` | Authenticated REST entry (`userID`), clear count, delete success, store errors |
| `[httpapi.Recognize]` | Recognize result (`ok`/`reject` + code), stroke/point/candidate counts — no coordinates |
| `[httpapi.Attempt.*]` | Attempt lifecycle (`op` create/submit/assess/abandon; no coordinates) |
| `[recognize]` / `[recognize.Assess]` | Startup `recognize_debug=…`; DEBUG assess/recognize; gated dumps when `RECOGNIZE_DEBUG=1` (non-production) |
| `INFO [main] recognizer=` | `target_compare` + `set=hiragana5` + `contentVersion=` at process start |
| `INFO [main] schema_version=` | Applied migration version + hiragana5 seed + contentVersion |
| `[db.migrate]` / `[db.seed]` / `[curriculum.*]` | Schema apply / curriculum seed / pack load |
| `[practiceCanvas]` | DEV-only client debug: attach/detach, resize css/dpr/backing, stroke start/commit/cancel (point counts only) |

Example privacy check while two users practice: user A's stroke logs should show `sendToUser` recipient counts only for A's open tabs, never B's.

Frontend (Vite dev): browser console uses `[wsClient]` and `[BoardPage.ws]` for connect/send/ignore reasons.

Default recognition logs are structured counts only (`[httpapi.Recognize] userID=… result=ok|reject mode=… code=… strokes=… candidates=…`) — **never** stroke coordinates or ASCII canvases. WS rejects log `[ws.Handle] WARN reject type=stroke userID=… code=…`.

For local handwriting diagnostics (features, ASCII preview, sample coords), set `RECOGNIZE_DEBUG=1` in non-production. Production ignores the flag and logs `INFO [recognize] recognize_debug=false` (with a WARN if the flag was set).

## Development

### Speckit Workflow (Cursor)

The repository is initialized for **Speckit** in Cursor. Use slash commands in
Cursor chat:

```text
/speckit.constitution
/speckit.specify <feature description>
/speckit.clarify
/speckit.plan
/speckit.tasks
/speckit.implement
```

Generated project artifacts live in `.specify/`, and Cursor commands are in
`.cursor/commands/`.

### Project Structure
```
drawing-board/
├── cmd/server/          # Go backend server
├── cmd/contentvalidate/ # Curriculum pack validator CLI
├── content/hiragana5/   # Reviewed curriculum packs (vN) + drafts/
├── internal/            # Go internal packages
│   ├── auth/           # Authentication logic
│   ├── curriculum/     # Pack load + validate
│   ├── db/             # SQLite store, versioned migrations, learning repos
│   │   └── migrations/ # Numbered schema Up steps
│   ├── learn/          # Learning-domain types + repository interfaces
│   ├── httpapi/        # HTTP API handlers
│   ├── limits/         # Shared stroke/recognize input bounds
│   ├── metrics/        # Process-local reject/ok counters
│   ├── recognize/      # hiragana5 target comparison (paths from content pack)
│   └── ws/             # WebSocket handling
├── web/                # Vue 3 frontend
│   ├── src/           # TypeScript / Vue source
│   │   └── curriculum/ # Trace rendering fixtures
│   └── public/        # Static assets
└── .ai-factory/        # Plans, patches, AI context
```

### Make Commands

#### Development Commands
```bash
make backend          # Run Go backend
make frontend         # Run Vue frontend
make build-web        # Build frontend for production
make run              # Run production server
make validate-content # Validate hiragana5 content pack
make test             # Go unit/integration (./test.sh all; no race)
make test-verbose     # Go tests with -v
```

#### Quality gates (local mirrors of CI)
```bash
make check            # PR-like: check-go + check-web + validate-content
make check-go         # gofmt + vet + golangci-lint (if installed) + build + Go tests
make test-race        # CGO race on ws/db/httpapi (override TEST_RACE_PKGS)
make check-web        # typecheck + Vitest + production build
make test-web         # Vitest only
make test-e2e         # Playwright learner journeys (needs Chromium once)
make coverage         # Go coverage HTML under coverage/ (signal only)
make coverage-web     # Vitest coverage under web/coverage/
make security-check   # govulncheck + npm audit --omit=dev
CHECK_E2E=1 make check  # include Playwright in the local PR gate
./test.sh recognize-fixtures  # hiragana5 Fixture/Eval + docguard
./test.sh race                # same packages as make test-race
```

#### Docker Commands
```bash
# Build containers
make docker-build           # Build all containers
make docker-build-backend   # Build backend container only
make docker-build-frontend  # Build frontend container only

# Run containers
make docker-run            # Run production containers
make docker-run-dev        # Run development containers

# Manage containers
make docker-stop           # Stop production containers
make docker-stop-dev       # Stop development containers
make docker-logs           # View all container logs
make docker-logs-backend   # View backend logs only
make docker-logs-frontend  # View frontend logs only
make docker-clean          # Remove containers, volumes, and images

# Debug containers
make docker-shell-backend  # Access backend container shell
make docker-shell-frontend # Access frontend container shell
```

## Quality gates / CI

Engineering confidence uses a **balanced test pyramid** (Go unit/integration → Vitest → small Playwright), not a vanity global coverage percentage. Coverage HTML/LCOV is uploaded as a **signal** artifact only.

| Tier | What it covers |
|------|----------------|
| Go | packages, SQLite migrations, HTTP/WS dial contracts, hiragana5 Fixture/Eval, `-race` on WS/db/httpapi |
| Vitest | components, composables, API/WS client contracts, axe a11y |
| Playwright | 2 Chromium learner journeys (register → assess → history/hub); non-required on PRs until stable |

### GitHub Actions (`.github/workflows/ci.yml`)

Required check **names** to enable under branch protection on `main`:

1. `Go quality`
2. `Go race`
3. `Web quality`
4. `Content validate`
5. `Security light`

`Playwright` runs on PRs with `continue-on-error: true` — **promote to required** when nightly stays green. Nightly also runs full `./... -race` and CodeQL (`.github/workflows/nightly.yml`). Dependabot covers Go modules, `web` npm, and Actions.

### Security triage

- `govulncheck ./...` — fix or document accepted Go advisories in the PR.
- `npm audit --omit=dev --audit-level=high` — production deps only; low/moderate noise is not a merge blocker. Prefer upgrades over force-audit silencing; if a false positive blocks CI, record the advisory ID and rationale in the PR.
- Gitleaks on PRs; never enable `RECOGNIZE_DEBUG` in CI (stroke coordinates must stay out of logs).

### Operator checklist (GitHub)

1. Push workflows to `main`.
2. Settings → Branches → protect `main` → require the five checks above.
3. Optionally require status checks to pass before merging; leave Playwright off until promoted.
## Docker Configuration

### Production Setup
The production Docker setup includes:
- **Multi-stage builds** for optimized image sizes
- **Nginx reverse proxy** for the frontend
- **Health checks** for both services
- **Persistent volumes** for database storage
- **Security headers** and optimizations

### Development Setup
The development setup provides:
- **Hot reload** capabilities
- **Volume mounting** for live code changes
- **Separate networks** for isolation
- **Debug-friendly** configuration

### Environment Variables
```bash
# Backend environment variables
ADDR=:8080                      # Listen address (preferred over unused PORT)
DB_PATH=/data/drawing-board.db  # Database file path
COOKIE_KEY=replace-me-with-a-long-random-cookie-key  # ≥32 bytes; required
APP_ENV=production              # Enables Secure cookies + COOKIE_KEY validation (prod compose)
ALLOWED_ORIGINS=http://localhost  # Exact browser origin(s); required in production; no wildcards
# COOKIE_SECURE=true            # Alternative to APP_ENV=production
# RECOGNIZE_DEBUG=1             # Local handwriting diagnostics only (ignored in production)
```

Production `docker-compose.yml` sets `APP_ENV=production`, `COOKIE_KEY`, and `ALLOWED_ORIGINS` (not `SESSION_SECRET`). The backend port is **not** published to the host; Nginx on `:80` is the public entrypoint. Dev compose uses a ≥32-byte `COOKIE_KEY` plus an explicit Vite/Nginx origin allowlist without production-secure flags so HTTP works. Pair production Secure cookies with HTTPS at the browser (`docker/nginx-tls.conf.example`). Local `APP_ENV=production` over plain `http://localhost` will drop Secure cookies in browsers — treat that compose path as a demo unless TLS is terminated in front.

## License
CC0 1.0 Universal — see `LICENSE` at the repository root. Curriculum stroke/trace data and short pedagogy glosses are also under CC0; pronunciation audio provenance/licenses are listed in `content/hiragana5/LICENSES.md`. UI fonts under `web/public/fonts/` are **SIL Open Font License** subsets: IBM Plex Sans and Noto Sans JP (vendored from Fontsource builds for self-hosting; `font-display: swap`).
