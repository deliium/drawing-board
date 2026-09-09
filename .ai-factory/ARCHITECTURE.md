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
├── cmd/contentvalidate/        # curriculum pack validator CLI
├── content/
│   └── hiragana5/              # reviewed curriculum packs (vN) + drafts/ quarantine
├── internal/
│   ├── auth/                   # register/login/logout/me, password hashing, sessions
│   ├── curriculum/             # load/validate hiragana5 content packs
│   ├── db/                     # SQLite store, versioned migrations, board + learning repos
│   │   └── migrations/         # numbered Up steps (baseline, learning, pedagogy, JA pedagogy)
│   ├── learn/                  # learning-domain types + repository interfaces
│   ├── httpapi/                # REST handlers (strokes, recognize, practice attempts)
│   ├── ws/                     # WebSocket hub, CheckOrigin, ingest, ack/echo
│   ├── limits/                 # shared validation bounds (stroke, recognize, opId)
│   ├── recognize/              # hiragana5 multi-criterion assess + heuristic free-board rank; correction catalog
│   ├── security/               # CORS / CSRF / origin policy helpers
│   ├── metrics/                # process-local counters
│   ├── features/               # FEATURE_* learning surface kill switches
│   └── docguard/               # README honesty regression tests
├── web/
│   ├── public/fonts/           # Self-hosted OFL subsets (IBM Plex Sans, Noto Sans JP)
│   ├── src/styles/             # Design tokens, focus, reduced-motion, font-face
│   ├── src/i18n/               # Lightweight EN/JA catalogs + correction display map
│   ├── src/pages/              # AuthPage, BoardPage, PracticeHub/History/Character
│   ├── src/components/         # AppShell (guestShell vs authed nav); DEV-only Dev metrics
│   ├── src/components/practice/# Journey chrome (intro, stroke-order, overlay, …)
│   ├── src/canvas/             # CSS/DPR coords, layout helpers, drawStrokes, hitTest
│   ├── src/curriculum/         # hiragana5 trace fixtures (geometry for UI)
│   ├── src/composables/        # usePracticeCanvas, usePracticeJourney, useLocale, useFeatureFlags
│   ├── src/services/           # apiFetch, attempts/curriculum/progress/features, wsClient, strokeSync
│   ├── src/stores/             # client state (free-board oriented)
│   ├── src/router/             # auth/guest guards; guestShell meta on login/register
│   ├── tests/                  # Vitest unit/contract/integration (+ axe a11y)
│   └── e2e/                    # Playwright learner-journey smoke (Chromium)
├── scripts/                    # e2e-webserver.sh and other CI helpers
├── .github/workflows/          # PR CI + nightly race/Playwright/CodeQL
├── docker/                     # Nginx examples, compose assets
└── .ai-factory/                # AI Factory plans, patches, context
```

## Dependency Rules

- ✅ `cmd/server` wires `internal/*` packages; packages do not import `cmd/`
- ✅ `httpapi` and `ws` may call `db`, `auth`, `limits`, `recognize`, `metrics`, `features`
- ✅ `httpapi` attempt handlers depend on `internal/learn` repos + `recognize.Assessor`; SQLite impl lives in `internal/db`
- ✅ Learning product surfaces are gated by `internal/features` kill switches; migrations stay forward-only (flags are not substitutes for schema rollback)
- ✅ Free-board stroke tables stay isolated from attempt stroke tables (no shared FK / clear coupling; no attempt↔board FK)
- ✅ Board `POST /api/recognize` stays heuristic-only (no `target`); single-character practice uses `/api/attempts`
- ✅ Practice journey UI uses local canvas ink + REST attempts; free-board WS/`boardRev` must not gate submit/assess
- ✅ Schema evolves only via versioned migrations in `internal/db/migrations` (fail-closed on `Open`)
- ✅ Trusted curriculum lives under `content/hiragana5/vN`; `internal/curriculum` loads/validates (including audioRef file presence); seed + recognize both consume the pack (no dual-maintained stroke JSON)
- ✅ Pronunciation audio is pack-owned + statically mirrored (`web/public/audio/hiragana5/`); play is client-side on demand with soft-fail missing/error paths
- ✅ Optional `kanjiExtensions` in pack JSON is a future hook only — hiragana5 keeps it empty
- ✅ `limits` is shared validation — keep free of HTTP/WS transport types when practical
- ✅ Vue `services/` owns network I/O; `canvas/` + `composables/` own drawing geometry/lifecycle; `i18n/` owns learner chrome strings; pages compose UI + call services
- ✅ Learner-facing builds must not mount internal DEV metrics (`MigrationHealthPanel` / “Dev metrics”) — `import.meta.env.DEV` only
- ❌ Do not add a global WS broadcast path — delivery is `sendToUser(userID, …)` only
- ❌ Do not put recognition rasterization / large allocations before `limits.CheckCanvas` / validators
- ❌ Frontend must not treat unmatched inbound stroke creates as authoritative canvas state
- ❌ Do not put lesson/attempt types into `internal/recognize` (recognize stays pure scoring)
- ❌ Do not seed or embed `content/hiragana5/drafts/` — AI/WIP only until human review publishes into `vN`
- ❌ Do not hardcode practice/board canvas CSS to fixed 300×300 — use `--canvas-size` + live logical size on submit
- ❌ Do not add SM-2/FSRS ease factors, streaks, notifications, or social-comparison features without a dedicated plan
- ❌ Do not put stroke point arrays in attempt history list payloads
- ✅ Leitner-style personal review (`review_box` / `due_at`) lives on progress and updates from assessed pass/fail only — schedule helpers stay in `internal/learn`

## Layer/Module Communication

- **REST:** cookie session → `httpapi` handlers → `db.Store` / `LearnStore` / `recognize` (list returns `{boardRev,strokes}`; board recognize is revision-gated; attempt assess uses submitted attempt strokes only)
- **WebSocket:** cookie + allowlisted Origin → `ws.Hub` → validate (`limits`) → `ApplyStrokeCreate|Delete|Clear` → `sendAck` to sender → `sendToUser` echo (includes `boardRev`)
- **Frontend:** `apiFetch` (CSRF header) for REST reads/recognize; `wsClient` queue/ack for create/delete/clear; `GET /api/strokes` on load clears session queue and sets `boardRev`

## Key Principles

1. **Per-user isolation** — strokes and live echoes stay within `user_id`
2. **Validate at the edge** — shared `limits` for recognize body params and WS stroke meta/points/`opId`
3. **Ack before trust** — client queue retries until ack/nack/budget; reload trusts REST
4. **Honest recognition** — MVP assessment is deterministic multi-criterion target comparison for five hiragana; free-board heuristic scores are match-score ranking aids, not calibrated confidence or ONNX/ML
5. **Assessor owns scoring + corrections** — `recognize` normalizes strokes, scores criteria, selects ≤2 learner feedback messages; `httpapi` persists engine `feedback` (does not invent a parallel correction map)
6. **Learning storage** — durable curriculum/attempts/progress behind versioned migrations; board scratchpad remains separate
7. **Reviewed content pack** — five-vowel `hiragana5` pedagogy + stroke/trace geometry + reviewed mora audio + guidance versioned under `content/`; seed and recognize agree on glyphs, stroke counts, and `contentVersion`
8. **Language-learning chrome** — on-demand pronunciation play, hideable romanization preference, concise bilingual guidance; kanji extension points in schema without kanji UI
9. **Attempt-scoped practice** — create/submit/assess/abandon via REST; assessment never loads free-board strokes; retry = new attempt row; UI should prefer score + feedback over candidates
10. **Guided journey UI** — `/practice` hub + `/practice/history` + `/practice/:characterId` stage machine; curriculum/progress GETs for pedagogy; compute-on-read mastery + schedule-aware next suggestion; practice-data clear is personal only; trace geometry from client fixtures; no WS for attempt ink
11. **Responsive bilingual accessible SPA** — mobile-first tokens (`--canvas-size`), client EN/JA preference (`web/src/i18n`), correction **display** by code (API EN message persisted), skip link / focus-visible / live regions / textual result summary; axe in Vitest; AppShell hides Practice/History/Board nav on `guestShell` auth routes; learner builds omit DEV diagnostics panel
12. **Forgiving personal review** — Leitner-style boxes + UTC wall-clock `due_at`; overdue without shame; next prefers due reviews before new introduction; never market as SM-2/FSRS or streak gamification
13. **Mastery honesty** — mastery labels are a simple practice summary from assessed attempts (not SM-2 stages, belts, grades, or calibrated scores); abandoned drafts never count
14. **Production perimeter** — fail-fast `COOKIE_KEY` / `ALLOWED_ORIGINS` when production-secure
15. **Staged learning flags** — `FEATURE_PRACTICE` / `PROGRESS` / `REVIEW` / `AUDIO` gate product surfaces for R1–R4 cutovers; schema remains forward-only with backup/restore rollback

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
- ❌ Marketing pronunciation clips as “AI pronunciation” / “perfect native TTS” without pack license + listen review
- ❌ Expanding open-set guesses to unrestricted kanji without measured evidence
- ❌ Using `Access-Control-Allow-Origin: *` with credentialed requests
