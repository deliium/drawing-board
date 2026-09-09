# Implementation Plan: Deterministic Target-Specific Handwriting Assessment (hiragana5 MVP)

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Deterministic target-specific handwriting assessment for hiragana5 MVP (Prompt 12)"
Rationale: Turns Prompt 08’s coarse target-compare engine into pedagogy-grade, explainable scoring (normalization + multi-criterion match + ≤2 actionable corrections) so attempt APIs return coaching feedback rather than opaque candidate glyphs — without claiming calibrated AI confidence.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep existing Prompt 08–11 entries unchanged (do not uncheck completed items; leave Prompt 11 checked/unchecked as found at implement time).

## Summary

Prompt 08 shipped `TargetCompareRecognizer.Assess` with **stroke-count + coarse resampled geometry** (weights 0.35 / 0.65), `scoreKind=match`, and sparse reason codes. Prompt 10 froze pack polylines; Prompt 11 wires assess onto attempt strokes and maps a few stable codes into `assessment_feedback` with **empty `message`**. Today learners who call assess mostly see a bounded score, `candidates` among five glyphs, and diagnostic float strings mixed into `reasons` — not coaching.

This plan delivers:

1. **Documented normalization** of learner and canonical strokes (shared unit space, aspect policy, short-stroke / one-point handling).
2. **Multi-criterion scoring** — stroke count, order, start/end direction, relative placement, proportions, overall shape — with explicit tolerances and weights.
3. **Bounded overall match score** + pass/fail against engineering `T_pass`, still `scoreKind=match`.
4. **At most two** ranked, learner-friendly corrections (`feedback[].code` + non-empty `message`); internal diagnostics stay separate from user-facing language.
5. **Fixture packs** of representative correct and incorrect samples; **eval harness** gates; **expert review** checklist for tolerances and copy.
6. **Retry guidance** in correction copy / docs (retry = new attempt row; what to practice next).
7. **Tests + verbose logs + README/axioms/docguard** honesty (no calibrated confidence; correctness only vs tested criteria).

**Out of scope:** Guided lesson Vue UI / overlays (Prompt 13); bilingual i18n of correction strings (Prompt 14 may externalize); SRS / mastery UX (Prompt 15–16); new characters beyond `hiragana5`; ML/ONNX; changing attempt REST status machine or WS/`boardRev`; persisting mid-draft strokes; classrooms.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Assessor | `TargetCompareRecognizer.Assess` in `internal/recognize/target_compare.go` |
| Normalization | Whole-ink AABB → unit box, aspect preserved via `max(w,h)` scale (letterbox); no per-stroke local norm |
| Criteria today | `strokeCountScore` + `geometryScore` (index-aligned resample distance); no explicit order / direction / placement / proportion terms |
| Weights | `weightStrokeCount=0.35`, `weightGeometry=0.65` (hard-coded) |
| Pass | `PassThreshold=0.70` and top ranked glyph == target |
| Reasons | Mix of diagnostic floats (`stroke_count=…`, `geometry=…`, `combined=…`) and sparse codes (`empty_strokes`, `stroke_count_mismatch`, `outranked_by_other`, `top_match`) |
| HTTP feedback | `feedbackFromReasons` keeps ≤2 **stable codes only**; `Message` always `""` |
| Short / one-point | `resampleStroke` duplicates single point; zero-length path treated as constant point — no pedagogy label |
| Fixtures | Gold = pack templates; wrong-char = other templates; one near-miss (drop stroke); no direction/order/placement corpora |
| Eval | `TestEvalHiragana5Summary` gold-pass / wrong-reject only |
| Content | `content/hiragana5/v1/strokes.json` normalized 0–1 polylines; `review.json` pedagogy checklist (not scoring tolerances) |
| Honesty | README + docguard: match scores, not confidence; attempt path only for practice |

## Design Decisions

### D1 — Pedagogy assessor extends target-compare; free-board heuristic stays separate

| Surface | Role after this plan |
|---------|----------------------|
| `Assessor.Assess` | Multi-criterion target assessment + diagnostics + ≤2 corrections |
| `Recognizer.Recognize` | Unchanged free-board heuristic ranking (`scoreKind=match`) |
| Attempt HTTP | Persist overall score/pass + feedback messages; optionally slim `reasons` to diagnostic codes only |

Do **not** invent a second recognizer type or ONNX path. Prefer evolving `TargetCompareRecognizer` (and split helpers into files) over a parallel engine.

### D2 — Normalization contract (learner + canonical)

Apply the **same** pipeline to learner strokes and pack templates before criterion scoring:

1. Drop empty-point strokes (treat as absent; count toward stroke-count mismatch).
2. Compute whole-ink axis-aligned bounds; reject / fail closed if no points (`empty_strokes`).
3. Translate so min corner → origin; scale by `1/max(w,h)` (preserve aspect; letterbox in unit square). Degenerate axis (`w` or `h` < ε) → clamp to ε (dots / vertical-only strokes).
4. **Per-stroke classification** after global norm:
   - **Dot / short:** path length ≤ `L_short` **or** ≤1 distinct point → compare by **centroid** only; skip direction scoring for that stroke.
   - **Normal:** resample to `N` points (keep `16` unless fixtures force change); score shape + direction.
5. Do **not** reverse stroke direction automatically — wrong direction is a scored defect (except dots).
6. Canvas `width`/`height` remain for limits validation only; comparison uses normalized geometry (canvas size must not change pass for uniformly scaled ink — already tested; keep that invariant).

Document ε, `L_short`, and `N` in README + code constants with names (not magic numbers buried in helpers).

### D3 — Criterion scores (internal diagnostics)

Each criterion produces a bounded `[0,1]` **match** component (never labeled confidence):

| Code | What it measures | Sketch |
|------|------------------|--------|
| `stroke_count` | \|got − want\| vs want | Keep / refine existing `strokeCountScore` |
| `stroke_order` | Whether stroke *i* best matches template *i* vs a permutation | If counts equal: greedy or small-n assignment on start-point distance; score = fraction of strokes in canonical order; if counts differ, score low and rely on count criterion |
| `start_end_direction` | Unit tangent at start / end of each paired normal stroke | Cosine similarity vs template; average; dots contribute `1` (N/A) or are excluded from the average denominator |
| `relative_placement` | Stroke centroids / start points in unit box vs template | Mean distance → match score with tolerance `T_place` |
| `proportions` | Relative stroke lengths and/or bounding-box aspect of parts | Compare length ratios and overall ink aspect within `T_prop` |
| `shape` | Resampled point-wise distance (existing `strokeSimilarity`) | Keep as overall shape term; optionally slightly lower weight once other terms exist |

**Overall score** = weighted sum of components, clamped to `[0,1]`. Publish weights in one const block / small `Weights` struct. Initial proposal (tune against fixtures in Task 5–6; do not treat as sacred):

| Criterion | Weight |
|-----------|--------|
| stroke_count | 0.15 |
| stroke_order | 0.15 |
| start_end_direction | 0.15 |
| relative_placement | 0.15 |
| proportions | 0.10 |
| shape | 0.30 |

**Pass rule (keep honesty):** `pass = (overall ≥ T_pass) && (top-of-set rank == target)` **or** (preferred simplification for attempt path): `pass = overall ≥ T_pass` **and** no hard-fail gate. Decide in Task 1 and stick to it:

- **Preferred for pedagogy:** hard-fail if `stroke_count` is exact-mismatch beyond `±0` when `want ≤ 3` (MVP glyphs are 2–3 strokes) → `pass=false` even if shape is high; still compute overall for progress UX.
- Keep ranking-within-set as **diagnostic** (`candidates`) for debugging; attempt UI should emphasize target score + corrections, not “you wrote い”.

Re-baseline `T_pass` only if gold fixtures still pass and wrong/incorrect suites still fail; document new value if changed.

### D4 — Tolerances (engineering, not marketing accuracy %)

| Symbol | Default (start) | Used for |
|--------|-----------------|----------|
| `T_pass` | `0.70` (revisit) | Overall pass bar |
| `ε_bounds` | `1e-9` | Degenerate bbox |
| `L_short` | `0.04` (unit space) | Dot / short-stroke classification |
| `T_dir` | cosine ≥ `0.5` soft; map to continuous score | Direction component |
| `T_place` | mean centroid distance `0.15` → score ~0 | Placement falloff |
| `T_prop` | relative length ratio error `0.35` | Proportions falloff |
| `N_resample` | `16` | Shape / direction sampling |

All thresholds live in `internal/recognize` constants (or `tolerances.go`) and are named in README “Engineering criteria” — never as “X% accurate handwriting recognition”.

### D5 — Diagnostics vs user-facing language

| Channel | Audience | Content |
|---------|----------|---------|
| `diagnostics` (new on `Assessment`, JSON optional / DEBUG) | Engineers, logs, tests | Per-criterion scores, weights, hard-fail flags, stroke indices involved — **no coordinates** in default logs |
| `reasons` | Backward-compatible API | Prefer **stable codes + optional compact diagnostic tokens**; stop advertising raw `stroke_count=0.123` as the primary learner signal (may remain under diagnostics or DEBUG-only) |
| `feedback` (≤2) | Learners | `{rank, code, message}` with **non-empty** English message for MVP; actionable; age-neutral |

**Selection of ≤2 corrections:**

1. Map failing / weak criteria (score < per-criterion soft threshold, e.g. `0.75`) to correction codes.
2. Priority order (highest first): `stroke_count_mismatch` → `stroke_order` → `start_direction` / `end_direction` → `relative_placement` → `proportions` → `shape`.
3. Emit at most two; if `pass` and all criteria strong → empty feedback (or single optional praise is **out of scope** — stay correction-oriented).
4. `empty_strokes` → one feedback item only.

Correction catalog (stable codes → messages) lives in `internal/recognize/corrections.go` (or `feedback_catalog.go`), not in Vue and not in SQL seeds. HTTP must stop inventing a parallel map: prefer `assessment.Feedback` from the engine, with `feedbackFromReasons` reduced to a thin adapter / removed.

Example messages (illustrative; finalize under expert review):

| Code | Message |
|------|---------|
| `empty_strokes` | “No strokes were submitted. Draw the character, then try again.” |
| `stroke_count_mismatch` | “Use {want} strokes for 「{glyph}」 (you used {got}).” |
| `stroke_order` | “Check stroke order — start with the stroke that begins at the top/left for 「{glyph}」.” |
| `start_direction` | “Start stroke {n} in the same direction as the model (see the tip of the first movement).” |
| `end_direction` | “Finish stroke {n} in the expected direction.” |
| `relative_placement` | “Place the strokes closer to their usual positions relative to each other.” |
| `proportions` | “Adjust the length or size of the strokes so parts of 「{glyph}」 match usual proportions.” |
| `shape` | “The overall shape of 「{glyph}」 still differs from the model — slow down and retrace.” |

Include **retry guidance** in messages or a short README/lesson note: assessed attempts are immutable; create a new attempt and focus on the listed corrections. Do not claim the score is scientific mastery.

### D6 — Dots and short strokes

- Classify after global normalization using `L_short` / single-point rule.
- Scoring: centroid placement contributes to `relative_placement` and (lightly) `shape`; **skip** `start_end_direction` for that stroke.
- Logging: `DEBUG` counts `shortStrokeCount` / `normalStrokeCount` — never point dumps.
- Note: MVP `hiragana5` has no dakuten dots; “dots” means one-point taps / near-zero ink (canvas lifecycle already allows taps). Still required so short marks on う/え first strokes are not over-penalized for direction.

### D7 — Explainability & honesty bounds

- Every `pass=false` should usually yield ≥1 feedback item when strokes non-empty (except pure outranked edge cases — convert those into `shape` / criterion feedback rather than “you look like another glyph” as the primary tip).
- Docs and API comments: score = **deterministic match vs pack templates + listed criteria**; correctness claims limited to **fixture-tested** behaviors.
- docguard: forbid presenting assessment as calibrated confidence / AI; allow “match score”, “engineering threshold”, “criteria”.
- Do **not** surface internal weight tables in learner JSON; weights may appear in README engineering section.

### D8 — Fixtures, expert review, evaluation

**Fixtures** under `internal/recognize/testdata/hiragana5/` (JSON), not only live pack clones:

| Suite | Intent |
|-------|--------|
| `gold/` | Correct (pack-aligned or lightly jittered within tolerances) → `pass=true`, empty or no critical feedback |
| `incorrect/count/` | Wrong stroke count → fail + `stroke_count_mismatch` |
| `incorrect/order/` | Swapped strokes → fail or non-pass + `stroke_order` |
| `incorrect/direction/` | Reversed stroke → direction correction |
| `incorrect/placement/` | Translated parts → placement |
| `incorrect/proportions/` | Distorted length ratios → proportions |
| `incorrect/wrong_glyph/` | Other character ink vs target → non-pass |
| `short_strokes/` | One-point / tiny strokes behave per D6 |

**Expert review:** add `content/hiragana5/v1/assessment_review.json` (or extend `review.json`) checklist: tolerances reviewed, correction copy age-neutral, codes stable, gold/incorrect suites spot-checked by a human. AI-drafted copy must not ship without checklist `correctionsReviewed=true`.

**Eval harness:** extend `eval_hiragana5_test.go` (or `eval_assessment_test.go`) to print suite counts under `-v` and **fail CI** if gold fails or incorrect suites unexpectedly pass / miss expected top correction code.

### D9 — HTTP / persistence mapping

- `SaveResult` already stores `reasons` + `feedback` (rank, code, message) — fill messages from engine.
- Assess response: include `feedback` with messages; keep `scoreKind=match`; `candidates` may remain on live assess for debugging but docs should say practice UI should prefer score + feedback (Prompt 13).
- Prefer putting structured diagnostics on `Assessment` for tests; HTTP may omit heavy diagnostics in GET assessment (persist reasons codes; full diagnostic floats optional / not required in DB).
- No schema migration required unless messages exceed practical TEXT size (they will not).

### D10 — Compatibility

- Attempt APIs, CSRF, rate limits, ownership: unchanged.
- Pack stroke paths: reuse `content/hiragana5/v1/strokes.json`; do not dual-maintain embeds.
- If weights/tolerances change gold behavior, update fixtures and README in the same change.
- Vue: optional thin update to `attemptsApi.ts` types for feedback messages only — **no** lesson UI.

## Logging Contract (verbose)

| Level | Events |
|-------|--------|
| DEBUG | `[recognize.Assess]` target, strokeCount, pointCount, shortStrokeCount, per-criterion scores, overall, pass, feedback codes; `[recognize.normalize]` degenerate-axis / empty drops (**no coordinates**); `[httpapi.Attempt.Assess]` attemptID, pass, score, feedbackCodes |
| INFO | Assess complete with pass/score/scoreKind/feedbackCount; eval summary lines in tests via `t.Logf` |
| WARN | Unsupported target; catalog miss for a code; unexpected hard-fail without feedback |
| ERROR | Template load failure; assess panics recovered as errors |

Respect `LOG_LEVEL` / existing `logf`. `RECOGNIZE_DEBUG` may dump extra feature summaries still without full point arrays in production.

## Out of Scope

- Prompt 13 guided lesson / comparison overlay UI
- Japanese localization of correction strings (Prompt 14)
- Mastery / history product UI (Prompt 15)
- Changing attempt state machine or board recognize
- Expanding glyph set; dakuten/handakuten curriculum
- ML models; claiming calibrated confidence
- Auto-praise / gamification streaks

## Tasks

### Phase 0: Contracts & axioms
- [x] Task 1: Lock assessment pedagogy axioms + API shape
  - Update `.ai-factory/RULES.md`: multi-criterion deterministic match; ≤2 learner corrections; diagnostics ≠ user copy; no confidence claims; correctness bounded to tested criteria.
  - Update `.ai-factory/ARCHITECTURE.md` / `DESCRIPTION.md`: assessor owns scoring + correction catalog; httpapi persists engine feedback.
  - Freeze pass rule + whether `candidates` remain first-class on attempt assess (document preference: diagnostics/candidates secondary).
  - LOGGING: none (docs only).
  - Files: `.ai-factory/RULES.md`, `.ai-factory/ARCHITECTURE.md`, `.ai-factory/DESCRIPTION.md`.

### Phase 1: Normalization & criterion engine
- [x] Task 2: Normalization helpers + short-stroke classification (depends on 1)
  - Extract/refine `normalizeStrokes` into documented helpers; add length/centroid utilities; classify short vs normal.
  - Unit tests: translation/scale invariance; degenerate vertical/horizontal; one-point stroke classification.
  - LOGGING: `DEBUG [recognize.normalize]` counts only.
  - Files: `internal/recognize/normalize.go` (new), `normalize_test.go`, touch `target_compare.go`.

- [x] Task 3: Implement criterion scorers + weighted overall (depends on 2)
  - Add scorers for count, order, direction, placement, proportions, shape; `Weights` + tolerances constants.
  - Wire `Assess` to compute diagnostics, overall, pass; keep `scoreKind=match`.
  - LOGGING: `DEBUG [recognize.Assess]` criterion breakdown; INFO not required per call (HTTP already INFO).
  - Files: `internal/recognize/criteria.go` (new), `tolerances.go` (new), `target_compare.go`, `interface.go` (extend `Assessment` with `Diagnostics` / structured fields as needed).

- [x] Task 4: Correction catalog + ≤2 feedback selection (depends on 3)
  - Implement catalog + selector priority; fill `FeedbackItem.Message`; ensure empty strokes / count mismatch paths.
  - Remove or shrink `feedbackFromReasons` in httpapi to use engine feedback.
  - LOGGING: `DEBUG` selected codes; `WARN` on unknown code.
  - Files: `internal/recognize/corrections.go`, `corrections_test.go`, `internal/httpapi/attempts.go`, tests under `internal/httpapi/attempts_*.go`.

### Phase 2: Fixtures, review, eval
- [x] Task 5: Representative fixture packs (depends on 3, 4)
  - Author JSON fixtures per D8; loader helper for tests.
  - Table-driven tests: expected pass/fail + expected primary feedback code where applicable.
  - LOGGING: `t.Logf` on assertion failure with diagnostic scores.
  - Files: `internal/recognize/testdata/hiragana5/**`, `fixtures_test.go` / extend `target_compare_test.go`.

- [x] Task 6: Eval harness + tolerance re-baseline (depends on 5)
  - Extend eval test to cover incorrect suites; enforce gates; adjust `T_pass`/weights only with fixture evidence; document final numbers.
  - LOGGING: `t.Logf` suite summary (`gold_pass`, `incorrect_unexpected_pass`, etc.).
  - Files: `internal/recognize/eval_hiragana5_test.go` (or sibling), `interface.go` thresholds, README engineering notes (draft for Task 8).

- [x] Task 7: Expert review artifact for tolerances + copy (depends on 4, 5)
  - Add/update review JSON checklist; ensure corrections and tolerances marked reviewed before claiming done.
  - LOGGING: none (content).
  - Files: `content/hiragana5/v1/assessment_review.json` and/or `review.json`, `content/hiragana5/README.md` snippet.

### Phase 3: HTTP integration hardening & regression
- [x] Task 8: Attempt assess response contract tests (depends on 4, 5)
  - Assert assess returns ≤2 feedback with non-empty messages on known incorrect submit; gold-ish submit → pass with empty/weak feedback; `scoreKind=match`; still no board stroke reads.
  - LOGGING: existing attempt assess logs; assert no panic under `LOG_LEVEL=debug`.
  - Files: `internal/httpapi/attempts_test.go`, `attempts_e2e_test.go` as needed.

### Phase 4: Docs & honesty
- [x] Task 9: README + docguard + optional client types (depends on 6, 7, 8)
  - Document normalization, criteria, tolerances, feedback rules, retry guidance, eval how-to, honesty limits.
  - Expand docguard if new forbidden phrasings appear; keep match-score language.
  - Optional: `web/src/services/attemptsApi.ts` feedback typing + Vitest contract for message presence.
  - LOGGING: document `[recognize.Assess]` diagnostic fields.
  - Files: `README.md`, `internal/docguard/*`, optionally `web/src/services/attemptsApi.ts`, `web/tests/contract/attempts-api.spec.ts`.

<!-- Commit checkpoint: tasks 1-4 -->
<!-- Commit checkpoint: tasks 5-7 -->
<!-- Commit checkpoint: tasks 8-9 -->

## Commit Plan
- **Commit 1** (after tasks 1–4): `feat(recognize): multi-criterion assessment and learner correction catalog`
- **Commit 2** (after tasks 5–7): `test(recognize): hiragana5 correct/incorrect fixtures and assessment review`
- **Commit 3** (after tasks 8–9): `docs: document assessment criteria, tolerances, and feedback honesty`

## Acceptance Criteria

1. `Assess` returns a bounded overall match score (`scoreKind=match`) derived from documented criteria (count, order, direction, placement, proportions, shape) after shared normalization.
2. Non-pass assessments with strokes yield **at most two** feedback items with stable codes and **non-empty** learner-facing messages; diagnostics/criterion scores are separable from that copy.
3. Short/one-point strokes follow D6 (no bogus direction failures); scale/translate invariance of gold ink remains.
4. Fixture suites: gold passes; representative incorrect samples fail with the expected primary correction code (within documented tolerance).
5. Eval harness fails CI on gold failure or incorrect unexpected pass; thresholds documented as engineering criteria only.
6. Expert review checklist records human review of tolerances + correction copy.
7. Attempt assess HTTP persists and returns feedback messages; practice path still uses attempt strokes only.
8. README + axioms + docguard: no calibrated AI confidence; no correctness claims beyond tested criteria; retry = new attempt focusing on listed corrections.
9. Verbose DEBUG logs include criterion breakdown and feedback codes without coordinates.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Weight/tolerance churn breaks Prompt 08 gold | Re-run eval every change; commit fixture updates with weight changes |
| Order assignment unstable for messy handwriting | Prefer simple greedy start-point matching; fixture the swap case; don’t overfit |
| User-facing copy sounds harsh or childish | Expert review checklist; age-neutral wording |
| Diagnostics leak into learner UI | Separate fields; Prompt 13 consumes `feedback` only |
| Over-claiming “correct handwriting” | docguard + README “tested criteria only” |

## Dependencies / Sequencing

- **Requires:** Prompt 08 engine + Prompt 10 pack + Prompt 11 attempt assess path (present on `main`).
- **Unblocks:** Prompt 13 (guided lesson showing ≤2 corrections + overlay), Prompt 15 (history that can display feedback codes/messages).
