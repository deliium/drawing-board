# Project Rules (Axioms)

Hard requirements for this repository. More specific files under `.ai-factory/rules/` override when they conflict.

- Strokes are **private per user** — never broadcast WebSocket stroke/delete events globally; use `sendToUser`.
- Recognition MVP is **deterministic target comparison** for a fixed five-hiragana set (`hiragana5`) — not unrestricted kanji OCR and not a loaded ML model.
- Free-board open-set ranking (if retained) is **heuristic match scores only** — never AI, ML confidence, calibrated confidence, or an ONNX/MNIST “upgrade” claim (`go test ./internal/docguard` must stay green).
- Practice **Assess** uses **multi-criterion deterministic match** (stroke count, order, start/end direction, relative placement, proportions, shape) after shared normalization — scores are match scores only; correctness claims are limited to fixture-tested criteria.
- Learner-facing assessment copy is **at most two** ranked corrections (`feedback[].code` + non-empty `message`); internal criterion diagnostics must not be used as user-facing language.
- Assessment **pass** = overall match ≥ engineering `T_pass`, no hard-fail gate (exact stroke-count mismatch when template stroke count ≤ 3), and target is the top ranked glyph in the MVP set; within-set `candidates` remain secondary diagnostics for debugging, not the primary coaching signal.
- **Attempt strokes ≠ board strokes** — practice attempts use separate tables; board clear/undo/erase must not mutate attempt history.
- Practice assessment reads **attempt** strokes only (never `ListStrokesWithRev` / board tables); free-board `POST /api/recognize` is heuristic playground and must not accept `target`.
- Retry always creates a **new** attempt row; never reopen `assessed` / `abandoned`. Assessed attempts are immutable — retry focuses on listed corrections.
- Draft attempts persist metadata only — stroke geometry is client-held until `SubmitStrokes`; attempt APIs must not require `boardRev`.
- Guided practice UI (`/#/practice`) draws attempt ink **locally** — do not enqueue practice strokes on WebSocket / `boardRev`; free-board remains the WS scratchpad.
- SQLite schema changes use **versioned, fail-closed migrations** (`schema_migrations`); do not reintroduce ad-hoc unversioned DDL on `Open`.
- Trusted curriculum is versioned under `content/hiragana5/vN` only; AI/WIP drafts stay in `content/hiragana5/drafts/` and must never be seeded or embedded for recognition.
- Seed and recognize must agree on the `hiragana5` glyph set, stroke counts, and pack `contentVersion` (single pack source of truth).
- Do **not** add SRS/classroom/cohort tables without a dedicated plan.
- Mutating WebSocket messages require a client `opId` (≤36) and `baseRev` (strict equality with per-user `boardRev`); creates are idempotent on active `(user_id, op_id)`; clear/tombstone yields `op_cancelled` on late creates.
- Prefer soft reject (`ack` nack / `error` frame) over disconnect for validation failures; log reject codes without stroke coordinates.
- Credentialed CORS and WebSocket upgrades allow **exact** origins only — no `*`.
- All `POST /api/*` require double-submit CSRF (`csrf` cookie + `X-CSRF-Token`).
- Passwords: bcrypt for new hashes; never log passwords, raw cookies, CSRF tokens, or full hashes.
- Production-secure mode: strong `COOKIE_KEY` (≥32, not sentinel) and explicit `ALLOWED_ORIGINS` or the process must refuse to start.
- Validate stroke/recognize inputs via `internal/limits` before persistence or rasterization.
- Frontend: unmatched inbound live stroke creates are non-authoritative; full reload uses `GET /api/strokes` (`boardRev` + strokes) and clears the in-memory WS queue; Vue clear/undo/erase mutate via WS (not REST-only clear).
