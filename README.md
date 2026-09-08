![drawing-board](http://counter.seku.su/cmoe?name=drawing-board&theme=rule34-big)

# Japanese Handwriting Practice (Vue + Go)

A **personal** Japanese handwriting training app: practice on a private canvas, persist your strokes, and get recognition feedback. Strokes are never shared with other users.

- **Vue 3 (TypeScript, Vite)** practice canvas with authenticated WebSocket persist/echo
- **Go backend** with Gorilla mux, WebSocket, SQLite persistence
- **User authentication** with session-based login/register/logout
- **Drawing tools**: Pencil and Eraser with hit-testing
- **Undo functionality**: Ctrl+Z to undo last stroke
- **Handwriting recognition**: heuristic pattern-based character suggestions for practice feedback (not a trained AI model)
- **Private stroke persistence**: drawings are scoped per user and restored only for that account

## Features

### Drawing Tools
- **Pencil**: Draw with customizable color and width
- **Eraser**: Remove individual strokes by clicking on them
- **Undo**: Press `Ctrl+Z` (or `Cmd+Z` on Mac) to undo the last stroke
- **Clear**: Remove all your drawings from the canvas and database

### Handwriting Recognition
- **Heuristic recognition**: Pattern-based ranking of candidate characters from stroke geometry
- **Supported Characters**: 一, 二, 三, 十, 丨, 丶, 人, 大, 小, 中, 国, 学, 生, and more
- **On-demand analysis**: Click "Recognize" for candidate suggestions with heuristic match scores (not calibrated confidence)
- **Pattern Detection**: Detects crosses (十), horizontal lines (三), and other simple stroke patterns

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
# Default: heuristic simple recognizer
go run ./cmd/server

# Optional ONNX_MODEL path (currently falls back to simple recognizer until model loading is implemented)
ONNX_MODEL=./models/handwriting.onnx go run ./cmd/server
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
3. **Practice** — after auth you are redirected to the private board. Use the pencil tool to write characters on the canvas.
4. **Save** — strokes are queued and persisted over WebSocket with per-operation `opId` acknowledgements (header shows Connecting / Saving / Saved / Offline / Sync error).
5. **Logout** — use Logout on the board to clear the session and return to the auth page.

Registration and login are **not** on the board header; they live only on the public auth routes.

### Drawing Tools
- **Color Picker**: Choose any color for your pencil
- **Width Slider**: Adjust line thickness from 1-20 pixels
- **Pencil Tool**: Default drawing tool
- **Eraser Tool**: Click on any stroke to remove it
- **Undo Button**: Click to undo the last stroke (or use Ctrl+Z)
- **Clear Button**: Remove all your drawings

### Handwriting Recognition
1. **Draw a character** on the canvas (try 一, 二, 三, 十)
2. **Click "Recognize"** button
3. **View results** showing possible characters with heuristic match scores (not calibrated confidence)
4. **Try different patterns** to see how the recognizer ranks candidates

### Keyboard Shortcuts
- **Ctrl+Z** (Windows/Linux) or **Cmd+Z** (Mac): Undo last stroke
- **Escape**: Cancel current drawing operation

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

# ONNX model for advanced recognition
ONNX_MODEL=./models/handwriting.onnx
```

Local `make run` without `APP_ENV=production` / `COOKIE_SECURE` keeps `Secure=false` so HTTP/Vite works. Startup logs `INFO [main] cookie_secure=true|false` and `INFO [main] origin_policy mode=… count=… origins=…`. In non-production mode a weak/default `COOKIE_KEY` only warns; in production-secure mode the process exits if `COOKIE_KEY` is missing, shorter than 32 bytes, or equal to a documented sentinel (`change-me-please-32-bytes-min` / `please-change-this-32-bytes-min`). With `APP_ENV=production`, missing/empty/`*`/`invalid` `ALLOWED_ORIGINS` also exits before listen.

If TLS terminates at Nginx in front of Go, the public site must still be HTTPS for browsers to send `Secure` cookies, and `ALLOWED_ORIGINS` must match the browser-facing origin exactly (e.g. `https://learn.example.com`). See `docker/nginx-tls.conf.example`. Forwarded headers may be set for logs; they are **not** used for origin allowlisting or auth.

### Production Build
```bash
# Build frontend
make build-web

# Run production server
ADDR=:8080 STATIC_DIR=web/dist DB_PATH=file:data.db?_fk=1 COOKIE_KEY=your-secure-key make run
```

### ONNX Model Setup (Optional)
Downloads a model artifact for the optional `ONNX_MODEL` path. Until honest model loading lands, the server still uses the simple heuristic recognizer:
```bash
# Optional download (does not by itself enable ML recognition today)
make onnx-model

ONNX_MODEL=./models/handwriting.onnx go run ./cmd/server
```

## API Reference

### Authentication Endpoints
Cookie session name: `sid` (`HttpOnly`, `SameSite=Lax`, `Path=/`; `Secure` when production-secure mode is on). Register/login **rotate** the session (`Sessions.New` after invalidating any prior `sid`). Auth handlers log with prefixes `[auth.Register]`, `[auth.Login]`, `[auth.Logout]`, `[auth.Me]`, `[auth.hash]`, `[auth.startSession]` (level filtered via `LOG_LEVEL`). Operator signals: `INFO [main] cookie_secure=…`, `INFO [main] origin_policy …`, `[cors]`, `[csrf]`, `[ws.CheckOrigin]`.

**CSRF (double-submit):** all `POST /api/*` require cookie `csrf` (readable by JS, `SameSite=Lax`, `Secure` in production-secure mode) plus matching header `X-CSRF-Token`. `GET /api/me` and `GET /api/csrf` ensure the cookie (including anonymous `401` on `/api/me`). The Vue client sends the header automatically after bootstrap. Failure: `403` `{ "error": "csrf_rejected", "message": "…" }`. WebSocket upgrades are not CSRF-token gated; they require a valid session cookie and an allowlisted `Origin`.

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
| 403 | `csrf_rejected` | Missing/mismatched CSRF cookie + `X-CSRF-Token` on `POST /api/*` |

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
- `GET /api/strokes` - Get user's saved strokes (authenticated)
- `POST /api/strokes/clear` - Clear all user's strokes (authenticated)
- `POST /api/strokes/delete?id={id}` - Delete specific stroke (authenticated)

### Recognition Endpoint
- `POST /api/recognize` — Recognize drawn characters `{ topN: 10, width: 300, height: 300 }` (authenticated). Body is params only (strokes come from the user store). Max body **4 KiB**.

| HTTP | Code | Meaning |
|------|------|---------|
| 400 | `bad_json` | Body not JSON |
| 400 | `payload_too_large` | Body exceeds 4 KiB |
| 400 | `invalid_dimensions` | `width`/`height` outside 1…2048 or pixel product too large |
| 400 | `invalid_top_n` | Present `topN` outside 1…32 (omitted → default 10) |
| 400 | `too_many_strokes` | More than 64 stored strokes |
| 400 | `too_many_points` | Per-stroke or total point caps exceeded |
| 400 | `invalid_stroke_data` | NaN/Inf/out-of-range coords in stored strokes |
| 401 | `unauthorized` | No session |
| 429 | `rate_limited` | Per-user recognize rate exceeded (30/min, burst 5; **per process replica**) |
| 503 | `recognizer_unavailable` | No recognizer configured |
| 500 | `internal_error` | Store/recognizer failure (no raw error text) |

Canvas bounds: width/height **1…2048**, max pixels **2048²**. Legitimate UI (`topN: 10`, ~300px canvas, width 1–20) is unchanged.

### WebSocket
- `WS /ws` - Authenticated **private persist + echo** channel (cookie session required)

The hub delivers messages only to connections belonging to the same `user_id` (multi-tab same account receives echoes; other users never see your strokes). This is **not** a collaborative/shared board.

Text frames are capped at **64 KiB**. Stroke ingest is rate-limited per user (**60/min**, burst 20; per process replica). Points per stroke ≤ **2048**; coords in **-512…4096**; line width **1…20**; color `#RGB` / `#RRGGBB` / `#RRGGBBAA`. Each mutating message requires a client `opId` (≤ **36** chars). Creates are idempotent on `(user_id, op_id)` in SQLite.

**WebSocket Messages:**
```json
// Send stroke (idempotent persist for the authenticated user; ack to sender; echo to that user's connections)
{"type":"stroke","opId":"550e8400-e29b-41d4-a716-446655440000","stroke":{"points":[{"x":10,"y":20}],"color":"#1d4ed8","width":4,"clientId":"abc","startedAtUnixMs":1690000000000}}

// Delete stroke (scoped to the authenticated user; ack to sender; echo to that user's connections)
{"type":"delete","opId":"550e8400-e29b-41d4-a716-446655440001","delete":123}

// Acknowledgement (confirmation for the originating connection)
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440000","ok":true,"strokeId":456}
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440001","ok":true,"delete":123}
{"type":"ack","opId":"550e8400-e29b-41d4-a716-446655440000","ok":false,"error":"rate_limited","message":"too many strokes"}

// Frame-level rejection without a bound opId (not persisted)
{"type":"error","error":"bad_json","message":"invalid JSON message"}
```

WS error / nack codes include `bad_json`, `payload_too_large`, `invalid_stroke`, `invalid_op_id`, `too_many_points`, `invalid_coordinates`, `rate_limited`, `internal_error`. Soft validation prefers an `ack` nack (when `opId` is known) or `error` frame over disconnect; oversize frames may close the connection after the read-limit error.

The Vue client keeps a **bounded in-memory queue** (32 ops), reconnects with exponential backoff, and retries until ack / nack / attempt budget. Header status: **Connecting… / Saving… / Saved / Offline — retrying… / Sync error**. Unmatched inbound strokes remain non-authoritative (ignores foreign live creates). Full page reload uses `GET /api/strokes` as source of truth and drops the session queue (no durable offline storage in this iteration). DEV builds `console.debug` WS frames without toasts.

**Operator extras (optional Nginx):** `client_max_body_size` on `/api/recognize`, `limit_req` for multi-instance deployments. In-process limits are per replica only.
## Recognition System

The application includes two recognition systems:

### 1. Simple Recognizer (Default)
- **Pattern-based analysis** of stroke shapes and directions
- **No external dependencies** — works out of the box
- **Supports basic characters**: 一, 二, 三, 十, 丨, 丶, 人, 大, 小, 中, 国, 学, 生
- **Returns heuristic match scores** — useful for ranking candidates, not calibrated confidence

### 2. ONNX Recognizer (Optional path)
- Configured via `ONNX_MODEL`; intended for a future model-backed recognizer
- **Current runtime:** model load is not fully implemented — the server falls back to the simple heuristic recognizer and logs that clearly
- Do not treat `ONNX_MODEL` / `make onnx-model` as an active ML accuracy upgrade until Prompt 08 (honest recognition) lands

## Troubleshooting

### Common Issues
1. **"Address already in use"**: Stop existing server processes with `pkill -f "go run"`
2. **Recognition not working**: Check `[httpapi.Recognize]` logs; enable `RECOGNIZE_DEBUG=1` locally only if you need feature dumps
3. **WebSocket connection failed**: Ensure backend is running on port 8080 and you are signed in
4. **Frontend not loading**: Check if `npm run dev` is running on port 5173
5. **Seeing another user's strokes**: Should not happen; verify you are on a build with per-user `sendToUser` (not global broadcast) and check logs below

### Debug Mode

Verbose server log prefixes for privacy and persistence:

| Prefix | Meaning |
|--------|---------|
| `[ws.Handle]` | Connect/disconnect (`userID`, remote), inbound stroke/delete, save/delete INFO, upgrade/read errors, stroke reject codes |
| `[ws.sendToUser]` | Delivery to one user's connections; `recipients=` should stay within that account (e.g. 1–N tabs) |
| `[httpapi.ListStrokes]` / `[httpapi.ClearStrokes]` / `[httpapi.DeleteStroke]` | Authenticated REST entry (`userID`), clear count, delete success, store errors |
| `[httpapi.Recognize]` | Recognize result (`ok`/`reject` + code), stroke/point/candidate counts — no coordinates |
| `[recognize]` | Startup `recognize_debug=…`; gated diagnostics when `RECOGNIZE_DEBUG=1` (non-production) |

Example privacy check while two users practice: user A's stroke logs should show `sendToUser` recipient counts only for A's open tabs, never B's.

Frontend (Vite dev): browser console uses `[wsClient]` and `[BoardPage.ws]` for connect/send/ignore reasons.

Default recognition logs are structured counts only (`[httpapi.Recognize] userID=… result=ok|reject code=… strokes=… candidates=…`) — **never** stroke coordinates or ASCII canvases. WS rejects log `[ws.Handle] WARN reject type=stroke userID=… code=…`.

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
├── internal/            # Go internal packages
│   ├── auth/           # Authentication logic
│   ├── db/             # Database layer
│   ├── httpapi/        # HTTP API handlers
│   ├── limits/         # Shared stroke/recognize input bounds
│   ├── metrics/        # Process-local reject/ok counters
│   ├── recognize/      # Recognition algorithms
│   └── ws/             # WebSocket handling
├── web/                # Vue 3 frontend
│   ├── src/           # TypeScript / Vue source
│   └── public/        # Static assets
└── models/            # ONNX model files
```

### Make Commands

#### Development Commands
```bash
make backend          # Run Go backend
make frontend         # Run Vue frontend
make build-web        # Build frontend for production
make run              # Run production server
make onnx-model       # Download ONNX model
make test             # Run all unit tests
make test-verbose     # Run tests with verbose output
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
ONNX_MODEL=./models/handwriting.onnx  # ONNX model path (optional)
```

Production `docker-compose.yml` sets `APP_ENV=production`, `COOKIE_KEY`, and `ALLOWED_ORIGINS` (not `SESSION_SECRET`). The backend port is **not** published to the host; Nginx on `:80` is the public entrypoint. Dev compose uses a ≥32-byte `COOKIE_KEY` plus an explicit Vite/Nginx origin allowlist without production-secure flags so HTTP works. Pair production Secure cookies with HTTPS at the browser (`docker/nginx-tls.conf.example`). Local `APP_ENV=production` over plain `http://localhost` will drop Secure cookies in browsers — treat that compose path as a demo unless TLS is terminated in front.

## License
MIT License - see LICENSE file for details.
