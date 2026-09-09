# Implementation Plan: Attempt History and Per-Character Mastery (Prompt 15)

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Attempt history and per-character mastery (Prompt 15)"
Rationale: Prompt 11–14 shipped attempt lifecycle, assessment, guided journey, and hub progress labels (`unseen`/`practicing`/`passed`) but learners still cannot browse past attempts, see an explainable mastery summary, get a humble next-character suggestion, or clear stored practice handwriting — this milestone closes that personal-progress gap without SRS or social features.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep Prompt 08–14 entries unchanged (do not uncheck completed items).

## Summary

After the guided hiragana5 MVP, durable rows already exist (`practice_attempts`, `assessment_results`, `user_character_progress`) and the hub shows coarse progress status. Gaps:

- **No attempt history API or UI** — only `GET /api/attempts/{id}` and assessment-by-id; `AttemptRepo` has no list/paginate.
- **Progress is operational, not explainable mastery** — counters update on submit/assess; unused `seen` constant; no derived label with a reason learners can trust as a simple practice aid (not science).
- **Abandoned drafts** are terminal but never counted (good) — undocumented in learner copy; history cannot surface them optionally.
- **No next-character recommendation** beyond lesson list order and “already passed” resume.
- **No privacy/retention control** for practice handwriting (stroke points cascade only with user delete via FK; no self-serve clear).

This plan delivers:

1. **Paginated attempt history** (metadata + assessment summary; **no** stroke points in list payloads).
2. **Explainable per-character mastery** derived from **assessed** attempts only, with stable `reasonCode` + facts for UI/i18n.
3. **Humble next-character recommendation** from lesson order + mastery (no due dates, no SRS).
4. **Privacy controls** — clear practice history / optional stroke-geometry purge policy; document retention.
5. **Hub + history UI** summaries, empty states, EN/JA strings, tests, verbose logs, docs.

**Out of scope:** Streaks; social comparison; leaderboards; teacher/classroom dashboards; SRS / Leitner / due dates / review queue (Prompt 16); pronunciation audio (Prompt 17); changing assess criteria / `T_pass` / correction codes (Prompt 12); free-board WS/`boardRev`; stroke replay canvas in history (keep list summary-only); exporting full handwriting archives; Playwright/Cypress (Vitest + Go tests).

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Schema | `practice_attempts` + assessment tables + `user_character_progress` (migration `0002`); indexes `(user_id, started_at DESC)` and `(user_id, character_id, started_at DESC)` ready for history |
| Progress upsert | `attempt_count` + `practicing` on **submit**; `pass_count` / `passed` sticky on **assess**; **abandon does not touch progress** |
| `ProgressStatusSeen` | Declared in `internal/learn/types.go` but **never written** |
| Progress HTTP | `GET /api/progress?lessonId=` / `?setId=` → status, counts, `lastAttemptId`, `lastPassedAt` |
| Attempt HTTP | create / get / submit / assess / get assessment / abandon — **no list** |
| Repos | `AttemptRepo` lacks `List`; `ProgressRepo` Get/ListForUser only |
| Frontend | Hub shows localized status; journey resume uses progress; `progressApi` / `attemptsApi` have no history client |
| Privacy | User cascade deletes learning rows; no learner “clear history” endpoint; README does not document practice retention |
| Prompt 09 note | Assessed attempts retained by default for Prompt 15; no auto-purge shipped |

## Design Decisions

### D1 — What learners can view (personal only)

| Surface | Visible | Hidden |
|---------|---------|--------|
| Hub row | Glyph, romanization, operational status **or** mastery label, short reason, attempt/pass counts (optional compact) | Other users; ranks; streaks |
| Suggested next | One character + one plain-language why | “Due”, “optimal”, scientific confidence |
| History list | Time, glyph/characterId, terminal status, pass/fail, match score + `scoreKind`, ≤2 feedback codes/messages (display via i18n map), link into journey | Stroke coordinates/points; diagnostics/`candidates`; board strokes |
| History empty | Localized empty copy + CTA to practice | Fake sample rows |
| Detail | Existing `GET …/assessment` (and attempt metadata) | New ink replay UI |

Ownership: every query `AND user_id=?`; foreign ids → `404 not_found` (no leakage).

### D2 — Retries, abandons, and what counts

| Event | History | `attempt_count` | Mastery inputs |
|-------|---------|-----------------|----------------|
| Create draft | No (until terminal, optional) | No | No |
| Abandon draft | Optional (`status=abandoned`); default history filter **excludes** | No (unchanged) | **No** |
| Submit | Not listed as assessed until assess | +1 | No until assess |
| Assess pass/fail | Yes (`status=assessed`) | already counted on submit | **Yes** |
| Retry | New attempt row (immutable prior) | New submit increments again | New assessed outcome |

Rules to document in README + UI microcopy:

- **Abandoned drafts never hurt or help mastery.**
- **Only assessed attempts** change mastery / recommendation signals.
- Stuck `submitted` without assess is rare (client always chains assess); if present, treat like incomplete — **exclude from mastery** until assessed (do not invent pass/fail).

### D3 — Explainable mastery (engineering heuristic, not science)

Keep operational `progress.status` (`practicing` / `passed` sticky) for resume compatibility. Add a **derived** mastery projection (compute-on-read preferred; no SRS columns):

```text
mastery.state:  not_started | learning | passed_once | steady
mastery.reasonCode: stable snake_case for i18n
mastery.facts: { assessedCount, passCount, failCount, lastPass, lastAssessedAt?, consecutivePassesEnding }
```

Deterministic rules (assessed attempts only, ordered by `assessed_at ASC` / `id ASC` for ties):

| `state` | When |
|---------|------|
| `not_started` | `assessedCount == 0` |
| `learning` | `assessedCount ≥ 1` and `passCount == 0` |
| `passed_once` | `passCount ≥ 1` and `consecutivePassesEnding < 2` |
| `steady` | `consecutivePassesEnding ≥ 2` |

`consecutivePassesEnding` = length of the trailing pass run on the assessed timeline (0 if last assessed failed).

Honesty requirements:

- UI + README: mastery is a **simple practice summary from your completed attempts**, not a validated proficiency or ML score.
- Never label as confidence, grade, belt, or SRS stage.
- `scoreKind` on history rows remains `match`.
- `go test ./internal/docguard` must stay green; add/adjust honesty strings if docs mention mastery.

Optional: if `progress.status == passed` but `assessedCount == 0` (should not happen) — treat as `not_started` and WARN log data inconsistency.

Unused `seen`: do **not** introduce learner-facing `seen` in this plan; leave constant or remove in a tiny cleanup if touch-safe — prefer leave unused to avoid migration churn.

### D4 — Next character without pretending science

`GET /api/progress/next` (or field on enriched progress list) for a published `lessonId` (default `lesson:hiragana5`):

Priority (first match in lesson `position` order):

1. `mastery.state == not_started`
2. `mastery.state == learning`
3. `mastery.state == passed_once` (encourage another successful attempt toward `steady`)
4. Else `suggestion: null` with `reasonCode: all_steady` — UI: “All five look steady — pick any character to keep practicing” (no forced review schedule)

Response sketch:

```json
{
  "lessonId": "lesson:hiragana5",
  "characterId": "hira:い",
  "glyph": "い",
  "reasonCode": "first_not_started",
  "masteryState": "not_started"
}
```

Copy maps `reasonCode` → EN/JA. No timezone, no `dueAt`, no interval math (Prompt 16).

### D5 — APIs and queries

All mutating routes: auth + CSRF. GETs: auth only. No `boardRev`.

#### Enrich progress (prefer extend existing)

`GET /api/progress` items gain:

```json
{
  "characterId": "hira:あ",
  "status": "passed",
  "attemptCount": 3,
  "passCount": 2,
  "lastAttemptId": 12,
  "lastPassedAt": "...",
  "updatedAt": "...",
  "mastery": {
    "state": "steady",
    "reasonCode": "two_consecutive_passes",
    "assessedCount": 3,
    "failCount": 1,
    "consecutivePassesEnding": 2
  }
}
```

Implementation: batch-load assessed outcomes for requested character set (avoid N+1); log timing DEBUG.

#### Attempt history

`GET /api/attempts?lessonId=&characterId=&status=assessed&limit=20&cursor=`

| Query | Rules |
|-------|-------|
| `status` | Default `assessed`. Allow `abandoned`, or `assessed,abandoned`. Reject `draft`/`submitted` in list (incomplete; use get-by-id). |
| `limit` | Default 20, max 50 |
| `cursor` | Opaque `started_at + id` (or base64 of both); stable DESC order |
| Filters | `characterId` and/or `lessonId` (published); always scoped to session user |

Item sketch (no points):

```json
{
  "id": 12,
  "characterId": "hira:あ",
  "glyph": "あ",
  "lessonId": "lesson:hiragana5",
  "status": "assessed",
  "startedAt": "...",
  "assessedAt": "...",
  "pass": true,
  "score": 0.82,
  "scoreKind": "match",
  "feedback": [{"rank":1,"code":"...","message":"..."}]
}
```

Abandoned items: omit pass/score/feedback.

Response: `{ items, nextCursor?, limit }`. Empty → `items: []` (200).

#### Next suggestion

`GET /api/progress/next?lessonId=lesson:hiragana5` → D4 payload or `{ characterId: null, reasonCode: "all_steady" }`.

#### Privacy / retention

| Control | Behavior |
|---------|----------|
| `DELETE /api/practice-data` (or `POST …/clear`) | CSRF; deletes **this user’s** `practice_attempts` (CASCADE strokes/assessments/feedback), resets/deletes `user_character_progress` rows; does **not** delete free-board strokes or account |
| Retention default | Keep until user clears or account deleted |
| Optional geometry purge | If cheap: `DELETE` attempt stroke points older than `N` days while keeping attempt + assessment summary — **only if** implemented behind explicit env (e.g. `PRACTICE_STROKE_RETENTION_DAYS`); default **off** / unlimited. Document clearly. |
| Logging | INFO userID + deleted attempt counts; **never** coordinates |

No teacher export; no cross-user admin UI.

### D6 — UI summaries, pagination, empty states

| UI | Behavior |
|----|----------|
| Practice hub | Show mastery label (localized) under each glyph; “Suggested next” banner with reason + link; quiet link “Attempt history” |
| History page | `/#/practice/history` (optional `?characterId=`); infinite scroll or “Load more” via `nextCursor`; filter chips: All assessed / This character |
| Empty history | “No completed attempts yet” + link to hub / suggested next |
| Empty suggestion | When all `steady`, banner still explains; no red urgency |
| Clear data | Confirm dialog (destructive); success refreshes hub/history; localized |
| A11y | Live region for clear result; list as list semantics; focus management on load-more; reuse Prompt 14 tokens/touch targets |
| i18n | New keys for mastery states, reasonCodes, history chrome, privacy confirm — EN/JA in `web/src/i18n` |

Do **not** add dashboard card grids, streak flames, or comparison bars.

### D7 — Logging (verbose)

| Prefix | Level | What |
|--------|-------|------|
| `[learn.mastery]` | DEBUG | userID, characterID, state, reasonCode, assessedCount (no scores dump spam) |
| `[learn.AttemptRepo.List]` | DEBUG | userID, filters, limit, resultCount, hasNext |
| `[httpapi.Attempts.List]` | INFO | userID, count, filters; WARN bad cursor/limit |
| `[httpapi.Progress.Next]` | INFO | userID, lessonId, characterId\|null, reasonCode |
| `[httpapi.PracticeData.Clear]` | INFO | userID, attemptsDeleted, progressRowsCleared |
| `[progressApi]` / `[attemptsApi]` | DEV debug | list/next/clear paths; never stroke bodies |

Production: no PII beyond userID; no point arrays.

### D8 — Testing strategy

**Go:**

- Mastery pure function table tests: empty, all fail, one pass, fail-after-pass, two consecutive passes, abandon-only ignored, submitted-only ignored.
- `AttemptRepo.List` pagination + filters + ownership.
- HTTP contract: list omits points; foreign user 404; default status; cursor round-trip; clear deletes cascades and progress.
- Next suggestion order with seeded lesson placements.
- Regression: abandon still does not change progress counters; assess sticky `passed`.

**Vue/Vitest:**

- Hub renders mastery + suggested next (mocked APIs).
- History empty + paginated load-more.
- Clear confirm flow.
- i18n keys for mastery/reasonCodes.
- Axe smoke on hub + history if cheap (extend parity suite lightly).

**Docs/honesty:** README API + privacy; ARCHITECTURE/AGENTS/DESCRIPTION; `docguard` green.

## Architecture touchpoints

```text
internal/learn/           # mastery types + pure DeriveMastery; AttemptList query types
internal/learn/repository.go  # AttemptRepo.List; optional Progress enrichment helpers
internal/db/learn_store.go    # ListAttempts SQL; clear practice data tx; batch assessed outcomes
internal/httpapi/attempts.go  # ListAttempts handler
internal/httpapi/curriculum.go / progress_*.go  # enrich ListProgress; Next; Clear
cmd/server/main.go        # route wiring
web/src/services/attemptsApi.ts   # listAttempts
web/src/services/progressApi.ts   # mastery fields + next + clear
web/src/pages/PracticeHubPage.vue
web/src/pages/PracticeHistoryPage.vue  # NEW
web/src/router/index.ts
web/src/i18n/locales/{en,ja}.ts
web/tests/...             # unit/contract/integration
README.md, ARCHITECTURE.md, AGENTS.md, DESCRIPTION.md, RULES.md (axioms: mastery honesty + no SRS)
```

**Dependency rules unchanged:** recognize stays scoring-only; attempts ≠ board; no SRS tables; practice ink not on WS.

## Acceptance Scenarios

### S1 — History after practice

1. Complete ≥2 assessed attempts on あ (mix pass/fail).
2. Open history → rows show times, pass/fail, match score, feedback; network payloads contain **no** `points`.
3. Pagination: with limit=1, `nextCursor` loads older row.
4. Abandoned-only drafts do not appear under default filter and do not change mastery.

### S2 — Mastery explainability

1. New character → `not_started` / reason `no_assessed_attempts`.
2. Fail only → `learning`.
3. One pass → `passed_once`; second consecutive pass → `steady`.
4. Fail after passes → leaves `steady` if trailing consecutive passes &lt; 2 (typically `passed_once`); UI reason mentions recent miss without shaming streaks.
5. Copy never says “scientifically validated,” “AI confidence,” or “due for review.”

### S3 — Next suggestion

1. Fresh user → suggests first lesson character (`あ`) with `first_not_started`.
2. After あ `steady` but い untouched → suggests い.
3. All `steady` → null suggestion + calm `all_steady` copy.

### S4 — Privacy clear

1. User clears practice data → history empty, hub mastery `not_started`, board strokes unchanged, account still logged in.
2. Foreign session cannot clear or list another user’s attempts.

### S5 — Regression

1. Guided journey submit/assess/retry unchanged; sticky `passed` resume still works.
2. Free-board recognize still playground; `docguard` green.

## Tasks

### Phase 1: Domain mastery + list queries

- [x] Task 1: Add pure mastery derivation (`DeriveMastery(assessedOutcomes []…)`) + reasonCode constants in `internal/learn`; document rules in godoc as engineering heuristic. Table-driven unit tests for D3 cases including abandons excluded by caller.

  LOGGING: `[learn.mastery]` DEBUG on derive when called from store/HTTP (gate noisy loops); tests may assert reasonCodes only.

  Files: `internal/learn/mastery.go` (new), `internal/learn/mastery_test.go` (new), `internal/learn/types.go` if needed

- [x] Task 2: Extend `AttemptRepo` with paginated `List(userID, filter)` joining assessment summary for assessed rows; ownership-enforced. Add `ClearPracticeData(userID)` transactional delete of attempts (CASCADE) + progress rows. Wire SQL using existing indexes; no stroke points in list SELECT.

  LOGGING: `[learn.AttemptRepo.List]` DEBUG filters/counts; `[learn.PracticeData.Clear]` INFO deleted counts; ERROR on tx failure.

  Files: `internal/learn/repository.go`, `internal/db/learn_store.go`, `internal/db/learn_store_*_test.go`

  Depends on: 1 (for optional batch helper used by progress enrichment)

<!-- Commit checkpoint: tasks 1–2 -->

### Phase 2: HTTP APIs

- [x] Task 3: Implement `GET /api/attempts` (list + cursor pagination + filters), enrich `GET /api/progress` with `mastery`, add `GET /api/progress/next`, add `DELETE /api/practice-data` (CSRF). Register routes in `cmd/server`. Handler tests + e2e coverage for list/next/clear; assert list JSON has no point arrays.

  LOGGING: `[httpapi.Attempts.List]`, `[httpapi.Progress.List]` (mastery attach counts), `[httpapi.Progress.Next]`, `[httpapi.PracticeData.Clear]` per D7; WARN invalid cursor/limit/status.

  Files: `internal/httpapi/attempts.go`, `internal/httpapi/curriculum.go` (or new `progress.go`), `cmd/server/main.go`, `internal/httpapi/*_test.go`, `attempts_e2e_test.go`

  Depends on: 2

<!-- Commit checkpoint: task 3 -->

### Phase 3: Frontend summaries + privacy UI

- [x] Task 4: Extend `progressApi` / `attemptsApi`; add history page + router; hub mastery labels, suggested-next banner, history link, clear-practice confirm. Empty/loading/error states; EN/JA strings for mastery states + reasonCodes + privacy copy.

  LOGGING: DEV `[progressApi]` / `[attemptsApi]` / `[PracticeHistoryPage]` debug for list cursor and clear result; never log feedback dumps at INFO.

  Files: `web/src/services/progressApi.ts`, `web/src/services/attemptsApi.ts`, `web/src/pages/PracticeHistoryPage.vue`, `web/src/pages/PracticeHubPage.vue`, `web/src/router/index.ts`, `web/src/i18n/locales/en.ts`, `web/src/i18n/locales/ja.ts`, `web/src/components/AppShell.vue` if nav link needed

  Depends on: 3

<!-- Commit checkpoint: task 4 -->

### Phase 4: Tests + docs

- [x] Task 5: Complete Vitest contract/integration for history pagination, hub mastery/next empty states, clear flow; Go mastery/list/next/clear gaps filled; optional light axe on history. Fix any Prompt 14 i18n key regressions.

  LOGGING: prefer behavior assertions; spy logs only where existing patterns do.

  Files: `web/tests/contract/*`, `web/tests/integration/*`, `web/tests/unit/*`, Go tests from tasks 1–3

  Depends on: 4

- [x] Task 6: Docs checkpoint — README (history/mastery/next/clear APIs, honesty wording, retention default, optional stroke retention env if shipped), `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, `.ai-factory/DESCRIPTION.md`, `.ai-factory/RULES.md` axiom that mastery is heuristic and SRS remains out of scope. `go test ./internal/docguard`. Note Prompt 15 milestone still unchecked until verify.

  LOGGING: n/a beyond docguard / existing prefixes documented in README table.

  Files: `README.md`, `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, `.ai-factory/DESCRIPTION.md`, `.ai-factory/RULES.md`, `internal/docguard/*` if strings asserted

  Depends on: 5

<!-- Commit checkpoint: tasks 5–6 -->

## Commit Plan

- **Commit 1** (after tasks 1–2): `feat(learn): derive explainable mastery and list/clear practice attempts`
- **Commit 2** (after task 3): `feat(api): attempt history, mastery progress, next suggestion, clear practice data`
- **Commit 3** (after task 4): `feat(web): hub mastery, attempt history, and privacy clear UI`
- **Commit 4** (after tasks 5–6): `docs: document attempt history, mastery heuristic, and practice retention`

## Risks / Notes

- **Sticky `passed` vs mastery `learning`:** operational status can remain `passed` after a later fail while mastery drops to `passed_once` — intentional; UI should prefer **mastery** for coaching chrome and keep `status` for journey “already passed” completion affordances. Document this split.
- **Compute-on-read cost:** five characters × small history is fine; if list enrichment scans all assessed rows, cap per-character window (e.g. last 20) for mastery facts and document the window.
- **Prompt 16:** may consume `mastery.state` / assessed timeline — avoid schema locks that assume Leitner boxes; no `due_at` columns here.
- **Optional stroke retention env:** ship only if Task 2/3 remain small; otherwise document “keep until clear” only and defer timed purge.
