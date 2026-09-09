# Project Roadmap

> Personal Japanese handwriting practice: private canvas, durable stroke sync, honest recognition feedback, production-safe auth/perimeter.

## Milestones

- [x] **Per-user stroke isolation** — WS `sendToUser` + frontend ignore foreign live creates; Japanese training rebrand
- [x] **Public onboarding** — login/register routes, anti-enumeration auth API, board logout-only
- [x] **Password and session security** — bcrypt, session rotation, production-secure cookies / `COOKIE_KEY`
- [x] **HTTP/WebSocket perimeter** — origin allowlist, CSRF, compose/Nginx defaults
- [x] **Recognition and stroke input hardening** — shared limits, rate limits, safe logging/metrics
- [x] **Reliable stroke WS persistence** — `opId` + ack + idempotent SQLite + client queue/reconnect/status UX
- [x] **Authoritative board operations (revision-consistent create/undo/erase/clear/recognize)** — per-user `boardRev` / `baseRev`, WS clear, recognize gated to Saved
- [x] **Vue canvas lifecycle modularization** — composable-owned pointer attach/detach, resize redraw, one-point taps/dots, DPR backing store with CSS-logical stroke coords
- [x] **Honest recognition strategy for five-character hiragana MVP (Prompt 08)** — deterministic target comparison for five hiragana; remove fake ONNX/MNIST upgrade path; heuristic free-board ranking stays labeled as match scores (not ML confidence)
- [x] **Learning domain model and versioned migrations (Prompt 09)** — versioned SQLite migrations; characters/lessons/attempts/assessments/progress separate from free-board strokes; idempotent hiragana5 seed
- [x] **Reviewed five-character hiragana starter curriculum (Prompt 10)** — versioned `hiragana5` content pack (あ行); pedagogy fields, stroke/trace data, human review gate, deterministic seed; AI drafts excluded until reviewed
- [x] **Practice attempt APIs replacing whole-board recognition (Prompt 11)** — start/submit/assess/retry attempt REST; assessment uses submitted attempt strokes only (not board store); idempotency + immutable assessed rows; coexist with WS/`boardRev`
- [x] **Deterministic target-specific handwriting assessment for hiragana5 MVP (Prompt 12)** — normalize learner/canonical strokes; multi-criterion match scoring; ≤2 actionable corrections; fixtures + expert review; diagnostics separate from learner copy; match scores only (not calibrated confidence)
- [x] **Core learner journey UI for one hiragana character (Prompt 13)** — guided intro (glyph/pronunciation/stroke count/example); stroke-order animation; trace + free-write; attempt submit/assess; comparison overlay + ≤2 corrections; retry or completion; progress resume; age-neutral Vue practice routes
- [x] **Responsive bilingual accessible learner UI (Prompt 14)** — mobile-first practice-notebook visual system; fluid canvases; EN/JA preference + externalized strings; focus/labels/live regions/reduced motion; axe + keyboard/zoom/SR/device smoke criteria; non-canvas textual attempt summary
- [x] **Attempt history and per-character mastery (Prompt 15)** — paginated personal attempt history; explainable mastery from assessed attempts; humble next-character suggestion; privacy clear / retention docs; no streaks, social comparison, or SRS
- [x] **Lightweight spaced-review queue (Prompt 16)** — Leitner-style personal review from assessed pass/fail; due dates + next/why; forgiving missed days; no streaks, notifications, or SM-2/FSRS
- [x] **Handwriting practice with basic language learning (Prompt 17)** — per-character pronunciation audio, example vocabulary, hideable romanization, concise learner guidance; kanji-ready content extension points without implementing kanji
- [x] **Test pyramid and CI confidence (Prompt 18)** — balanced Go unit/integration + SQLite migrations + HTTP/WS contracts + Vitest/axe/canvas + small Playwright learner journeys; WS race detection; coverage as signal; lint/type/build; GitHub Actions with caching/artifacts/security scanning and required PR checks; local Makefile equivalents
- [x] **Product copy, shell, and documentation honesty (Prompt 19)** — gate/remove learner-visible migration diagnostics; consolidate app shell/nav/branding; accurate keyboard/feature/recognition copy; verified setup/env/Makefile/Docker docs; privacy retention + deletion/export honesty; troubleshooting + architecture notes + concise contributor workflow; command verification + docguard/tests
- [x] **R1: Trustworthy guided lesson for five hiragana** — release cutover sequencing Prompts 08–13 (+ perimeter baseline); schema ≤0004; attempt-scoped guided journey for あいうえお; feature-flag / migration backup-restore / observability / E2E gates (see `japanese-learning-release-sequencing.md`)
- [ ] **R2: Product-ready bilingual accessible learner shell** — independently releasable cutover of Prompt 14 (+ shell honesty); no review/audio requirement; a11y + EN/JA gates
- [ ] **R3: Personal progress, mastery, and spaced review** — independently releasable cutover of Prompts 15–16; schema ≤0005; history/mastery/clear + Leitner due/next; privacy gates
- [ ] **R4: Language-learning chrome, CI confidence, and docs honesty** — independently releasable cutover of Prompts 17–19; schema ≤0006; audio/guidance + required CI/docs honesty gates

## Completed

| Milestone | Date |
|-----------|------|
| Per-user stroke isolation | 2026-09-07 |
| Public onboarding | 2026-09-07 |
| Password and session security | 2026-09-08 |
| HTTP/WebSocket perimeter | 2026-09-08 |
| Recognition and stroke input hardening | 2026-09-08 |
| Reliable stroke WS persistence | 2026-09-08 |
| Authoritative board operations (revision-consistent create/undo/erase/clear/recognize) | 2026-09-08 |
| Vue canvas lifecycle modularization | 2026-09-08 |
| Honest recognition strategy for five-character hiragana MVP (Prompt 08) | 2026-09-08 |
| Learning domain model and versioned migrations (Prompt 09) | 2026-09-08 |
| Reviewed five-character hiragana starter curriculum (Prompt 10) | 2026-09-08 |
| Practice attempt APIs replacing whole-board recognition (Prompt 11) | 2026-09-09 |
| Deterministic target-specific handwriting assessment for hiragana5 MVP (Prompt 12) | 2026-09-09 |
| Core learner journey UI for one hiragana character (Prompt 13) | 2026-09-09 |
| Responsive bilingual accessible learner UI (Prompt 14) | 2026-09-09 |
| Attempt history and per-character mastery (Prompt 15) | 2026-09-09 |
| Lightweight spaced-review queue (Prompt 16) | 2026-09-09 |
| Handwriting practice with basic language learning (Prompt 17) | 2026-09-09 |
| Test pyramid and CI confidence (Prompt 18) | 2026-09-09 |
| Product copy, shell, and documentation honesty (Prompt 19) | 2026-09-09 |
| R1: Trustworthy guided lesson for five hiragana | 2026-09-10 |
