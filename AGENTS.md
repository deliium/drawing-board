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
internal/recognize/   # hiragana5 multi-criterion assess + heuristic ranking + correction catalog
internal/security/    # CORS/CSRF/origins
internal/metrics/     # counters
internal/docguard/    # README honesty tests
web/src/              # Vue SPA
web/src/pages/        # BoardPage, PracticeHub/Character, LoginPage
web/src/components/practice/  # intro, stroke-order, canvas, overlay, journey chrome
web/src/canvas/       # CSS/DPR coords, layout helpers, draw, hit-test helpers
web/src/composables/  # usePracticeCanvas + usePracticeJourney + useLocale
web/src/i18n/         # EN/JA catalogs + correction display map
web/src/curriculum/   # hiragana5 trace fixtures (geometry only)
web/src/services/     # apiFetch, attempts/curriculum/progress clients, wsClient
web/public/fonts/     # self-hosted OFL font subsets
web/tests/            # Vitest (+ axe a11y)
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
| `internal/httpapi/handlers.go` | REST strokes / free-board recognize |
| `internal/recognize/target_compare.go` | Multi-criterion hiragana5 assess + heuristic free-board rank |
| `internal/recognize/normalize.go` | Shared unit-space normalization + short-stroke classification |
| `internal/recognize/criteria.go` / `corrections.go` | Criterion scorers + ≤2 learner correction catalog |
| `internal/httpapi/attempts.go` | Practice attempt REST lifecycle |
| `internal/httpapi/curriculum.go` | Lesson + progress read endpoints |
| `internal/db/board.go` | boardRev transactional create/delete/clear |
| `internal/db/migrate.go` | versioned schema runner + `schema_migrations` |
| `internal/db/learn_store.go` | SQLite learning repos (attempts/assessments/progress) |
| `internal/learn/` | learning-domain types + repository interfaces |
| `web/src/pages/BoardPage.vue` | Free-board canvas UI (tools/WS/recognize) |
| `web/src/pages/PracticeHubPage.vue` | Hiragana5 lesson hub + progress |
| `web/src/pages/PracticeCharacterPage.vue` | Guided single-character journey shell |
| `web/src/composables/usePracticeJourney.ts` | Stage machine, session resume, attempt orchestration |
| `web/src/composables/usePracticeCanvas.ts` | Pointer lifecycle, DPR resize redraw, Escape cancel |
| `web/src/composables/useLocale.ts` | EN/JA preference + reactive `t` |
| `web/src/canvas/*` | CSS-logical coords, layout helpers, stroke paint (incl. dots), hit-test |
| `web/src/i18n/*` | Lightweight EN/JA catalogs + correction display by code |
| `web/src/curriculum/*` | Trace template fixtures for animation/overlay |
| `web/src/services/wsClient.ts` | WS queue / reconnect / status / baseRev |
| `web/src/services/strokeSync.ts` | Merge ack/echo/clear into local strokes |
| `web/src/services/attemptsApi.ts` | Thin typed client for practice attempt REST |
| `web/src/services/curriculumApi.ts` | Lesson pedagogy fetch |
| `web/src/services/progressApi.ts` | Progress list fetch |
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
