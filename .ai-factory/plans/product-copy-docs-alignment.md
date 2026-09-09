# Implementation Plan: Product Copy, Shell, and Documentation Honesty (Prompt 19)

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Product copy, shell, and documentation honesty (Prompt 19)"
Rationale: Prompts 08–18 shipped the real hiragana5 learner product and CI pyramid, but learner-facing chrome still exposes a Vue-migration diagnostics panel, README overstates keyboard scope and Docker DX, and operational docs drift from Makefile/env reality — this milestone makes UI, config, and documentation match what the app actually does, with command verification and honesty gates.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep Prompt 08–18 entries unchanged (do not uncheck completed items).

## Summary

The product is a **personal Japanese handwriting practice** app (guided hiragana5 journey, private free-board, deterministic match scoring). Shipping docs already reject ML/ONNX marketing (`internal/docguard`), but several **learner-visible and operator-facing** surfaces still describe a earlier migration / ONNX / board-only era:

| Surface | Problem |
|---------|---------|
| `MigrationHealthPanel` | Always mounted in `App.vue` — “Migration Health” metrics visible on every route including login/practice |
| App shell | Brand + full Practice/History/Board nav on auth pages; page-local links re-state shell destinations; History label inconsistent (“History” vs “Attempt history”) |
| Keyboard docs | README lists Ctrl+Z as global; only free-board `BoardPage` wires it; Escape is correctly shared via `usePracticeCanvas` |
| Docker / Make | “hot reload” oversells `docker-compose.dev.yml`; `.PHONY` still lists `dev` / `zinnia-*` with no recipes; Dockerfile `models/` leftover |
| Privacy docs | Accurate on `DELETE /api/practice-data` and no export; “account is deleted” implies a product path that does not exist (no self-serve account delete) |
| Historical plans | Pre-Prompt-08 “Current State” still narrates ONNX as live (docguard does not scan `.ai-factory/plans/`) |

This plan **gates or removes** internal diagnostics, **consolidates** shell/nav/copy, **rewrites** README/env/setup/privacy/troubleshooting/contributor sections to verified commands, and **extends** tests + docguard so regressions fail CI.

**Out of scope:** New curriculum characters; ONNX/ML; SM-2/FSRS; collaboration; implementing account-delete or data-export APIs (document honestly as unsupported unless a tiny follow-up is explicitly pulled in); redesigning the practice-notebook visual system; rewriting completed Prompt 08–18 plans’ task checkboxes; raising coverage vanity gates.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Diagnostics UI | `web/src/components/MigrationHealthPanel.vue` + `migrationHealth.ts`; mounted unconditionally in `App.vue`; writers only on `BoardPage` (`ws.message`, `recognize.*`); **no** `import.meta.env.DEV` gate |
| Shell | `AppShell.vue` — brand, 3-link nav, locale, romanization; used for all routes; auth pages still show practice nav |
| Branding | `index.html` title + `brand.name` + Login `h1`/lede; static `document.title` (no per-route titles) |
| Keyboard runtime | Ctrl/Cmd+Z → `BoardPage` only; Escape → `usePracticeCanvas` (board + practice); practice undo is button-only |
| README honesty | Target comparison / match scores / deprecated `ONNX_MODEL` WARN — largely correct; keyboard + Docker hot-reload + account-delete implication are the main learner/operator drifts |
| Env (server) | `ADDR`, `STATIC_DIR`, `DB_PATH`, `COOKIE_KEY`, `APP_ENV`, `COOKIE_SECURE`, `ALLOWED_ORIGINS`, `RECOGNIZE_DEBUG`, deprecated `ONNX_MODEL`; plus `LOG_LEVEL` in packages |
| Privacy APIs | `DELETE /api/practice-data` (practice only); free-board clear via WS/REST; **no** export; **no** `DELETE /api/account` |
| Docguard | `internal/docguard` scans **README.md only**; forbids AI/ONNX marketing phrases; run via `./test.sh recognize-fixtures` |
| Make | Working quality targets exist; phantom `.PHONY`: `dev`, `zinnia-build`, `zinnia-model`; `make sync-audio` real but under-documented in Make Commands block |
| Contributor | README quality gates exist; **no** concise `CONTRIBUTING.md` (or equivalent short section) for clone → `make check` → PR checks |

## Design Decisions

### D0 — Product goal

Make every **learner-visible** string and every **operator/contributor** doc describe the implemented hiragana5 experience: private handwriting, guided practice, honest match scores, verified commands — with no developer migration chrome and no unsupported capability claims.

### D1 — Diagnostics: remove from learner builds (prefer gate, allow delete)

| Decision | Detail |
|----------|--------|
| Default | **Development-only gate** (`import.meta.env.DEV`) for `MigrationHealthPanel`, **or** delete panel + service if implement finds zero remaining value |
| Production | Panel must not appear in `npm run build` / Playwright learner journeys |
| Telemetry | Keep `trackMetric` calls optional behind DEV, or remove with panel; do not invent a new metrics backend |
| Naming | If kept for DEV, rename copy away from “Migration Health” to something like “Dev metrics” so it cannot be mistaken for product UI |
| Logging | DEV `console.debug('[MigrationHealth] …')` only when gated panel mounts/updates; never INFO spam in prod |

### D2 — Shell and navigation consolidation

| Decision | Detail |
|----------|--------|
| Auth routes | Minimal chrome: brand + locale (and maybe skip link); **hide** Practice/History/Board nav and romanization toggle on `/login` / `/register` (guest layout) |
| Authenticated shell | Single primary nav (Practice / History / Board); drop redundant page-local “Attempt history” / “Free board” / “Back to lesson” **duplicates** where shell already covers them — keep one contextual back affordance on character journey only if needed for wayfinding |
| Labels | Unify History naming in i18n (`nav.history` ↔ hub/history titles) — pick one learner phrase (“History” or “Attempt history”) and use consistently |
| Branding | Brand remains shell-level hero signal; Login `h1` should not restate the full product name if shell brand is visible — prefer stage/action headline (“Sign in” / “Create account”) under brand |
| Titles | Optional cheap win: per-route `document.title` = `"<page> · <brand>"` via router meta — only if low-cost; do not block the milestone |
| Logging | DEV `[AppShell] layout=guest|authed` on route change; no PII |

### D3 — Feature and recognition copy (UI + README)

| Decision | Detail |
|----------|--------|
| Recognition | Keep honest language already in README Features / Handwriting Recognition; sweep UI i18n for any “confidence”, “AI”, “ONNX”, “migration” leftovers |
| Keyboard | Document Ctrl/Cmd+Z as **free board only**; Escape as cancel in-progress stroke on board **and** practice canvases; do **not** claim practice Ctrl+Z unless Task wires it (prefer docs fix over new shortcut unless trivial) |
| Auto-restore | Clarify free-board restore vs practice resume (`sessionStorage` + attempt status) — README “practice strokes” line must not blur attempt vs board |
| Logging | N/A for static copy; docguard + Vitest assert key strings |

### D4 — Setup, env, Docker, Makefile honesty

| Decision | Detail |
|----------|--------|
| Env table | Single README table: every **runtime** var (`cmd/server` + `LOG_LEVEL`) with purpose, production requirements, and deprecated `ONNX_MODEL` (warn+ignore — **no** `ONNX_MODEL=` assignment that trips docguard) |
| Docker | Replace “hot reload” with accurate behavior of `docker-compose.dev.yml`; recommend host `make backend` + `make frontend` for real DX |
| Makefile | Remove phantom `.PHONY` entries **or** implement tiny aliases; document `sync-audio`; keep quality targets accurate |
| Dockerfile | Drop leftover `models/` mkdir if unused |
| `.gitignore` | Optionally prune dead `models/*.onnx` entries in same cleanup (cosmetic) |
| Verify | Automated check: every `make <target>` cited in README exists as a real recipe (see D8) |
| Logging | Scripts/Makefile already use `[check]` markers; preserve |

### D5 — Privacy, retention, deletion, export

| Decision | Detail |
|----------|--------|
| Explain | Dedicated README subsection: what is stored (free-board strokes; practice attempts/assessments/progress/review schedule); isolation (`sendToUser`); no cross-user visibility |
| Deletion | Document hub UI + `DELETE /api/practice-data` scope; free-board clear (WS primary / REST helper); **explicitly** state self-serve account deletion and data export are **not** supported |
| Retention | keep-until-clear (practice) / clear-on-board-clear (free-board); no timed auto-purge |
| RULES | Align “account is deleted” wording with operator/DB cascade reality, not a missing API |
| Logging | Existing `[httpapi.PracticeData.Clear]` counts only; never log stroke points |

### D6 — Troubleshooting, architecture notes, contributor workflow

| Decision | Detail |
|----------|--------|
| Troubleshooting | Refresh WS sync / Saved gate / CSRF / origin / recognize reject codes / missing audio soft-fail — verify each step against code; remove dead ONNX troubleshooting if any remains |
| Architecture | Small README + `ARCHITECTURE.md` / `AGENTS.md` touch-ups only where inventory or shell layout changes (MigrationHealth removal, guest shell) |
| Contributor | Concise workflow: clone → Node/Go versions → `make frontend`/`make backend` → `make check` → optional `make test-e2e` → PR required checks list — either short `CONTRIBUTING.md` **or** a README “Contributing” section (prefer one place; avoid duplicating full CI matrix) |
| Historical plans | Do **not** rewrite Prompt 08–18 bodies; optional one-line “superseded” note only if implement touches a plan for other reasons — not required |

### D7 — Tests and honesty gates

| Layer | What to add |
|-------|-------------|
| Vitest | Panel absent when `DEV=false` (or component deleted); guest shell hides practice nav on login; history label consistency smoke; axe still green on login/hub |
| Playwright | Assert “Migration Health” text **absent** on practice journey; optional assert shell nav counts |
| Docguard | Keep README AI/ONNX bans; extend only if new dishonest phrases appear (e.g. “hot reload” is **not** a docguard concern — command verification covers it) |
| Command verify | Small Go test or `scripts/verify-readme-make.sh` + CI/`make check` step: parse README fenced `make …` tokens and assert Makefile recipes exist |
| Go | No product logic change expected; run existing suites green |

### D8 — Command verification policy

**Rule:** Do not document a command unless it is runnable in this repo (or clearly marked external, e.g. `git clone`).

| Check | Mechanism |
|-------|-----------|
| `make <target>` in README | Script/test fails if target missing from Makefile |
| `./test.sh <cmd>` | Script/test fails if unknown `test.sh` subcommand |
| npm scripts cited | Optional: verify against `web/package.json` |
| Evidence bar | No latency/accuracy/coverage % claims without measured CI/fixture evidence; no “hot reload” without observing reload behavior |

### D9 — Logging (verbose)

| Area | Guidance |
|------|----------|
| UI gate | DEV-only `console.debug` with `[AppShell]`, `[MigrationHealth]` prefixes |
| Docs scripts | `echo "[docs.verify] start|ok|fail …"` |
| Server | No new production log noise; keep deprecated `ONNX_MODEL` single WARN |

## Acceptance Criteria

1. Learner-facing builds do **not** show Migration Health / internal diagnostics (DEV-only or removed); Vitest + Playwright assert absence in prod-like mode.
2. App shell uses a coherent guest vs authenticated layout; nav labels consistent; redundant page links reduced without breaking wayfinding on character journey.
3. README keyboard, recognition, restore, Docker DX, env vars, privacy/deletion/export, troubleshooting, and Make Commands match runtime — **verified** by automated command check + manual scenario pass.
4. Phantom Makefile targets cleaned; `sync-audio` documented; no ONNX setup instructions as active capability.
5. Privacy section explains stored handwriting, practice clear, free-board clear, and explicitly states no export / no self-serve account delete.
6. Concise contributor workflow documented; local `make check` (and docguard/fixtures) remain green.
7. `go test ./internal/docguard` green; no new dishonest AI/ONNX/confidence marketing in README.
8. `AGENTS.md` / `ARCHITECTURE.md` updated only as needed for shell/diagnostics inventory truth.
9. Prompt 19 left **unchecked** until implement + verify finish.

## Scenario Checklist

### S1 — Learner never sees diagnostics

1. Production build or Playwright journey: page text has no “Migration Health”.
2. DEV server: gated panel may appear; production bundle does not.

### S2 — Auth chrome

1. Open `/#/login` → brand present; Practice/History/Board nav hidden (or clearly guest-safe).
2. After login → full nav; practice hub usable.

### S3 — Keyboard honesty

1. Free board: Ctrl/Cmd+Z undoes; Escape cancels in-progress stroke.
2. Practice free-write: Escape cancels; Ctrl+Z either works (if wired) or is **not** documented for practice.

### S4 — Privacy clear

1. Hub clear practice data → history empty / mastery reset; free-board strokes remain.
2. README states no export and no self-serve account delete.

### S5 — Command verify

1. Introduce a fake `make not-a-real-target` in README → verify script/test fails.
2. `make check` (or dedicated target) runs verify.

### S6 — Recognition honesty

1. README + UI still say Match / match scores; docguard fails if AI/ONNX marketing returns.

## Tasks

### Phase 1: Inventory lock + diagnostics gate

- [x] Task 1: Freeze inventory notes into implement checklist — confirm MigrationHealth mount points, AppShell guest/auth needs, README sections to rewrite (keyboard, Docker, env, privacy, Make, troubleshooting), phantom Make targets, Dockerfile `models/`. Capture findings in plan Notes or a short comment block in the PR description (no new long markdown file unless needed).

  LOGGING: N/A; optional DEV note in PR.

  Files: (read-only) `App.vue`, `MigrationHealthPanel.vue`, `AppShell.vue`, `README.md`, `Makefile`, `Dockerfile.backend`, `docker-compose.dev.yml`

  Depends on: —

- [x] Task 2: Gate or remove `MigrationHealthPanel` — prefer `import.meta.env.DEV` mount in `App.vue` (and rename DEV heading); or delete component + `migrationHealth.ts` + `BoardPage` `trackMetric` calls. Ensure production bundle has no learner-visible diagnostics.

  LOGGING: if gated, `console.debug('[MigrationHealth] mount')` only in DEV; remove dead imports cleanly.

  Files: `web/src/App.vue`, `web/src/components/MigrationHealthPanel.vue`, `web/src/services/migrationHealth.ts`, `web/src/pages/BoardPage.vue`

  Depends on: 1

- [x] Task 3: Vitest for diagnostics absence — assert panel not rendered when DEV false / component gone; axe login+hub still pass.

  LOGGING: test names only; no stroke dumps.

  Files: `web/tests/unit/*` or `web/tests/integration/*`

  Depends on: 2

<!-- Commit checkpoint: tasks 1–3 -->

### Phase 2: Shell, nav, i18n consolidation

- [x] Task 4: Guest vs authenticated shell — hide practice nav (and romanization if irrelevant) on guest routes; keep skip link + brand + locale; authenticated routes keep full nav. Prefer router meta (`guestShell: true`) over brittle path string lists.

  LOGGING: DEV `console.debug('[AppShell] layout=', …)`.

  Files: `web/src/components/AppShell.vue`, `web/src/router/index.ts`, possibly `web/src/App.vue`

  Depends on: 1

- [x] Task 5: Deduplicate branding and page links — adjust Login headlines so brand is not triple-stated; unify History i18n labels; remove redundant hub/history/board cross-links that duplicate shell nav while keeping character-journey back-to-lesson.

  LOGGING: N/A for copy; DEV locale toggle logs already exist.

  Files: `web/src/i18n/locales/en.ts`, `web/src/i18n/locales/ja.ts`, `web/src/pages/LoginPage.vue`, `PracticeHubPage.vue`, `PracticeHistoryPage.vue`, `PracticeCharacterPage.vue`

  Depends on: 4

- [x] Task 6: Shell/nav Vitest (+ optional document.title) — guest login has no Practice/History/Board links; authed hub has them; optional router title meta if cheap.

  LOGGING: assert debug only if existing tests already do.

  Files: `web/tests/**`, optionally `web/src/router/index.ts`

  Depends on: 4, 5

<!-- Commit checkpoint: tasks 4–6 -->

### Phase 3: Keyboard + feature copy accuracy

- [x] Task 7: Align keyboard docs (and optional practice Ctrl+Z) — **default:** scope README Features + Keyboard Shortcuts to free-board Ctrl/Cmd+Z; document Escape for board and practice. Only wire practice Ctrl+Z if ≤ small change in `usePracticeCanvas` / stage canvas and covered by unit test.

  LOGGING: if shortcut added, DEV `console.debug('[practice.canvas] undo shortcut')`.

  Files: `README.md`, optionally `web/src/composables/usePracticeCanvas.ts`, `PracticeStageCanvas.vue`, tests

  Depends on: 1

- [x] Task 8: Sweep feature/restore copy — fix README bullets that blur free-board restore vs practice resume; ensure Features list matches guided practice + mastery/review without overclaiming; grep UI i18n for confidence/AI/ONNX/migration strings.

  LOGGING: N/A.

  Files: `README.md`, `web/src/i18n/locales/*.ts`, `internal/docguard/*` if new forbidden phrases needed

  Depends on: 7

<!-- Commit checkpoint: tasks 7–8 -->

### Phase 4: Config, Makefile, Docker, privacy, troubleshooting, contributor docs

- [x] Task 9: Env + Docker + Makefile cleanup — single accurate env table; fix Docker “hot reload”; remove phantom `.PHONY` / Dockerfile `models/` cruft; document `make sync-audio`; keep deprecated ONNX note honest (warn+ignore, no `ONNX_MODEL=`).

  LOGGING: preserve `[main]` ONNX WARN contract (`cmd/server/onnx_env_test.go`).

  Files: `README.md`, `Makefile`, `Dockerfile.backend`, optionally `.gitignore`, `docker-compose*.yml` comments only if needed

  Depends on: 1

- [x] Task 10: Privacy / deletion / export guidance — rewrite retention subsection: stored data classes; practice clear UI+API; free-board clear; **unsupported** export and self-serve account delete; align `.ai-factory/RULES.md` account-delete wording with cascade/operator reality.

  LOGGING: reference existing clear logs; no new PII logs.

  Files: `README.md`, `.ai-factory/RULES.md` (minimal), optionally hub confirm copy in i18n if unclear

  Depends on: 9

- [x] Task 11: Troubleshooting + architecture notes + contributor workflow — verify troubleshooting steps against code; update `AGENTS.md` / `ARCHITECTURE.md` for shell/diagnostics; add concise Contributing (README section or `CONTRIBUTING.md`) with verified commands and CI check names.

  LOGGING: N/A.

  Files: `README.md`, `AGENTS.md`, `.ai-factory/ARCHITECTURE.md`, optionally `CONTRIBUTING.md`, `.ai-factory/DESCRIPTION.md` if product summary drifts

  Depends on: 9, 10, 2, 5

- [x] Task 12: Automated command verification — add `scripts/verify-readme-commands.sh` (or Go test under `internal/docguard`) that extracts `make …` / `./test.sh …` from README and asserts targets/subcommands exist; wire into `make check` or `check-go` / docs gate. Fail loudly with `[docs.verify] missing=…`.

  LOGGING: `[docs.verify] start|ok|fail` on stdout.

  Files: `scripts/*` or `internal/docguard/*`, `Makefile`, `README.md` (if listing the verify command)

  Depends on: 9, 11

<!-- Commit checkpoint: tasks 9–12 -->

### Phase 5: E2E, CI parity, final honesty pass

- [x] Task 13: Playwright assertion — learner journeys assert no “Migration Health” (and optionally guest login has no Practice nav). Keep journeys stable; do not expand to full visual suite.

  LOGGING: Playwright default CI logging; no `RECOGNIZE_DEBUG`.

  Files: `web/e2e/**`

  Depends on: 2, 4, 3

- [x] Task 14: Final docs honesty pass — grep README for Ctrl+Z scope, hot reload, ONNX setup, export, account delete; run `go test ./internal/docguard`, `./test.sh recognize-fixtures`, `make check`, and the new verify script; fix any drift found.

  LOGGING: CI/`[docs.verify]` markers only.

  Files: `README.md`, tests as needed

  Depends on: 8, 10, 11, 12, 13

<!-- Commit checkpoint: tasks 13–14 -->

## Commit Plan

- **Commit 1** (tasks 1–3): `fix(web): gate migration health panel away from learners`
- **Commit 2** (tasks 4–6): `fix(web): consolidate guest shell nav and branding copy`
- **Commit 3** (tasks 7–8): `docs: scope keyboard shortcuts to actual canvas behavior`
- **Commit 4** (tasks 9–12): `docs: align env, privacy, Makefile, and verify README commands`
- **Commit 5** (tasks 13–14): `test: assert no learner diagnostics; final honesty pass`

## Implement Notes

- Stay on **main** (user requested no new branch for this plan).
- Prefer **docs-accurate** over adding features (no account-delete/export unless explicitly expanded).
- Do not uncheck Prompt 08–18 roadmap items.
- `Docs: yes` → `/aif-implement` mandatory docs checkpoint; keep `internal/docguard` green.
- Verbose logging = DEV/`[docs.verify]` markers, not production stroke dumps.

### Inventory lock (Task 1) — confirmed 2026-09-09

| Item | Finding | Action taken |
|------|---------|--------------|
| Diagnostics | `MigrationHealthPanel` + `migrationHealth.ts`; writers on `BoardPage` only | DEV gate + rename “Dev metrics”; `trackMetric` no-op in prod |
| Shell | Full nav on auth pages | `guestShell` meta; hide nav + romanization on login/register |
| Keyboard | Ctrl+Z free-board only; Escape shared | README scoped; practice Ctrl+Z not wired |
| Make/Docker | Phantom `dev`/`zinnia-*`; Dockerfile `models/`; “hot reload” | PHONY cleaned; models mkdir removed; Docker DX honest |
| Privacy | README implied account-delete path | Explicit no export / no self-serve account delete |
| RULES.md | “keep-until-clear (or account delete)” | Left for `$aif-rules` (implement must not edit RULES); README corrected |

### Verify evidence

- `make verify-docs` → `[docs.verify] ok`
- `go test ./internal/docguard` green
- `./test.sh recognize-fixtures` green
- Vitest: 132 passed (incl. `app-shell-diagnostics.spec.ts`)
