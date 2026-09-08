# AGENTS.md

> Structural map for AI agents. Keep factual; update when layout changes. Product detail lives in `.ai-factory/DESCRIPTION.md` and `README.md`.

## Project Overview

Personal Japanese handwriting practice app (Vue + Go + SQLite). Strokes are private per user; WebSocket is persist/echo, not collaboration.

## Tech Stack

- **Programming language:** Go 1.22+, TypeScript
- **Framework:** Gorilla mux/WebSocket; Vue 3 + Vite
- **Database:** SQLite
- **ORM:** none (raw SQL in `internal/db`)

## Project Structure

```text
cmd/server/           # backend entrypoint
cmd/contentvalidate/  # curriculum pack validator CLI
content/hiragana5/    # reviewed curriculum packs (vN) + drafts/
internal/auth/        # sessions, passwords
internal/curriculum/  # load/validate hiragana5 content packs
internal/db/          # SQLite store + versioned migrations + learning repos
internal/learn/       # learning-domain types + repository interfaces
internal/httpapi/     # REST API
internal/ws/          # WebSocket hub
internal/limits/      # shared validators
internal/recognize/   # hiragana5 target comparison + heuristic ranking
internal/security/    # CORS/CSRF/origins
internal/metrics/     # counters
internal/docguard/    # README honesty tests
web/src/              # Vue SPA
web/src/canvas/       # CSS/DPR coords, draw, hit-test helpers
web/src/composables/  # usePracticeCanvas lifecycle
web/src/curriculum/   # hiragana5 trace fixtures
web/tests/            # Vitest
.ai-factory/          # plans, patches, AI context
docker/               # compose / nginx helpers
```

## Key Entry Points

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Server wiring, env, listen |
| `cmd/contentvalidate/main.go` | Curriculum pack validate / hash CLI |
| `internal/curriculum/` | Load + validate published hiragana5 pack |
| `content/hiragana5/v1/` | Reviewed curriculum source of truth |
| `internal/ws/handler.go` | WS upgrade, ingest, ack/echo, boardRev mutates |
| `internal/httpapi/handlers.go` | REST strokes / recognize |
| `internal/db/board.go` | boardRev transactional create/delete/clear |
| `internal/db/migrate.go` | versioned schema runner + `schema_migrations` |
| `internal/db/learn_store.go` | SQLite learning repos (attempts/assessments/progress) |
| `internal/learn/` | learning-domain types + repository interfaces |
| `web/src/pages/BoardPage.vue` | Practice canvas UI (tools/WS/recognize) |
| `web/src/composables/usePracticeCanvas.ts` | Pointer lifecycle, DPR resize redraw, Escape cancel |
| `web/src/canvas/*` | CSS-logical coords, stroke paint (incl. dots), hit-test |
| `web/src/curriculum/*` | Trace template fixtures for Vitest / future lesson UI |
| `web/src/services/wsClient.ts` | WS queue / reconnect / status / baseRev |
| `web/src/services/strokeSync.ts` | Merge ack/echo/clear into local strokes |
| `Makefile` | Dev/build/docker/`validate-content` targets |
| `README.md` | Operator + API contract |

## Documentation

| Document | Path | Description |
|----------|------|-------------|
| README | `README.md` | Product, API, WS, security, troubleshooting |
| Plans | `.ai-factory/plans/` | Completed feature plans |
| Patches | `.ai-factory/patches/` | Self-improvement notes |

## AI Context Files

| File | Purpose |
|------|---------|
| `AGENTS.md` | This map |
| `.ai-factory/DESCRIPTION.md` | Stack and product summary |
| `.ai-factory/ARCHITECTURE.md` | Module boundaries and dependency rules |
| `.ai-factory/RULES.md` | Hard project axioms |
| `.ai-factory/rules/base.md` | Day-to-day conventions |
| `.ai-factory/ROADMAP.md` | Milestone checklist |
| `.ai-factory/config.yaml` | AI Factory paths/language/git |

## Agent Rules

- Prefer small, focused diffs; do not expand scope beyond the asked task.
- Decompose shell commands — avoid `cd x && y` when sequential tool calls work.
- Example incorrect: `git checkout main && git pull`
- Example correct: first `git checkout main`, then `git pull origin main`
- Follow `.ai-factory/RULES.md` and architecture dependency rules when implementing.
