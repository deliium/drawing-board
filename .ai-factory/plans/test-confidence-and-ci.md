# Implementation Plan: Test Pyramid and CI Confidence (Prompt 18)

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Test pyramid and CI confidence (Prompt 18)"
Rationale: Product milestones through Prompt 17 shipped a full learner path, but confidence rests on uneven Go tests, thin Vitest placeholders, no browser E2E, and zero repository CI — this milestone installs a balanced pyramid, race-aware WS checks, coverage-as-signal, and required PR gates with local Makefile equivalents.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep Prompt 08–17 entries unchanged (do not uncheck completed items).

## Summary

Prompts 01–17 delivered private stroke sync, perimeter security, hiragana5 curriculum/assessment, guided practice UI, mastery/review, and language-learning chrome. Verification today is **local-only** and uneven:

| Layer | Current state | Gap |
|-------|---------------|-----|
| Go unit/integration | ~33 `*_test.go` files across auth/db/httpapi/ws/recognize/learn/… | No CI; `./test.sh race` exists but is unused in default `make test`; WS tests mostly use fake `*websocket.Conn` without upgrade/dial contracts |
| SQLite migrations | `migrate_test.go` covers fresh open + some upgrades | Need explicit upgrade-from-baseline / fail-closed / seed idempotency matrix documented as CI gate |
| Recognition fixtures | Solid `internal/recognize/testdata/hiragana5` + Eval/Fixture runners | Keep deterministic; expand only where gaps; wire as named CI step |
| Frontend Vitest | Good unit/contract/a11y for practice; **legacy US1–US3 specs are route/metric stubs** | Replace placeholders; deepen canvas + critical components; keep axe |
| Browser E2E | None (Playwright/Cypress absent) | Add a **small** Playwright suite for 2–3 learner journeys |
| Lint / type / build | `tsc` only via `npm run build`; no ESLint; no `golangci-lint` | Add pragmatic Go + TS checks without vanity rule floods |
| Coverage | `test.sh report` writes `coverage/` for Go only | Collect Go + Vitest coverage as **artifacts/signal**, not a global % gate |
| CI | **No `.github/`** | GitHub Actions: fast PR jobs + optional nightly; caching; artifacts; security scans; required checks |

This plan delivers a **balanced test pyramid** and **CI that stays fast for routine PRs**, with documented local equivalents.

**Out of scope:** Rewriting product features; raising global coverage to a vanity percentage; full visual-regression farm; load/chaos testing; multi-browser matrix beyond Chromium (+ optional Firefox nightly); rewriting all Vitest into Playwright; SM-2/FSRS/collaboration; new curriculum characters; ONNX/ML.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| CI | No `.github/workflows`; no Dependabot/CodeQL config |
| Makefile | `test` / `test-verbose` → `./test.sh all` (Go only); `validate-content`; no web test / race / lint targets as first-class CI mirrors |
| `test.sh` | `all`, package filters, `coverage`, `report`, `bench`, `race` — race **not** in default `all` |
| Go CGO | `mattn/go-sqlite3` → race + integration need CGO-enabled runners |
| Vitest | `web/vite.config.ts` `environment: 'jsdom'`; no dedicated `vitest.config`; scripts: `test` only (no lint/typecheck/coverage script) |
| Placeholder FE tests | `web/tests/integration/us1-*.spec.ts`, `us2-*.spec.ts`, `us3-route-parity.spec.ts` assert router names/meta or trivial metrics — migration leftovers |
| A11y | `accessibility-parity.spec.ts` + README manual device checklist |
| Canvas | Unit tests for coords/draw/hit + practice-canvas; jsdom stubs (`stubCanvasContext.ts`); real pointer/DPR behavior limited |
| WS Go | Hub unit tests + concurrent add/remove; origin httptest; no `-race` in default path; limited real dial/`httptest` upgrade coverage |
| Docs | README Make Commands list Go tests only; no “CI / quality gates” section |

## Design Decisions

### D0 — Product goal

Raise **engineering confidence** that regressions in privacy, perimeter, board sync, assessment honesty, and the guided learner journey are caught **before merge**, while keeping the default feedback loop fast enough for day-to-day development.

### D1 — Test pyramid (balanced, not inverted)

```text
        /\
       /E2\     2–3 Playwright learner journeys (Chromium); optional nightly Firefox
      /----\
     / Vitest\  unit + contract + axe; replace US stubs; canvas/composables/components
    /----------\
   / Go unit+int \  packages, migrations, httptest/WS dial, fixtures, -race on WS/hot paths
  /----------------\
```

| Tier | What belongs | What does **not** |
|------|--------------|-------------------|
| Go unit | Pure logic: limits, normalize, criteria, mastery/review, password | Full browser |
| Go integration | SQLite migrate/seed, httpapi with real Store, WS upgrade+ingest | Pixel painting |
| Vitest | Components, composables, services, axe, API/WS **client** contracts | Full multi-tab WS against live server (prefer Go + one E2E) |
| Playwright | Login → practice submit/assess; history/mastery smoke; optional board Saved gate | Exhaustive criterion matrix (that stays in Go fixtures) |

### D2 — Race detection

| Decision | Detail |
|----------|--------|
| Default local | `make test-race` runs `go test -race` on **WS + db board helpers + httpapi stroke/attempt paths** (or `./...` if still fast) |
| CI PR | Dedicated job `go-race` with CGO; may be slower — allow ~10–15 min budget; fail on race |
| Focus tests | Extend concurrent hub tests; add real-conn stress that hits `sendToUser` / ack paths under `-race` |
| Logging | On race fail, CI uploads race log artifact; tests use `t.Logf` with `[ws.race]` style markers when verbose |

### D3 — Deterministic recognition fixtures

| Decision | Detail |
|----------|--------|
| Source of truth | Keep `internal/recognize/testdata/hiragana5/**` + pack templates; do not invent ML |
| CI | Named step `go test ./internal/recognize -run 'Eval|Fixture|Hiragana5' -count=1` |
| Expansion | Only add fixtures for known brittle criteria or missing correction codes — not a bulk dump |
| Honesty | `go test ./internal/docguard` remains required; coverage never excuses marketing language |

### D4 — Coverage as signal (not vanity)

| Decision | Detail |
|----------|--------|
| Collect | Go `coverprofile` + Vitest `--coverage` (add `@vitest/coverage-v8`) |
| Publish | Upload HTML/`lcov` as CI **artifacts**; optional PR comment only if cheap |
| Gates | **No** repo-wide “must be ≥80%” gate |
| Optional floors | Soft or hard floors **only** on critical packages if implement finds them stable: e.g. `internal/recognize`, `internal/limits`, `internal/learn` — document chosen floors in README; prefer signal dashboards over blocking global % |
| Local | `make coverage` / `make coverage-web` generate reports under `coverage/` (gitignored) |

### D5 — Lint / type / build

| Check | Tool | Policy |
|-------|------|--------|
| Go fmt | `gofmt -l` (or `go fmt`) | Fail if dirty |
| Go vet | `go vet ./...` | Required |
| Go lint | `golangci-lint` with **small** config (errcheck, staticcheck, govet, ineffassign) — avoid enabling everything day one | Required once config lands |
| TS | `vue-tsc --noEmit` or existing `tsc` from build | Required on web |
| ESLint | Optional light Vue/TS config **or** skip if time-boxed — prefer typecheck+build over a huge lint ruleset | Prefer yes if cheap |
| Build | `go build ./...` + `npm run build` (web) | Required |
| Content | `make validate-content` | Required |

### D6 — CI topology (fast PR + optional nightly)

```text
PR (required, target ≤8–12 min wall with parallelism):
  ├─ go-quality     fmt/vet/golangci + build + unit/integration (no race or short race subset)
  ├─ go-race        -race on WS-focused packages (required; may be longest)
  ├─ web-quality    npm ci → typecheck → vitest → build
  ├─ content        validate-content
  └─ security-light govulncheck + npm audit --omit=dev (or production audit)

Nightly / manual (not required for every PR):
  ├─ playwright     Chromium journeys against compose or scripted server
  ├─ go-race-full   ./... -race
  └─ security-deep  CodeQL or fuller audit; Dependabot separate
```

**Speed rules:**

- Parallel jobs; cache Go modules + build cache; cache `~/.npm` / `web/node_modules` via `actions/setup-node` cache
- Playwright browsers cached; install only in E2E job
- Do **not** run Docker image rebuilds on every PR unless needed for E2E; prefer `go run` + `vite preview` or small compose profile
- Fail-fast per job; continue-on-error **only** for explicitly experimental steps

### D7 — Artifacts

| Artifact | Retention | Purpose |
|----------|-----------|---------|
| Go coverage HTML/out | 7–14 days | Signal |
| Vitest coverage | 7–14 days | Signal |
| Playwright report + traces on failure | 7–14 days | Debug |
| Race detector logs on failure | 7–14 days | Debug |
| `contentvalidate` hash output (optional) | short | Audit |

Never upload stroke coordinates from `RECOGNIZE_DEBUG` in CI (debug flag off).

### D8 — Security scanning

| Scan | Placement |
|------|-----------|
| `govulncheck ./...` | PR job (light) |
| `npm audit` (production deps) | PR job; document how to triage false positives |
| Dependabot or Renovate | Enable for Go modules + npm (config only) |
| Secret scan | `gitleaks` or GitHub secret scanning; fail on high-confidence |
| CodeQL | Nightly or weekly Go+JS — optional if setup cost is high; prefer at least one SAST |

Do not block merges on low/informational npm advisories without a documented allowlist process.

### D9 — Required pull-request checks

Branch protection on `main` (document exact check names):

1. `Go quality`
2. `Go race`
3. `Web quality`
4. `Content validate`
5. `Security light`

Playwright may start as **non-required** until stable green, then promote — plan Task should leave a clear “promote when green” note.

### D10 — Local equivalents (mandatory docs)

Every CI job maps to a Makefile target (or script):

| CI job | Local |
|--------|-------|
| Go quality | `make check-go` |
| Go race | `make test-race` |
| Web quality | `make check-web` |
| Content | `make validate-content` |
| Security light | `make security-check` |
| Playwright | `make test-e2e` |
| All PR-like | `make check` (orchestration; may skip e2e by default) |

Verbose logging: CI steps print job start/end, cache hit/miss, test package counts; Go tests keep bracket prefixes; Vitest uses existing DEV debug patterns only in tests that assert them.

### D11 — Frontend placeholder cleanup

Replace or delete legacy migration specs:

- `us1-core-workflows.spec.ts`, `us1-deeplink.spec.ts`
- `us2-rollout.spec.ts`, `us2-rollback.spec.ts`
- `us3-route-parity.spec.ts`

Either fold real assertions into `auth-guards` / practice integration specs, or rewrite into meaningful smoke (router + guard behavior with stubbed `me`). **No** `expect(route).toBeTruthy()`-only files remaining.

### D12 — Playwright learner journeys (small set)

Minimum journeys (Chromium):

1. **Auth + practice assess** — register/login (or seed user), open `/#/practice/:id`, advance to free-write with fixture strokes (inject via evaluate or draw helpers), submit, see match score + ≤2 corrections (or fail path with fixture).
2. **History / hub** — after an assessed attempt, history lists metadata **without** point arrays; hub shows mastery or next without crashing.
3. **Optional board sync smoke** — login, draw one stroke, reach Saved (if flaky, demote to nightly).

Constraints: use test DB file; CSRF-aware helpers; no dependence on external network audio (mock or local static); honesty: assert “Match” not “confidence”.

## Acceptance Criteria

1. `.github/workflows/` exists with PR workflow(s) matching D6; caches + artifacts per D7.
2. Required checks documented; branch-protection checklist in README (operator applies on GitHub).
3. `make check` / `make test-race` / `make check-web` / `make test-e2e` / `make security-check` / `make coverage` work locally and mirror CI.
4. Go `-race` job covers WebSocket package (and related) and is green.
5. Migration tests cover fresh DB + at least one upgrade path + fail-closed behavior; run in CI.
6. Recognition Fixture/Eval tests are an explicit CI step; fixtures remain deterministic.
7. Legacy US1–US3 placeholder Vitest specs removed or rewritten with real assertions.
8. Vitest covers critical Vue practice components + canvas helpers; axe suite still green.
9. Playwright suite with ≥2 learner journeys runs in CI (required or documented non-required with promote path).
10. Coverage reports uploaded as artifacts; README states coverage is a **signal**, not a vanity gate.
11. `govulncheck` + npm audit (+ secret scan and/or Dependabot) wired; triage notes in docs.
12. README + AGENTS/ARCHITECTURE updated; Prompt 18 left unchecked until verify.

## Scenario Checklist

### S1 — Local PR-equivalent

1. Developer runs `make check` → Go quality + web quality + content pass without GitHub.

### S2 — Race catch

1. Intentional data race in hub map (temporary) → `make test-race` fails; CI race job would fail.

### S3 — Migration fail-closed

1. Corrupt/partial migrations scenario (existing or new test) fails closed; CI runs migrate tests.

### S4 — Recognition fixtures

1. Flip a gold fixture expectPass → Fixture test fails in CI recognize step.

### S5 — Placeholder gone

1. No US1–US3 stub-only files; `npm test` still green.

### S6 — E2E journey

1. Playwright journey 1 completes against local server; failure uploads trace.

### S7 — Security light

1. `make security-check` runs; known clean baseline documented.

### S8 — Coverage signal

1. `make coverage` produces Go HTML; web coverage artifact path documented; no global % hard fail unless critical-package floors explicitly chosen.

## Tasks

### Phase 1: Inventory, toolchain, Makefile contracts

- [x] Task 1: Inventory & gitignore — confirm `coverage/`, Playwright report dirs, and any CI caches are ignored; list current Go/Vitest entrypoints in a short “Quality” subsection draft (final docs in Task 10). Decide golangci-lint version pin and Playwright location (`web/e2e` vs `e2e/`).

  LOGGING: N/A for inventory; document chosen paths in plan notes if they differ from defaults (`web/e2e`, `coverage/`, `playwright-report/`).

  Files: `.gitignore`, optionally `web/.gitignore`, Makefile sketch comments

  Notes: Chose `web/e2e`, root `coverage/` + `web/coverage/`, golangci-lint via CI install (local optional). Ignored playwright-report/test-results/blob-report and `*.race.log`.

- [x] Task 2: Makefile / scripts — add `check-go`, `check-web`, `test-race`, `test-web`, `test-e2e`, `security-check`, `coverage`, `coverage-web`, `check` (PR-like, e2e optional via `CHECK_E2E=1`). Keep `test` backward compatible or redirect to clearer targets with README note. Ensure `test.sh race` and package filters remain usable.

  LOGGING: each target echoes `[check] start|ok|fail name=…`; verbose `-v` passthrough for Go.

  Files: `Makefile`, `test.sh` (extend, do not delete useful commands)

  Depends on: 1

<!-- Commit checkpoint: tasks 1–2 -->

### Phase 2: Strengthen Go pyramid (migrations, HTTP/WS, race, fixtures)

- [x] Task 3: SQLite migrations — extend `internal/db/migrate_test.go` (or sibling) for: fresh open → `LatestVersion`; reopen idempotent; upgrade from older fixture DB if practical; fail-closed on bad migration; seed hiragana5 idempotent; board vs attempt table isolation smoke. Keep tests hermetic (`t.TempDir`).

  LOGGING: `t.Logf` version before/after; `[db.migrate]` style messages already in code — assert no panic; DEBUG not required in production path.

  Files: `internal/db/migrate_test.go`, possibly `internal/db/testdata/`

  Depends on: 2

- [x] Task 4: HTTP + WebSocket contracts — fill gaps with `httptest` REST tests (auth CSRF, attempts lifecycle already partially covered — close holes) and **real WebSocket dial** against `httptest.Server` + hub: create/ack/`boardRev`, isolation `sendToUser`, stale_board / nack paths as applicable. Prefer table-driven cases; no stroke coordinate spam in logs.

  LOGGING: `[ws.contract]` / `[httpapi.contract]` via `t.Logf` on case name + codes; WARN paths assert reject codes only.

  Files: `internal/ws/*_test.go`, `internal/httpapi/*_test.go`

  Depends on: 2

- [x] Task 5: Race detection — ensure concurrent hub/store tests are meaningful under `go test -race`; add focused stress if needed; wire `make test-race` to packages that exercise WS + SQLite board ops (CGO). Fix any races found (product bugfixes allowed if small).

  LOGGING: on failure rely on race detector output; add `t.Logf("[ws.race] …")` for scenario labels.

  Files: `internal/ws/handler_test.go` (+ board/db tests as needed), `Makefile`

  Depends on: 4

- [x] Task 6: Recognition fixtures — audit fixture pack; add any missing correction/pass cases called out by recent assess changes; ensure `-run 'Eval|Fixture|Hiragana5'` is documented and CI-ready; keep `docguard` green.

  LOGGING: fixture runner already logs names; keep deterministic `-count=1` in CI.

  Files: `internal/recognize/testdata/**`, `*_test.go`, `internal/docguard/*` if copy changes

  Depends on: 2

  Notes: 14 fixtures across gold/incorrect/short_strokes already green; `./test.sh recognize-fixtures` wired. No bulk expansion needed.

<!-- Commit checkpoint: tasks 3–6 -->

### Phase 3: Frontend Vitest quality + a11y + canvas

- [x] Task 7: Replace placeholder US1–US3 Vitest files; deepen practice component/composable tests and canvas behavior (resize/DPR/logical coords, pointer cancel via existing composable tests). Keep axe integration green; extend only if a critical surface lacks coverage (history, audio button).

  LOGGING: DEV `console.debug` assertions only where existing style does; no new production log noise.

  Files: delete/rewrite `web/tests/integration/us*.spec.ts`; `web/tests/unit/*`; `web/tests/integration/accessibility-parity.spec.ts`; `web/package.json` scripts (`test`, `test:coverage`, `typecheck`)

  Depends on: 2

- [x] Task 8: Web toolchain — add `typecheck` script; coverage provider; optional thin ESLint if low-cost. Ensure `npm run build` remains the build gate.

  LOGGING: N/A; CI prints npm script names.

  Files: `web/package.json`, `web/vite.config.ts` (coverage config), lockfile

  Depends on: 7

<!-- Commit checkpoint: tasks 7–8 -->

### Phase 4: Playwright E2E (small)

- [x] Task 9: Add Playwright (Chromium) with ≥2 journeys from D12; helpers for CSRF/session; isolated SQLite path; scripts `test:e2e` / `make test-e2e`. Start CI job as non-required if flaky, with promote note. Artifacts: HTML report + traces on failure.

  LOGGING: Playwright config `quiet: false` in CI; server logs to file artifact on failure; never enable `RECOGNIZE_DEBUG` in CI.

  Files: `web/e2e/**` or `e2e/**`, `playwright.config.ts`, `Makefile`, `web/package.json`

  Depends on: 7, 3

<!-- Commit checkpoint: task 9 -->

### Phase 5: CI, security, docs

- [x] Task 10: GitHub Actions — PR workflow with parallel jobs (go-quality, go-race, web-quality, content, security-light); caching (Go modules/build, npm); artifact uploads (coverage, playwright on failure, race logs on failure); pin Action versions. Optional nightly workflow for full race + Playwright + CodeQL. Add Dependabot config for gomod + npm.

  LOGGING: each step `echo "::group::…"` / clear start-end markers; fail messages name the Makefile equivalent.

  Files: `.github/workflows/ci.yml`, optionally `nightly.yml`, `.github/dependabot.yml`, `.golangci.yml`

  Depends on: 2, 5, 6, 8, 9

- [x] Task 11: Security scanning — `govulncheck`, npm production audit, secret scan approach (gitleaks action or GitHub native); document triage/allowlist. Keep scans from dumping secrets into logs.

  LOGGING: tool stdout only; redact tokens; `[security-check] ok|fail`.

  Files: `Makefile`, workflow YAML, short README subsection

  Depends on: 10

- [x] Task 12: Docs checkpoint — README “Quality gates / CI” (pyramid, local `make` table, coverage-as-signal, required checks names, how to run E2E); update `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, `.ai-factory/DESCRIPTION.md` non-functional testing note; leave Prompt 18 unchecked until verify.

  LOGGING: document race/CI log locations and that stroke coordinates stay out of CI logs.

  Files: `README.md`, `AGENTS.md`, `.ai-factory/ARCHITECTURE.md`, `.ai-factory/DESCRIPTION.md`

  Depends on: 10, 11

<!-- Commit checkpoint: tasks 10–12 -->

## Commit Plan

- **Commit 1** (after tasks 1–2): `chore: add local quality Makefile targets and coverage gitignore`
- **Commit 2** (after tasks 3–6): `test(go): harden migrations, WS/HTTP contracts, race, and fixtures`
- **Commit 3** (after tasks 7–8): `test(web): replace placeholder specs and add typecheck/coverage scripts`
- **Commit 4** (after task 9): `test(e2e): add Playwright learner journey smoke suite`
- **Commit 5** (after tasks 10–12): `ci: GitHub Actions, security scans, and quality-gate docs`

## Risks / Notes

- **CGO + race + sqlite3** can be slow on CI — keep race job focused; cache aggressively.
- **Playwright flake** — start non-required; stabilize before branch protection.
- **golangci-lint noise** — enable a small set first; fix or narrowly exclude with comments.
- **npm audit** false positives — document omit/allowlist; do not soft-fail silently forever.
- **jsdom canvas limits** — do not pretend Playwright replaces unit geometry tests; keep both tiers.
- **No new product scope** — if a race reveals a real WS bug, fix minimally and note in commit.
- **Stay on main** — user requested no new branch for this plan.

## Implementation Notes for `/aif-implement`

- Stay on **main** (user requested no new branch).
- Prefer small diffs: Makefile/tooling → Go tests → Vitest cleanup → Playwright → CI/docs.
- Every task needs logging as specified; CI must not enable coordinate debug dumps.
- Docs policy: **mandatory** checkpoint (Settings Docs: yes).
- Do not mark ROADMAP Prompt 18 complete in implementation commits — leave for verify/roadmap.
- Coverage floors, if any, must be justified per critical package — never a single global vanity %.
