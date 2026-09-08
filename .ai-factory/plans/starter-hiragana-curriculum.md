# Implementation Plan: Reviewed Five-Character Hiragana Starter Curriculum

Branch: main
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Reviewed five-character hiragana starter curriculum (Prompt 10)"
Rationale: Freezes a pedagogically coherent, human-reviewed `hiragana5` content pack (glyphs, romanization, pronunciation metadata, descriptions, stroke paths, trace templates, example words) behind a versioned format with validation, licensing, and deterministic seed — so later attempt APIs and guided lessons consume trusted curriculum, not stubs or unreviewed AI copy.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep existing completed Prompt 09 entry unchanged.

## Summary

Prompt 09 delivered durable learning tables and an idempotent **stub** seed for `あ い う え お`. Recognition still embeds placeholder polylines in `internal/recognize/testdata/hiragana5/templates.json`. Pedagogical fields (`romanization`, descriptions, example words, pronunciation metadata, UI trace templates) are missing or NULL; stroke geometry is duplicated manually between seed stroke counts and recognize JSON.

This plan delivers:

1. A **coherent first set** — keep the **あ行 vowels** (`あ い う え お`) — with documented pedagogy.
2. A **versioned content pack** under `content/hiragana5/` as the single reviewed source of truth for curriculum metadata + canonical stroke paths + trace templates.
3. **Per-character authored fields**: glyph, romanization, pronunciation metadata, learner-friendly description, canonical stroke count + ordered paths, trace template, one example word + English meaning.
4. **Validation tooling**, **content-schema tests**, **rendering fixtures**, **font/stroke licensing**, and a **Japanese-speaker review workflow**.
5. **Deterministic seed/import** via stable `SeedHiragana5` entrypoint reading the pack; migration `0003` for pedagogy columns; recognize embeds the same stroke paths.
6. Hard rule: **AI-generated prose/paths never enter the trusted pack** unless a qualified reviewer signs them off in the review record.

**Out of scope:** Attempt REST APIs (Prompt 11), scoring-copy / two-corrections pedagogy (Prompt 12), guided lesson Vue UI (Prompt 13), pronunciation audio playback/CDN (Prompt 17 — this plan only stores **metadata + asset refs**), SRS, classrooms, expanding beyond five glyphs.

## Pedagogical Set Choice

### Selected set: あ行 (a-row vowels) — `あ い う え お`

| Glyph | ID | Romanization | Stroke count |
|-------|-----|--------------|--------------|
| あ | `hira:あ` | a | 3 |
| い | `hira:い` | i | 2 |
| う | `hira:う` | u | 2 |
| え | `hira:え` | e | 2 |
| お | `hira:お` | o | 3 |

### Why this set (not か行, mixed “easy” glyphs, or random five)

1. **Gojuon foundation** — The vowel row is the first column of the standard hiragana chart. Every later mora is consonant + vowel; teaching vowels first matches how textbooks and teachers introduce kana.
2. **Phonological completeness** — Five distinct Japanese vowels cover the full vowel inventory used in later rows without digraphs or dakuten.
3. **Stroke-load ramp** — Counts are 3/2/2/2/3: enough variation to practice order and short strokes, without early complexity like き (4), さ (3 with discontinuous feel), or ぬ (2 but dense).
4. **Product continuity** — Prompt 08/09 already froze `set_id=hiragana5` and stable ids `hira:あ`…`hira:お`. Changing the set would break recognize honesty axioms, seed FKs, and eval fixtures for no pedagogical gain.
5. **Reviewability** — A single chart row is easy for a Japanese-speaking reviewer to check stroke order, example words, and romanization consistently.

**Rejected alternatives (documented for reviewers):**

| Alternative | Why not for MVP |
|-------------|-----------------|
| かきくけこ | Adds consonant onset before vowels are solid; denser strokes |
| “Easiest five” mix (e.g. の つ し …) | No chart coherence; harder to extend into ordered lessons |
| Full 46 hiragana | Out of scope; explodes review + fixture cost |

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Glyph set | Stub seed + recognize templates already use あいうえお |
| DB columns | `characters`: id, set_id, glyph, romanization (NULL), stroke_count, sort_key, status, timestamps — **no** description / example / pronunciation / trace columns |
| Seed source | Hardcoded `hiragana5Seed` in `internal/db/seed_hiragana5.go`; romanization forced NULL; lesson title still `(stub)` |
| Stroke paths | Only in `internal/recognize/testdata/hiragana5/templates.json` (`hiragana5-v1`); engineering placeholders |
| Content pack | **None** — no `content/` tree, no JSON Schema, no review record |
| Validation | `TestHiragana5SeedMatchesRecognizeGlyphs` only (glyph ⊆ assessable set) |
| Frontend | No trace/glyph overlay; BoardPage free-board only |
| Fonts | CSS stacks name Noto / IBM Plex; **not vendored**; no stroke-font attribution |
| Repo license | Root `LICENSE` is **CC0 1.0**; README still says “MIT” (docs fix in this plan’s docs task) |
| Learn repos | `Character` type lacks pedagogy fields; `CharacterRepo` list/get only |

## Design Decisions

### D1 — Keep あ行; freeze ids and set

- Do **not** rename `hiragana5`, `hira:あ`…`hira:お`, or `lesson:hiragana5`.
- Content pack **version** advances (`hiragana5-content-v1` → …) independently of schema migrations.
- Retiring a glyph later uses `status=retired` + new content version — never rewrite historical attempt FKs.

### D2 — Single trusted content pack; dual consumers

```text
content/hiragana5/v1/          ← reviewed source of truth
        │
        ├─► internal/curriculum (load + validate)
        │         ├─► SeedHiragana5 (SQLite pedagogy + stroke_count)
        │         └─► recognize embed (canonical assessment paths)
        └─► web fixtures (trace rendering tests; Prompt 13 UI later)
```

- Move/replace recognize `testdata/.../templates.json` so assessment paths are **copied or embedded from the pack** (build-time embed of `content/hiragana5/v1/strokes.json`, or generate into `internal/recognize` via checked-in sync — prefer **direct embed from `content/`** to avoid drift).
- Trace templates may equal assessment paths for MVP, or be a separate `trace` polyline set with the same stroke count/order (allow denser sampling for animation). Validator requires `len(trace)==stroke_count` and same order semantics.

### D3 — Versioned content format (JSON + schema)

Prefer JSON (matches existing recognize tooling) with a JSON Schema for CI.

```text
content/hiragana5/
  README.md                 # authoring + review instructions
  LICENSES.md               # font + stroke-data + example-word notes
  v1/
    manifest.json           # pack version, setId, review status, contentHash
    schema/                 # or content/schemas/hiragana5-character.schema.json
    characters.json         # array of 5 character records
    strokes.json            # canonical ordered paths (assessment)
    traces.json             # UI trace templates (may alias strokes in v1)
    review.json             # signed-off review record (human, not AI)
  drafts/                   # UNTRUSTED — AI or WIP; never seeded
```

**`manifest.json` (sketch):**

| Field | Notes |
|-------|-------|
| `packId` | `hiragana5` |
| `contentVersion` | `hiragana5-content-v1` |
| `setId` | must equal `recognize.SetIDHiragana5` |
| `characterCount` | must be `5` |
| `schemaVersion` | integer for format evolution |
| `reviewStatus` | `draft` \| `reviewed` \| `published` — seed **refuses** unless `published` |
| `contentHash` | SHA-256 of canonical serialized characters+strokes+traces (deterministic) |
| `licenses` | refs into `LICENSES.md` sections |

**Per-character record in `characters.json`:**

| Field | Type | Example / rule |
|-------|------|----------------|
| `id` | string | `hira:あ` |
| `glyph` | string (1 rune) | `あ` |
| `romanization` | string | `a` (Hepburn; lowercase) |
| `sortKey` | int | 1..5 |
| `strokeCount` | int | must match `len(strokes[glyph])` |
| `status` | string | `active` |
| `pronunciation` | object | see below |
| `description` | object | `{ "en": "…" }` learner-friendly; age-neutral |
| `example` | object | `{ "word": "あさ", "romanization": "asa", "meaningEn": "morning" }` |
| `review` | object | optional per-glyph notes; pack-level review is authoritative |

**`pronunciation` object (MVP metadata — no audio binary required):**

| Field | Notes |
|-------|-------|
| `ipa` | e.g. `/a/` — optional but preferred |
| `jaHint` | short JP tip, e.g. 「あ」の音 |
| `audioRef` | nullable string path/id for Prompt 17 (`null` in v1) |
| `pitch` | optional simple tag `flat` / `unknown` — do not invent accent without reviewer |

### D4 — Planned character content (authoring draft → must pass human review)

> These rows are the **implementation starting draft** for reviewers. Until `manifest.reviewStatus=published` and `review.json` is filled, they must not be treated as final learner-facing truth. Implementers may refine wording during review without changing ids/glyphs/sort order.

| Glyph | Romaji | Strokes | Description (en, draft) | Example word | Meaning |
|-------|--------|---------|-------------------------|--------------|---------|
| あ | a | 3 | Open vowel “ah”, as in father (shorter). First chart vowel. | あさ | morning |
| い | i | 2 | Close front vowel “ee”, as in see (shorter). | いぬ | dog |
| う | u | 2 | Close back vowel; lips unrounded vs English “oo”. | うみ | sea |
| え | e | 2 | Mid front vowel “eh”, as in get. | えき | station |
| お | o | 3 | Mid back vowel “oh”, pure (not a diphthong). | おと | sound |

**Stroke-order teaching notes (for reviewer + optional description footnotes):** follow standard school stroke order (あ: horizontal → vertical with hook → looping left stroke; い: left then right; う: short top then curve; え: top bar then zigzag; お: horizontal → vertical → right hook). Exact polylines in `strokes.json` must be redrawn/reviewed — current recognize templates are placeholders.

### D5 — Schema migration `0003` for pedagogy columns

Add nullable/text columns so `CharacterRepo` can serve Prompt 11/13 without a second churn:

| Column | Type | Source field |
|--------|------|--------------|
| `romanization` | TEXT NOT NULL after seed | already exists; seed fills |
| `description_en` | TEXT NOT NULL DEFAULT '' | `description.en` |
| `pronunciation_json` | TEXT NOT NULL DEFAULT '{}' | serialized `pronunciation` |
| `example_word` | TEXT NOT NULL DEFAULT '' | `example.word` |
| `example_romanization` | TEXT NOT NULL DEFAULT '' | `example.romanization` |
| `example_meaning_en` | TEXT NOT NULL DEFAULT '' | `example.meaningEn` |
| `content_version` | TEXT NOT NULL DEFAULT '' | pack `contentVersion` |
| `trace_ref` | TEXT NOT NULL DEFAULT '' | e.g. `traces.json#あ` or inline hash key |

**Do not** store full stroke polylines in SQLite for MVP (keep geometry in embedded pack for recognize + future API that can stream templates). Optionally add `strokes_sha256` column for ops/debug — nice-to-have, not required.

Update `internal/learn.Character` + `learn_store` SELECT/scan accordingly.

Lesson: replace stub title with reviewed English title, e.g. `Hiragana vowels (あいうえお)` (final string from review).

### D6 — AI content quarantine

| Location | Allowed |
|----------|---------|
| `content/hiragana5/v1/` | **Only** human-reviewed published content |
| `content/hiragana5/drafts/` | AI drafts, WIP, experimental paths — **never** imported by seed |
| PR description | May attach AI drafts for reviewer convenience |

Rules:

1. Seed and recognize load **only** `v1/` (or pinned published version).
2. CI fails if `manifest.reviewStatus != published` on main, or if `review.json` missing required fields.
3. `review.json` must name a **human** reviewer (display name or handle), date (ISO), and checklist results — not an AI model id as sole approver.
4. docguard / content tests forbid README claims like “expert-reviewed curriculum” unless `reviewStatus=published`.

### D7 — Font and stroke-data licensing

Document in `content/hiragana5/LICENSES.md` and README:

| Asset | Requirement |
|-------|-------------|
| Canonical stroke polylines | Project-authored under repo root license (**CC0** per `LICENSE`) **or** third-party paths with explicit compatible license + attribution file |
| Trace templates | Same as stroke data |
| Display fonts (Noto Sans JP / Serif JP) | If vendored later: ship OFL license text under `web/public/fonts/` or `third_party/`; until then document “system/font CDN — not bundled” and do not claim we ship the font files |
| Example words / meanings | Original short glosses under CC0; avoid copying textbook paragraphs |
| Audio (future Prompt 17) | Separate license section; do not invent TTS-as-trusted without review |

Validator does not download fonts; docs task must reconcile README “MIT” vs root CC0 (prefer correcting README to match `LICENSE`).

## Content Review Workflow

```text
Author draft → validate schema locally → open PR with checklist
    → Japanese-speaker review (stroke order, glyph, romanization,
       example word naturalness, description accuracy)
    → fill review.json + set manifest.reviewStatus=published
    → merge → seed imports published pack only
```

**Reviewer checklist (`content/hiragana5/README.md` + `review.json` fields):**

> v1 sign-off lives in `content/hiragana5/v1/review.json` (all checklist keys `true`) with `manifest.reviewStatus=published`. The markdown list below is the workflow template; `content/hiragana5/README.md` mirrors completed v1 checks.

- [x] Exactly five characters; glyphs あいうえお in chart order
- [x] Romanization Hepburn lowercase; matches pronunciation
- [x] Stroke counts and order match standard handwriting pedagogy
- [x] Polylines readable at 0–1 normalized coords; no self-intersecting nonsense
- [x] Trace templates align with stroke order (for animation)
- [x] Example words common, age-neutral, correct spelling/meaning
- [x] Descriptions helpful, not childish, not claiming ML/AI grading
- [x] No unreviewed AI text remains in `v1/`
- [x] License notes accurate

**Qualified reviewer:** native or near-native Japanese literacy sufficient to judge kana stroke order and elementary vocabulary (document expected bar in README; repo does not gate on credentials files).

Until first publish, implementers may land format + validator + fixture plumbing with `reviewStatus=draft` on a branch, but **main seed must fail closed** or keep previous stub only if pack not published — prefer fail-closed once pack path is wired: `Open` returns error if published pack missing/invalid (same severity as migration failure).

**Transition policy:** On first merge of published v1, seed upserts pedagogy fields; recognize switches embed to pack strokes; eval gold fixtures update to new paths (re-baseline gold = templates).

## Deterministic Seed / Import

1. `internal/curriculum` loads pinned pack path (`content/hiragana5/v1`).
2. Validates schema + invariants (see Tests).
3. Computes `contentHash`; compares to `manifest.contentHash` (mismatch → error).
4. `SeedHiragana5`:
   - Single `BEGIN IMMEDIATE` transaction (unchanged pattern).
   - Upsert 5 characters with all pedagogy columns + `content_version`.
   - Upsert lesson title + `lesson_characters` positions = `sortKey`.
   - **Do not** rewrite `romanization`/descriptions from NULL stubs after publish — always overwrite from pack (pack is source of truth).
5. Log: `INFO [db.seed] set=hiragana5 characters=5 lesson=lesson:hiragana5 contentVersion=… contentHash=…`; `DEBUG` per id + strokeCount (no polyline dumps).
6. Re-open is idempotent: same hash → same row field values.

Recognize startup logs: `INFO [main] recognizer=target_compare set=hiragana5 contentVersion=…`.

## Logging Contract (verbose)

| Level | Events |
|-------|--------|
| DEBUG | `[curriculum.load] path=… files=…`; `[curriculum.validate] glyph=あ strokes=3`; `[db.seed] character id=… romanization=… strokeCount=…` (**no coordinates**) |
| INFO | `[curriculum.load] contentVersion=… hash=… reviewStatus=published`; `[db.seed] …`; `[main] schema_version=N learn_seed=hiragana5 contentVersion=…` |
| WARN | Loading draft pack in non-production test harness only; deprecated embed path removed |
| ERROR | Schema validation failure; hash mismatch; seed refused for non-published pack; stroke_count ≠ path length |

Never log full point arrays outside existing `RECOGNIZE_DEBUG` gated paths.

## Out of Scope

- Attempt HTTP APIs / state machine (Prompt 11)
- Learner-facing correction copy and score weights (Prompt 12)
- Guided lesson Vue routes/animation UX (Prompt 13) — only **rendering fixtures** here
- Audio file hosting and `<audio>` UI (Prompt 17)
- Expanding character set beyond five
- SRS / classrooms
- Changing board WS / `boardRev` contracts
- Calibrated confidence / ONNX claims (`docguard` stays green)

## Tasks

### Phase 0: Contracts & licensing docs skeleton
- [x] Task 1: Lock pedagogy set + content axioms in AI context
  - Update `.ai-factory/ARCHITECTURE.md`: add `content/hiragana5/` and `internal/curriculum` (load/validate); recognize + seed both consume the pack; free-board remains separate.
  - Update `.ai-factory/RULES.md` axioms: (1) trusted curriculum is versioned under `content/hiragana5/vN` only; (2) AI drafts stay in `drafts/` until human review publishes; (3) seed/recognize must agree on glyph set, stroke counts, and contentVersion.
  - Update `.ai-factory/DESCRIPTION.md`: mention reviewed five-vowel starter curriculum pack.
  - Add `content/hiragana5/LICENSES.md` + authoring `README.md` (review checklist).
  - LOGGING: none (docs only).

### Phase 1: Format, schema, loader
- [x] Task 2: Author JSON Schema + pack skeleton (depends on 1)
  - Create `content/hiragana5/v1/{manifest,characters,strokes,traces,review}.json` and schema files under `content/schemas/` (or `v1/schema/`).
  - Encode the five あ行 rows (draft descriptions/examples from D4); copy current recognize polylines into `strokes.json` / `traces.json` as starting geometry marked for review.
  - Set `reviewStatus=draft` until Task 8; local tests may use a `testdata` published fixture.
  - Files: `content/hiragana5/**`, schema JSON.
  - LOGGING: n/a (data files).

- [x] Task 3: `internal/curriculum` load + validate library (depends on 2)
  - Package API: `LoadPack(fs, versionDir) (Pack, error)`, `Validate(Pack) error`, `ContentHash(Pack) string`.
  - Invariants: exactly 5 chars; ids/glyphs/sortKeys; `strokeCount==len(strokes)==len(traces)`; coords in [0,1]; unique glyphs; `setId==hiragana5`; manifest hash matches.
  - Reject `drafts/` path in production loader.
  - LOGGING: DEBUG per glyph; INFO on successful load; ERROR with field path on validation failure.
  - Files: `internal/curriculum/*.go`.

### Phase 2: DB + seed + recognize wiring
- [x] Task 4: Migration `0003_curriculum_pedagogy.go` (depends on 1)
  - ALTER `characters` add pedagogy columns (D5); keep forward-only; fail-closed.
  - LOGGING: existing `[db.migrate]` DEBUG/INFO/ERROR.
  - Files: `internal/db/migrations/0003_*.go`, registry bump `LatestVersion`.

- [x] Task 5: Extend `learn.Character` + `LearnStore` (depends on 4)
  - Scan new columns; keep API read-only for curriculum.
  - LOGGING: DEBUG `[learn.CharacterRepo.Get] id=… contentVersion=…` (no PII).
  - Files: `internal/learn/types.go`, `internal/db/learn_store.go`, tests.

- [x] Task 6: Deterministic `SeedHiragana5` from pack (depends on 3, 4, 5)
  - Replace hardcoded slice with curriculum pack load; upsert all pedagogy fields; update lesson title from pack/lesson metadata.
  - Fail `Open` if pack not `published` in production builds; for unit tests, embed a minimal published testdata pack OR set review published after Task 8.
  - Preserve `Hiragana5Glyphs()` via pack glyphs.
  - LOGGING: per D7/logging contract (`contentHash`, `contentVersion`).
  - Files: `internal/db/seed_hiragana5.go`, `internal/db/db.go` if needed.

- [x] Task 7: Point recognize embed at pack strokes (depends on 3, 6)
  - Embed `content/hiragana5/v1/strokes.json` (or shared file); delete or thin-wrap old `testdata` path to avoid dual sources.
  - Re-baseline `target_compare_test` / `eval_hiragana5_test` gold paths.
  - LOGGING: `INFO [main] recognizer=… contentVersion=…`; recognize debug unchanged.
  - Files: `internal/recognize/target_compare.go`, tests, embed paths.

### Phase 3: Review publish + validation tooling
- [x] Task 8: Human review gate + publish v1 (depends on 2, 3)
  - Run checklist with qualified Japanese speaker; correct stroke paths, copy, examples.
  - Fill `review.json`; set `manifest.reviewStatus=published`; refresh `contentHash`.
  - Record reviewer identity/date; ensure no AI-only approval.
  - LOGGING: n/a (process); seed logs published version after merge.

- [x] Task 9: Validation tooling + Makefile target (depends on 3, 8)
  - `go test ./internal/curriculum` as primary gate; optional `go run ./cmd/contentvalidate` wrapping same lib.
  - Makefile: `validate-content` → tests/validator; wire into `make test` or document in README.
  - LOGGING: validator prints INFO summary + ERROR details on stderr.

### Phase 4: Tests, fixtures, docs
- [x] Task 10: Content-schema & alignment tests (depends on 3, 6, 7)
  - Schema validation tests; hash stability test; seed↔pack↔recognize triangle (glyphs, counts, set id, contentVersion).
  - Refuse draft pack in seed test; published pack upserts romanization non-empty.
  - Migration test: fresh Open → schema v3 → 5 characters with pedagogy fields.
  - LOGGING: use `t.Logf` for hash/version in verbose tests.
  - Files: `internal/curriculum/*_test.go`, `internal/db/migrate_test.go`, recognize tests.

- [x] Task 11: Rendering fixtures (frontend) (depends on 2, 7)
  - Export or copy strokes/traces JSON into a Vitest-accessible module (e.g. `web/src/curriculum/hiragana5.json` generated or imported).
  - Unit test: load 5 glyphs; stroke counts match; helper draws template polylines to a mock/canvas context (extend `web/tests` patterns from canvas draw tests).
  - Do **not** build full lesson UI (Prompt 13).
  - LOGGING: n/a (Vitest assertions).
  - Files: `web/src/curriculum/*`, `web/tests/unit/curriculum-*.spec.ts`.

- [x] Task 12: README + honesty docs (depends on 8–11)
  - Document content pack layout, validate command, review workflow, licensing, seed determinism, contentVersion logging.
  - Replace stub seed wording; fix License section to match root `LICENSE` (CC0) or explicitly dual-note if intentional.
  - Extend `docguard` if needed: forbid “AI-authored curriculum” / require honesty about human review.
  - Update `AGENTS.md` tree with `content/` + `internal/curriculum`.
  - LOGGING: document `[curriculum.*]` / `[db.seed]` fields in README logging table.

## Commit Plan
- **Commit 1** (after tasks 1–3): "docs: add hiragana5 content pack axioms and curriculum loader skeleton"
- **Commit 2** (after tasks 4–7): "feat: import reviewed curriculum fields via migration 0003 and pack-backed seed"
- **Commit 3** (after tasks 8–9): "chore: publish hiragana5 content v1 with validation tooling"
- **Commit 4** (after tasks 10–12): "test: add curriculum schema, seed alignment, and trace rendering fixtures"

## Acceptance Criteria

1. Trusted pack exists at `content/hiragana5/v1/` with exactly five あ行 characters and `reviewStatus=published` after human review.
2. Each character has glyph, romanization, pronunciation metadata, description_en, strokeCount, ordered strokes, trace template, example word + English meaning.
3. `SeedHiragana5` is deterministic/idempotent and populates new DB columns from the pack; ids/set unchanged.
4. Recognize assessment paths and seed stroke counts derive from the same pack (no manual dual maintenance).
5. `make validate-content` / `go test ./internal/curriculum` fails on schema breakage, count mismatch, or hash drift.
6. Vitest rendering fixtures load all five trace templates.
7. AI-only content cannot seed: drafts directory ignored; published review record required.
8. README documents pedagogy rationale, format, licensing, review workflow, and logging; `docguard` green.
9. Roadmap milestone remains unchecked until verify.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Reviewer unavailable blocks publish | Land format+tests with testdata published fixture; keep main seed on stub until review — **or** pair-program review early in Task 8 |
| Stroke path changes break eval gold | Expect re-baseline in Task 7/8; keep engineering thresholds documented |
| Embed path fragility (`content/` from multiple packages) | Single `internal/curriculum` embed FS; recognize imports pack bytes via that package |
| README license mismatch (MIT vs CC0) | Explicit docs fix in Task 12 |
| Scope creep into lesson UI | Fixtures only; Prompt 13 owns UX |

## Dependencies

- **Requires:** Prompt 09 complete (migrations, seed entrypoint, learn repos) — done on `main`.
- **Unblocks:** Prompt 11 (attempt APIs reading character metadata), Prompt 12 (target assessment against frozen paths), Prompt 13 (guided lesson copy/trace), Prompt 17 (audioRef fulfillment).
