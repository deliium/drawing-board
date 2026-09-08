# Implementation Plan: Learning Domain Model and Versioned Migrations

Branch: main
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Learning domain model and versioned migrations (Prompt 09)"
Rationale: Introduces the minimum durable learning schema (characters, lessons, attempts, assessments, feedback, progress) behind a versioned SQLite migration runner so later attempt APIs and curriculum content can land without inventing ad-hoc DDL.

> Note for `$aif-roadmap` / implement: append an **unchecked** milestone to `.ai-factory/ROADMAP.md` with this exact title (Prompt 09). Do not mark it complete until implement + verify finish. Keep existing completed Prompt 08 entry unchanged.

## Summary

Today `internal/db.migrate` applies **unversioned** `CREATE TABLE IF NOT EXISTS` / best-effort `ALTER TABLE` on every `Open`. That is enough for free-board strokes (`users`, `strokes`, `stroke_points`, `user_board_state`, `stroke_op_tombstones`, `board_ops`) but cannot safely evolve a learning curriculum, attempt history, or progress without drift between environments.

This plan delivers:

1. A **versioned migration runner** with recorded applied versions, baseline for existing board schema, and fail-closed startup.
2. The **minimum learning-domain schema** for the five-character hiragana MVP: characters, lessons + ordering, practice attempts, attempt strokes/points, assessment results, feedback rows, and per-user character progress.
3. **Repository interfaces** (learning package) with SQLite implementations, deliberately **separate** from free-board stroke persistence.
4. **Migration tests**, rollback/recovery policy, and **idempotent seed** of `hiragana5` stubs (glyph freeze remains Prompt 10).

**Out of scope:** SRS / Leitner scheduling, classrooms/teachers/cohorts, attempt HTTP APIs (Prompt 11), rich pedagogy copy (Prompt 12), guided lesson UI (Prompt 13), mastery UX polish (Prompt 15). This plan ships durable storage + boundaries so those prompts have tables and repos to call.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Schema bootstrap | `internal/db/db.go` `migrate()` — single multi-statement `CREATE IF NOT EXISTS` + opportunistic `ALTER strokes ADD op_id` + board tables |
| Version tracking | **None** — no `schema_migrations` / version table |
| Free-board tables | `users`, `strokes` (+ nullable `op_id`), `stroke_points`, `user_board_state.board_rev`, `stroke_op_tombstones`, `board_ops` |
| Board tx | `BEGIN` + reserved-lock upgrade in `beginImmediate`; create/delete/clear bump `board_rev` |
| Stroke delete | Hard `DELETE` cascades points via FK; clear tombstones create `op_id`s |
| Learning entities | **Absent** in SQLite; recognition uses in-process `hiragana5` templates only (`internal/recognize`) |
| Seed | None for curriculum; tests create users/strokes ad hoc |
| Docs | README documents board/recognize contracts; no learning schema section |

## Design Decisions

### D1 — Free-board strokes stay separate from attempt strokes

| Concern | Decision |
|---------|----------|
| Storage | New `attempt_strokes` / `attempt_stroke_points` tables. **No FK** to `strokes` / `stroke_points`. |
| Clear / undo / erase | Board `ApplyClear` / `ApplyStrokeDelete` **must not** mutate attempt rows. |
| Recognize today | `POST /api/recognize` may keep reading **board** strokes for free-board heuristic / optional `target` demo. Attempt assessment APIs (later) read **attempt** strokes only. |
| Why | Board is a scratchpad with `boardRev` / `opId` semantics. Attempts are immutable pedagogical submissions. Merging them would couple WS idempotency to lesson lifecycle and risk losing practice history on clear. |

Preserve existing free-board data in place: migrations are **additive**; no rewrite or copy of historical board strokes into attempts.

### D2 — Versioned, forward-only migrations in production

- Record applied versions in `schema_migrations(version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`.
- Each migration is a numbered Go-registered step (SQL file embedded or Go function) with unique monotonic `version` (e.g. `1`, `2`, …).
- Production path is **forward-only**. Optional `Down` functions exist **only for tests / local recovery drills**, never auto-run on startup.
- Startup: apply pending versions in order; on any error **abort `Open` / refuse listen** (same severity as corrupt perimeter config).

### D3 — Thin learning package + SQLite repos

- New package `internal/learn` owns domain types + **repository interfaces**.
- `internal/db` (or `internal/db/learnstore`) implements interfaces against `*sql.DB` / `*Store`.
- `httpapi` / future attempt handlers depend on interfaces, not raw SQL.
- Do **not** put lesson/attempt types into `internal/recognize` (recognize stays pure scoring).

### D4 — Minimum character/lesson content now; rich content later

Prompt 10 owns reviewed glyphs, romanization, audio metadata, trace templates, example words. This plan seeds **stub** `hiragana5` characters (`あ い う え お` matching current recognize set id) and one ordered lesson so FKs and progress rows work. Columns that Prompt 10 will fill may exist as nullable TEXT/JSON stubs **or** be deferred — prefer nullable stubs listed below to avoid a second DDL churn if cheap.

### D5 — No SRS / classroom columns

Do not add `due_at`, `interval`, `ease`, `box`, `class_id`, `teacher_id`, or shared “assignment” tables. Progress statuses stay simple and explainable.

## Domain Model

```text
users 1──* practice_attempts *──1 characters
                │                    ▲
                │                    │
                *── attempt_strokes  lesson_characters ──* lessons
                │
                1──? assessment_results 1──* assessment_feedback
users 1──* user_character_progress *──1 characters
```

Free-board subgraph unchanged and disconnected:

```text
users 1──* strokes 1──* stroke_points
users 1──1 user_board_state
users 1──* stroke_op_tombstones | board_ops
```

### Entity contracts

#### `characters`

| Column | Type | Notes |
|--------|------|-------|
| `id` | TEXT PK | Stable id, e.g. `hira:あ` (not autoincrement — content-addressable) |
| `set_id` | TEXT NOT NULL | e.g. `hiragana5` (`recognize.SetIDHiragana5`) |
| `glyph` | TEXT NOT NULL | Unicode character |
| `romanization` | TEXT NULL | stub for Prompt 10 |
| `stroke_count` | INTEGER NOT NULL | canonical count; must match templates when content freezes |
| `sort_key` | INTEGER NOT NULL | default order within set |
| `status` | TEXT NOT NULL | `active` \| `retired` (MVP: all `active`) |
| `created_at` / `updated_at` | TIMESTAMP NOT NULL | |

Constraints/indexes:
- `UNIQUE(set_id, glyph)`
- `INDEX idx_characters_set_sort ON characters(set_id, sort_key)`

Ownership: **global curriculum** (not per-user).

#### `lessons`

| Column | Type | Notes |
|--------|------|-------|
| `id` | TEXT PK | e.g. `lesson:hiragana5` |
| `code` | TEXT NOT NULL UNIQUE | stable machine code |
| `title` | TEXT NOT NULL | English stub OK |
| `set_id` | TEXT NOT NULL | curriculum set |
| `sort_order` | INTEGER NOT NULL | multi-lesson future |
| `status` | TEXT NOT NULL | `draft` \| `published` (seed `published`) |
| `created_at` / `updated_at` | TIMESTAMP NOT NULL | |

#### `lesson_characters` (ordering)

| Column | Type | Notes |
|--------|------|-------|
| `lesson_id` | TEXT NOT NULL FK → lessons ON DELETE CASCADE | |
| `character_id` | TEXT NOT NULL FK → characters ON DELETE RESTRICT | |
| `position` | INTEGER NOT NULL | 1-based order |

Constraints: `PRIMARY KEY (lesson_id, character_id)`; `UNIQUE (lesson_id, position)`.

#### `practice_attempts`

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK AUTOINCREMENT | |
| `user_id` | INTEGER NOT NULL FK → users ON DELETE CASCADE | owner |
| `character_id` | TEXT NOT NULL FK → characters ON DELETE RESTRICT | target |
| `lesson_id` | TEXT NULL FK → lessons ON DELETE SET NULL | optional context |
| `status` | TEXT NOT NULL | see statuses |
| `client_attempt_id` | TEXT NULL | ≤36; idempotency for later API |
| `canvas_width` / `canvas_height` | INTEGER NULL | set on submit |
| `started_at` | TIMESTAMP NOT NULL | |
| `submitted_at` | TIMESTAMP NULL | |
| `assessed_at` | TIMESTAMP NULL | |
| `abandoned_at` | TIMESTAMP NULL | |
| `created_at` / `updated_at` | TIMESTAMP NOT NULL | |

**Statuses:** `draft` → `submitted` → `assessed`; or `draft` → `abandoned`. Terminal: `assessed`, `abandoned`. Do not reopen `assessed` (Prompt 11 may add retry as a **new** attempt row).

Indexes:
- `INDEX idx_attempts_user_started ON practice_attempts(user_id, started_at DESC)`
- `INDEX idx_attempts_user_char ON practice_attempts(user_id, character_id, started_at DESC)`
- `UNIQUE INDEX idx_attempts_user_client ON practice_attempts(user_id, client_attempt_id) WHERE client_attempt_id IS NOT NULL`

#### `attempt_strokes` / `attempt_stroke_points`

Mirror board stroke geometry **without** `op_id` / boardRev:

| `attempt_strokes` | |
|-------------------|--|
| `id` | INTEGER PK |
| `attempt_id` | FK CASCADE |
| `seq` | INTEGER NOT NULL (0-based stroke order) |
| `color` | TEXT NOT NULL DEFAULT `#000000` |
| `width` | INTEGER NOT NULL DEFAULT 2 |
| `started_at_unix_ms` | INTEGER NOT NULL DEFAULT 0 |
| `UNIQUE(attempt_id, seq)` | |

| `attempt_stroke_points` | |
|-------------------------|--|
| `id` | INTEGER PK |
| `attempt_stroke_id` | FK CASCADE |
| `seq` | INTEGER NOT NULL |
| `x` / `y` | REAL NOT NULL |
| `UNIQUE(attempt_stroke_id, seq)` | |
| `INDEX` on `attempt_stroke_id` | |

Ownership: via attempt → user. Validate point counts/coords with `internal/limits` at write time (same bounds as board).

#### `assessment_results`

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK | |
| `attempt_id` | INTEGER NOT NULL UNIQUE FK → practice_attempts ON DELETE CASCADE | 1:1 |
| `pass` | INTEGER NOT NULL | 0/1 |
| `score` | REAL NOT NULL | match score |
| `score_kind` | TEXT NOT NULL | `match` (`recognize.ScoreKindMatch`) |
| `assessor` | TEXT NOT NULL | e.g. `target_compare` |
| `set_id` | TEXT NOT NULL | e.g. `hiragana5` |
| `reasons_json` | TEXT NOT NULL | JSON array of strings (engine reasons); keep small |
| `created_at` | TIMESTAMP NOT NULL | |

No calibrated-confidence naming in columns or docs.

#### `assessment_feedback`

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK | |
| `assessment_id` | INTEGER NOT NULL FK CASCADE | |
| `rank` | INTEGER NOT NULL | 1..N; MVP allow 0–2 rows |
| `code` | TEXT NOT NULL | stable machine code (`stroke_count`, `order`, …) |
| `message` | TEXT NOT NULL DEFAULT '' | learner copy; may be empty until Prompt 12 |
| `UNIQUE(assessment_id, rank)` | | |

#### `user_character_progress`

| Column | Type | Notes |
|--------|------|-------|
| `user_id` | INTEGER NOT NULL FK CASCADE | |
| `character_id` | TEXT NOT NULL FK RESTRICT | |
| `status` | TEXT NOT NULL | `unseen` \| `seen` \| `practicing` \| `passed` |
| `attempt_count` | INTEGER NOT NULL DEFAULT 0 | |
| `pass_count` | INTEGER NOT NULL DEFAULT 0 | |
| `last_attempt_id` | INTEGER NULL FK → practice_attempts ON DELETE SET NULL | |
| `last_passed_at` | TIMESTAMP NULL | |
| `updated_at` | TIMESTAMP NOT NULL | |
| `PRIMARY KEY (user_id, character_id)` | | |

**No** mastery/SRS fields. `passed` means ≥1 assessed pass; finer mastery is Prompt 15.

## Transaction Boundaries

| Operation | Transaction | Notes |
|-----------|-------------|-------|
| Apply pending migrations | One transaction **per migration version** | Commit version row in same tx as DDL/DML; failure rolls back that version only |
| Seed curriculum | Single tx | Idempotent `INSERT OR IGNORE` / upsert |
| Create draft attempt | Single tx | Insert attempt only |
| Submit attempt + strokes | Single tx | Insert strokes/points; set `submitted`; reject if not `draft` |
| Persist assessment + feedback + progress | Single tx | Insert assessment/feedback; set attempt `assessed`; upsert progress counters |
| Abandon draft | Single tx | Status → `abandoned` |
| Board create/delete/clear | Existing board txs | **Must not** touch learning tables |
| User delete | FK CASCADE | Attempts, progress, board strokes all cascade from `users` |

SQLite: use `BEGIN IMMEDIATE` (or existing reserved-lock pattern) for write txs that race with board mutations on the same DB file.

## Deletion / Retention

| Data | Behavior |
|------|----------|
| Board strokes | Unchanged: hard delete on undo/erase/clear; points CASCADE |
| Attempt strokes | Deleted only with attempt (`ON DELETE CASCADE`) or user delete |
| Assessed attempts | **Retain** by default (history for Prompt 15). No auto-purge in this plan. |
| Abandoned drafts | Retain; optional future GC out of scope |
| Curriculum rows | `characters` / `lessons`: prefer `status=retired` over hard delete; `ON DELETE RESTRICT` from attempts/progress blocks accidental wipe |
| Operator wipe | Document: deleting a user removes all private learning + board data via CASCADE |

Privacy axiom unchanged: learning rows are **per-user** via `user_id`; never shared across accounts.

## Migration Framework Design

### Layout

```text
internal/db/
  migrate.go          # runner: list, apply pending, current version
  migrations/
    0001_baseline_board.go      # or .sql.go embed
    0002_learning_domain.go
  migrate_test.go
  bootstrap_legacy_test.go
internal/learn/
  types.go            # structs + statuses
  repository.go       # interfaces
  errors.go
internal/db/learn_store.go   # SQLite impl (or learn/sqlite/)
internal/db/seed_hiragana5.go
```

### Bootstrap algorithm (`Open`)

1. Open SQLite, PRAGMA foreign_keys / busy_timeout / WAL (unchanged).
2. Ensure `schema_migrations` exists (bootstrap DDL, version 0 meta — not counted as a learning migration).
3. If DB has board tables but **empty** `schema_migrations` (legacy install):
   - **Stamp** baseline version `1` as applied **without re-running** destructive DDL (detect via `sqlite_master`), **or** run baseline only as `CREATE IF NOT EXISTS` identical to today’s schema then insert version `1`.
   - Prefer: baseline migration is idempotent `IF NOT EXISTS` matching current production DDL (including `op_id` + board tables); then `INSERT OR IGNORE` version 1.
4. Apply all registered migrations with `version > max(applied)` in ascending order.
5. Run seed (idempotent) after migrations succeed.
6. Log final version.

### Rollback / recovery policy

| Scenario | Policy |
|----------|--------|
| Migration fails mid-tx | Rollback that version; process exits; DB remains at previous version |
| Bad deploy needs undo | **Restore from file backup** taken before upgrade (document `cp data.db data.db.bak`). Do not ship automatic `Down` on startup |
| Test-only `Down` | Optional reverse SQL for 0002 in tests to assert clean teardown; never registered for prod runner |
| Partial manual DDL | Unsupported; operator restores backup |
| Forward fix | Add `0003_…` corrective migration; never edit applied migration bodies after release |

Document operator steps in README: backup → migrate on start → on failure restore bak and pin previous binary.

## Repository Interfaces

```go
// internal/learn/repository.go (sketch)

type CharacterRepo interface {
  ListBySet(ctx context.Context, setID string) ([]Character, error)
  Get(ctx context.Context, id string) (*Character, error)
}

type LessonRepo interface {
  GetPublished(ctx context.Context, id string) (*Lesson, error)
  ListCharacters(ctx context.Context, lessonID string) ([]LessonCharacter, error)
}

type AttemptRepo interface {
  CreateDraft(ctx context.Context, in CreateDraft) (Attempt, error)
  Get(ctx context.Context, userID, attemptID int64) (*Attempt, error)
  SubmitStrokes(ctx context.Context, userID, attemptID int64, strokes []StrokeInput, w, h int) error
  MarkAssessed(ctx context.Context, userID, attemptID int64) error // usually inside Assess tx
  Abandon(ctx context.Context, userID, attemptID int64) error
}

type AssessmentRepo interface {
  // SaveResult writes assessment + feedback and updates attempt + progress in one store tx.
  SaveResult(ctx context.Context, userID int64, in SaveAssessment) (AssessmentResult, error)
  GetByAttempt(ctx context.Context, userID, attemptID int64) (*AssessmentResult, error)
}

type ProgressRepo interface {
  Get(ctx context.Context, userID int64, characterID string) (*Progress, error)
  ListForUser(ctx context.Context, userID int64) ([]Progress, error)
}
```

Implementations enforce `user_id` on every read/write of private rows (return not-found on mismatch — no leakage).

HTTP wiring for attempts is **out of scope**; `cmd/server` may construct repos and leave them unused or behind a no-op health check until Prompt 11.

## Seed-Data Handling

- After migrations: `SeedHiragana5(store)`:
  - Upsert 5 characters (`hira:あ` … `hira:お`), `set_id=hiragana5`, `stroke_count` from recognize templates / documented placeholders.
  - Upsert lesson `lesson:hiragana5`, `code=hiragana5`, `status=published`.
  - Upsert `lesson_characters` positions 1..5 matching `sort_key`.
- Idempotent on every startup (safe for tests and restarts).
- Do **not** invent unreviewed pedagogical prose; empty `romanization` / feedback messages OK.
- Prompt 10 may replace seed source with versioned content files; keep seed entrypoint stable.
- LOGGING: `INFO [db.seed] set=hiragana5 characters=5 lesson=lesson:hiragana5`; `DEBUG` per glyph id.

## Logging Contract (verbose)

| Level | Events |
|-------|--------|
| DEBUG | `[db.migrate] apply version=N name=…`; `[learn.AttemptRepo.CreateDraft] userID=… characterID=…`; stroke/point counts on submit (**no coordinates**) |
| INFO | `[db.migrate] up-to-date version=N` or `applied=N→M`; `[db.seed] …`; `[main] schema_version=N` at startup |
| WARN | Legacy DB stamped to baseline; skipped deprecated env (none expected); seed no-op counts if useful |
| ERROR | Migration failure with version + error string; FK/constraint failures on repos |

Never log passwords, session cookies, or full point arrays. Reuse `LOG_LEVEL` gating like `board.go` `dbLog`.

## Out of Scope

- Attempt REST/WS APIs and state-machine handlers (Prompt 11)
- Pedagogical “two corrections” copy and scoring weight changes (Prompt 12)
- Vue guided lesson UI (Prompt 13)
- SRS / review queue / classroom entities
- Migrating historical free-board strokes into attempts
- Changing `boardRev` / WS contracts
- Editing frozen recognize honesty axioms except docs cross-links

## Tasks

### Phase 0: Contracts & docs skeleton
- [x] Task 1: Lock schema + separation decisions in plan-adjacent axioms notes for implement
  - Update `.ai-factory/ARCHITECTURE.md` dependency note: learning repos live under `internal/learn` + `internal/db`; free-board strokes remain isolated; versioned migrations replace ad-hoc `migrate()`.
  - Update `.ai-factory/RULES.md` with axioms: (1) attempt strokes ≠ board strokes; (2) migrations are versioned and fail-closed; (3) no SRS/classroom tables without a dedicated plan.
  - Update `.ai-factory/DESCRIPTION.md` one-liner: durable learning schema for five-hiragana MVP (attempts/progress) separate from free-board.
  - LOGGING: none (docs only). Implementer logs when wiring lands in Task 8.

### Phase 1: Versioned migration runner
- [x] Task 2: Introduce `schema_migrations` + migration registry (depends on 1)
  - Replace monolithic `migrate()` body with runner in `internal/db/migrate.go` (name flexible).
  - Register migrations as ordered list `{Version int, Name string, Up func(*sql.Tx) error}`.
  - Bootstrap `schema_migrations` before applying numbered steps.
  - Files: `internal/db/db.go` (`openAndInit` calls runner), new migrate files.
  - LOGGING: `DEBUG [db.migrate] apply version=…`; `INFO [db.migrate] applied …` / `up-to-date version=…`; `ERROR [db.migrate] failed version=… err=…`.

- [x] Task 3: Baseline migration `0001` capturing current board schema (depends on 2)
  - Encode today’s `users` / `strokes` / `stroke_points` / indexes / `op_id` partial unique / `user_board_state` / `stroke_op_tombstones` / `board_ops` as idempotent `CREATE IF NOT EXISTS` + safe `ALTER` duplicate-column handling.
  - Legacy DBs without version rows must end at version ≥1 without data loss.
  - LOGGING: `INFO [db.migrate] baseline ready version=1`; `WARN` if stamping legacy detected (optional detection log once).

### Phase 2: Learning-domain DDL + repos
- [x] Task 4: Migration `0002` learning tables (depends on 3)
  - Create `characters`, `lessons`, `lesson_characters`, `practice_attempts`, `attempt_strokes`, `attempt_stroke_points`, `assessment_results`, `assessment_feedback`, `user_character_progress` with keys/indexes/FKs as specified.
  - Enable FK pragma remains ON.
  - LOGGING: included in generic migrate logs (`name=learning_domain`).

- [x] Task 5: Define `internal/learn` types + repository interfaces (depends on 4)
  - Status constants, domain structs, sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrInvalidStatus`).
  - No HTTP types.
  - LOGGING: N/A (types only); package comment references log prefix `[learn.*]`.

- [x] Task 6: SQLite repository implementation (depends on 5)
  - Implement Character/Lesson/Attempt/Assessment/Progress repos on `*db.Store`.
  - `SaveResult` must be one transaction: assessment + feedback + attempt status + progress upsert.
  - Enforce ownership checks.
  - LOGGING: `DEBUG` entry/exit with ids/status; `INFO` on assess persist `pass=… score=…`; `WARN` on conflict/stale status; never coordinates.

- [x] Task 7: Idempotent `hiragana5` seed (depends on 4, 5)
  - `SeedHiragana5` after migrations in `Open` (or explicit call from `main` — prefer `Open` so tests get seed).
  - Align glyph set with `recognize` MVP (`あいうえお`) and `set_id=hiragana5`.
  - LOGGING: `INFO [db.seed] …`; `DEBUG` per character upsert.

### Phase 3: Wiring, tests, recovery docs
- [x] Task 8: Startup wiring + schema version log (depends on 2, 7)
  - `cmd/server/main.go`: after `db.Open`, log `INFO [main] schema_version=N learn_seed=hiragana5`.
  - Ensure repos can be constructed (even if unused by HTTP yet).
  - LOGGING: as above; migration errors already fail `Open`.

- [x] Task 9: Migration & bootstrap tests (depends on 3, 4, 7)
  - Empty file → Open → version=2 (or latest), all learning tables present, seed count=5.
  - Simulate pre-versioned DB: create old-style tables without `schema_migrations`, Open → stamps/applies without dropping strokes; assert stroke rows survive.
  - Apply `0002` twice via re-Open → idempotent.
  - Optional: test-only Down for `0002` then Up again.
  - LOGGING: `t.Logf` version transitions on failure.

- [x] Task 10: Learning store unit tests (depends on 6)
  - Draft → submit strokes → save assessment → progress `passed`; second assess on same attempt → conflict.
  - Board `ApplyClear` leaves attempts intact.
  - User delete cascades attempts/progress.
  - List characters by set; lesson order positions stable.
  - FK: cannot delete character referenced by attempt (RESTRICT).
  - LOGGING: failure-only `t.Logf`.

- [x] Task 11: README + operator recovery docs (depends on 8, 9)
  - Document schema versioning, backup/restore rollback, learning vs free-board separation, seed behavior, log prefixes.
  - Cross-link recognize honesty (scores still match scores).
  - Update ARCHITECTURE folder tree for `learn/` + `migrations/`.
  - Ensure `go test ./internal/docguard` still passes (no ONNX regressions).
  - LOGGING: document `[db.migrate]`, `[db.seed]`, `[main] schema_version=`.

## Commit Plan
- **Commit 1** (after tasks 1–3): "chore: add versioned SQLite migrations and board schema baseline"
- **Commit 2** (after tasks 4–7): "feat: add learning-domain tables, repos, and hiragana5 seed"
- **Commit 3** (after tasks 8–10): "test: cover migration bootstrap and learning store transactions"
- **Commit 4** (after task 11): "docs: describe learning schema and migration recovery"

## Acceptance Criteria
1. Fresh and legacy DBs open via versioned migrations; `schema_migrations` records applied versions; startup fails closed on migrate error.
2. Existing free-board stroke data survives upgrade; board clear/undo does not delete practice attempts.
3. Learning tables exist with documented keys, statuses, indexes, and FK ownership; no SRS/classroom entities.
4. Repository interfaces in `internal/learn` with SQLite impl; assessment+progress write is transactional.
5. Idempotent `hiragana5` seed creates 5 characters + ordered lesson without duplicating rows.
6. Tests cover empty migrate, legacy bootstrap, seed idempotency, attempt/assessment/progress flows, and clear-isolation.
7. README documents backup/restore rollback policy and learning vs board separation; ARCHITECTURE/RULES/DESCRIPTION updated; roadmap milestone linked (Prompt 09).
8. Verbose logs follow `[db.migrate]` / `[db.seed]` / `[learn.*]` / `[main] schema_version=` without stroke coordinates.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Editing baseline after release breaks stamp logic | Freeze `0001` body; only add new versions |
| Seed glyphs diverge from recognize templates | Shared constant / test asserting glyph set equality with `recognize` set |
| Over-building attempt API in this plan | Repos only; no new HTTP routes |
| SQLite DDL + version insert not atomic | Same `*sql.Tx` for Up + insert version row |
| Accidental JOIN of board strokes into attempts | Code review + test that clear ≠ attempt delete; no shared table |
