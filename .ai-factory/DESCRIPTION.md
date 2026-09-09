# Project Description

## Overview

Personal Japanese handwriting practice app: authenticated users draw on a private canvas, persist strokes, and get heuristic recognition feedback. Strokes are never shared across users (not a collaborative board).

## Tech Stack

| Layer | Choice |
|-------|--------|
| Backend | Go 1.22+, Gorilla mux, Gorilla WebSocket |
| Frontend | Vue 3, TypeScript, Vite |
| Database | SQLite (`mattn/go-sqlite3`), per-user stroke store |
| Auth | Cookie sessions (`sid`), bcrypt passwords (legacy SHA-256 upgrade path) |
| Deploy | Docker Compose + Nginx reverse proxy (`/api`, `/ws`) |

## Architecture

See `.ai-factory/ARCHITECTURE.md` — Go `cmd/` + `internal/` packages with a Vue SPA. Structured as a modular monolith adapted to the existing layout.

## Core Features

- Public login/register; private board after session
- Pencil / eraser / undo / clear on canvas
- WebSocket stroke persist + echo with `opId` ack, `boardRev`/`baseRev` ordering, and idempotent SQLite creates
- REST list strokes (`boardRev` envelope); recognize revision-gated; clear available via WS (primary) or REST helper
- Recognition: multi-criterion deterministic target assessment for five hiragana (`hiragana5`) with ≤2 actionable corrections; optional free-board heuristic ranking with match scores (not ML/ONNX)
- Durable learning schema (characters/lessons/attempts/assessments/progress) via versioned SQLite migrations, separate from free-board strokes
- Reviewed five-vowel starter curriculum pack (`content/hiragana5/`) with pedagogy fields, guidance, reviewed mora audio, stroke/trace templates, and deterministic seed (kanji extension points reserved empty)
- Attempt-scoped practice assessment REST (`/api/attempts`) — submit strokes for one character, assess via target comparison + correction catalog, persist engine feedback without reading the free-board
- Guided learner journey UI (`/#/practice`) for hiragana5 — intro with pronunciation play + guidance, stroke-order animation, trace/free-write, comparison overlay with ≤2 corrections; curriculum/progress read APIs
- Attempt history (`/#/practice/history`), explainable per-character mastery, Leitner-style personal review schedule (pass/fail → box/`due_at`), schedule-aware next suggestion, and self-serve clear of practice data (not free-board)
- Responsive bilingual (EN/JA) learner chrome with hideable romanization, fluid shared canvas sizing, focus-visible / live regions, and axe-covered critical surfaces
- Perimeter: origin allowlist, CSRF on mutating `/api/*`, production-secure cookies
- Staged learning feature flags (`FEATURE_PRACTICE` / `PROGRESS` / `REVIEW` / `AUDIO`) for independently releasable R1–R4 cutovers

## Non-Functional Requirements

- **Privacy:** per-user WS delivery (`sendToUser`); no cross-user stroke visibility
- **Security:** bcrypt, session rotation, CSRF, CORS/WS origin allowlist, input bounds + rate limits
- **Honesty:** MVP is multi-criterion match vs pack templates; never market scores as calibrated AI confidence or claim an ONNX upgrade; correctness limited to fixture-tested criteria; keep `internal/docguard` green
- **Reliability:** bounded in-memory WS queue (32), reconnect/backoff; reload uses REST as source of truth (no durable offline vault yet)
- **CI / quality:** GitHub Actions PR jobs (`Go quality`, `Go race`, `Web quality`, `Content validate`, `Security light`) with Makefile mirrors (`make check`, `make test-race`, …); coverage is a signal artifact, not a vanity gate; Playwright learner journeys are non-required until promoted

## Constraints

- Personal practice product — do not reintroduce collaborative/shared-board semantics
- Production requires strong `COOKIE_KEY` and explicit `ALLOWED_ORIGINS` (no `*`)
- In-process rate limits are per replica only
