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
- WebSocket stroke persist + echo with `opId` ack and idempotent SQLite creates
- REST list/clear strokes; recognize via `POST /api/recognize`
- Heuristic pattern recognizer (default); optional `ONNX_MODEL` path currently falls back to heuristic
- Perimeter: origin allowlist, CSRF on `POST /api/*`, production-secure cookies

## Non-Functional Requirements

- **Privacy:** per-user WS delivery (`sendToUser`); no cross-user stroke visibility
- **Security:** bcrypt, session rotation, CSRF, CORS/WS origin allowlist, input bounds + rate limits
- **Honesty:** never market recognition as calibrated AI confidence; keep `internal/docguard` green
- **Reliability:** bounded in-memory WS queue (32), reconnect/backoff; reload uses REST as source of truth (no durable offline vault yet)

## Constraints

- Personal practice product — do not reintroduce collaborative/shared-board semantics
- Production requires strong `COOKIE_KEY` and explicit `ALLOWED_ORIGINS` (no `*`)
- In-process rate limits are per replica only
