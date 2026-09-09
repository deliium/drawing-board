# Implementation Plan: Practice Attempt APIs Replacing Whole-Board Recognition

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Practice attempt APIs replacing whole-board recognition (Prompt 11)"
Rationale: Moves single-character practice assessment onto explicit attempt rows whose strokes are submitted and frozen per attempt, so scoring never loads the free-board stroke store — unlocking honest retry/history for later pedagogy UI without coupling to `boardRev` / WS.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep existing completed Prompt 09/10 entries unchanged.

## Summary

Prompt 09 shipped durable learning tables + repos (`CreateDraft` / `SubmitStrokes` / `SaveResult` / `Abandon`). Prompt 10 froze the reviewed `hiragana5` pack and wired `recognize.Assessor.Assess` to pack strokes. Today **`POST /api/recognize` still loads every saved board stroke** at `boardRev` (optional `target` assesses that whole board dump). `LearnStore` is constructed in `cmd/server` but unused by HTTP.

This plan delivers:

1. **REST practice-attempt APIs** — start (draft), submit ordered strokes for one target character, assess, get result, abandon, retry as a **new** attempt.
2. **Assessment input = attempt strokes only** — never `ListStrokesWithRev` / board tables.
3. **Contracts** for ownership, idempotency (`clientAttemptId`), validation, transactional persistence, stale/conflict handling, and coexistence with reliable WS / `boardRev`.
4. **Draft policy** — draft **rows** are persisted; stroke geometry is **not** draft-persisted; immutability begins at `submitted`.
5. **Tests** — contract, handler, DB/repo gaps, and end-to-end HTTP lifecycle against SQLite.
6. **Docs + axioms** — README API section, ARCHITECTURE/RULES/DESCRIPTION updates; deprecate board-loaded **target** recognition as the practice path.

**Out of scope:** Arbitrary multi-character / open-set “what did I write?” as the practice product path; guided lesson Vue UI (Prompt 13); learner-facing correction copy / score weights (Prompt 12); SRS; classrooms; changing WS `opId`/`baseRev`/`boardRev` contracts; durable offline vault; expanding beyond `hiragana5`.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Schema | `practice_attempts`, `attempt_strokes` / `attempt_stroke_points`, `assessment_results`, `assessment_feedback`, `user_character_progress` exist (migration `0002`) |
| Repos | `AttemptRepo` / `AssessmentRepo` / `CharacterRepo` / `ProgressRepo` implemented in `internal/db/learn_store.go` |
| Gaps in repos | **No** `ListStrokes(attemptID)` (or equivalent) — assess HTTP cannot reload submitted geometry yet; `CreateDraft` returns `ErrConflict` on duplicate `client_attempt_id` instead of idempotent replay |
| Recognize HTTP | `POST /api/recognize` requires `boardRev`; loads **all** user board strokes; optional `target` → `Assessor.Assess` on that set |
| Assessor | `TargetCompareRecognizer.Assess(glyph, strokes, w, h)` ready; set = `hiragana5`; `scoreKind=match` |
| Server wiring | `_ = db.NewLearnStore(store)` unused; routes only board + recognize |
| Body limits | Recognize params body **4 KiB**; general `MaxAPIJSONBodyBytes` = **64 KiB** (use for attempt submit) |
| Frontend | BoardPage free-board Recognize (heuristic); no attempt client |
| WS / board | Unchanged; clear/undo must not touch attempts (already tested at store level) |

## Design Decisions

### D1 — Practice assessment is attempt-scoped REST; board recognize is not the practice path

| Path | Role after this plan |
|------|----------------------|
| `POST /api/attempts…` | **Canonical** single-character practice: submit strokes → assess → history |
| `POST /api/recognize` | **Free-board playground only**: heuristic ranking from board strokes + `boardRev` gate. **Remove or hard-reject `target`** so practice cannot silently assess the whole board |
| WebSocket create/delete/clear | Unchanged scratchpad; **not** used to submit attempt strokes |

Rationale: Prompt 08 already labeled free-board scores as heuristic match ranking. Attempt APIs are the honest “practice one glyph” product surface. Keeping board `target` would reintroduce whole-board assessment.

### D2 — State machine (unchanged statuses; HTTP maps 1:1)

```text
                  CreateDraft
                      │
                      ▼
                   draft ──────────────► abandoned
                      │                      ▲
                      │ SubmitStrokes        │ (only from draft)
                      ▼                      │
                  submitted ── Assess/SaveResult ──► assessed
                      │
                      └── (no reopen; retry = new CreateDraft)
```

| Transition | Who | Immutable after |
|------------|-----|-----------------|
| → `draft` | `POST /api/attempts` | Metadata mutable only via abandon |
| → `submitted` | `POST …/submit` | **Stroke set + canvas size frozen** |
| → `assessed` | `POST …/assess` | Attempt + assessment **terminal** |
| → `abandoned` | `POST …/abandon` | Terminal; no strokes |

**Retry:** always a **new** `practice_attempts` row (new `clientAttemptId`). Never reopen `assessed` / `abandoned`.

### D3 — Drafts: persist row, not stroke drafts

| Persist on create? | Yes — `practice_attempts` row (`status=draft`, character, optional lesson, optional `client_attempt_id`) |
| Persist strokes before submit? | **No** — client holds ordered strokes (may draw on free-board UI as scratchpad, but must **copy** geometry into submit JSON) |
| Why not server draft strokes? | Avoids inventing attempt-WS, partial rewrite races, and `boardRev`-like revision for in-progress ink; matches Prompt 09 repo (`SubmitStrokes` is the first stroke write) |

Document clearly: abandoning a draft discards client-only ink; board clear still does not imply attempt abandon.

### D4 — Idempotency and stale handling (not `boardRev`)

Attempts **must not** require or return `boardRev`. Coexistence with reliable transport:

| Concern | Attempt semantics |
|---------|-------------------|
| Create retry | Optional `clientAttemptId` (≤36, same charset rules as `opId`). Duplicate for same user → **return existing attempt** if `character_id` (and `lesson_id` if both set) match; else `409 conflict` |
| Submit retry | If already `submitted`/`assessed` with same attempt id → `409` `invalid_status` / `already_submitted` (strokes not replaced). Client must not “patch” strokes |
| Assess retry | If already `assessed` → **`200` with existing assessment** (safe replay). If `submitted` → assess once. Else `409` |
| Stale client | No revision field; “stale” = wrong status (e.g. assess while still `draft`, submit after abandon). Codes: `invalid_status`, `not_found`, `conflict` |
| Board WS | Independent; empty queue / Saved gating stays for **board** recognize only |

### D5 — Ownership and character binding

- Every handler: session `user_id`; repo queries always `AND user_id=?`; foreign attempt id → `404 not_found` (no leakage).
- Create: `characterId` must exist and `status=active` (and preferably `set_id=hiragana5` for MVP); resolve glyph for assess from character row (not client-supplied free text).
- Optional `lessonId`: if present must be published and contain the character; else `400`.
- Assess: load attempt strokes in `seq` order; call `Assessor.Assess(character.Glyph, …)`; persist via `SaveResult` with `assessor=target_compare`, `set_id` from recognizer/character, `score_kind=match`. Map engine `reasons` into `assessment_feedback` ranks 1..≤2 with stable codes when possible (`stroke_count_mismatch`, `empty_strokes`, …); empty `message` until Prompt 12.

### D6 — API surface (contracts)

All mutating routes: auth + CSRF. GETs: auth only.

#### `POST /api/attempts`

Request:

```json
{
  "characterId": "hira:あ",
  "lessonId": "lesson:hiragana5",
  "clientAttemptId": "optional-uuid-or-op-like-id"
}
```

Response `201` (or `200` on idempotent replay):

```json
{
  "id": 42,
  "characterId": "hira:あ",
  "glyph": "あ",
  "lessonId": "lesson:hiragana5",
  "status": "draft",
  "clientAttemptId": "…",
  "startedAt": "RFC3339"
}
```

Errors: `400` `invalid_input` / `character_not_active`; `404` `not_found` (unknown character/lesson); `409` `conflict` (clientAttemptId reused for different character).

#### `GET /api/attempts/{id}`

Response: attempt metadata; if `submitted`|`assessed`, include `canvasWidth`/`canvasHeight` and `strokeCount` (optional: omit full points on GET to keep payloads small — full points only needed internally for assess). Do **not** require returning points to clients in MVP unless useful for debug; prefer metadata-only GET.

#### `POST /api/attempts/{id}/submit`

Body max **`limits.MaxAPIJSONBodyBytes` (64 KiB)**:

```json
{
  "width": 300,
  "height": 300,
  "strokes": [
    {
      "color": "#000000",
      "width": 2,
      "startedAtUnixMs": 0,
      "points": [{"x": 10.5, "y": 20.0}]
    }
  ]
}
```

- Validate with existing `limits.CheckCanvas` / `ValidateStrokeSet` / `ValidateStrokeMeta` / `ValidateStrokePoints` (same CSS-logical coords as board).
- Stroke **order** in the array is authoritative (`seq`).
- Success `200`: `{ "id", "status": "submitted", "submittedAt", "strokeCount", "width", "height" }`.
- Failures: `400` limit/validation codes; `404`; `409` `invalid_status` if not `draft`.

#### `POST /api/attempts/{id}/assess`

Empty body (or `{}`). Server:

1. Load attempt (owner); require `submitted` **or** already `assessed` (idempotent).
2. If `assessed` → return existing assessment `200`.
3. Else `ListAttemptStrokes` → `Assessor.Assess(glyph, …)` → `SaveResult` (one tx already) → `200`.

Response:

```json
{
  "attemptId": 42,
  "characterId": "hira:あ",
  "glyph": "あ",
  "status": "assessed",
  "pass": true,
  "score": 0.82,
  "scoreKind": "match",
  "assessor": "target_compare",
  "setId": "hiragana5",
  "reasons": ["…"],
  "feedback": [{"rank": 1, "code": "stroke_count_mismatch", "message": ""}],
  "candidates": [{"text": "あ", "score": 0.82, "scoreKind": "match"}]
}
```

`candidates` may be returned from the live assess call; persisted table does not need candidate rows (reasons/feedback only) — document that GET assessment may omit live `candidates` or recompute only if needed; prefer store what `SaveResult` already persists and return candidates only on the assess response that just ran.

#### `GET /api/attempts/{id}/assessment`

`200` persisted result; `404` if not assessed / wrong owner.

#### `POST /api/attempts/{id}/abandon`

Only `draft` → `abandoned`. `200` `{ "id", "status": "abandoned" }`; else `409`.

#### Retry

Client: `POST /api/attempts` with a **new** `clientAttemptId` and same `characterId`.

### D7 — Transactional persistence (reuse Prompt 09 boundaries)

| HTTP | Store ops | Tx |
|------|-----------|-----|
| Create | `CreateDraft` | Existing single tx |
| Submit | `SubmitStrokes` (+ progress upsert on submit) | Existing single tx |
| Assess | `ListAttemptStrokes` (read) + `Assess` (CPU) + `SaveResult` | `SaveResult` remains one tx (assessment + feedback + attempt `assessed` + progress). Do **not** split MarkAssessed outside SaveResult |
| Abandon | `Abandon` | Existing |

Add **`AttemptRepo.ListStrokes(ctx, userID, attemptID) ([]StrokeInput, width, height, error)`** (or return strokes + canvas from attempt row). Ownership-checked. Used only after `submitted`/`assessed`.

Upgrade **`CreateDraft`** (or HTTP layer): on unique `(user_id, client_attempt_id)` conflict, `GetByClientAttemptID` and return existing if character/lesson match.

### D8 — Rate limits, metrics, body size

- Apply `RecognizeLimiter` (or dedicated `AttemptAssessLimiter` with same defaults) to **`POST …/assess`** and optionally submit.
- Metrics: `attempt_requests_total{op=create|submit|assess|…,result=ok|reject|error}` and reject codes — mirror recognize style; **no coordinates**.
- Submit uses `MaxAPIJSONBodyBytes`; reject with `body_too_large`.

### D9 — Compatibility with reliable transport

Explicit non-goals / guarantees:

1. Do **not** send attempt strokes over WS; do **not** invent attempt `opId` ack on the hub.
2. Do **not** gate attempt submit/assess on WS queue empty / `syncStatus === 'Saved'` (that gate stays for **board** recognize only).
3. Board `stale_revision` / `baseRev` remain board-only.
4. Attempt submit may occur while board has unsynced ink — irrelevant; assess uses POST body strokes only.
5. CSRF + cookie session identical to other `POST /api/*`.
6. docguard: never describe attempt scores as calibrated AI confidence / ONNX.

### D10 — Free-board `target` deprecation

- Remove `target` handling from `Recognize` **or** return `400` `use_attempt_api` with message pointing at `/api/attempts`.
- Prefer **hard reject** of non-empty `target` to prevent dual practice paths.
- Keep heuristic recognize + `boardRev` for playground.

## Logging Contract (verbose)

| Level | Events |
|-------|--------|
| DEBUG | `[httpapi.Attempt.Create]` userID, characterID, clientAttemptID present?; `[httpapi.Attempt.Submit]` attemptID, strokeCount, pointCount, w, h; `[httpapi.Attempt.Assess]` attemptID, status, pass, score; `[learn.AttemptRepo.ListStrokes]` attemptID, strokeCount (**no coordinates**) |
| INFO | Successful create/submit/assess/abandon with ids + status + pass/scoreKind; idempotent create/assess replay |
| WARN | `invalid_status`, `conflict`, rate limit, unsupported/inactive character |
| ERROR | Store failures, assessor failures |

Never log point arrays; respect `LOG_LEVEL` like existing `apiLog` / `learnLog`.

## Out of Scope

- Multi-character / page-level recognition APIs
- Guided lesson Vue routes, trace overlay UX (Prompt 13) — optional thin `attemptsApi.ts` only if needed for contract tests
- Pedagogy copy / two-corrections messaging (Prompt 12)
- Persisting mid-draft stroke geometry on the server
- Attempt WebSocket channel
- Schema redesign / SRS fields
- Changing board WS reliability semantics

## Tasks

### Phase 0: Contracts & axioms
- [x] Task 1: Lock attempt API axioms in AI context
  - Update `.ai-factory/ARCHITECTURE.md`: attempt REST under `httpapi`; depends on `learn` repos + `recognize.Assessor`; board recognize stays heuristic-only; no attempt↔board FK.
  - Update `.ai-factory/RULES.md`: (1) practice assessment reads **attempt** strokes only; (2) retry = new attempt row; (3) drafts persist metadata without stroke drafts; (4) attempt APIs must not require `boardRev`.
  - Update `.ai-factory/DESCRIPTION.md`: one-liner for attempt-scoped practice assessment.
  - LOGGING: none (docs only).

### Phase 1: Repository gaps
- [x] Task 2: `ListStrokes` + idempotent create-by-client id (depends on 1)
  - Add `AttemptRepo.ListStrokes` / `GetByClientAttemptID` to `internal/learn/repository.go`.
  - Implement in `internal/db/learn_store.go`: ordered strokes+points; ownership checks; `CreateDraft` conflict → return existing when character/lesson match else `ErrConflict`.
  - Extend `learn_store_test.go`: list after submit; idempotent create; mismatch conflict; other-user 404.
  - LOGGING: DEBUG list counts; WARN conflict/mismatch; no coordinates.
  - Files: `internal/learn/repository.go`, `internal/db/learn_store.go`, `internal/db/learn_store_test.go`.

### Phase 2: HTTP handlers & wiring
- [x] Task 3: Attempt handler contracts + helpers (depends on 2)
  - Add `internal/httpapi/attempts.go` (or split files): request/response types, status→HTTP mapping (`ErrNotFound`→404, `ErrConflict`→409, `ErrInvalidStatus`→409, `ErrInvalidInput`→400 with stable codes).
  - Wire `API` with `Learn *db.LearnStore` (or interface), keep `Assessor`.
  - LOGGING: per logging contract; shared `apiLog`.
  - Files: `internal/httpapi/attempts.go`, `internal/httpapi/handlers.go` (API struct).

- [x] Task 4: Implement routes — create / get / submit / assess / assessment / abandon (depends on 3)
  - Register under `cmd/server/main.go` with `RequireAuth`; CSRF covers POSTs via existing middleware.
  - Submit: `MaxBytesReader(…, MaxAPIJSONBodyBytes)`; validate limits; call `SubmitStrokes`.
  - Assess: rate limit; list strokes; `Assessor.Assess(glyph)`; map to `SaveAssessment` + feedback codes; idempotent if assessed.
  - Reject empty `target` usage on recognize (D10).
  - LOGGING: INFO/DEBUG/WARN/ERROR per D logging contract; metrics counters.
  - Files: `internal/httpapi/attempts.go`, `internal/httpapi/handlers.go` (Recognize target reject), `cmd/server/main.go`.

### Phase 3: Tests
- [x] Task 5: Handler / contract tests (depends on 4)
  - Table-driven tests: auth 401; CSRF 403 on POST; create→submit→assess→get; idempotent create; idempotent assess; submit twice → 409; assess while draft → 409; abandon; other user → 404; inactive/unknown character; body too large; stroke limit violations; assess does **not** call board list (assert board stroke count irrelevant / clear board before assess still works).
  - Recognize with `target` → 400 `use_attempt_api` (or chosen code).
  - Files: `internal/httpapi/attempts_test.go`, update `recognize_handler_test.go` / `recognize_target_test.go`.

- [x] Task 6: End-to-end HTTP lifecycle test (depends on 4, 5)
  - Single test opening real SQLite via `db.Open`, full mux or handler chain with session cookie + CSRF: register/login (or session helper) → create → submit gold-ish strokes for `あ` → assess → GET assessment → retry new attempt → board `ApplyClear` → assessment still present.
  - LOGGING: tests may set `LOG_LEVEL=debug` optionally; assert no panic.
  - Files: `internal/httpapi/attempts_e2e_test.go` (or `cmd/server` integration if project prefers — prefer `httpapi` + real store like existing tests).

### Phase 4: Docs & thin client (optional contract aid)
- [x] Task 7: README + operator docs (depends on 4)
  - Document attempt endpoints, status machine, draft policy, immutability, idempotency, CSRF, body limit, relationship to WS/boardRev, deprecation of recognize `target`.
  - Note progress counters update on submit/assess as today.
  - Keep docguard honesty (match scores, not ML confidence).
  - Files: `README.md`; touch `internal/docguard` only if new forbidden phrases appear.

- [x] Task 8: Optional `web/src/services/attemptsApi.ts` + Vitest contract stub (depends on 7)
  - Thin typed client for create/submit/assess/get; **no** lesson UI.
  - Contract test asserting request shapes / error codes (mock fetch) so Prompt 13 has a stable client.
  - LOGGING: `console.debug` in DEV only if matching existing apiFetch patterns; no stroke dumps.
  - Files: `web/src/services/attemptsApi.ts`, `web/tests/contract/attempts-api.spec.ts`.

<!-- Commit checkpoint: tasks 1-4 -->
<!-- Commit checkpoint: tasks 5-8 -->

## Commit Plan
- **Commit 1** (after tasks 1–2): `feat(learn): list attempt strokes and idempotent clientAttemptId create`
- **Commit 2** (after tasks 3–4): `feat(httpapi): practice attempt REST lifecycle and drop board target assess`
- **Commit 3** (after tasks 5–6): `test(httpapi): attempt contract and e2e lifecycle coverage`
- **Commit 4** (after tasks 7–8): `docs: document attempt APIs; add thin attempts client`

## Acceptance Criteria

1. Learner can start a draft for one `hiragana5` character, submit ordered strokes, assess, and retrieve a persisted pass/score/`scoreKind=match` result without reading board strokes.
2. Clearing or mutating the free-board does not change attempt strokes or assessments.
3. Retry creates a new attempt; assessed attempts are immutable; draft abandon works; duplicate submit rejected; duplicate assess is idempotent.
4. `clientAttemptId` create is idempotent for the same character; mismatched reuse conflicts.
5. `POST /api/recognize` no longer assesses via board-loaded `target`.
6. WS/`boardRev` contracts unchanged; attempt routes do not require `boardRev`.
7. Verbose structured logs without coordinates; tests cover contract, handler, DB, and e2e lifecycle; README documents the API.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Clients keep using recognize `target` | Hard reject + README + migrate any internal tests |
| 64 KiB too small for dense strokes | Same limits as recognize stroke caps; document; align with `ValidateStrokeSet` |
| Double progress increment if submit+assess miscounted | Keep existing store progress rules; e2e assert attempt_count/pass_count |
| Feeding board strokes into assess by mistake | E2E clears board before assess; code review: assess path only `ListStrokes` on attempt |
| Scope creep into lesson UI | Explicit out of scope; optional thin client only |

## Unblocks

- Prompt 12 (pedagogy copy on `assessment_feedback.message`)
- Prompt 13 (guided lesson Vue consuming attempt APIs + traces)
- Prompt 15 (history / mastery UX over assessed attempts)
