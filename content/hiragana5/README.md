# Hiragana5 content pack (あ行)

Trusted curriculum for the five-vowel starter set `あ い う え お` (`set_id=hiragana5`).

## Layout

| Path | Role |
|------|------|
| `v1/` | Published (or pending) pack version — **only** this tree is seeded / embedded |
| `drafts/` | Untrusted AI or WIP material — **never** imported by seed or recognize |
| `LICENSES.md` | Font / stroke-data / example-word notes |

| `v1/assessment_review.json` | Human review of scoring tolerances + correction copy (Prompt 12) |

## Authoring

1. Edit files under `v1/` (or stage WIP in `drafts/`).
2. Run `make validate-content` (or `go test ./internal/curriculum` / `go run ./cmd/contentvalidate`).
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
- [x] No unreviewed AI text remains in `v1/`
- [x] License notes accurate

Reuse this list (unchecked) when publishing a future `vN` pack.

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
