# Implementation Plan: Core Learner Journey for One Hiragana Character (Prompt 13)

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Core learner journey UI for one hiragana character (Prompt 13)"
Rationale: Turns Prompt 10–12 curriculum + attempt/assess APIs into the first guided product path — introduce one glyph, animate stroke order, practice (trace then free-write), submit an attempt, show comparison + ≤2 corrections, then retry or mark completion — without gamification or free-board coupling.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep existing Prompt 08–12 entries unchanged (do not uncheck completed items).

## Summary

Prompts 09–12 shipped durable learning tables, the reviewed `hiragana5` pack, attempt REST (`create` → `submit` → `assess` / `abandon`), multi-criterion assessment with ≤2 actionable `feedback` messages, and Vue fixtures for traces + a thin `attemptsApi.ts`. The SPA still only exposes free-board `BoardPage` (WS scratchpad + heuristic Recognize). There is **no** lesson route, no pedagogy chrome, no guided stage machine, and **no** HTTP for lesson/character catalog or progress reads (repos exist; UI cannot resume “passed” state from the server).

This plan delivers:

1. **Guided single-character journey UI** — target glyph, pronunciation, stroke count, example word; animated canonical stroke order; **trace** then **free-write**; explicit submit; comparison overlay; ≤2 corrections; retry or completion.
2. **Vue routes + focused components** — lesson hub + per-character practice; age-neutral, non-gamified chrome; loading / error / empty states.
3. **Thin curriculum + progress read APIs** — so the UI does not dual-maintain pedagogy fields and can show persisted progress after refresh.
4. **Client journey state machine** — transitions, cancellation (`abandon`), resume rules (session + server), API integration via `attemptsApi` + new curriculum/progress clients.
5. **Tests** — Vitest component + journey integration for **all five** starter glyphs; Go handler tests for new reads; docs/honesty updates.

**Out of scope:** Pronunciation audio playback/CDN (Prompt 17 — show text IPA/romanization only; ignore null `audioRef`); bilingual i18n of UI strings (Prompt 14); SRS / mastery dashboards (Prompt 15–16); changing assess scoring / correction catalog (Prompt 12); WS/`boardRev` for practice strokes; persisting mid-draft stroke geometry on the server; classrooms; new characters beyond `hiragana5`; Playwright/Cypress (stay on Vitest + jsdom like the rest of `web/tests`).

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Routes | Hash router: `/` BoardPage (auth), `/login`, `/register` only (`web/src/router/index.ts`) |
| Free-board | `BoardPage.vue` + WS + heuristic Recognize; not attempt-scoped |
| Attempts client | `web/src/services/attemptsApi.ts` — create/get/submit/assess/getAssessment/abandon |
| Curriculum UI data | `web/src/curriculum/hiragana5.ts` + JSON traces; `drawTraceTemplates` only (no pedagogy pack mirror) |
| Canvas | `usePracticeCanvas` pointer/DPR lifecycle — reusable for local (non-WS) practice canvases |
| Backend attempts | Wired in `cmd/server`; statuses `draft` → `submitted` → `assessed` / `abandoned`; draft = metadata only |
| Progress | Updated inside `SaveResult`; `ProgressRepo.Get` / `ListForUser` exist — **no HTTP** |
| Characters/lessons | Seeded from pack; `CharacterRepo` / `LessonRepo` exist — **no catalog HTTP** |
| Assessment UX contract | Prefer `pass` + `score` (`scoreKind=match`) + ≤2 `feedback[].message`; do **not** lead with `candidates` / diagnostics |
| Tests | Vitest unit/contract/integration; Go attempt e2e; **no** browser e2e runner |

## Design Decisions

### D1 — Practice path is REST attempt-scoped; free-board stays a playground

| Surface | Role after this plan |
|---------|----------------------|
| `/practice` + `/practice/:characterId` | **Canonical** learner journey for one glyph |
| `/` BoardPage | Unchanged free-board scratchpad + heuristic Recognize |
| Attempt strokes | Drawn **locally** (no WS enqueue); submitted once via `POST …/submit` |
| WS / `boardRev` | Must not gate submit/assess; practice clear/undo is local only |

Link from AppShell/BoardPage to Practice with a quiet text control (“Practice hiragana”), not a game lobby.

### D2 — Minimal curriculum + progress read APIs (no dual pedagogy source)

Ship authenticated read endpoints that project seeded DB / pack fields (same source as seed):

| Method | Path | Response (sketch) |
|--------|------|-------------------|
| `GET` | `/api/lessons/{id}` | `{ id, code, title, setId, contentVersion, characters:[{ id, glyph, romanization, strokeCount, pronunciation, descriptionEn, example:{word,romanization,meaningEn}, sortKey/position }] }` |
| `GET` | `/api/progress` | `{ items:[{ characterId, status, attemptCount, passCount, lastAttemptId?, lastPassedAt?, updatedAt }] }` — optionally filter `?lessonId=` / `?setId=hiragana5` by joining lesson characters client-side if filter is omitted |

Rules:

- Only **published** lessons (`lesson:hiragana5`); 404 otherwise.
- Pronunciation JSON from DB (`PronunciationJSON`) decoded to object; `audioRef` may be null (UI must not require audio).
- Trace/stroke **geometry** stays on the client pack fixtures (`hiragana5Traces`) for animation/overlay; assert `contentVersion` from lesson response matches fixture pack when both present (warn in DEV if mismatch).
- Do **not** invent a second pedagogy JSON authoring path in Vue.

### D3 — Journey stages (UI state machine)

```text
load ──► intro ──► animate ──► trace ──► freewrite ──► submitting ──► result
  │         │         │          │           │              │            │
  │         └─ cancel/leave ─────┴───────────┴── abandon draft ──────────┤
  │                                                                      ▼
  └─ error / empty                                              complete | retry
```

| Stage | Learner sees | Canvas | Server |
|-------|--------------|--------|--------|
| `loading` | Skeleton / “Loading…” | off | `GET lesson` + `GET progress` |
| `error` | Short age-neutral error + Retry load | off | last failure |
| `empty` | Character missing / not in lesson | off | — |
| `intro` | Glyph, romanization, IPA/hint, stroke count, example word + meaning | optional static preview | optional create draft deferred to first draw or on “Start” |
| `animate` | Stroke-order animation (canonical polylines); Continue | no input | — |
| `trace` | Faint guide + learner ink; “Next” when ≥1 stroke (or strokeCount strokes — prefer **allow Next when strokeCount met**, soft warn if under) | local pencil | draft exists |
| `freewrite` | Blank (no guide); explicit **Submit** | local pencil; clear/undo local | draft |
| `submitting` | Disabled controls; “Checking…” | locked | submit → assess (sequential) |
| `result` | Comparison overlay + pass/fail match score + ≤2 corrections | read-only overlay | assessed |
| `complete` | Quiet confirmation; link next glyph / lesson hub | off | progress `passed` (or already passed) |

**Retry:** always `createAttempt` with a **new** `clientAttemptId`; never reopen assessed. Prefer returning to `animate` or `trace` (product choice: default **`trace`** so stroke order reminder is one tap away via “Show order again”).

**Completion:** available when `pass === true` **or** progress already `passed` and learner chooses Done; do not require a streak/XP. Failed attempt: Retry primary, Done secondary (leave without pass).

### D4 — Create draft timing, cancellation, resume

**Create draft:** On entering `trace` (or pressing Start on intro) call `createAttempt({ characterId, lessonId: 'lesson:hiragana5', clientAttemptId })`. Persist `{ attemptId, clientAttemptId, characterId, stage }` in `sessionStorage` under a stable key (e.g. `practice:v1:{characterId}`).

**Cancellation / leave:**

- In-progress pencil: Escape cancels the open stroke (existing composable behavior).
- Leaving the route or “Cancel practice” while status is `draft`: best-effort `abandonAttempt`; clear session key; ignore 409/invalid_status.
- After `submitted`/`assessed`: abandon is invalid — do not call it.

**Resume after refresh:**

| Prior state | Behavior |
|-------------|----------|
| `draft` + session has attemptId | `GET /api/attempts/{id}`; if still `draft`, restore stage (default `freewrite` or saved stage); restore local strokes from session **if** present, else empty canvas + notice “Drawing was not saved — continue or restart” |
| `draft` + missing session | Start fresh (new `clientAttemptId`); do not list orphan drafts (no list API in MVP) |
| `submitted` (crash mid-assess) | `GET attempt` → call `assessAttempt` (idempotent) → `result` |
| `assessed` | `GET …/assessment` → `result` |
| Progress `passed` + no active attempt | Offer intro with “Practice again” / “Done” |

**Do not** persist stroke points on the server until Submit (Prompt 11 invariant).

### D5 — Comparison overlay + corrections

- Overlay: learner strokes + canonical template (distinct styles; faint template). Optional side-by-side toggle if overlay is cluttered — default **overlay**.
- Show `pass`, overall **match** score (label “Match”, never “confidence” / “AI”).
- Render at most two `feedback` items (`rank`, `message`); hide empty.
- Do not surface `reasons`, `diagnostics`, or `candidates` in the learner panel (DEV-only optional collapse OK).

### D6 — Age-neutral presentation

- No points, levels, streaks, confetti, cartoon mascots, or childish praise.
- Copy: calm instructional English (“Check stroke order”, “Try again”, “Completed”).
- Visual: reuse existing quiet shell; practice canvas as the focus; one job per stage.

### D7 — Logging (verbose)

| Layer | Prefix | What |
|-------|--------|------|
| HTTP curriculum/progress | `[httpapi.Lesson.Get]`, `[httpapi.Progress.List]` | INFO on success counts; WARN 404/unauthorized; DEBUG ids (no PII beyond userID) |
| Attempt (existing) | `[httpapi.Attempt.*]` | Keep; journey should not spam coordinates |
| Vue journey | `[practiceJourney]` DEV `console.debug` | stage transitions, attemptId, pass, feedback codes (not points) |
| Animation | `[strokeOrder]` DEV | glyph, stroke index play/pause |
| Overlay | `[compareOverlay]` DEV | stroke counts learner vs template |

Production: client debug gated on `import.meta.env.DEV`; server respects existing `LOG_LEVEL`.

## Routes & Components

### Routes (`web/src/router/index.ts`)

| Path | Name | Auth | Component |
|------|------|------|-----------|
| `/practice` | `practice-hub` | requiresAuth | `PracticeHubPage.vue` — five glyphs + progress status |
| `/practice/:characterId` | `practice-character` | requiresAuth | `PracticeCharacterPage.vue` — journey shell (`characterId` like `hira:あ` URL-encoded) |
| `/` | `board` | requiresAuth | existing BoardPage |

Guards: reuse `requireAuth`. Invalid `characterId` → empty state (not a hard router block).

### Component map

```text
PracticeHubPage
PracticeCharacterPage
  ├── CharacterIntroPanel          # glyph, pronunciation, stroke count, example
  ├── StrokeOrderPlayer            # animate canonical strokes
  ├── PracticeStageCanvas          # wraps usePracticeCanvas; mode=trace|free|readonly
  ├── ComparisonOverlay            # template + learner; score + feedback list
  ├── JourneyActions               # Start / Next / Submit / Retry / Done / Cancel
  └── JourneyStatusBanner          # loading / error / submitting / resume notice
```

Composables / services:

- `usePracticeJourney.ts` — stage machine, session resume, API orchestration
- `web/src/services/curriculumApi.ts` — lesson fetch
- `web/src/services/progressApi.ts` — progress list
- Extend `attemptsApi.ts` only if types need tweak

Keep `workflowStore` free-board-oriented; prefer journey state in the composable (Pinia optional if prop-drilling hurts).

## Acceptance Scenarios

### S1 — First-time completion (happy path)

1. Auth user opens `/practice`, sees five characters with `unseen`/`seen` progress.
2. Opens あ → intro shows glyph, romanization, pronunciation, stroke count **3**, example あさ / morning.
3. Animate plays all strokes in order; Continue → trace with guides → free-write → Submit.
4. Server create/submit/assess succeeds; overlay + empty or weak feedback; `pass=true`.
5. Done → hub shows progress `passed` for あ.

### S2 — Retry after incorrect writing

1. Free-write submits incorrect strokes → `pass=false`, **1–2** correction messages, no candidates panel.
2. Retry → new `clientAttemptId` / attempt id; returns to practice stages; previous assessed row unchanged.
3. Second pass → complete.

### S3 — Refresh during an attempt

1. Mid-`freewrite` with local strokes + draft id in session → refresh → resume draft, strokes restored from session **or** empty + notice if storage missing.
2. Mid-`submitting` after submit succeeded but before assess response → refresh → assess idempotently → `result`.
3. On `result` → refresh → same assessment via GET.

### S4 — Backend failure

1. `GET /api/lessons/…` 5xx/network → error state + Retry (no blank crash).
2. `submit`/`assess` 429/5xx → stay on freewrite or submitting with error banner; do not claim pass; allow retry submit if still draft/submitted per API rules.
3. `abandon` failure on leave → log WARN; still clear local session (best-effort).

### S5 — All five starter characters

Hub + journey work for `hira:あ`…`hira:お` with correct stroke counts and examples from curriculum; automated tests cover each glyph (table-driven).

## Tasks

### Phase 1: Curriculum & progress HTTP

- [x] Task 1: Lesson + progress read handlers and routes

### Phase 2: Frontend API clients & curriculum fixtures alignment

- [x] Task 2: `curriculumApi` + `progressApi` + contract tests

### Phase 3: Journey state machine & components

- [x] Task 3: `usePracticeJourney` state machine (depends on 2)
- [x] Task 4: Presentational components (depends on 3)
- [x] Task 5: Pages + router + shell links (depends on 3, 4)

### Phase 4: Tests for five characters + failure/resume

- [x] Task 6: Component + journey integration tests for あいうえお (depends on 5)
- [x] Task 7: Go regression for curriculum/progress + attempt coexistence (depends on 1)

### Phase 5: Docs & honesty

- [x] Task 8: README + architecture + roadmap notes + AGENTS (depends on 5, 6)

## Commit Plan

- **Commit 1** (after tasks 1–2): `feat(api): lesson and progress read endpoints for practice UI`
- **Commit 2** (after tasks 3–5): `feat(web): guided hiragana practice journey UI`
- **Commit 3** (after tasks 6–8): `test+docs: hiragana5 practice journey coverage and README`

## Acceptance Criteria

1. Authenticated learner can open a focused journey for any of あいうえお showing target, pronunciation, stroke count, and example word from curriculum API.
2. Canonical stroke order animates; trace (with guides) and free-write (no guides) stages work on a local canvas without WS attempt traffic.
3. Explicit Submit runs attempt submit+assess; UI shows comparison overlay, match score, and **at most two** corrections; retry = new attempt; completion updates/reads progress.
4. Loading, error, and empty states are defined and covered by tests; cancellation abandons drafts when applicable.
5. Resume behavior matches D4 for refresh during draft / mid-assess / result.
6. Acceptance scenarios S1–S5 pass via Vitest component + integration suites for all five characters; Go tests cover new GETs.
7. README/ARCHITECTURE/AGENTS document the journey; honesty rules preserved (no calibrated confidence / AI claims).
8. Verbose DEV/server logs exist for stage and HTTP boundaries without stroke coordinates.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Dual pedagogy sources drift | Lesson API is source of truth; fixtures only for geometry; DEV contentVersion check |
| Scope creep into audio/SRS/i18n | Explicit out of scope; text pronunciation only |
| Resume loses ink (by design) | Clear copy + sessionStorage best-effort; never pretend server held drafts strokes |
| Free-board confusion | Separate routes; BoardPage link labeled practice; no Recognize on journey |
| Animation a11y | `prefers-reduced-motion` → skip to final frame |
| Gamification pressure in UI polish | D6 checklist in review; no XP/streak components |

## Dependencies / Sequencing

- **Requires:** Prompt 10 pack + Prompt 11 attempt APIs + Prompt 12 feedback messages (present on `main`).
- **Unblocks:** Prompt 14 (i18n of UI/corrections), Prompt 15 (history/mastery surfaces), Prompt 17 (audio on intro).
