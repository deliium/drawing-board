# Hiragana5 content pack (あ行)

Trusted curriculum for the five-vowel starter set `あ い う え お` (`set_id=hiragana5`).

## Layout

| Path | Role |
|------|------|
| `v1/` | Published (or pending) pack version — **only** this tree is seeded / embedded |
| `v1/audio/` | Reviewed mora MP3s (`a.mp3`…`o.mp3`); mirrored to `web/public/audio/hiragana5/` |
| `drafts/` | Untrusted AI or WIP material — **never** imported by seed or recognize |
| `LICENSES.md` | Font / stroke-data / example-word / **audio** notes |

| `v1/assessment_review.json` | Human review of scoring tolerances + correction copy (Prompt 12) |

## Authoring

1. Edit files under `v1/` (or stage WIP in `drafts/`). For audio, stage under `drafts/audio/` until listen-reviewed, then copy into `v1/audio/` and set `pronunciation.audioRef` to `/audio/hiragana5/<file>.mp3`.
2. Run `make sync-audio` then `make validate-content` (or `go test ./internal/curriculum` / `go run ./cmd/contentvalidate`). Refresh `contentHash` with `go run ./cmd/contentvalidate -hash` after pedagogy/geometry edits.
3. Open a PR with the reviewer checklist below.
4. After a qualified Japanese-speaker review, fill `v1/review.json`, set `manifest.reviewStatus` to `published`, refresh `contentHash`.
5. For assessment tolerance/copy changes, also update `v1/assessment_review.json` (`correctionsReviewed=true`) and re-run `go test ./internal/recognize -run 'Eval|Fixture' -v`.
6. Merge — `SeedHiragana5` and recognize load the published pack only.

Seed and recognize **fail closed** if the pinned pack is missing, invalid, or not `published`.

## Reviewer checklist

Completed for `v1` (see `v1/review.json`; `manifest.reviewStatus=published`):

- [x] Exactly five characters; glyphs あいうえお in chart order
- [x] Romanization Hepburn lowercase; matches pronunciation
- [x] Stroke counts and order match standard handwriting pedagogy
- [x] Polylines readable at 0–1 normalized coords; no self-intersecting nonsense
- [x] Trace templates align with stroke order (for animation)
- [x] Example words common, age-neutral, correct spelling/meaning
- [x] Descriptions helpful, not childish, not claiming ML/AI grading
- [x] Concise EN/JA guidance distinct from longer descriptions
- [x] Non-null `audioRef` for all five; files present, short, loudness OK; human listen pass
- [x] Audio license/provenance recorded in `LICENSES.md` (no TTS-only without review note)
- [x] `kanjiExtensions` absent/empty on `hira:*`
- [x] No unreviewed AI text remains in `v1/`
- [x] License notes accurate

Reuse this list (unchecked) when publishing a future `vN` pack.

### Schema notes (Prompt 17)

- `schemaVersion` ≥ 2 requires `guidance.{en,ja}` and non-null `audioRef` resolved under `vN/audio/`.
- Optional `kanjiExtensions` (`readings`, `meanings`, `radicals`, `exampleSentences`) is a **future** kanji pack hook — hiragana packs must leave it empty.

### Assessment review (Prompt 12)

See `v1/assessment_review.json`. Before claiming pedagogy-grade scoring:

- [x] Tolerances / weights reviewed as engineering criteria only
- [x] Correction catalog age-neutral; codes stable; ≤2 feedback
- [x] Diagnostics separate from learner-facing messages
- [x] Gold / incorrect / short-stroke fixture suites spot-checked
- [x] Retry guidance present (new attempt; focus on listed corrections)
- [x] No calibrated confidence / AI accuracy claims

**Qualified reviewer:** native or near-native Japanese literacy sufficient to judge kana stroke order and elementary vocabulary.


## Pedagogy rationale

The あ行 vowel row is the first gojuon column. Teaching vowels first matches textbook order, covers the full vowel inventory, and keeps stroke load modest (3/2/2/2/3) before denser consonant rows.
