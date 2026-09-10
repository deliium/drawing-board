# Implementation Plan: UUID Primary Keys for Surrogate IDs

Branch: feature/uuid-primary-keys
Created: 2026-09-10

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "UUID primary keys for all surrogate IDs"
Rationale: Replace every INTEGER AUTOINCREMENT / integer FK entity ID with UUID TEXT end-to-end (SQLite → Go domain → REST/WS → Vue) so IDs are non-enumerable strings and schema/API types stay aligned.

> Note for `$aif-roadmap` / implement: append an **unchecked** milestone to `.ai-factory/ROADMAP.md` with this exact title. Do not mark it complete until implement + verify finish. Keep existing completed R1–R4 entries unchanged.

## Summary

Today free-board and learning surrogate keys are SQLite `INTEGER PRIMARY KEY AUTOINCREMENT` (`users`, `strokes`, `stroke_points`, `practice_attempts`, attempt stroke/point rows, assessment rows) with matching `int64` Go types and **JSON numbers** on the wire (`/api/me.id`, `strokes[].id`, WS `delete` / `strokeId`, `/api/attempts/{id}`, history cursor `id`, `lastAttemptId`). Curriculum natural keys (`characters.id` like `hira:あ`, `lessons.id` like `lesson:hiragana5`), client `op_id` / `client_attempt_id`, and composite PKs stay TEXT as today. Counters such as `board_rev` / `at_rev` / widths / counts remain integers (they are not entity IDs).

This plan delivers:

1. **Migration `0007`** — rebuild affected tables as `TEXT` UUID PKs/FKs, **rewrite existing rows** with freshly generated UUIDs (mapping old INTEGER → new UUID), preserve board strokes, attempts, assessments, progress, and board ops/tombstones; fail-closed on error.
2. **Domain + store rewrite** — Go `int64` entity IDs → `string` (UUID); generate UUID on insert; parse/validate UUID on path/query/WS delete.
3. **Auth session** — cookie `user_id` becomes string UUID; existing sessions invalidate on deploy.
4. **REST / WS / Vue** — all entity IDs are JSON **strings**; frontend types and call sites use `string` only (no numeric entity IDs).
5. **Tests + README** — migrate/rewrite coverage, contract updates, docguard-honest API/WS examples.

**Out of scope:** Changing curriculum natural keys to UUIDs; changing `board_rev` / `baseRev` / `at_rev` to UUID; collaborative multi-user IDs; external SSO subject mapping; optional UUID v7 time-ordering (v4 is fine).

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Schema versions | `migrations.LatestVersion() == 6`; next Up is **7** |
| INTEGER surrogate PKs | `users`, `strokes`, `stroke_points`, `practice_attempts`, `attempt_strokes`, `attempt_stroke_points`, `assessment_results`, `assessment_feedback` |
| INTEGER FKs | All `user_id`, `stroke_id`, `attempt_id`, `attempt_stroke_id`, `assessment_id`, `last_attempt_id` |
| Already TEXT (keep) | `characters.id`, `lessons.id`, `lesson_characters`, `op_id`, `client_attempt_id`, pedagogy columns |
| Non-ID integers (keep) | `board_rev`, `at_rev`, `width`, `started_at_unix_ms`, `seq`, `pass` 0/1, `review_box`, counts, canvas sizes |
| Auth session | `sess.Values["user_id"]` as `int64` (+ `float64` fallback) in `internal/auth/auth.go` |
| Wire numbers | `userView.id`, stroke `id`, WS `delete`/`strokeId`, attempt path `{id}`, history item `id`, cursor payload `id`, `lastAttemptId` |
| Frontend | `number` on session, strokes, attempts, WS delete/ack; `sessionStorage` practice snapshot `attemptId: number` |
| UUID lib | Not in `go.mod` yet — add `github.com/google/uuid` |

## Design Decisions

### D1 — UUID TEXT format

| Concern | Decision |
|---------|----------|
| Format | RFC 4122 UUID **v4**, canonical string: lowercase hex with hyphens, length 36 (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`) |
| Storage | SQLite `TEXT NOT NULL` primary keys / FKs (no AUTOINCREMENT) |
| Generation | Server-side on insert via `github.com/google/uuid` (`uuid.NewString()`); never accept client-supplied PK for `users` / `strokes` / attempts / assessments |
| Validation | Shared helper (e.g. `internal/limits` or small `internal/ids`) — reject empty / non-UUID on path params, `?id=`, WS `delete` stroke id |
| Why not BINARY(16) | TEXT matches existing `op_id` style, readable logs, and JSON strings without encoding helpers |

### D2 — Forward-only migration 0007 (do not rewrite 0001–0006)

- Keep historical migrations immutable (already applied / stamped).
- Add `up0007UuidPrimaryKeys` registered as version **7**.
- SQLite cannot `ALTER` PK type in place: **CREATE new tables → copy with UUID map → drop old → rename** inside one transaction (or ordered sub-steps in the same migration `tx`).
- Fresh DBs still run 1→6 then 7 (acceptable one-time cost); do **not** squash baseline in this plan.
- Optional `Down` for tests only (destructive recreate INTEGER schema) — only if cheap; otherwise migration tests cover Up + data integrity without Down.

### D3 — Natural keys and composite keys unchanged

Leave as-is (not INTEGER; not in scope to UUID-ify):

- `characters.id` (`hira:あ` …), `lessons.id` (`lesson:hiragana5`), `lesson_characters` composite PK
- `strokes.op_id`, `board_ops` / `stroke_op_tombstones` PK `(user_id, op_id)` — after migration `user_id` becomes UUID TEXT; `op_id` stays client string
- `client_attempt_id` stays client string ≤36

### D4 — Non-ID integers stay integers

`board_rev`, `baseRev`, `at_rev`, widths, timestamps-ms, `seq`, `review_box`, `pass`, deleted/cleared **counts** remain numeric. Do not convert counters to UUID.

### D5 — Wire + session: strings only for entity IDs

| Surface | Before | After |
|---------|--------|-------|
| `GET /api/me` `id` | JSON number | JSON string UUID |
| `GET /api/strokes` `strokes[].id` | number | string |
| WS `delete` / ack `strokeId` / echo `stroke.id` | number | string |
| `/api/attempts/{id}` | decimal path | UUID path |
| History `items[].id`, cursor inner `id` | number | string |
| `lastAttemptId` | number \| null | string \| null |
| Cookie session `user_id` | int64 | string |

Breaking change: existing sessions and any cached numeric attempt IDs in `sessionStorage` must be treated as invalid (clear / re-login / new practice session).

### D6 — Rewrite mapping order

Inside migration 0007, assign UUIDs and copy in FK-safe order:

1. `users` (build `old_user_id → new_uuid` map)
2. Board: `user_board_state`, `strokes` (+ stroke id map), `stroke_points`, `stroke_op_tombstones`, `board_ops`
3. Learning: `practice_attempts` (+ attempt id map), `attempt_strokes` (+ map), `attempt_stroke_points`, `assessment_results` (+ map), `assessment_feedback`, `user_character_progress` (remap `user_id` + `last_attempt_id`)

Curriculum tables (`characters`, `lessons`, `lesson_characters`) are untouched.

Preserve `ON DELETE CASCADE` / `RESTRICT` / `SET NULL` semantics identical to current schema. Recreate indexes (`idx_strokes_user`, partial unique `(user_id, op_id)`, attempt indexes, progress due index, etc.).

### D7 — Logging

With Settings Logging = verbose:

- INFO on migration 0007 start/complete with counts rewritten per table (no emails, no stroke coords)
- DEBUG on insert ID generation only when useful (avoid per-point spam)
- Change existing `userID=%d` / `attemptID=%d` / `strokeId=%d` log formats to `%s`
- Never log password hashes, cookies, CSRF, or full stroke point arrays

## Schema Target (illustrative)

```sql
-- users (after 0007)
id TEXT PRIMARY KEY,  -- UUID
email TEXT NOT NULL UNIQUE,
password_hash TEXT NOT NULL,
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP

-- strokes
id TEXT PRIMARY KEY,
user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
...
op_id TEXT

-- practice_attempts
id TEXT PRIMARY KEY,
user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE RESTRICT,
...
```

`schema_migrations.version` remains INTEGER (migration version number, not an entity ID).

## Commit Plan

- **Commit 1** (after tasks 1–3): `feat(db): add UUID helpers and migration 0007 rewrite`
- **Commit 2** (after tasks 4–6): `feat: use string UUID IDs in store, auth, HTTP, and WS`
- **Commit 3** (after tasks 7–8): `feat(web): treat entity IDs as UUID strings`
- **Commit 4** (after tasks 9–10): `test+docs: cover UUID IDs and update API/WS contract`

## Tasks

### Phase 1: UUID helpers + migration 0007

- [x] Task 1: Add UUID dependency and shared ID helpers
  - Add `github.com/google/uuid` to Go module.
  - Introduce a small helper (prefer `internal/ids` or extend `internal/limits`) with: `New()` → string, `Parse`/`Valid` for path/query/WS, and clear error codes mappable to `invalid_input` / WS soft reject.
  - LOGGING: DEBUG on invalid parse with reason code only (no raw secrets); no per-call success spam.
  - Files: `go.mod`, `go.sum`, `internal/ids/ids.go` (or `internal/limits/uuid.go`), tests for Valid/Parse.

- [x] Task 2: Implement `up0007UuidPrimaryKeys` table rebuild + data rewrite
  - Register version 7 in `internal/db/migrations/migrations.go` (`LatestVersion` becomes 7).
  - Rebuild every table that has INTEGER surrogate PK/FK (list in Current State); copy rows with new UUIDs using in-memory maps; drop/rename; recreate indexes and FK behavior.
  - Leave curriculum natural-key tables and non-ID integer columns unchanged.
  - Fail the migration transaction on any copy/FK inconsistency (fail-closed `Open`).
  - LOGGING: INFO `[migrate.0007] start` / `done users=%d strokes=%d attempts=%d …`; ERROR with step name on failure (no PII dumps).
  - Files: `internal/db/migrations/0007_uuid_primary_keys.go`, `migrations.go`.

- [x] Task 3: Migration tests for rewrite integrity
  - Extend `internal/db/migrate_test.go` (and/or dedicated test): seed legacy-shaped data (or open at v6 fixture), run Open → v7, assert: row counts preserved, all PKs/FKs match UUID regex, FK joins still resolve (user→strokes, attempt→assessment→feedback, progress.last_attempt_id), board_rev unchanged, curriculum IDs unchanged, re-Open idempotent at version 7.
  - Cover empty DB fresh path (1…7) still seeds hiragana5.
  - LOGGING: use existing test log patterns; assert migration INFO path if log capture is already used — otherwise rely on assertions.
  - Files: `internal/db/migrate_test.go`, possibly `internal/db/migrations/0007_*_test.go`.
  <!-- Commit checkpoint: tasks 1-3 -->

### Phase 2: Domain + persistence + auth/HTTP/WS

- [x] Task 4: Convert Go domain and Store/LearnStore signatures to string UUID
  - Change `User.ID`, `Stroke.ID`/`UserID`, `learn.Attempt` IDs, `AssessmentResult` IDs, `Progress.UserID` / `LastAttemptID`, create/list filters (`AfterID`), etc. from `int64` → `string`.
  - Inserts: generate UUID before `INSERT` (stop using `LastInsertId` for entity PKs).
  - Update all SQL placeholders/scans; fix `CreateUser` return type; board apply results `StrokeID string`.
  - LOGGING: replace `%d` with `%s` for entity IDs in `internal/db/*.go`; keep `boardRev=%d`.
  - Files: `internal/db/db.go`, `board.go`, `learn_store.go`, `internal/learn/types.go`, `list.go`, `repository.go`, related tests.

- [x] Task 5: Auth session + REST API string IDs
  - Session `user_id` store/read as `string`; remove int64/float64 identity path (document session invalidation).
  - `userView.ID string`; attempt path `parseAttemptID` → UUID parse; stroke delete query `id` as UUID string; history cursor payload `id` string; progress `lastAttemptId` *string; JSON responses for attempts/history use string ids.
  - LOGGING: `userID=%s`, `attemptID=%s` throughout `internal/auth`, `internal/httpapi`.
  - Files: `internal/auth/auth.go`, `internal/httpapi/handlers.go`, `attempts.go`, `progress_history.go`, `curriculum.go` (progress DTO), tests.

- [x] Task 6: WebSocket message types string stroke IDs
  - `Stroke.ID`, `delete`, `strokeId` become `string` / `*string` as appropriate; validate UUID on delete-by-id; keep `opId`/`baseRev`/`boardRev` semantics unchanged; hub maps keyed by string user ID.
  - Soft-reject invalid delete id (prefer nack/error frame over disconnect).
  - LOGGING: `userID=%s`, `strokeId=%s` in `internal/ws/handler.go`.
  - Files: `internal/ws/handler.go`, WS tests/contracts.
  <!-- Commit checkpoint: tasks 4-6 -->

### Phase 3: Frontend

- [x] Task 7: Vue services and types — entity IDs as `string`
  - Update `sessionContext`, `attemptsApi`, `progressApi`, `strokeSync`, `wsClient`, `usePracticeJourney` snapshot: `id` / `attemptId` / `strokeId` / `delete` / `lastAttemptId` → `string`.
  - Auth `me`/login/register typing: `id: string`.
  - Invalidate or migrate practice `sessionStorage` snapshots that still hold numeric attempt ids (treat as missing → new draft).
  - LOGGING: DEV `console.debug` may print UUID strings; no change to production logging.
  - Files: `web/src/services/*`, `web/src/composables/usePracticeJourney.ts`, `web/src/main.ts`, `web/src/pages/LoginPage.vue`, `BoardPage.vue`, history page keys.

- [x] Task 8: Frontend unit/contract/e2e fixtures
  - Replace numeric ID literals in Vitest + Playwright helpers with UUID strings; fix assertions comparing numbers.
  - Files: `web/tests/**`, `web/e2e/helpers.ts`, related e2e specs.
  <!-- Commit checkpoint: tasks 7-8 -->

### Phase 4: Docs + verification gates

- [x] Task 9: README + architecture honesty for UUID IDs
  - Update API/WS examples (`delete`, `strokeId`, attempt paths, auth `{ id }`) to UUID strings; note session invalidation / breaking ID type change; mention migration 0007 rewrite.
  - Touch `.ai-factory/ARCHITECTURE.md` / `DESCRIPTION.md` only if they claim integer user/stroke IDs.
  - Keep recognition honesty / non-ML language intact (`go test ./internal/docguard`).
  - LOGGING: N/A for docs; ensure README does not invent new log claims.
  - Files: `README.md`, optionally `.ai-factory/ARCHITECTURE.md`, `DESCRIPTION.md`, `AGENTS.md` if it lists integer ID assumptions.

- [x] Task 10: Run quality gates and fix regressions
  - `go test ./...` (include race-sensitive packages if touched), `make check-web` or project-equivalent Vitest, migration tests green, `go test ./internal/docguard`.
  - Grep for leftover entity `int64` ID fields / `ParseInt` on attempt/stroke/user IDs / Vue `: number` on entity ids; fix stragglers.
  - LOGGING: spot-check that new code paths emit `%s` for IDs under verbose/dev log level.
  - Files: any remaining call sites from grep.
  <!-- Commit checkpoint: tasks 9-10 -->

## Acceptance Criteria

- [x] No surrogate entity PK/FK remains INTEGER in schema after version 7 (except `schema_migrations.version` and non-ID counters).
- [x] Existing data survives Open→0007 with intact relationships (strokes per user, attempt assessments, progress `last_attempt_id`).
- [x] REST/WS/Vue expose entity IDs only as strings (UUID form); no JSON numbers for user/stroke/attempt/lastAttempt IDs.
- [x] Curriculum IDs (`hira:あ`, `lesson:hiragana5`) and `op_id` / `client_attempt_id` behavior unchanged.
- [x] `boardRev` / `baseRev` still integer revision counters.
- [x] Tests + README updated; `docguard` green.

## Risks / Notes

- **Breaking API/session:** clients and cookies holding numeric IDs break — acceptable for this personal app; document clearly.
- **SQLite rewrite cost:** large stroke_points tables make 0007 heavier; keep work in one transaction carefully (SQLite lock); consider batched inserts if tests show timeout — prefer correctness first.
- **Enumeration:** UUID removes sequential ID guessing on `/api/attempts/{id}` (still must enforce ownership).
- **Do not** change integer `id` keys to UUID in curriculum natural keys; do not UUID-ify `board_rev`.
