# Implementation Plan: Japanese Learning Release Sequencing (hiragana5)

Branch: main
Created: 2026-09-10

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "R1: Trustworthy guided lesson for five hiragana"
Rationale: First independently releasable product cutover — honest assessment + reviewed curriculum + attempt APIs + guided journey for あいうえお — with migration, feature-flag, rollback, observability, and E2E gates defined for the full R1–R4 train.

> Note for `$aif-roadmap` / implement: append **unchecked** milestones R1–R4 (titles below) to `.ai-factory/ROADMAP.md`. Do not uncheck completed Prompt 08–19 entries. This plan owns cutover sequencing and release gates; it does **not** re-own feature implementation tasks inside individual Prompt plans.

## Summary

Prompts 08–19 are approved (and largely implemented) as separate feature plans under `.ai-factory/plans/`. Shipping them as one undifferentiated blob couples schema, protocol, UI, and security risks and blocks rollback. This plan **sequences** those plans into **independently releasable product milestones (R1–R4)**, resolves overlap, names hard dependencies, and defines shared **migration / feature-flag / rollback / observability / E2E acceptance** contracts.

**Do not duplicate** detailed implementation tasks already owned by:

| Prompt | Plan file |
|--------|-----------|
| 08 | `honest-recognition-strategy.md` |
| 09 | `learning-domain-versioned-migrations.md` |
| 10 | `starter-hiragana-curriculum.md` |
| 11 | `practice-attempt-apis.md` |
| 12 | `target-specific-handwriting-assessment.md` |
| 13 | `core-learner-journey-hiragana.md` |
| 14 | `responsive-bilingual-accessible-ui.md` |
| 15 | `attempt-history-and-mastery.md` |
| 16 | `spaced-review-queue.md` |
| 17 | `handwriting-language-learning.md` |
| 18 | `test-confidence-and-ci.md` |
| 19 | `product-copy-docs-alignment.md` |

This plan’s deliverables are: dependency/conflict matrix, R1–R4 cut definitions, flag + migration policy, operator runbooks, observability cutover signals, and release-gate tests/docs. Feature work stays in the referenced plans.

**Baseline (already released / required before R1):** per-user stroke isolation, public onboarding, password/session security, HTTP/WS perimeter, input hardening, reliable WS + `boardRev`, Vue canvas lifecycle. Cite: `feature-japanese-training-no-collaboration.md`, `public-onboarding.md`, `password-session-security.md`, `http-websocket-perimeter-security.md`, `recognition-stroke-input-hardening.md`, `reliable-stroke-ws-persistence.md`, `feature-authoritative-board-ops.md`, `feature-vue-canvas-lifecycle.md`.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Feature plans 08–19 | Present under `.ai-factory/plans/`; ROADMAP marks Prompt milestones complete |
| Schema | Versioned migrations `0001`…`0006` (`baseline_board` → `curriculum_guidance`); fail-closed on `Open`; production path forward-only |
| Feature flags | **None** for practice/history/review/audio — routes and APIs are always-on once binary ships |
| Practice product | Attempt REST + `/#/practice*` + mastery/review/audio already wired on `main` |
| Observability | Structured logs (`[db.migrate]`, `[db.seed]`, `[httpapi.*]`, `[recognize.*]`, …); `RECOGNIZE_DEBUG` (non-prod); `LOG_LEVEL`; process metrics; no release-cutover counters |
| Quality gates | `make check`, `test-race`, `check-web`, `test-e2e`, `security-check`, `validate-content`, `verify-docs` |
| Gap | No documented **releasable cuts**, flag rollback knobs, or per-milestone E2E acceptance matrices tying Prompt plans together |

## Design Decisions

### D0 — Release train shape (product milestones)

| Release | Product outcome | Owned feature plans (reference only) | Schema ceiling (must be applied) | Default flags (see D2) |
|---------|-----------------|--------------------------------------|----------------------------------|------------------------|
| **R1** | Trustworthy guided lesson for **five** hiragana (あいうえお) | 08, 09, 10, 11, 12, 13 | ≤ `0004` (`curriculum_ja_pedagogy`) | `FEATURE_PRACTICE=1` |
| **R2** | Mobile-first bilingual accessible learner chrome | 14 (+ shell/copy honesty slices from 19 that touch practice/auth UX) | still ≤ `0004` (no new DDL required) | `FEATURE_PRACTICE=1` (+ UI always with practice) |
| **R3** | Personal history, explainable mastery, Leitner review | 15, 16 | ≤ `0005` (`review_schedule`) | `FEATURE_PROGRESS=1`, `FEATURE_REVIEW=1` |
| **R4** | Pronunciation/guidance + CI/docs honesty hardening | 17, 18, 19 (remainder) | ≤ `0006` (`curriculum_guidance`) | `FEATURE_AUDIO=1` |

**R1 definition of done (product):** authenticated learner can complete intro → stroke-order → trace → free-write → submit/assess → ≤2 corrections → retry/complete for **each** of あいうえお, with attempt strokes ≠ board strokes, match scores only (not confidence), reviewed pack content, and perimeter/CSRF intact. Bilingual polish, history, SRS-like review, and audio are **explicitly not** required for R1.

### D1 — Hard dependency graph (do not reorder)

```text
Perimeter + board + canvas lifecycle (baseline)
        │
        ▼
Prompt 08 honest target-compare contract
        │
        ├──────────────────────┐
        ▼                      ▼
Prompt 09 migrations/repos   (08 fixtures may land in parallel after strategy lock)
        │
        ▼
Prompt 10 reviewed hiragana5 pack + seed
        │
        ├────────────┐
        ▼            ▼
Prompt 11 attempts  Prompt 12 multi-criterion + corrections
        │            │
        └─────┬──────┘
              ▼
        Prompt 13 guided journey UI (all five glyphs)
              │
              ▼
        ★ R1 CUTOVER ★
              │
              ▼
        Prompt 14 responsive bilingual a11y
              │
              ▼
        ★ R2 CUTOVER ★
              │
        ┌─────┴─────┐
        ▼           ▼
   Prompt 15     (15 before 16: next/mastery without due_at)
        │
        ▼
   Prompt 16 review schedule (migration 0005)
        │
        ▼
        ★ R3 CUTOVER ★
              │
        ┌─────┼─────────┐
        ▼     ▼         ▼
       17    18        19
   (audio) (CI)  (copy/docs)
        │     │         │
        └─────┴─────────┘
              ▼
        ★ R4 CUTOVER ★
```

**Hard constraints:**

1. **09 before 10/11/13/15/16** — learning tables + versioned runner.
2. **10 before 11/12/13/17** — pack is seed + assess + pedagogy source of truth.
3. **08 before 12** — honesty contract and target-compare API shape.
4. **11 before 13/15** — attempt lifecycle HTTP.
5. **12 before 13** — non-empty correction `message` for coaching UI.
6. **13 before 14/15/17** — practice routes/components to restyle/enrich.
7. **15 before 16** — mastery/next/clear exist; 16 extends schedule + next why.
8. **14 before treating R2 done** — a11y/i18n gates are R2 acceptance, not R1.
9. **18 may start anytime after R1 APIs exist** but **R4** waits for required CI checks green; do not block R1 on Playwright promotion.
10. **19 honesty gates** apply continuously; R4 requires learner-visible diagnostics gone and README/docguard green.

### D2 — Overlap resolution (schema / protocol / UI / security)

#### Schema

| Concern | Owner plan | Resolution |
|---------|------------|------------|
| `schema_migrations` + learning tables | 09 (`0002`) | Only 09 may define baseline learning DDL |
| Pedagogy columns EN | 10 (`0003`) | Additive; seed from pack |
| Pedagogy columns JA | 10 / follow-on (`0004`) | Additive; no attempt API change |
| `review_box` / `due_at` | 16 (`0005`) | **Must not** ship in R1/R2 binaries that expose review UI; migration may ship early only if APIs/UI stay flag-off |
| `guidance` / audio-related pack fields | 17 (`0006`) | Additive; UI gated by `FEATURE_AUDIO` |
| Free-board strokes | baseline board plans | **Never** FK/clear-coupled to attempts |

**Policy:** Migrations are **always applied forward** on `Open` (fail-closed). Product surfaces are gated by flags. Never ship a binary that **requires** `0005`/`0006` columns for R1/R2 request paths.

#### Protocol

| Surface | Owner | Cut |
|---------|-------|-----|
| `Assessor.Assess` + match scores | 08 → 12 | R1 |
| `POST /api/recognize` heuristic only (no `target`) | 08 + 11 | R1 |
| Attempt CRUD/submit/assess/abandon | 11 | R1 |
| `GET /api/lessons/{id}`, `GET /api/progress` | 13 | R1 |
| History list, mastery fields, `DELETE /api/practice-data`, next | 15 | R3 |
| Review fields on progress + schedule-aware next | 16 | R3 |
| Lesson payload `audioRef` / guidance | 17 | R4 |
| WS / `boardRev` / `opId` | baseline | Unchanged all cuts; practice ink never enqueued |

**Conflict rule:** Prompt 11 removes board `target` assess; Prompt 13 must not reintroduce board-loaded practice. Prompt 15/16 extend `GET /api/progress/next` — 16 must remain backward compatible when `FEATURE_REVIEW=0` (no due preference / omit schedule fields or null them).

#### UI

| Surface | Owner | Cut |
|---------|-------|-----|
| `/#/practice`, `/#/practice/:characterId` | 13 | R1 |
| Fluid canvas, EN/JA, a11y | 14 | R2 |
| Hub mastery + `/#/practice/history` | 15 | R3 |
| Due/review chrome | 16 | R3 |
| Audio button, romaji hide, guidance | 17 | R4 |
| AppShell guest vs authed; hide DEV diagnostics | 19 | R2 minimum (diagnostics), R4 complete |

**Conflict rule:** 14 owns tokens/i18n; later plans only add catalog keys. 15/16 must not invent a second hub layout. 17 must soft-fail missing audio (R1/R2 remain valid without clips).

#### Security / privacy

| Concern | Owner | Cut |
|---------|-------|-----|
| CSRF + origin allowlist + session cookies | baseline | All |
| Attempt authz = session `user_id` | 11 | R1 |
| History payloads without stroke points | 15 | R3 |
| `DELETE /api/practice-data` clears practice only | 15 | R3 |
| Retention / no self-serve account delete honesty | 15 + 19 | R3 docs; R4 docguard |
| No stroke coordinates in logs | all recognize/httpapi plans | All |
| Production `COOKIE_KEY` + `ALLOWED_ORIGINS` | baseline | All deploys |

### D3 — Feature flags (introduce; currently absent)

Env-driven booleans parsed at process start (and mirrored to SPA via a tiny authenticated `GET /api/features` **or** build-time `import.meta.env` + server 404 when off — prefer **server enforce** + hide nav):

| Flag | Default (dev) | Default (prod recommend) | Gates |
|------|---------------|--------------------------|-------|
| `FEATURE_PRACTICE` | `1` | `0` until R1 accepted, then `1` | Attempt + lesson/progress read routes; Practice nav; `/practice*` |
| `FEATURE_PROGRESS` | `1` | `0` until R3 | History list API/UI; mastery enrichment; practice-data DELETE; mastery-only next extras |
| `FEATURE_REVIEW` | `1` | `0` until R3 (requires `FEATURE_PROGRESS=1`) | Schedule fields; due chrome; schedule-aware next |
| `FEATURE_AUDIO` | `1` | `0` until R4 | Pronunciation play control; require non-null audio only when on |

**Rules:**

- Flags are **kill switches**, not substitutes for migrations.
- `FEATURE_REVIEW=1` with `FEATURE_PROGRESS=0` is invalid → refuse start (`FATAL`) or coerce review off + `WARN`.
- When `FEATURE_PRACTICE=0`, mutating attempt routes return **404** (not 403) to avoid feature enumeration; Vue omits Practice nav.
- Free-board + auth remain available regardless of learning flags.
- Verbose startup log: `INFO [main] features practice=… progress=… review=… audio=… schema_version=N`.

### D4 — Migration & rollback policy

| Action | Procedure |
|--------|-----------|
| Upgrade | Take SQLite backup → deploy binary → `Open` applies pending versions → verify `schema_version` log == expected ceiling for that cut |
| R1 ceiling | `N >= 4` (pedagogy present); review/guidance columns **may** already be present if later migrations shipped early — R1 code must not require them |
| R3 ceiling | `N >= 5` |
| R4 ceiling | `N >= 6` |
| Rollback binary | Redeploy previous image; set flags off for surfaces not in prior cut |
| Rollback schema | **Restore DB backup** taken pre-upgrade (forward-only migrations; no production `Down`) |
| Seed | Idempotent `SeedHiragana5`; content pack path immutable for a cut (`content/hiragana5/vN`) |

Document backup command examples in README (operator section) without inventing new tools.

### D5 — Observability for cutovers

Required log/metric signals (verbose-friendly, no coordinates):

| Signal | Level / sink | Purpose |
|--------|--------------|---------|
| `INFO [main] schema_version=N features=…` | startup | Cut verification |
| `INFO [db.migrate] applied version=… name=…` | migrate | Upgrade audit |
| `INFO [db.seed] set=hiragana5 contentVersion=…` | seed | Pack alignment |
| `INFO [httpapi.Attempt*] …` / assess pass/scoreKind | request | R1 health |
| `WARN [httpapi] feature_disabled name=…` | gated 404 path | Flag misuse / probe |
| Counters (extend `internal/metrics`) | `practice_attempt_assess_total`, `practice_feature_disabled_total`, `practice_clear_total` | Cut dashboards |
| `RECOGNIZE_DEBUG` | non-prod only | Criterion diagnostics; never learner-facing |

Cutover checklist must print expected signals after deploy smoke.

### D6 — E2E acceptance gates (per release)

Gates **reference** tests owned by feature plans / Prompt 18; this plan adds a **matrix + Makefile entry** that maps release → commands, not a second Playwright suite.

| Release | Automated gate (minimum) | Manual / smoke |
|---------|--------------------------|----------------|
| **R1** | `make check-go`; recognize fixture eval; attempt HTTP e2e; Vitest journey for all five glyphs; `validate-content`; `go test ./internal/docguard` | Login → practice each of あいうえお → assess → ≤2 corrections; confirm board clear does not wipe attempts; CSRF still required |
| **R2** | `make check-web` including axe suites; i18n catalog keys present | Phone/tablet smoke; EN↔JA; keyboard + reduced motion; no DEV diagnostics in production build |
| **R3** | Go history/progress/review tests; Vitest history/hub; clear practice-data test | History list has no stroke points; next prefers due when review on; clear removes practice only |
| **R4** | `make check` + `test-e2e` + `security-check`; audio sync/`validate-content`; verify-docs | Audio play soft-fail; romaji hide; README commands match Makefile; Playwright learner journeys green |

Promote Playwright to required PR check only at R4 (per Prompt 18), not R1.

## Approach

1. Write the dependency/conflict matrix into this plan (done in Design Decisions) and mirror a short operator table into README / release runbook.
2. Append R1–R4 milestones to `ROADMAP.md`; keep Prompt 08–19 history intact.
3. Implement D3 feature-flag parsing + route/nav gating + startup logs + metrics counters (minimal code owned by **this** plan).
4. Add `make gate-r1` … `gate-r4` (or one `gate-release RELEASE=r1`) wrapping existing targets.
5. Document migration backup/restore, flag rollback, and cutover observability.
6. Tests: flag matrix (404 when off; happy path when on); invalid flag combo; startup log assertions.
7. Docs checkpoint: README release train section, ARCHITECTURE note on flags, AGENTS pointer, docguard unchanged honesty rules.

## Tasks

### Phase 1: Sequencing artifact & roadmap

- [x] Task 1: Freeze R1–R4 cut definitions in-repo
  - Ensure this plan is the canonical sequencing source; cross-link each Prompt plan file in a short “Release train” subsection of README (or `docs/` if present) **without** copying Prompt task lists.
  - Append unchecked ROADMAP milestones:
    - `R1: Trustworthy guided lesson for five hiragana`
    - `R2: Product-ready bilingual accessible learner shell`
    - `R3: Personal progress, mastery, and spaced review`
    - `R4: Language-learning chrome, CI confidence, and docs honesty`
  - LOGGING: N/A (docs); if a small generator script is added, `INFO` only.
  - Files: `.ai-factory/ROADMAP.md`, `README.md`, optionally `AGENTS.md` (one-line pointer).

### Phase 2: Feature flags + server/UI gating

- [x] Task 2: Parse and validate learning feature flags (depends on 1)
  - Implement env parsing for `FEATURE_PRACTICE`, `FEATURE_PROGRESS`, `FEATURE_REVIEW`, `FEATURE_AUDIO` with D3 defaults/validation.
  - Startup: `INFO [main] features practice=%v progress=%v review=%v audio=%v schema_version=%d`.
  - LOGGING: `DEBUG` raw env values (not secrets); `WARN` on coerce; `FATAL`/`ERROR` on invalid combo if refuse-start chosen.
  - Files: `cmd/server/main.go`, small helper under `internal/` (e.g. `internal/features` or `internal/limits` — prefer dedicated `internal/features` to avoid overloading limits).

- [x] Task 3: Enforce flags on HTTP + Vue nav (depends on 2)
  - Server: 404 gated attempt/lesson/progress/history/review/audio-dependent behaviors per D3; emit `WARN [httpapi] feature_disabled` + metric increment.
  - Vue: hide Practice/History nav and block routes when features off (match server).
  - Free-board + auth unaffected.
  - LOGGING: one WARN per disabled hit (rate-limit if noisy); DEBUG route decisions in DEV.
  - Files: `internal/httpapi/*`, `cmd/server` route wiring, `web/src/router/index.ts`, `web/src/components/AppShell.vue`, thin `features` client if needed.

### Phase 3: Observability + release gates

- [x] Task 4: Cutover metrics + log contract tests (depends on 2)
  - Add counters in `internal/metrics` for assess totals, feature-disabled hits, practice-data clears.
  - Test startup feature log / invalid combo; never log stroke points.
  - LOGGING: document prefixes in README; verbose DEBUG on counter increments behind `LOG_LEVEL`.
  - Files: `internal/metrics/metrics.go`, tests under `cmd/server` or `internal/features`.

- [x] Task 5: Makefile release gates wrapping existing quality targets (depends on 1)
  - Add `gate-r1` … `gate-r4` (or `gate-release`) invoking the D6 command sets; do not reimplement Prompt 18 pyramid.
  - Wire README “Release gates” to these targets; keep `scripts/verify-readme-commands.sh` green.
  - LOGGING: gate scripts echo which release and which commands run (`INFO`-style stdout).
  - Files: `Makefile`, `README.md`, optionally `scripts/gate-release.sh`.

### Phase 4: Operator runbook + acceptance mapping

- [x] Task 6: Migration / rollback / cutover runbook (depends on 1, 5)
  - Document backup → migrate → verify schema_version → flag flip → smoke → rollback (binary + DB restore).
  - Map each Rn to Prompt plans + schema ceiling + flags + D6 gates (table only).
  - LOGGING: runbook lists exact log lines operators must see.
  - Files: `README.md` (operator/release section); touch `.ai-factory/ARCHITECTURE.md` flag boundary note; `.ai-factory/RULES.md` only if a new axiom is required (prefer “flags gate surfaces; migrations stay forward-only”).

- [x] Task 7: Automated tests for flag matrix + R1 smoke linkage (depends on 3, 5)
  - Go: practice routes 404 when `FEATURE_PRACTICE=0`; assess path works when on; review endpoints respect flags.
  - Vitest: router/nav hides practice when features off.
  - Assert R1 gate target includes five-glyph journey tests already owned by Prompt 13/14 plans (call them, don’t fork).
  - LOGGING: `t.Logf` feature env under test; failure messages name the release gate.
  - Files: `internal/httpapi/*_test.go`, `web/tests/**`, `Makefile`.

### Phase 5: Docs checkpoint

- [x] Task 8: Docs honesty + contributor pointers (depends on 6, 7)
  - README release train; verify-docs; docguard still green; no ONNX/ML/confidence marketing.
  - Note that Prompt plans remain implementation owners; this plan owns sequencing/gates only.
  - LOGGING: document `[main] features=` and `[httpapi] feature_disabled` in troubleshooting.
  - Files: `README.md`, `.ai-factory/DESCRIPTION.md` (one sentence on staged flags if accurate), `AGENTS.md` entry for release plan.

## Commit Plan

- **Commit 1** (after tasks 1): `docs: add hiragana5 release train roadmap milestones and sequencing plan`
- **Commit 2** (after tasks 2–4): `feat: add learning feature flags, gating, and cutover metrics`
- **Commit 3** (after tasks 5–7): `test: cover feature-flag matrix and release gate targets`
- **Commit 4** (after task 8): `docs: operator runbook for migration, flags, and R1–R4 gates`

## Acceptance Criteria

1. ROADMAP lists unchecked R1–R4; this plan links to R1; Prompt 08–19 remain historical completed entries.
2. Hard dependency order and overlap resolutions (schema/protocol/UI/security) are documented and match D1–D2; feature plans are referenced, not rewritten.
3. Feature flags exist, are logged at startup, enforce server+UI gating, and can roll back product surfaces without running migration `Down`.
4. Migration policy is backup/restore + forward-only; schema ceilings per release are documented.
5. Observability signals for cutover are defined and covered by tests where code was added.
6. `gate-r1` (or equivalent) is sufficient to accept **trustworthy guided lesson for five hiragana**; R2–R4 gates match D6.
7. Verbose logging follows existing bracket prefixes; no stroke coordinates/passwords/cookies in logs.
8. Docs checkpoint complete; `make verify-docs` / docguard green.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Re-implementing Prompt 08–19 inside this plan | Tasks limited to sequencing, flags, gates, runbooks; reference plan files |
| Migrations already at 0006 on main | Flags still provide kill switches; R1 acceptance does not require disabling DB columns |
| Flag/UI drift (nav shows practice, API 404) | Shared feature payload or mirrored defaults + integration test |
| Operators attempt production `Down` | Runbook forbids it; only backup restore |
| R1 scope creep into i18n/SRS/audio | D0 explicit non-goals; gates exclude those suites |
| Dual “next” semantics 15 vs 16 | `FEATURE_REVIEW=0` preserves mastery-only next |

## Dependencies / Sequencing

- **Requires:** Approved Prompt plans 08–19 (and perimeter baseline plans) as referenced inputs.
- **Unblocks:** Controlled production cutovers R1→R4, flag-based rollback, operator confidence independent of further curriculum prompts.
- **Out of scope:** New characters beyond hiragana5; SM-2/FSRS; collaboration; ONNX/ML; implementing unfinished Prompt task checkboxes (those stay with their plans).
