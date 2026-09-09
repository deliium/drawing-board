# Implementation Plan: Lightweight Spaced-Review Queue (Prompt 16)

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Lightweight spaced-review queue (Prompt 16)"
Rationale: Prompt 15 shipped explainable mastery and a humble next-character pick without due dates; learners still lack a forgiving review queue that says which character to practice next and why after characters become `steady` — this milestone adds the smallest justified schedule on assessed attempts without streaks, notifications, or gamification.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep Prompt 08–15 entries unchanged (do not uncheck completed items).

## Summary

Prompt 15 left `GET /api/progress/next` as a mastery-priority walk (`not_started` → `learning` → `passed_once` → calm `all_steady`) with **no** `dueAt`, intervals, or review state. Once all five hiragana look `steady`, the product has nothing useful to suggest. Gaps:

- **No durable review schedule** — `user_character_progress` has counters/`last_passed_at` only; no box, interval, or `due_at`.
- **`all_steady` dead-end** — suggestion goes null; no overdue/soon-due signal.
- **No controllable clock** — `LearnStore` uses `time.Now().UTC()` inline; schedule tests cannot freeze time.
- **Honesty/docs axioms** still say “no SRS / due dates without a dedicated plan” — this plan **is** that plan; axioms must be rewritten carefully (lightweight Leitner-style schedule allowed; streaks/notifications/SM-2 still forbidden).

This plan delivers:

1. **Algorithm choice (Leitner-style fixed boxes)** — compare vs SM-2; pick Leitner as least complex for five characters.
2. **Automatic schedule updates** from assessed pass/fail only (no manual Again/Hard/Good/Easy UI).
3. **UTC `due_at` + box state** on progress; wall-clock intervals; overdue without penalty.
4. **Next + why** extended to prefer overdue reviews, then new introduction, then learning — with explainable `reasonCode`.
5. **Hub UI** due/ready labels + calm empty/caught-up states; EN/JA; tests with fake clock; verbose logs; docs.

**Out of scope:** Streaks; push/email notifications; daily goals; XP/badges; SM-2/FSRS ease factors or rating buttons; classroom/cohort tables; pronunciation audio (Prompt 17); changing assess criteria / `T_pass` / corrections (Prompt 12); free-board WS/`boardRev`; stroke points in history; Playwright/Cypress.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Schema | `user_character_progress`: `status`, `attempt_count`, `pass_count`, `last_attempt_id`, `last_passed_at`, `updated_at` — **no** review columns (migration `0002`) |
| Mastery | Compute-on-read `DeriveMastery` from assessed outcomes; states `not_started` / `learning` / `passed_once` / `steady` |
| Next | `SuggestNextCharacter` mastery priority only; `all_steady` → null `characterId` |
| Assess path | `SaveResult` → `upsertProgressOnAssessTx` updates sticky `passed` + counts with `time.Now().UTC()` |
| Clear | `ClearPracticeData` deletes attempts + progress rows (will clear schedule with progress) |
| Clock | No learn-domain clock injection; `limits.Limiter` already uses injectable `now` (pattern to mirror) |
| UI | Hub suggested-next banner + mastery labels; no due/overdue chrome |
| Docs/rules | README/ARCHITECTURE/RULES: mastery ≠ SRS; due dates require dedicated plan |

## Design Decisions

### D0 — Algorithm comparison and selection

| Option | How it works | Fit for hiragana5 |
|--------|--------------|-------------------|
| **Leitner-style (chosen)** | Small integer **box** (0–3); fixed interval table; pass → promote; fail → demote one box (forgiving); `due_at = now + interval(box)` | Tiny set (5 glyphs); binary assess outcome already exists; reason copy maps cleanly to box/due; few knobs; deterministic tests trivial |
| **SM-2 (rejected)** | Ease factor + quality grades (0–5) + growing intervals | Needs rating UI or invented grades from score; ease is hard to explain honestly; over-parameterized for five characters; risks “science” marketing pressure |

**Decision:** Implement a **Leitner-style fixed-interval queue**, not SM-2/FSRS. Product copy may say “simple review schedule” / “practice reminder timing” — **never** “scientifically optimized SRS,” “Anki-grade,” or “SM-2.” Keep `docguard` honest.

**Why not “mastery-only delays without boxes”?** Boxes give a stable, inspectable state for APIs/UI/tests (`box`, `dueAt`) and survive a single fail without recomputing from the full timeline each write. Mastery remains the **explainable practice summary**; the schedule is the **when**.

### D1 — Scheduling inputs

| Input | Role |
|-------|------|
| Assessed `pass` boolean | Sole automatic rating (promote / demote) |
| Current `box` on progress | Stage before update (default 0 if never scheduled) |
| Controllable `now` (UTC) | Compute `due_at`; tests freeze/advance |
| Lesson placements | Order for introducing `not_started` and tie-breaks |
| Mastery (compute-on-read) | Still shown on hub; used only as secondary signal for introduction priority, not for interval math |

**Not inputs:** match score magnitude, consecutive-pass streak length as a gamified counter, wall-clock “calendar day streaks,” user-typed ratings, timezone preference rows.

### D2 — Automatic outcomes (no rating UI)

On every successful `SaveResult` (assessed attempt only):

| Outcome | Box transition | New `due_at` |
|---------|----------------|--------------|
| **Pass** | `box = min(box+1, BoxMax)` | `now + Interval(box)` after promotion |
| **Fail** | `box = max(box-1, 0)` (forgiving demote-by-one, **not** reset to 0 unless already 0/1) | `now + Interval(box)` after demotion — usually soon |

Recommended constants (tunable in one Go file, documented in README):

| Box | Meaning (learner-facing paraphrase) | Interval after landing in box |
|-----|-------------------------------------|-------------------------------|
| 0 | Fresh / needs practice soon | **0** (due immediately / next look) |
| 1 | Early review | **1 day** (`24h`) |
| 2 | Settling | **3 days** (`72h`) |
| 3 | Comfortable review | **7 days** (`168h`) |

`BoxMax = 3`. First-ever pass from unscheduled row: start at box 0 → promote to 1 → due in 1 day. First-ever fail: stay/land box 0, due now.

**Abandoned / submitted-only:** never touch schedule (same as mastery).

**Manual early practice:** allowed anytime via hub/journey. Assess still runs the same transition from **current** box using **assess `now`**, not the previous `due_at` (practicing early is fine; no “broke the schedule” shame).

### D3 — Due dates, timezone, missed days

| Topic | Rule |
|-------|------|
| Storage | `due_at TIMESTAMP` UTC (nullable until first assessed attempt that schedules the character) |
| Comparison | Character is **due** iff `due_at IS NOT NULL AND due_at <= now` |
| Interval unit | **Wall-clock durations** from assess time (`24h` / `72h` / `168h`), **not** local midnight boundaries — avoids storing IANA TZ for MVP |
| Display | API returns RFC3339 UTC; Vue formats with browser locale/time zone |
| Missed days | Overdue is **ready when you are** — no penalty multiplier, no pile-on urgency, no “you broke a streak.” Sorting among overdue: earliest `due_at` first, then lesson `position` |
| Null `due_at` | Unintroduced / never assessed for schedule — treated as not in review queue; introduction path uses mastery `not_started` |

Do **not** ship notifications, badges for catching up, or red “overdue!” pressure chrome.

### D4 — New-character introduction vs reviews

`GET /api/progress/next` priority (first match):

1. **Due review** — among lesson characters with `due_at <= now`, pick earliest `due_at`, tie-break lesson `position`. `reasonCode: due_review` (or `overdue_review` if `now - due_at >= 24h` — optional single code is fine if copy stays calm).
2. **Introduce new** — first lesson character with mastery `not_started` (and no schedule / null due). `reasonCode: first_not_started` (reuse Prompt 15).
3. **Continue learning** — mastery `learning` (fails, not yet passed). Prefer earliest due if scheduled, else lesson order. `reasonCode: continue_learning`.
4. **Encourage steadiness** — mastery `passed_once` and not due yet? Prefer soonest future due among them, else lesson order. `reasonCode: encourage_steady`.
5. **Caught up** — nothing due and no introduction/learning left: `characterId: null`, `reasonCode: all_caught_up` (replace calm dead-end `all_steady` for schedule-aware copy; keep `all_steady` as mastery-only fact on progress items). Include `nextDueAt` / `nextDueCharacterId` when a future review exists so UI can say “Next review around …”.

**Introduction policy for five glyphs:** do **not** hard-cap concurrent new cards (set is tiny). Still prefer **due reviews before** introducing the next untouched character so early vowels get a light revisit.

### D5 — Manual practice

- Hub/character routes remain freely clickable — schedule never locks a glyph.
- Practicing a non-due character updates box/`due_at` on assess exactly as D2.
- UI may show quiet “Not due yet — practice anyway” on character page if `due_at > now`; never block Submit/Assess.
- Clear practice data wipes progress (including box/`due_at`) — same privacy path as Prompt 15.

### D6 — Data model

Additive migration `0005_review_schedule` (name flexible):

```sql
ALTER TABLE user_character_progress ADD COLUMN review_box INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_character_progress ADD COLUMN due_at TIMESTAMP;
-- optional denormalized for debugging / UI without join:
ALTER TABLE user_character_progress ADD COLUMN last_reviewed_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_progress_user_due ON user_character_progress(user_id, due_at);
```

| Column | Notes |
|--------|-------|
| `review_box` | 0–3; default 0 |
| `due_at` | UTC; NULL = not in review queue yet |
| `last_reviewed_at` | Set on each assess that updates schedule (optional but useful) |

**No** separate `review_log` table required for MVP (attempt history already is the audit trail). **No** ease/stability columns.

Domain type `learn.Progress` gains fields; JSON progress items expose:

```json
{
  "characterId": "hira:あ",
  "mastery": { "...": "..." },
  "review": {
    "box": 2,
    "dueAt": "2026-09-12T15:00:00Z",
    "isDue": true,
    "intervalDays": 3
  }
}
```

`intervalDays` is derived from current box constants (informational), not a stored free-form value.

`DELETE /api/practice-data` already deletes progress rows — no extra clear logic beyond migration defaults for fresh users.

### D7 — Clock injection

Introduce a small `learn.Clock` / `func() time.Time` (UTC) used by:

- `upsertProgressOnAssess` schedule update
- `SuggestNext` / due comparisons in HTTP or pure functions
- Tests: fixed instant + `Advance(d)`

Pattern: mirror `limits.Limiter.now`. Wire default `time.Now().UTC` in `LearnStore` / API; tests set fake clock on store or pass `now` into pure `ApplyReviewOutcome(box, pass, now) (newBox, dueAt)`.

**Pure functions (preferred):**

```text
ApplyReviewOutcome(box int, pass bool, now time.Time) (newBox int, dueAt time.Time)
IsDue(dueAt *time.Time, now time.Time) bool
SuggestNextWithReview(lessonOrder, masteryBy, reviewBy, now) → id, glyph, reasonCode, …
```

Keep interval table in one constant map for docguard-friendly documentation.

### D8 — APIs

All mutating: auth + CSRF. GETs: auth. No `boardRev`.

| Endpoint | Change |
|----------|--------|
| `GET /api/progress` | Add `review` object per item (`box`, `dueAt`, `isDue`, `intervalDays`) |
| `GET /api/progress/next` | Schedule-aware priority (D4); response gains optional `dueAt`, `reviewBox`, `nextDueAt`, `nextDueCharacterId`; new/updated `reasonCode`s |
| Assess (`POST …/assess`) | Side effect: update box/`due_at` in same tx as existing progress upsert |
| `DELETE /api/practice-data` | Unchanged semantics (progress wipe includes schedule) |

Optional (only if hub needs a list without N calls): `GET /api/progress/due?lessonId=` → `{ items:[{ characterId, glyph, dueAt, box, reasonCode }] }` sorted by `dueAt`. Prefer enriching list + next first; add due-list **only** if UI cannot derive from `GET /api/progress`.

Next response sketch:

```json
{
  "lessonId": "lesson:hiragana5",
  "characterId": "hira:う",
  "glyph": "う",
  "reasonCode": "due_review",
  "masteryState": "steady",
  "dueAt": "2026-09-08T12:00:00Z",
  "reviewBox": 2,
  "nextDueAt": null,
  "nextDueCharacterId": null
}
```

Caught up with a future review:

```json
{
  "lessonId": "lesson:hiragana5",
  "characterId": null,
  "reasonCode": "all_caught_up",
  "masteryState": "steady",
  "nextDueAt": "2026-09-14T12:00:00Z",
  "nextDueCharacterId": "hira:あ"
}
```

### D9 — UI states

| Surface | Behavior |
|---------|----------|
| Hub suggested-next | Prefer schedule reasons; calm copy for `due_review` / `all_caught_up` (+ optional “next review around {date}”) |
| Hub glyph row | Quiet review hint when `review.isDue` (“Ready for a light review”) — no red urgency, no streak flames |
| Character journey | Optional non-blocking note if not due yet |
| Empty / caught up | Friendly; invite free pick; never “you’re behind” |
| i18n | New EN/JA keys for review reasons, due hint, caught-up + nextDue; keep mastery strings |
| A11y | Suggested-next stays `aria-live="polite"`; dates in accessible text, not color-only |

Do **not** add notification permission prompts, countdown guilt meters, or gamified box ladders as primary chrome (box may appear in subtle secondary text if useful for debug/honesty — prefer plain language).

### D10 — Logging (verbose)

| Prefix | Level | What |
|--------|-------|------|
| `[learn.review]` | DEBUG | userID, characterID, pass, oldBox→newBox, dueAt (RFC3339); no scores dump |
| `[learn.review.Suggest]` | DEBUG | userID, lessonId, reasonCode, characterId\|null, dueCount |
| `[httpapi.Progress.Next]` | INFO | extend existing line with reasonCode + dueAt/box when present |
| `[httpapi.Progress.List]` | DEBUG | how many items marked `isDue` |
| `[learn.AssessmentRepo.SaveResult]` | INFO/DEBUG | keep pass/score; add box/dueAt on schedule update |
| `[progressApi]` | DEV debug | next/list review fields; never stroke bodies |

Production: userID only; no point arrays.

### D11 — Testing strategy (deterministic clock)

**Go:**

- Table tests for `ApplyReviewOutcome`: promote/demote clamps; intervals; fail from box 3 → 2 with 3-day due; pass from 0 → 1 with 1-day due.
- `SuggestNextWithReview` with frozen `now`: overdue beats not_started; earliest due wins; null due introduction; `all_caught_up` + `nextDueAt`.
- `SaveResult` integration with fake clock: assess pass writes `review_box`/`due_at`; second assess after `Advance` updates correctly; abandon does not schedule.
- HTTP: progress JSON includes `review`; next returns `due_review`; clear wipes due fields; foreign user isolation unchanged.
- Regression: mastery derivation unchanged; sticky `passed` unchanged; list attempts still omit points.

**Vue/Vitest:**

- Hub shows due hint + schedule reason copy (mocked progress/next).
- Caught-up banner with optional nextDue text.
- i18n keys present EN/JA.
- Light axe smoke if cheap (hub).

**Docs:** README schedule honesty; RULES/ARCHITECTURE allow Leitner-style personal review, still forbid streaks/notifications/SM-2 marketing; `docguard` green.

## Architecture touchpoints

```text
internal/db/migrations/0005_review_schedule.go   # NEW — box + due_at (+ index)
internal/db/migrations/migrations.go             # register 0005
internal/learn/review.go                         # NEW — boxes, intervals, ApplyReviewOutcome, SuggestNextWithReview
internal/learn/review_test.go                    # NEW — fake clock table tests
internal/learn/mastery.go                        # keep DeriveMastery; next may move to review.go or call into it
internal/learn/types.go                          # Progress.ReviewBox, DueAt, LastReviewedAt
internal/learn/repository.go                     # Progress fields; no new repo required if columns on progress
internal/db/learn_store.go                       # upsertProgressOnAssessTx schedule; list progress SELECT; injectable now
internal/httpapi/progress_history.go             # enrich review; next uses schedule suggest
internal/httpapi/attempts.go                     # assess path inherits store update
cmd/server/main.go                               # routes unchanged unless due-list added
web/src/services/progressApi.ts                  # review types; next fields
web/src/pages/PracticeHubPage.vue                # due hints + caught-up copy
web/src/i18n/locales/{en,ja}.ts
web/tests/...                                    # hub/next/review contract + unit
README.md, ARCHITECTURE.md, AGENTS.md, DESCRIPTION.md, RULES.md, docguard
```

**Dependency rules:** recognize stays scoring-only; schedule lives in `learn` + `db` progress upsert; practice ink still not on WS; no classroom tables.

## Acceptance Scenarios

### S1 — First passes create a gentle schedule

1. Fresh user assesses あ pass → `review_box=1`, `due_at ≈ now+24h`.
2. Hub next still prefers other `not_started` characters before that future due.
3. After freezing clock past `due_at`, next returns あ with `due_review` and plain-language why.

### S2 — Fail is forgiving

1. Character in box 3 fails assess → box 2 (not 0), due in 3 days (or interval for box 2).
2. Copy mentions needing another look — no streak shame, no “reset to zero” messaging.

### S3 — Missed days

1. Advance clock 10 days past due → still one calm due suggestion; no penalty interval blow-up; no notification.

### S4 — Manual early practice

1. Practice a character with `due_at` in the future → assess allowed; schedule recomputed from assess `now`.

### S5 — Caught up

1. All five scheduled with future `due_at`, mastery steady → `all_caught_up` + `nextDueAt` when applicable; UI invites free pick.

### S6 — Privacy + regression

1. Clear practice data → no due/box left; board strokes intact.
2. Mastery labels, history list (no points), sticky `passed`, assess corrections unchanged.
3. `docguard` green; docs do not call the queue SM-2 or calibrated science.

## Tasks

### Phase 1: Domain schedule + migration + clock

- [x] Task 1: Add pure Leitner helpers in `internal/learn` (`ApplyReviewOutcome`, interval table, `IsDue`, reasonCode constants for `due_review` / `all_caught_up`, optional `SuggestNextWithReview`). Document as engineering heuristic (not SM-2). Table-driven tests with fixed `now` and `Advance`.

  LOGGING: `[learn.review]` DEBUG on apply when called from store (gate in loops); tests assert boxes/dueAt/reasonCodes.

  Files: `internal/learn/review.go` (new), `internal/learn/review_test.go` (new), `internal/learn/types.go`, optionally slim `mastery.go` next helper delegation

- [x] Task 2: Migration `0005` add `review_box`, `due_at`, `last_reviewed_at` + index; extend `Progress` scan/upsert; inject `LearnStore` clock (`now func() time.Time`, default UTC); update `upsertProgressOnAssessTx` to apply review outcome in the same tx; ensure `ClearPracticeData` still wipes rows. Store tests with fake clock.

  LOGGING: `[learn.review]` on upsert; `[learn.AssessmentRepo.SaveResult]` include newBox/dueAt at DEBUG; ERROR on tx failure.

  Files: `internal/db/migrations/0005_*.go`, `migrations.go`, `internal/db/learn_store.go`, `internal/db/learn_store_*_test.go`, `internal/learn/repository.go` if signatures need `now`

  Depends on: 1

<!-- Commit checkpoint: tasks 1–2 -->

### Phase 2: HTTP next/progress enrichment

- [x] Task 3: Enrich `GET /api/progress` with `review`; rewrite `GetProgressNext` to schedule-aware priority (D4) using store clock; keep ownership/`lessonId` validation; handler + e2e tests with fake clock (overdue vs introduce; `all_caught_up` + `nextDueAt`). Do not add rating endpoints.

  LOGGING: `[httpapi.Progress.Next]` INFO with reasonCode/characterId/dueAt; `[httpapi.Progress.List]` DEBUG due counts; WARN invalid lessonId as today.

  Files: `internal/httpapi/progress_history.go`, `internal/httpapi/progress_history_test.go`, `internal/httpapi/curriculum.go` (progress list enrichment if shared), `attempts_e2e_test.go` if needed

  Depends on: 2

<!-- Commit checkpoint: task 3 -->

### Phase 3: Hub UI + i18n

- [x] Task 4: Extend `progressApi` types; hub due hints + suggested-next reasons (`due_review`, `all_caught_up`, nextDue sentence); character-page optional non-blocking “not due yet”; EN/JA strings; empty/loading/error unchanged patterns.

  LOGGING: DEV `[progressApi]` / `[PracticeHubPage]` debug for next reason/dueAt; never log assessment dumps at INFO.

  Files: `web/src/services/progressApi.ts`, `web/src/pages/PracticeHubPage.vue`, `web/src/pages/PracticeCharacterPage.vue` (light touch), `web/src/i18n/locales/en.ts`, `web/src/i18n/locales/ja.ts`

  Depends on: 3

<!-- Commit checkpoint: task 4 -->

### Phase 4: Tests + docs

- [x] Task 5: Fill Vitest hub/contract coverage for due/caught-up; Go gaps for migration/upsert/next; regression mastery/history/clear; optional axe hub smoke.

  LOGGING: prefer behavior assertions; spy only where existing patterns do.

  Files: `web/tests/unit/*`, `web/tests/contract/*`, `web/tests/integration/*`, Go tests from tasks 1–3

  Depends on: 4

- [x] Task 6: Docs checkpoint — README (Leitner-style personal review, box/interval table, UTC wall-clock due, missed-day forgiveness, auto pass/fail ratings, APIs, honesty vs SM-2/streaks/notifications); update `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, `.ai-factory/DESCRIPTION.md`, `.ai-factory/RULES.md` so dedicated-plan gate is satisfied and streaks/notifications remain forbidden; `go test ./internal/docguard`. Leave Prompt 16 milestone unchecked until verify.

  LOGGING: n/a beyond documenting new `[learn.review]` prefixes in README table.

  Files: `README.md`, `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, `.ai-factory/DESCRIPTION.md`, `.ai-factory/RULES.md`, `internal/docguard/*` as needed

  Depends on: 5

<!-- Commit checkpoint: tasks 5–6 -->

## Commit Plan

- **Commit 1** (after tasks 1–2): `feat(learn): Leitner-style review boxes with injectable clock`
- **Commit 2** (after task 3): `feat(api): schedule-aware progress next and review fields`
- **Commit 3** (after task 4): `feat(web): hub due hints and caught-up review copy`
- **Commit 4** (after tasks 5–6): `docs: document forgiving spaced-review queue for hiragana5`

## Risks / Notes

- **Mastery vs box drift:** mastery is compute-on-read from attempts; box is write-on-assess. After clear + restore impossibilities — clear deletes both. If assess path fails mid-tx, existing rollback covers both.
- **Replacing `all_steady`:** keep mastery state `steady`; next reason becomes `all_caught_up` when schedule-aware. Update i18n + tests that asserted `all_steady` on next.
- **RULES/ARCHITECTURE wording:** replace “no SRS/due dates without a plan” with “Leitner-style personal review only as shipped; no SM-2/FSRS, streaks, or notifications.”
- **Timezone support later:** if calendar-local due is ever needed, add explicit preference — out of scope now; wall-clock UTC intervals are intentional.
- **Prompt 17:** audio/content must not depend on review boxes.
- **Five-character justification:** SM-2 ease learning and rating UX add complexity without queue-size pressure — Leitner is enough.

## Implementation Notes for `/aif-implement`

- Stay on **main** (user requested no new branch).
- Prefer pure schedule functions + one migration; avoid a second review table.
- Reuse Prompt 15 progress/next/clear surfaces; do not invent a parallel “SRS service” package outside `learn`.
- Every task needs logging as specified; tests must control time (no `time.Sleep` for due assertions).
- Docs policy: **mandatory** checkpoint (Settings Docs: yes).
