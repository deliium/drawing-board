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
- Recognition: deterministic target comparison for five hiragana (`hiragana5`); optional free-board heuristic ranking with match scores (not ML/ONNX)
- Durable learning schema (characters/lessons/attempts/assessments/progress) via versioned SQLite migrations, separate from free-board strokes
- Perimeter: origin allowlist, CSRF on `POST /api/*`, production-secure cookies

## Non-Functional Requirements

- **Privacy:** per-user WS delivery (`sendToUser`); no cross-user stroke visibility
- **Security:** bcrypt, session rotation, CSRF, CORS/WS origin allowlist, input bounds + rate limits
- **Honesty:** MVP is target comparison for five hiragana; never market scores as calibrated AI confidence or claim an ONNX upgrade; keep `internal/docguard` green
- **Reliability:** bounded in-memory WS queue (32), reconnect/backoff; reload uses REST as source of truth (no durable offline vault yet)

## Constraints

- Personal practice product — do not reintroduce collaborative/shared-board semantics
- Production requires strong `COOKIE_KEY` and explicit `ALLOWED_ORIGINS` (no `*`)
- In-process rate limits are per replica only
