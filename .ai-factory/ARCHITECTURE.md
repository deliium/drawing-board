# Architecture: Structured Modules (adapted to existing layout)

## Overview

This project is a small modular monolith: a Go HTTP/WebSocket server under `cmd/server` + `internal/*`, and a Vue 3 SPA under `web/`. Packages are organized by responsibility (auth, db, httpapi, ws, limits, recognize, security, metrics). The document describes **current reality** so agents extend the app without inventing a parallel folder scheme.

## Decision Rationale

- **Project type:** personal training web app, single deployable
- **Tech stack:** Go + Vue + SQLite
- **Key factor:** existing `internal/<package>` boundaries already match a lightweight structured-modules style; team size and domain complexity do not justify Explicit Architecture or microservices

## Folder Structure

```text
drawing-board/
├── cmd/server/                 # process entry (wiring, env, listen)
├── internal/
│   ├── auth/                   # register/login/logout/me, password hashing, sessions
│   ├── db/                     # SQLite store, versioned migrations, board + learning repos
│   │   └── migrations/         # numbered Up steps (baseline board, learning domain)
│   ├── learn/                  # learning-domain types + repository interfaces
│   ├── httpapi/                # REST handlers (strokes, recognize, CSRF helpers)
│   ├── ws/                     # WebSocket hub, CheckOrigin, ingest, ack/echo
│   ├── limits/                 # shared validation bounds (stroke, recognize, opId)
│   ├── recognize/              # hiragana5 target comparison + heuristic free-board ranking
│   ├── security/               # CORS / CSRF / origin policy helpers
│   ├── metrics/                # process-local counters
│   └── docguard/               # README honesty regression tests
├── web/
│   ├── src/pages/              # AuthPage, BoardPage
│   ├── src/canvas/             # CSS/DPR coords, drawStrokes, hitTest
│   ├── src/composables/        # usePracticeCanvas (pointer/resize lifecycle)
│   ├── src/services/           # apiFetch, wsClient, strokeSync
│   ├── src/stores/             # client state
│   ├── src/router/             # auth/guest guards
│   └── tests/                  # Vitest unit/contract/integration
├── docker/                     # Nginx examples, compose assets
└── .ai-factory/                # AI Factory plans, patches, context
```

## Dependency Rules

- ✅ `cmd/server` wires `internal/*` packages; packages do not import `cmd/`
- ✅ `httpapi` and `ws` may call `db`, `auth`, `limits`, `recognize`, `metrics`
- ✅ `httpapi` / future attempt handlers depend on `internal/learn` interfaces; SQLite impl lives in `internal/db`
- ✅ Free-board stroke tables stay isolated from attempt stroke tables (no shared FK / clear coupling)
- ✅ Schema evolves only via versioned migrations in `internal/db/migrations` (fail-closed on `Open`)
- ✅ `limits` is shared validation — keep free of HTTP/WS transport types when practical
- ✅ Vue `services/` owns network I/O; `canvas/` + `composables/` own drawing geometry/lifecycle; pages compose UI + call services
- ❌ Do not add a global WS broadcast path — delivery is `sendToUser(userID, …)` only
- ❌ Do not put recognition rasterization / large allocations before `limits.CheckCanvas` / validators
- ❌ Frontend must not treat unmatched inbound stroke creates as authoritative canvas state
- ❌ Do not put lesson/attempt types into `internal/recognize` (recognize stays pure scoring)

## Layer/Module Communication

- **REST:** cookie session → `httpapi` handlers → `db.Store` / `recognize` (list returns `{boardRev,strokes}`; recognize is revision-gated)
- **WebSocket:** cookie + allowlisted Origin → `ws.Hub` → validate (`limits`) → `ApplyStrokeCreate|Delete|Clear` → `sendAck` to sender → `sendToUser` echo (includes `boardRev`)
- **Frontend:** `apiFetch` (CSRF header) for REST reads/recognize; `wsClient` queue/ack for create/delete/clear; `GET /api/strokes` on load clears session queue and sets `boardRev`

## Key Principles

1. **Per-user isolation** — strokes and live echoes stay within `user_id`
2. **Validate at the edge** — shared `limits` for recognize body params and WS stroke meta/points/`opId`
3. **Ack before trust** — client queue retries until ack/nack/budget; reload trusts REST
4. **Honest recognition** — MVP assessment is deterministic target comparison for five hiragana; free-board heuristic scores are match-score ranking aids, not calibrated confidence or ONNX/ML
5. **Learning storage** — durable curriculum/attempts/progress behind versioned migrations; board scratchpad remains separate
6. **Production perimeter** — fail-fast `COOKIE_KEY` / `ALLOWED_ORIGINS` when production-secure

## Code Organization Note

- **New Features:** Prefer placing code in the matching `internal/<package>` or `web/src/services|pages` module above.
- **Existing Code:** Documented as-is. Prefer conventions here when touching files; do not rewrite unrelated packages for purity.
- **Interoperability:** Keep store interfaces thin; avoid leaking WS message types into `db`.

## Code Examples

### Idempotent stroke save (backend)

```go
id, created, err := store.SaveStrokeIdempotent(userID, opID, color, width, startedAt, points)
// created=false on (user_id, op_id) hit — still ack with same stroke id
```

### Ack then echo (WebSocket)

```go
hub.sendAck(conn, opID, true, &boardRev, &id, nil, nil, "", "")
hub.sendToUser(uid, strokeMessage) // includes boardRev
```

### Client enqueue (frontend)

```ts
ws.send({ type: 'stroke', opId, baseRev: ws.getBoardRev(), stroke: payload })
// never silently drop when socket is closed — queue + reconnect
```

## Anti-Patterns

- ❌ Reintroducing collaborative multi-user live canvas semantics
- ❌ Dual-writing deletes/clear via REST and WS for the same UI action (Vue uses WS; REST clear is scripts/tests only)
- ❌ Allocating `width×height` recognize buffers before canvas bounds checks
- ❌ Documenting recognition as “AI”, calibrated confidence, or an active ONNX/MNIST handwriting upgrade
- ❌ Expanding open-set guesses to unrestricted kanji without measured evidence
- ❌ Using `Access-Control-Allow-Origin: *` with credentialed requests
