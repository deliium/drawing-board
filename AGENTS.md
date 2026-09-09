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
internal/features/    # FEATURE_* learning kill switches
internal/docguard/    # README honesty tests
web/src/              # Vue SPA
web/src/pages/        # BoardPage, PracticeHub/History/Character, LoginPage
web/src/components/   # AppShell (guest vs authed nav); DEV-only Dev metrics panel
web/src/components/practice/  # intro, stroke-order, canvas, overlay, journey chrome
web/src/canvas/       # CSS/DPR coords, layout helpers, draw, hit-test helpers
web/src/composables/  # usePracticeCanvas + usePracticeJourney + useLocale + useRomanizationPreference + useFeatureFlags
web/src/i18n/         # EN/JA catalogs + correction display map
web/src/curriculum/   # hiragana5 trace fixtures (geometry only)
web/src/services/     # apiFetch, attempts/curriculum/progress/features, wsClient
web/src/router/       # auth/guest guards; guestShell meta on login/register; feature-gated practice routes
web/public/fonts/     # self-hosted OFL font subsets
web/public/audio/hiragana5/  # mirrored mora clips (canonical under content pack)
web/tests/            # Vitest (+ axe a11y)
web/e2e/              # Playwright learner-journey smoke
scripts/              # e2e-webserver, verify-readme-commands, CI helpers
.github/workflows/    # PR CI + nightly
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
| `internal/httpapi/curriculum.go` | Lesson + progress read (+ mastery + review enrichment) |
| `internal/httpapi/progress_history.go` | Attempt history, schedule-aware progress next, practice-data clear |
| `internal/learn/mastery.go` | Explainable mastery derive + mastery-only next helper |
| `internal/learn/review.go` | Leitner-style boxes, ApplyReviewOutcome, SuggestNextWithReview |
| `internal/db/board.go` | boardRev transactional create/delete/clear |
| `internal/db/migrate.go` | versioned schema runner + `schema_migrations` |
| `internal/db/learn_store.go` | SQLite learning repos (attempts/assessments/progress + review upsert) |
| `internal/learn/` | learning-domain types + repository interfaces |
| `web/src/pages/BoardPage.vue` | Free-board canvas UI (tools/WS/recognize) |
| `web/src/pages/PracticeHubPage.vue` | Hiragana5 lesson hub + mastery / due hints / next / clear |
| `web/src/pages/PracticeHistoryPage.vue` | Paginated personal attempt history |
| `web/src/pages/PracticeCharacterPage.vue` | Guided single-character journey shell |
| `web/src/composables/usePracticeJourney.ts` | Stage machine, session resume, attempt orchestration |
| `web/src/composables/usePracticeCanvas.ts` | Pointer lifecycle, DPR resize redraw, Escape cancel |
| `web/src/composables/useLocale.ts` | EN/JA preference + reactive `t` |
| `web/src/composables/useRomanizationPreference.ts` | Hide/show Hepburn romanization (`romanizationVisible:v1`) |
| `web/src/components/practice/PronunciationAudioButton.vue` | Accessible on-demand mora playback + soft-fail |
| `web/src/canvas/*` | CSS-logical coords, layout helpers, stroke paint (incl. dots), hit-test |
| `web/src/i18n/*` | Lightweight EN/JA catalogs + correction display by code |
| `web/src/curriculum/*` | Trace template fixtures for animation/overlay |
| `web/src/services/wsClient.ts` | WS queue / reconnect / status / baseRev |
| `web/src/services/strokeSync.ts` | Merge ack/echo/clear into local strokes |
| `web/src/services/attemptsApi.ts` | Practice attempt REST + history list |
| `web/src/services/curriculumApi.ts` | Lesson pedagogy fetch |
| `web/src/services/progressApi.ts` | Progress list / next suggestion / clear practice data |
| `web/src/components/AppShell.vue` | Brand + guest/authed nav; locale + romanization (authed only) |
| `internal/metrics/metrics.go` | Process counters (incl. practice assess / feature-disabled / clear) |
| `internal/features/features.go` | `FEATURE_*` parse + startup cutover log |
| `Makefile` | Dev/build/docker/`validate-content`/`verify-docs` + quality gates (`check`, `test-race`, `check-web`, `test-e2e`, `security-check`, `gate-r1`…`gate-r4`) |
| `scripts/verify-readme-commands.sh` | Fail if README cites missing `make` / `./test.sh` commands |
| `scripts/e2e-webserver.sh` | Temp SQLite + static SPA for Playwright |
| `README.md` | Operator + API contract + CI / quality gates + contributing |

## Documentation

| Document | Path | Description |
|----------|------|-------------|
| README | `README.md` | Product, API, WS, security, troubleshooting |
| Plans | `.ai-factory/plans/` | Completed feature plans + release sequencing (`japanese-learning-release-sequencing.md`) |
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
