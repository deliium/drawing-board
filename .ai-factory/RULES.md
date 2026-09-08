# Project Rules (Axioms)

Hard requirements for this repository. More specific files under `.ai-factory/rules/` override when they conflict.

- Strokes are **private per user** — never broadcast WebSocket stroke/delete events globally; use `sendToUser`.
- Recognition is **heuristic** — do not describe it as AI, ML confidence, or calibrated confidence in code comments, UI, or README (`go test ./internal/docguard` must stay green).
- Mutating WebSocket messages require a client `opId` (≤36) and `baseRev` (strict equality with per-user `boardRev`); creates are idempotent on active `(user_id, op_id)`; clear/tombstone yields `op_cancelled` on late creates.
- Prefer soft reject (`ack` nack / `error` frame) over disconnect for validation failures; log reject codes without stroke coordinates.
- Credentialed CORS and WebSocket upgrades allow **exact** origins only — no `*`.
- All `POST /api/*` require double-submit CSRF (`csrf` cookie + `X-CSRF-Token`).
- Passwords: bcrypt for new hashes; never log passwords, raw cookies, CSRF tokens, or full hashes.
- Production-secure mode: strong `COOKIE_KEY` (≥32, not sentinel) and explicit `ALLOWED_ORIGINS` or the process must refuse to start.
- Validate stroke/recognize inputs via `internal/limits` before persistence or rasterization.
- Frontend: unmatched inbound live stroke creates are non-authoritative; full reload uses `GET /api/strokes` (`boardRev` + strokes) and clears the in-memory WS queue; Vue clear/undo/erase mutate via WS (not REST-only clear).
