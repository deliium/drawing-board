# Implementation Plan: Recognition and Stroke Input Hardening

Branch: main
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Summary

Recognition and stroke ingestion trust client-controlled sizes. `POST /api/recognize` ignores JSON decode errors, does not bound `width` / `height` / `topN`, and the ONNX-path recognizer allocates grayscale buffers and float tensors as `O(width×height)`. WebSocket stroke messages accept arbitrary point arrays, widths, and coordinates (only a coarse `SetReadLimit(1<<20)` exists). Both paths emit detailed handwriting diagnostics (stroke point counts, ASCII canvas dumps, coordinates) unconditionally via `fmt.Printf`.

Deliver **shared limit constants and validators** for REST recognition and WebSocket stroke ingest, **safe allocation checks** before any canvas/tensor allocation, **consistent structured validation errors**, **production-safe logging** (no handwriting coordinates/images), gated **local-dev diagnostics**, light **rate-limit considerations** (in-process + documented edge options), and **boundary tests**. Preserve normal Vite practice flow (≈300–viewport canvas, `topN: 10`, width 1–20).

This is roadmap item **4. Recognition and stroke input hardening** (Prompt 04). Perimeter CSRF/CORS/WS origin (Prompt 03) and password/session (Prompt 02) stay out of scope. Recognition accuracy / honest ONNX loading remains Prompt 08.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Recognize decode | `internal/httpapi/handlers.go`: `_ = json.NewDecoder(r.Body).Decode(&req)` — errors ignored; zero-value request proceeds |
| Recognize body size | No `http.MaxBytesReader` / `LimitReader` on `/api/recognize` (or other stroke REST) |
| Dimensions / topN | `RecognizeRequest{TopN, Width, Height}` unbounded; `SimpleRecognizer` / `ONNXRecognizer` only coerce `topN <= 0` → `10` |
| Allocation | `ONNXRecognizer.strokesToTensor`: `image.NewGray(0,0,width,height)` + `make([]float32, width*height)` with **no** overflow/size guard |
| Recognize data source | Handler loads **all** user strokes from SQLite (`ListStrokesByUser`) — not from request body; hostile growth comes from prior WS ingest + huge canvas params |
| Recognize logs | `fmt.Printf` stroke counts / candidates in handler; ONNX path prints feature maps, ASCII canvas, first/last coordinates |
| WS read limit | `conn.SetReadLimit(1 << 20)` (1 MiB) — large for a single stroke JSON frame |
| WS stroke validate | No checks on `len(points)`, coords (NaN/Inf/range), `width`, `color`, `clientId`; nil stroke skipped silently |
| WS bad JSON | WARN log + `continue` — no client-visible error frame |
| DB persist | `SaveStroke` inserts every point in a loop — unbounded rows per stroke |
| Auth error shape | Auth uses `{ "error": "<code>", "message": "…" }` (`bad_json`, etc.); httpapi still uses ad-hoc `map[string]string{"error": …}` / raw `err.Error()` |
| Metrics | Client-only `trackMetric` in `web/src/services/migrationHealth.ts`; no server counters for reject/ok |
| Rate limit | None (Prompt 03 explicitly deferred WAF/rate limiting) |
| Frontend | `BoardPage`: width slider 1–20; recognize body `{ topN: 10, width: canvas.width, height: canvas.height }` (canvas sized to layout, often ~300 CSS px) |
| Nginx | No explicit `client_max_body_size` for `/api/recognize`; WS proxied without message-size policy |

## Approach

1. **Centralize limits** in a small shared package (prefer `internal/limits` or `internal/strokevalidate`) used by `httpapi`, `ws`, and `recognize`.
2. **Validate at the edges**: REST body + params before store/recognizer; WS stroke messages before `SaveStroke`; recognizer entry points refuse unsafe `width`/`height`/`topN` even if a caller forgets validation (defense in depth).
3. **Fail closed** on malformed JSON and oversize payloads (do not silently default hostile requests).
4. **Gate handwriting diagnostics** behind an explicit local-dev setting (e.g. `RECOGNIZE_DEBUG=1`); production/default logs only structured counts and outcome codes — never coordinates, tensors, or ASCII canvases.
5. **Align error JSON** with auth-style `{ "error", "message" }` for REST; add a small WS `type:"error"` frame for reject reasons so the client can ignore or toast without protocol breakage.
6. **Rate limiting**: document considerations; implement a **minimal in-process per-user limiter** for recognize + stroke ingest (token bucket / sliding window). Optional Nginx `limit_req` remains operator-owned notes, not required for acceptance.
7. **Metrics**: increment simple process-local counters (e.g. `expvar` or a tiny `internal/metrics` map) for `recognize.ok` / `recognize.reject.*` / `ws.stroke.ok` / `ws.stroke.reject.*`; keep values free of PII and handwriting content.

## Decision: Shared validators + defense-in-depth recognizer guards

| Option | Pros | Cons |
|--------|------|------|
| Validate only in HTTP/WS handlers | Thin change | Future callers / ONNX still allocate unsafely |
| Validate only inside `Recognizer.Recognize` | Protects allocation | WS/DB still accept hostile strokes; REST decode still ignored |
| **Shared limits + edge validation + recognizer guards** | One source of truth; prevents DB growth and OOM | Slightly more files |

**Decision:** Shared limit constants + `ValidateRecognizeParams` / `ValidateStroke` helpers; handlers and WS call them; `Recognize` implementations (and/or a wrapper) reject unsafe dimensions/`topN` before allocation. Cap strokes/points considered for a recognition attempt when loading from DB.

## Limit Contract

Canonical defaults (tune only with tests + README; keep frontend within bounds):

| Limit | Symbol (suggested) | Value | Notes |
|-------|--------------------|-------|-------|
| REST recognize body | `MaxRecognizeBodyBytes` | **4 KiB** | Body is only JSON params, not strokes |
| General API JSON body (optional shared) | `MaxAPIJSONBodyBytes` | **64 KiB** | Future POSTs; strokes clear/delete stay small |
| WS text frame | `MaxWSMessageBytes` | **64 KiB** | Replace `1<<20`; enough for dense stroke JSON |
| Canvas width | `MinCanvasDim` / `MaxCanvasDim` | **1 … 2048** | Covers viewport/retina practice canvases; blocks huge tensors |
| Canvas height | same | **1 … 2048** | |
| Max pixels (`width*height`) | `MaxCanvasPixels` | **2_097_152** (2048²) | Guard multiplication overflow + allocation |
| `topN` | `MinTopN` / `MaxTopN` | **1 … 32** | UI uses 10; default **10** only when field omitted via explicit `omitempty` handling — **invalid present values still 400** |
| Strokes per recognize attempt | `MaxStrokesPerRecognize` | **64** | After this, 400 `too_many_strokes` (do not silently truncate — predictable security UX) |
| Points per stroke | `MaxPointsPerStroke` | **2048** | WS reject; recognize reject if stored stroke exceeds |
| Total points per recognize | `MaxPointsPerRecognize` | **16_384** | Sum across strokes |
| Coordinate range | `MinCoord` / `MaxCoord` | **-512 … 4096** | Reject NaN/Inf; allows slight overshoot outside canvas |
| Line width | `MinStrokeWidth` / `MaxStrokeWidth` | **1 … 20** | Matches UI slider |
| Color string | `MaxColorLen` | **32** | Prefer `#RGB` / `#RRGGBB` / `#RRGGBBAA`; reject control chars |
| `clientId` | `MaxClientIDLen` | **64** | Non-empty recommended; empty allowed only if product already permits |
| Recognize rate (per user) | `RecognizeRate` | **30 / minute** | Burst 5 |
| Stroke ingest rate (per user) | `StrokeIngestRate` | **60 / minute** | Burst 20; personal practice, not collaborative flood |

**Integer overflow:** Before `width*height`, use checked multiply (`uint64` or reject if `width > MaxCanvasPixels/height`). Never allocate when product exceeds `MaxCanvasPixels`.

**`topN` semantics:**
- Missing / JSON null → default `10`.
- Present `0`, negative, or `> MaxTopN` → `400` `invalid_top_n` (do **not** silently clamp hostile values).
- Recognizer internal clamp remains only as a last-resort safety net for non-HTTP callers/tests with documented helper that already validated.

## Validation Error Contract

### REST (`POST /api/recognize`)

Align with auth: `{ "error": "<code>", "message": "<safe human text>" }`.

| HTTP | Code | When |
|------|------|------|
| 400 | `bad_json` | Decode error |
| 400 | `payload_too_large` | Body exceeds `MaxRecognizeBodyBytes` (`http.MaxBytesReader`; map `ErrHandlerTimeout`/max-bytes to this code) |
| 400 | `invalid_dimensions` | width/height out of range or pixel product too large |
| 400 | `invalid_top_n` | topN out of range |
| 400 | `too_many_strokes` | Stored strokes for user exceed `MaxStrokesPerRecognize` |
| 400 | `too_many_points` | A stroke or total points exceed caps |
| 400 | `invalid_stroke_data` | NaN/Inf/out-of-range coords found in stored strokes used for recognition |
| 401 | `unauthorized` | Existing |
| 429 | `rate_limited` | Recognize rate exceeded |
| 503 | `recognizer_unavailable` | Existing nil recognizer (normalize underscore style) |
| 500 | `internal_error` | Store/recognizer failure — **never** return raw `err.Error()` to client |

Do not log request bodies or stroke coordinates on these failures.

### WebSocket (`type: "stroke"`)

On validation failure: **do not** `SaveStroke`; **do not** echo as a successful stroke. Send:

```json
{"type":"error","error":"<code>","message":"<safe text>"}
```

| Code | When |
|------|------|
| `bad_json` | Unmarshal failure (optional: still send once per bad frame; rate-limit spam) |
| `payload_too_large` | Read limit exceeded / message too large |
| `invalid_stroke` | Missing stroke object, zero points (if required), bad width/color/clientId |
| `too_many_points` | `len(points) > MaxPointsPerStroke` |
| `invalid_coordinates` | NaN/Inf or outside `MinCoord`/`MaxCoord` |
| `rate_limited` | Stroke ingest rate exceeded |

Frontend: ignore unknown types already; extend WS client to `console.debug` error frames in DEV only (no toast required for MVP). Legitimate drawing stays unchanged.

Close codes: prefer application error frames over hard disconnect for soft validation; hard-close only on repeated abuse or read-limit violations if simpler (document choice in tasks).

## Safe Allocation and Recognizer Guards

1. `strokesToTensor` / any future rasterizer: call `limits.CheckCanvas(width, height)` first; return error on failure (handler maps to `invalid_dimensions`).
2. Prefer allocating at most `MaxCanvasPixels` floats/bytes; never trust client dims after a single check elsewhere.
3. Simple recognizer does not allocate canvases today but must still validate `topN`/`width`/`height` for API consistency (width/height unused in simple path — still reject out-of-range so clients cannot probe path differences).
4. Recognition attempt: after `ListStrokesByUser`, run `ValidateStrokeSet(strokes)` for count/points/coords before calling `Recognizer.Recognize`.

## Logging Contract

| Mode | Behavior |
|------|----------|
| Default / production | Structured `log.Printf` with level prefixes: userID, strokeCount, pointCount totals, candidateCount, error **codes** — **no** coordinates, colors optional as length-only, no ASCII art, no tensor dumps |
| `RECOGNIZE_DEBUG=1` (or `true`) **and** not `APP_ENV=production` | Allow existing detailed handwriting diagnostics (features, ASCII preview, sample coords) behind a helper `recognize.Debugf` |
| Production + `RECOGNIZE_DEBUG` | **Ignore** debug flag (or WARN once at startup that debug is disabled in production) |

Replace raw `fmt.Printf` in `handlers.go` / `onnx.go` with the gated helper. Align with existing `LOG_LEVEL` filtering where practical (`DEBUG` lines suppressed when `LOG_LEVEL=info|warn|error`).

**Never log:** full point arrays, password/cookie/CSRF material (already), handwritten images.

## Rate-Limiting Considerations

| Layer | Plan |
|-------|------|
| In-process (required light) | Per-`userID` token bucket for `POST /api/recognize` and WS `stroke` using the rates in Limit Contract; store in `sync.Map` or small mutex map; no Redis |
| Nginx (optional doc) | Note `limit_req_zone` on `/api/recognize` for multi-instance deployments; not implemented in Go acceptance |
| Auth endpoints | Out of scope (separate brute-force plan) |
| Multi-instance | Document that in-process limits are per-replica; operators needing global limits add edge rate limiting |

Failure → REST `429 rate_limited`; WS error frame `rate_limited`.

## Metrics

Minimal server-side counters (process-local):

| Name | Labels / fields |
|------|-----------------|
| `recognize_requests_total` | `result=ok\|reject\|error` |
| `recognize_reject_total` | `code=…` |
| `ws_stroke_total` | `result=ok\|reject` |
| `ws_stroke_reject_total` | `code=…` |

Exposure: `expvar` on existing process **or** DEBUG/INFO log lines `metric name=… value=…` on increment is enough if adding `/debug/vars` is undesirable for a private app. Prefer a tiny `internal/metrics` package with `Add(name string, delta int64)` and optional `expvar` publish behind non-production or authenticated health — **do not** expose unauthenticated admin metrics on the public listen address without auth. Default: in-memory + structured logs; document that Prometheus scrape is out of scope.

Frontend may keep `trackMetric('recognize.success')` and add `recognize.reject` on 4xx if trivial.

## Acceptance Criteria

1. Malformed recognize JSON → `400 bad_json` (not 200 with empty/default params).
2. Oversize recognize body → `400 payload_too_large`.
3. `width`/`height`/`topN` outside contract → `400` with the matching code; no large allocation occurs (test with absurd dims must not OOM; guard returns error).
4. WS stroke with too many points, illegal width, or NaN coords → not persisted; error frame (or documented close); no echo as success.
5. WS read limit ≤ `MaxWSMessageBytes`.
6. Production/default logs never include stroke coordinates or ASCII canvas dumps; with `RECOGNIZE_DEBUG=1` in non-production, diagnostics remain available.
7. Rate limit: burst over recognize/stroke limits yields `429` / WS `rate_limited`.
8. Boundary unit tests cover limits helpers, recognize handler rejects, recognizer allocation guard, WS validation; happy-path recognize + stroke still pass.
9. README documents limits, error codes, `RECOGNIZE_DEBUG`, rate-limit caveats; no AI/confidence marketing regressions (`internal/docguard`).
10. Legitimate UI flow (width 1–20, `topN: 10`, normal canvas size, normal stroke density) unchanged.

## Security Implications

- Stops easy DoS via huge canvas dimensions (`width*height` allocation) and oversized WS stroke payloads.
- Stops unbounded SQLite growth from hostile point floods (at ingest).
- Reduces handwriting PII leakage through logs (coordinates / visual dumps).
- Rate limits blunt authenticated abuse; residual risk: distributed multi-account abuse, multi-replica bypass, and already-stored oversized strokes until users clear — recognition path must still refuse oversized stored sets.
- Does not replace WAF, auth lockout, or XSS defenses.

## Failure Handling

| Failure | Behavior |
|---------|----------|
| JSON decode | 400 `bad_json`; DEBUG log code only |
| MaxBytes exceeded | 400 `payload_too_large` |
| Invalid params / stroke | 400 or WS error frame; WARN/DEBUG with code + userID |
| Rate limited | 429 / WS `rate_limited`; INFO/WARN count |
| Recognizer alloc guard | Error → 400 `invalid_dimensions` (client fault) not 500 |
| Store error | 500 `internal_error`; ERROR log without stroke payload |
| `RECOGNIZE_DEBUG` in production | Disabled; optional single WARN at startup |

## Commit Plan
- **Commit 1** (after tasks 1–2): `feat(limits): shared stroke and recognize input bounds`
- **Commit 2** (after tasks 3–5): `feat(security): validate recognize and WS stroke ingestion`
- **Commit 3** (after tasks 6–8): `chore(obs): safe recognize logging, metrics, and docs`

## Tasks

### Phase 1: Limits package and recognizer guards

- [x] Task 1: Shared limit constants + validators
  - Add `internal/limits` (or `internal/strokevalidate`) with the Limit Contract constants and helpers:
    - `CheckCanvas(width, height int) error`
    - `NormalizeOrValidateTopN(topN int, present bool) (int, error)` — exact API up to implementer if using pointer/`json.RawMessage`; document omitted vs invalid
    - `ValidateStrokePoints(points []T) error` (generic via float X/Y accessors or duplicated thin wrappers for `db`/`ws`/`recognize` point types)
    - `ValidateStrokeMeta(width int, color, clientID string) error`
    - `ValidateStrokeSet(strokeCount, totalPoints int) error` / per-stroke point caps
    - Checked `width*height` against `MaxCanvasPixels`
  - Reject NaN/Inf via `math.IsNaN` / `math.IsInf`.
  - Table-driven unit tests for boundaries (min-1, max, max+1, 0, negative, overflow pair like `width=MaxCanvasDim, height=MaxCanvasDim` OK; larger product rejected).
  - LOGGING REQUIREMENTS: none in pure helpers
  - Files: `internal/limits/*.go`, `internal/limits/*_test.go`
  - Depends on: none

- [x] Task 2: Recognizer safe allocation + debug diagnostics
  - In `ONNXRecognizer.strokesToTensor` / `Recognize`: call `CheckCanvas` before `image.NewGray` / `make([]float32, …)`.
  - In `SimpleRecognizer.Recognize`: validate dimensions/`topN` consistently (return error on violation).
  - Replace unconditional `fmt.Printf` diagnostics with `recognize.debugf` gated by `RECOGNIZE_DEBUG` and non-production `APP_ENV`.
  - Startup (optional): INFO log `recognize_debug=true|false`.
  - LOGGING REQUIREMENTS:
    - Default: no coordinate/ASCII dumps
    - Debug gate: preserve useful local diagnostics
    - Production forces debug off
  - Files: `internal/recognize/onnx.go`, `internal/recognize/simple.go`, small `internal/recognize/log.go`, tests in `internal/recognize/*_test.go`
  - Depends on: Task 1

### Phase 2: REST and WebSocket enforcement

- [x] Task 3: Harden `POST /api/recognize`
  - Wrap body with `http.MaxBytesReader(w, r.Body, MaxRecognizeBodyBytes)`.
  - Decode JSON; on error → `400 bad_json`.
  - Validate width/height/topN via limits helpers; map errors to contract codes.
  - After `ListStrokesByUser`, enforce stroke/point/coord caps → `400` codes (not silent truncate).
  - Map rate limit → `429 rate_limited`.
  - Stop ignoring decode errors; stop returning raw `err.Error()`; use `{error,message}` helper (share style with auth or small `writeAPIError`).
  - Remove/replace `fmt.Printf` request dumps with structured safe logs (`stroke_count`, `candidate_count`).
  - LOGGING REQUIREMENTS:
    - INFO/DEBUG: `[httpapi.Recognize] userID=… result=ok|reject code=… strokes=… candidates=…`
    - Never log points/coords
  - Files: `internal/httpapi/handlers.go`, `internal/httpapi/handlers_test.go` (or `recognize_handler_test.go`), rate-limiter wiring
  - Depends on: Tasks 1–2

- [x] Task 4: Harden WebSocket stroke ingestion
  - Set `SetReadLimit(MaxWSMessageBytes)`.
  - Validate stroke messages with shared helpers before `SaveStroke`.
  - On failure: skip persist/echo-as-stroke; send `type:error` frame (and/or close on read-limit — document).
  - Apply per-user stroke ingest rate limit.
  - Keep delete-path id validation (`<=0` reject) consistent if trivial.
  - LOGGING REQUIREMENTS:
    - WARN: `[ws.Handle] reject type=stroke userID=… code=…`
    - INFO: successful save unchanged (`id=`, not points)
  - Files: `internal/ws/handler.go`, `internal/ws/handler_test.go` / new `stroke_validate_test.go`
  - Depends on: Task 1

- [x] Task 5: In-process rate limiter + metrics counters
  - Implement tiny per-user limiter used by Tasks 3–4 (may live in `internal/limits/ratelimit.go` or `internal/security/ratelimit.go`).
  - Add `internal/metrics` (or expvar registration) for the Metrics table; increment from recognize + WS paths.
  - Do not expose unauthenticated public metric dump on `:8080` unless gated; structured logs acceptable.
  - Unit-test limiter burst/deny behavior.
  - LOGGING REQUIREMENTS: optional `DEBUG [metrics] name=… delta=…` only if useful; prefer silent counters + test accessors
  - Files: new small packages + call sites in httpapi/ws
  - Depends on: Tasks 3–4 (or land stub in Phase 1 and wire in 3–4)

### Phase 3: Frontend awareness, docs, verification

- [x] Task 6: Frontend WS/REST tolerance (minimal)
  - Ensure recognize errors surface a short safe message (existing catch path).
  - Optionally handle WS `type:"error"` with DEV `console.debug` only.
  - Do not change normal draw/recognize happy path; keep `topN: 10` and width 1–20.
  - LOGGING REQUIREMENTS: DEV-only; no coordinate dumps
  - Files: `web/src/pages/BoardPage.vue`, `web/src/services/wsClient.ts` (as needed), light unit test if pattern exists
  - Depends on: Tasks 3–4

- [x] Task 7: Documentation
  - README: document limit table (or summary), recognize/WS error codes, `RECOGNIZE_DEBUG`, rate-limit per-replica caveat, removal of always-on handwriting log dumps from Troubleshooting “Debug Mode” (point to `RECOGNIZE_DEBUG` instead).
  - Note Nginx optional `client_max_body_size` / `limit_req` as operator extras.
  - Keep heuristic honesty (`internal/docguard` must pass).
  - LOGGING REQUIREMENTS: document prefixes `[httpapi.Recognize]`, `[ws.Handle] reject`, recognize debug gate
  - Files: `README.md`
  - Depends on: Tasks 1–6

- [x] Task 8: Verification against acceptance criteria
  - `go test ./…` including new boundary tests; `cd web && npm test`.
  - Manual smoke:
    1. Login → draw normal stroke → recognize → candidates.
    2. `curl` recognize with `{` → `bad_json`.
    3. Recognize with `width: 999999, height: 999999` → `invalid_dimensions`, process stable.
    4. WS (or unit) oversize points → reject, DB unchanged.
    5. Default logs: no ASCII canvas / coordinates; with `RECOGNIZE_DEBUG=1` locally, diagnostics appear.
    6. Rapid-fire recognize → eventually `429`.
  - Files: none required
  - Depends on: Tasks 6–7

## Out of Scope
- CSRF/CORS/WS origin policy changes (Prompt 03 — done)
- Password/session/`COOKIE_KEY` (Prompt 02)
- Honest ONNX model loading / accuracy work (Prompt 08)
- Redis/global distributed rate limiting, full Prometheus stack, WAF
- Migrating already-oversized historical strokes (beyond reject-on-recognize / user Clear)
- Changing drawing UX limits beyond staying inside server caps
- Collaborative board features

## Risks & Notes
- **Stored hostile strokes:** If a DB already has huge strokes from before this change, recognize must reject with `too_many_*` rather than allocate; users can Clear. Optional one-time cleanup is out of scope.
- **Canvas 2048 cap:** Very large monitors / zoomed canvases must stay ≤ 2048 CSS/backing pixels; frontend already sizes to layout (~300–1200 typical). If product later needs larger, raise `MaxCanvasDim` deliberately with pixel-product still capped.
- **Silent topN default vs clamp:** Prefer reject invalid explicit values; only default when omitted — avoids masking attackers and keeps tests crisp.
- **WS error frames:** Old clients ignoring unknown types remain safe; ensure error frames never include submitted coordinates.
- **Rate limiter memory:** Bound map size (e.g. max tracked users / TTL GC) so unbounded distinct userIDs cannot grow the limiter map forever.
- **Simple vs ONNX path:** Both must enforce the same dimension/`topN` errors so clients cannot fingerprint which recognizer is active via status codes alone.
- **Parallel Prompt 08:** When real model I/O arrives, reuse `CheckCanvas` and never rasterize before validation.
