# Project Roadmap

> Personal Japanese handwriting practice: private canvas, durable stroke sync, honest recognition feedback, production-safe auth/perimeter.

## Milestones

- [x] **Vue frontend migration** — Vite/Vue 3 practice board replaces prior UI
- [x] **Per-user stroke isolation** — WS `sendToUser` + frontend ignore foreign live creates; Japanese training rebrand
- [x] **Public onboarding** — login/register routes, anti-enumeration auth API, board logout-only
- [x] **Password and session security** — bcrypt, session rotation, production-secure cookies / `COOKIE_KEY`
- [x] **HTTP/WebSocket perimeter** — origin allowlist, CSRF, compose/Nginx defaults
- [x] **Recognition and stroke input hardening** — shared limits, rate limits, safe logging/metrics
- [x] **Reliable stroke WS persistence** — `opId` + ack + idempotent SQLite + client queue/reconnect/status UX
- [x] **Authoritative board operations (revision-consistent create/undo/erase/clear/recognize)** — per-user `boardRev` / `baseRev`, WS clear, recognize gated to Saved
- [x] **Vue canvas lifecycle modularization** — composable-owned pointer attach/detach, resize redraw, one-point taps/dots, DPR backing store with CSS-logical stroke coords
- [x] **Honest recognition strategy for five-character hiragana MVP (Prompt 08)** — deterministic target comparison for five hiragana; remove fake ONNX/MNIST upgrade path; heuristic free-board ranking stays labeled as match scores (not ML confidence)
- [ ] **Durable offline stroke vault** — IndexedDB/service-worker queue surviving full reload (explicitly out of scope for reliable-WS v1)
- [ ] **Cross-tab live create sync** — optional same-account multi-tab create fan-in (deletes/clear already echo)

## Completed

| Milestone | Date |
|-----------|------|
| Vue frontend migration | 2026-03 (approx; PR merge) |
| Per-user stroke isolation | 2026-09-07 |
| Public onboarding | 2026-09-07 |
| Password and session security | 2026-09-08 |
| HTTP/WebSocket perimeter | 2026-09-08 |
| Recognition and stroke input hardening | 2026-09-08 |
| Reliable stroke WS persistence | 2026-09-08 |
| Authoritative board operations (revision-consistent create/undo/erase/clear/recognize) | 2026-09-08 |
| Vue canvas lifecycle modularization | 2026-09-08 |
| Honest recognition strategy for five-character hiragana MVP (Prompt 08) | 2026-09-08 |
