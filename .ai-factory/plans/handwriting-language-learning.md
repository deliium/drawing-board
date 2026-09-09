# Implementation Plan: Handwriting Practice with Basic Language Learning (Prompt 17)

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Handwriting practice with basic language learning (Prompt 17)"
Rationale: Prompts 10–16 shipped reviewed hiragana5 pedagogy text, guided journey, bilingual chrome, mastery, and review — but `audioRef` is still null, romanization cannot be hidden, and language learning beyond handwriting cues is thin; this milestone connects sound, vocabulary, and concise guidance to each glyph while reserving kanji-shaped content fields without implementing kanji.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep Prompt 08–16 entries unchanged (do not uncheck completed items).

## Summary

Today each hiragana5 character already has glyph, Hepburn romanization, IPA/`jaHint`, EN/JA description, and one example word — but:

- **`pronunciation.audioRef` is always `null`** — intro shows text only; no playback.
- **Romanization is always visible** — no learner preference to hide it (useful once sound/glyph association improves).
- **“Language learning” is thin** — one example gloss exists, but there is no dedicated short guidance tying sound ↔ writing, and no asset/review pipeline for audio.
- **Kanji-shaped fields are absent** — future readings/meanings/radicals/sentences need explicit schema extension points so Prompt 17 does not paint us into a kana-only JSON shape.

This plan delivers:

1. **Content ownership + review gate** for audio, guidance, and any vocabulary/copy changes under `content/hiragana5/`.
2. **Licensed, reviewed pronunciation audio** for all five vowels (not TTS-as-sole-trusted-source).
3. **Asset format, packaging, caching, and missing-asset fallbacks**.
4. **Accessible play control** on the character intro (and light hub affordance if cheap).
5. **Hideable romanization** (character + example) via client preference.
6. **Concise bilingual learner guidance** per character.
7. **Kanji extension points** in pack schema (validated empty/absent for hiragana5; no kanji UI/seed).
8. **Tests** for missing assets, null `audioRef`, playback errors, romanization toggle, contentvalidate; verbose logs; docs.

**Out of scope:** Implementing kanji glyphs/sets/UI; TTS generation pipelines as production source of truth; CDN/third-party streaming hosts; autoplay; pitch-accent teaching UI; multi-speaker selection; example-word audio (glyph audio only); changing assess criteria / `T_pass` / corrections (Prompt 12); review-box / mastery math (Prompt 15–16); free-board WS/`boardRev`; Playwright/Cypress; new characters beyond hiragana5.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Pack | `content/hiragana5/v1/characters.json` — pedagogy complete; `pronunciation.audioRef: null` for all five |
| Types | `internal/curriculum.Pronunciation` has `AudioRef *string`; `Example` has word/romaji/meanings; no `guidance`; no kanji hooks |
| Validate | Requires descriptions/examples; **does not** require or resolve audio files |
| DB | `pronunciation_json` TEXT; pedagogy columns from migrations `0003`/`0004` — no separate audio table |
| API | `GET /api/lessons/{id}` returns `pronunciation` as JSON blob (includes null `audioRef`); example + `descriptionEn`/`Ja` |
| UI | `CharacterIntroPanel.vue` shows glyph, romanization, IPA, jaHint, description, example — **no audio control, no romaji hide** |
| Static | Go `FileServer` on `-static`; nginx caches `js\|css\|png|…` — **no audio extensions** |
| License | `content/hiragana5/LICENSES.md` placeholders Prompt 17 audio section |
| Prefs | Locale `locale:v1` only — no romanization visibility key |

## Design Decisions

### D0 — Product goal

Connect **handwriting practice** with **basic language learning** for each supported hiragana:

| Element | MVP behavior |
|---------|--------------|
| Pronunciation audio | One reviewed clip per glyph; play on demand from intro |
| Example vocabulary | Keep/enrich the single age-neutral example (word + meanings + romanization) already in pack |
| Hideable romanization | Toggle hides character + example romanization (IPA/jaHint remain; glyph/audio stay) |
| Concise guidance | Short EN/JA tip (sound, mouth shape, or writing cue) separate from longer `description` |
| Kanji future | Schema extension points only — empty for hiragana5 |

### D1 — Content ownership

| Concern | Owner |
|---------|--------|
| Source of truth | Reviewed pack under `content/hiragana5/vN/` (continue `v1/` tree with bumped `contentVersion` / `schemaVersion`, **or** publish `v2/` if implement prefers clean cut — pick one in Task 1 and stick to it) |
| Audio binaries | Same pack tree: `vN/audio/<id-or-glyph>.<ext>` (+ optional mirror copy into `web/public/audio/hiragana5/` for SPA/dev static) |
| Pedagogy copy | `characters.json` — human-edited; AI drafts only in `drafts/` until review |
| Licensing | `content/hiragana5/LICENSES.md` + per-file or per-clip attribution table |
| Runtime DB | Seed upsert from pack (existing `SeedHiragana5`); no dual-maintained pedagogy |
| Client geometry | Trace fixtures unchanged unless `contentVersion` bump requires fixture sync note |

**Rule:** Do not invent unreviewed glosses or claim audio is “native quality” without checklist sign-off in `review.json` (extend checklist).

### D2 — Audio source and licensing (non-negotiable)

| Allowed as trusted MVP source | Disallowed as sole trusted source |
|-------------------------------|-----------------------------------|
| Human-recorded clips by a qualified speaker, reviewed for clarity | Machine TTS (browser `speechSynthesis`, cloud TTS, AI voice) **without** explicit human quality review documented in pack review |
| Third-party clips under a **compatible** license (prefer CC0 / CC-BY with attribution), quality-reviewed | Scraped dictionary audio of unknown license |
| Project-recorded under CC0 (same as pedagogy) | Shipping placeholder silence / beep labeled as pronunciation |

**Process:**

1. Stage candidates under `drafts/audio/` if needed.
2. Move into published `vN/audio/` only after listening review (checklist items: correct phone for glyph, no clipping, adequate loudness, age-neutral, license recorded).
3. If a clip originated as TTS, review record **must** state that a human approved it as teaching-quality **and** license allows redistribution — still prefer real recordings when practical.
4. `docguard` / README: never market audio as “AI pronunciation” or “perfect native TTS.”

### D3 — Asset formats, packaging, caching

| Topic | Decision |
|-------|----------|
| Container/codec | Prefer **one primary** browser-friendly format for all five: **Opus in `.ogg` or `.webm`**, **or** `.mp3` if tooling is simpler. Document the choice in LICENSES/README. Optional secondary `<source>` only if needed for a documented browser gap — avoid format sprawl. |
| Naming | Stable logical id in `audioRef`, e.g. `hiragana5/a.ogg` or `/audio/hiragana5/a.ogg` — **same string** in pack JSON and on disk relative to the static root. |
| Duration | Short isolated mora (~0.3–1.0s) + tiny leading/trailing silence OK; not full vocabulary utterances. |
| Loudness | Roughly normalized across the five clips (document target; no need for broadcast loudness certification). |
| Packaging | Pack tree owns canonical files; Vite/dev serves via `web/public/audio/…` (copy or documented sync in Makefile/`validate-content`). Production static dir must include the same paths. |
| Caching | Hash **or** `contentVersion` in cache story: nginx/static `Cache-Control` for audio (`immutable` only if filename includes content hash; otherwise shorter max-age or versioned path). Extend nginx static regex to include `ogg|mp3|webm|wav` as applicable. |
| Size | Keep each clip tiny (target well under ~50KB compressed); fail review if unexpectedly large. |

**Validator:** When `audioRef` is non-null, resolve path under pack (and/or public mirror) and **fail closed** if file missing or empty. When null, warn in tests for hiragana5 MVP that production pack **should** have audio (MVP gate: all five non-null after this plan).

### D4 — Accessible playback

| Requirement | Behavior |
|-------------|----------|
| Control | Explicit **Play pronunciation** button (not hover-only); optional native `<audio controls>` is acceptable if styled to notebook system — prefer one clear button + hidden/managed `<audio>` for consistent a11y |
| Autoplay | **Never** |
| Name | `aria-label` / visible text from i18n (`intro.playAudio`, includes glyph when useful) |
| State | Pressed/playing vs idle; disable or announce when unavailable |
| Keyboard | Focusable, activatable with Enter/Space |
| Screen readers | On play start / error, update an existing or intro-local **live region** (`polite`) — e.g. “Playing あ” / “Audio unavailable” |
| Reduced motion | N/A for sound; do not tie playback to CSS animation |
| Fallback | IPA + `jaHint` + guidance always visible without audio |

### D5 — Offline / missing / error behavior

| Case | UI / API |
|------|----------|
| `audioRef` null or empty | Omit play control (or disabled + `intro.audioUnavailable`); journey continues |
| 404 / network / decode error | Catch `error` on `<audio>`; live region message; keep text pedagogy; log WARN once per ref |
| Partial pack (dev broken mirror) | Contentvalidate fails in CI; runtime still soft-fails playback |
| Service worker / true offline vault | **Out of scope** — browser HTTP cache only |

Do **not** block stroke-order / trace / assess on audio failure.

### D6 — Bilingual copy and romanization preference

| Surface | Rule |
|---------|------|
| Guidance | New `guidance: { en, ja }` (required non-empty for hiragana5 after bump) — concise (≈1 short sentence); distinct from `description` |
| Descriptions / meanings | Remain pack-owned; locale picker already chooses EN/JA |
| Romanization visibility | Client preference `localStorage` key e.g. `romanizationVisible:v1` (`true` default); toggle in practice shell or intro; when hidden, still expose romanization to SR **only if** product decides — **prefer:** hide visually and from accessibility tree when user opted to practice without romaji (`aria-hidden` on romaji spans), since that is the point of the toggle |
| `lang` | Keep `lang="ja"` on glyphs/examples; romaji/`ipa` use `lang="en"` when parent is JA (existing Prompt 14 pattern) |

### D7 — Kanji extension points (schema only)

Extend pack `Character` (JSON) with an optional object, e.g. `kanjiExtensions` **or** top-level optional fields that hiragana5 leaves absent/null:

| Field | Purpose (future) | Hiragana5 now |
|-------|------------------|---------------|
| `readings` | on/kun (or typed reading list) | omit / empty |
| `meanings` | gloss list beyond single example | omit / empty |
| `radicals` | radical ids/labels | omit / empty |
| `exampleSentences` | short sentence + reading + gloss | omit / empty |

**Validation:** For `setId=hiragana5` / ids prefixed `hira:`, reject **non-empty** kanji extension payloads (keeps pack honest). Types + docs describe the shape so a future kanji pack can fill them without another breaking rename. **No** DB columns required if extensions stay inside `pronunciation_json`-style JSON **or** a new `extensions_json` column — prefer **pack JSON only + pass-through in lesson API as `extensions`/`kanji` object** so SQLite stays thin unless seed already flattens everything. Practical recommendation:

- Keep flattened pedagogy columns as today.
- Store `guidance_en` / `guidance_ja` as new columns **or** embed guidance inside an extended pedagogy JSON — prefer **two columns** mirroring `description_*` for query simplicity.
- Pass `kanjiExtensions` (raw object) only via pack → optional API field; **do not** flatten into many nullable SQLite columns yet.

Migration: only if new guidance columns (or `extensions_json`) are chosen — `0006_*`.

### D8 — Schema / API / UI changes

**Pack (`schemaVersion` bump, e.g. 1 → 2):**

- `pronunciation.audioRef`: non-null string for all five.
- `guidance: { en, ja }`.
- Optional `kanjiExtensions` (empty/absent).
- Refresh `contentHash`, extend `review.json` checklist (audio + guidance + license).
- Bump `contentVersion` (e.g. `hiragana5-content-v2`).

**DB / seed:** Upsert guidance; pronunciation JSON includes audioRef; fail closed if pack invalid.

**API `GET /api/lessons/{id}`:** Add `guidanceEn` / `guidanceJa` (or nested `guidance`); ensure `pronunciation.audioRef` string reaches clients; optional `kanjiExtensions: null` or omit.

**UI:**

- `CharacterIntroPanel`: play control, guidance paragraph, respect romanization visibility.
- Preference toggle (hub header or intro) with i18n EN/JA.
- Hub list: honor hide-romaji for row romanization.
- `curriculumApi` types updated.

**Logging (verbose):**

| Level | Examples |
|-------|----------|
| DEBUG | `[curriculum.validate] audioRef=… ok bytes=…`; `[CharacterIntroPanel] play ref=…`; romanization pref changes |
| INFO | `[curriculum.load] contentVersion=… audioFiles=5`; seed contentVersion |
| WARN | `[curriculum.validate] audio missing`; `[audio] load error ref=… status=…` (no PII) |
| ERROR | seed/validate fail-closed |

Never log raw audio bytes.

## Acceptance Criteria

1. All five hiragana have reviewed non-null `audioRef` pointing at real licensed files; `make validate-content` fails if a ref is dangling.
2. Intro play control is keyboard-accessible, labeled, no autoplay; errors soft-fail with live-region copy; text pedagogy remains.
3. Example vocabulary + concise EN/JA guidance visible; locale switch works.
4. Learner can hide/show romanization; preference persists in `localStorage`; hub + intro honor it.
5. Pack/docs define kanji extension points; hiragana5 does not populate them; validator rejects non-empty kanji extensions on `hira:*`.
6. TTS is not the only undocumented audio source — LICENSES + review checklist record provenance and human review.
7. Go + Vitest cover missing asset validation, null-ref UI fallback, playback error path, romaji toggle; `docguard` green.
8. README / ARCHITECTURE / AGENTS / content README / LICENSES updated; Prompt 17 milestone left unchecked until verify.

## Scenario Checklist

### S1 — Happy path intro

1. Open あ → hear play control; press → mora audio plays; guidance + example + description show in active locale.

### S2 — Hide romanization

1. Toggle hide → character `a` and example `asa` disappear from UI; glyph/audio/IPA policy per D6 remain usable.

### S3 — Missing audio soft-fail

1. Fixture character with `audioRef: null` → no play button (or disabled unavailable); journey stages still work.

### S4 — Broken URL

1. Non-null ref to missing file → play attempt → error live region; no throw killing the page.

### S5 — Contentvalidate

1. Pack with dangling `audioRef` → validate fails; published pack with five good refs → passes + hash match.

### S6 — Kanji hooks inert

1. Pack with non-empty `kanjiExtensions` on `hira:あ` → validate error; empty/absent → OK.

## Tasks

### Phase 1: Content schema, licensing, assets, validation

- [x] Task 1: Decide pack layout (`v1` in-place bump vs `v2/` directory), bump `schemaVersion`/`contentVersion`, add `guidance`, require non-null `audioRef` for hiragana5, add optional `kanjiExtensions` shape + reject non-empty on `hira:*`. Extend `Validate` to resolve audio files (size > 0). Update `contentvalidate` CLI help if needed. Refresh hash workflow.

  LOGGING: DEBUG per glyph audioRef/path/bytes; INFO load summary with audio count; ERROR on missing file.

  Files: `internal/curriculum/pack.go`, `internal/curriculum/*_test.go`, `cmd/contentvalidate/main.go`, `content/hiragana5/vN/manifest.json`, `characters.json`, `review.json`, `content/hiragana5/README.md`

- [x] Task 2: Source or record five mora clips; write `vN/audio/*`; document provenance/license/review in `LICENSES.md` + review checklist (human listen pass; **no TTS-only without explicit review note**). Mirror/sync into `web/public/audio/hiragana5/` (Makefile target or documented copy). Ensure nginx/static cache rules cover chosen extension(s).

  LOGGING: n/a for binaries; validate logs cover presence.

  Files: `content/hiragana5/LICENSES.md`, `content/hiragana5/vN/audio/*`, `web/public/audio/hiragana5/*`, `Makefile`, `docker/nginx.conf` (+ example TLS conf if mirrored), optionally `content/hiragana5/embed.go` if embed strategy chosen

  Depends on: 1

- [x] Task 3: DB migration if guidance columns / `extensions_json` needed; extend `learn.Character`, seed upsert, lesson API DTO (`guidanceEn`/`guidanceJa`, audioRef passthrough, optional extensions). Handler tests assert audioRef + guidance for all five.

  LOGGING: `[db.seed]` INFO contentVersion + audio refs present count; DEBUG per character id (no paths spam at INFO); `[httpapi.Lesson.Get]` DEBUG character count / missing guidance WARN.

  Files: `internal/db/migrations/0006_*.go` (if needed), `migrations.go`, `internal/learn/types.go`, `internal/db/seed_hiragana5.go`, `internal/db/learn_store.go`, `internal/httpapi/curriculum.go`, `internal/httpapi/curriculum_test.go`

  Depends on: 1, 2

<!-- Commit checkpoint: tasks 1–3 -->

### Phase 2: Learner UI — audio, guidance, romanization preference

- [x] Task 4: Add `useRomanizationPreference` (or extend `useLocale` sibling) with `localStorage` key; EN/JA i18n for toggle + audio strings; wire toggle into practice chrome (hub and/or journey).

  LOGGING: DEV `console.debug` on pref read/write; never at production INFO.

  Files: `web/src/composables/useRomanizationPreference.ts` (new), `web/src/i18n/locales/en.ts`, `web/src/i18n/locales/ja.ts`, `web/src/pages/PracticeHubPage.vue`, practice layout/shell if present

  Depends on: 3

- [x] Task 5: Extend `curriculumApi` types; update `CharacterIntroPanel` with guidance, hideable romanization, accessible audio play control + error live region + null-ref fallback. Light hub romaji hide. Prefer existing a11y patterns from Prompt 14.

  LOGGING: DEV debug play/error/ref; WARN once on media error.

  Files: `web/src/services/curriculumApi.ts`, `web/src/components/practice/CharacterIntroPanel.vue`, `web/src/pages/PracticeHubPage.vue`, optional tiny `PronunciationAudioButton.vue`

  Depends on: 2, 4

<!-- Commit checkpoint: tasks 4–5 -->

### Phase 3: Tests + docs

- [x] Task 6: Tests — curriculum validate missing/empty audio file; reject kanji extensions on hiragana; API guidance/audioRef; Vitest intro: null audioRef hides control, broken src soft-fails, romaji toggle hides text, guidance locale; axe smoke on intro with play button if practical; regression journey/hub.

  LOGGING: assert WARN/debug spies only where existing test style does.

  Files: `internal/curriculum/*_test.go`, `internal/httpapi/curriculum_test.go`, `web/tests/unit/practice-components.spec.ts`, `web/tests/unit/*romanization*`, `web/tests/integration/accessibility-parity.spec.ts`, `web/tests/contract/curriculum-progress-api.spec.ts`

  Depends on: 5

- [x] Task 7: Docs checkpoint — README (audio ownership, license, formats, caching, a11y playback, offline/error fallbacks, romanization preference, kanji extension points, honesty: no TTS-only trust without review); update `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, `.ai-factory/DESCRIPTION.md`, `.ai-factory/RULES.md` (trusted audio under pack + review); `content/hiragana5/README.md` authoring steps; `go test ./internal/docguard`. Leave Prompt 17 milestone unchecked until verify.

  LOGGING: document `[curriculum.validate]` / audio WARN prefixes in README log table if one exists.

  Files: `README.md`, `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, `.ai-factory/DESCRIPTION.md`, `.ai-factory/RULES.md`, `content/hiragana5/README.md`, `content/hiragana5/LICENSES.md`, `internal/docguard/*` as needed

  Depends on: 6

<!-- Commit checkpoint: tasks 6–7 -->

## Commit Plan

- **Commit 1** (after tasks 1–3): `feat(curriculum): hiragana5 audio refs, guidance, and validation`
- **Commit 2** (after tasks 4–5): `feat(web): pronunciation playback, guidance, hideable romanization`
- **Commit 3** (after tasks 6–7): `docs: language-learning audio and content extension points`

## Risks / Notes

- **License friction:** If suitable CC0/CC-BY clips are unavailable, record in-house; do not ship scraped audio.
- **TTS temptation:** Explicitly blocked as sole trusted source without review — implement must leave paper trail in `LICENSES.md` / `review.json`.
- **Static path drift:** Pack ref vs `web/public` mirror can diverge — single Makefile/`validate-content` check should compare or validate both.
- **Cache immutability:** Do not mark audio `immutable` unless filenames are content-hashed.
- **Kanji scope creep:** Extension points are types + validation + docs only — no kanji routes, seeds, or assess templates.
- **Independence from Prompt 16:** Audio/content must not depend on review boxes/`due_at`.

## Implementation Notes for `/aif-implement`

- Stay on **main** (user requested no new branch).
- Prefer small diffs: pack + validate first, then API/seed, then Vue.
- Reuse Prompt 14 a11y/i18n patterns; do not redesign the notebook visual system.
- Every task needs logging as specified; media errors soft-fail.
- Docs policy: **mandatory** checkpoint (Settings Docs: yes).
- Do not mark ROADMAP Prompt 17 complete in this plan’s implementation commits — leave for verify/roadmap.
