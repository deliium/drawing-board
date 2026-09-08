# Implementation Plan: Honest Recognition Strategy (Five-Character Hiragana MVP)

Branch: main
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Honest recognition strategy for five-character hiragana MVP (Prompt 08)"
Rationale: Replaces the misleading “Honest ONNX recognition” framing with a measured strategy decision—deterministic target comparison for the five-character MVP—plus removal of unsupported model claims.

> Note for `$aif-roadmap` / implement: update `.ai-factory/ROADMAP.md` so the unchecked Prompt 08 item uses this milestone title (retire “Honest ONNX recognition” wording).

## Summary

`internal/recognize/onnx.go` does **not** load or run an ONNX model. `NewONNXRecognizer` ignores the file contents, keeps `session == nil`, and runs the same style of heuristic feature matching as the simple path. `make onnx-model` downloads **MNIST digit** `mnist-12.onnx` into `models/handwriting.onnx`. Heuristic scores in `[0,1]` are presented like confidence. README still documents an optional “ONNX recognizer” upgrade path.

This plan **recommends one MVP strategy**, removes or relabels unsupported ONNX claims, defines evaluation/fallback/docs/logging contracts, and scopes work to a **testable five-character hiragana** path. **Broad unrestricted kanji recognition is out of scope** until measured evidence exists.

## Strategy Evaluation (required decision)

Evaluate three options for the five-character hiragana MVP:

| Criterion | A. Deterministic target comparison | B. Real Japanese handwriting ONNX model | C. Staged combination (heuristic now → model later as default) |
|-----------|------------------------------------|------------------------------------------|----------------------------------------------------------------|
| **Accuracy (5 known chars)** | High when learner practices a known target against templates; fails closed off-target | Potentially high *if* a licensed JP handwriting model is obtained and validated; current MNIST artifact is **wrong domain** | Mid accuracy until stage 2; stage 1 still misleads if sold as ML |
| **Explainability** | Excellent — stroke count/order/geometry vs template; scores map to named criteria | Poor — softmax-like outputs are not calibrated confidence and are hard to explain to learners | Worse UX: two scoring languages |
| **Model licensing** | N/A (project-owned templates / stroke fixtures) | Must clear model + redistributable runtime licenses; MNIST is irrelevant | Defers licensing pain but keeps dead packaging |
| **Packaging** | Pure Go; drop `onnxruntime_go` and model curl | Native ORT shared libs + model blob in image/CI | Pays packaging cost twice |
| **Latency** | Sub-millisecond on typical stroke sets | Model load + inference (ms–tens of ms); cold start risk | Stage 2 adds latency |
| **Offline** | Always | Yes only if model+runtime bundled and validated | Same as B eventually |
| **Test data** | Small deterministic fixtures per character (correct / near-miss / wrong) | Needs labeled JP handwriting corpora; expensive to build honestly | Fixtures + later corpora |
| **Maintenance** | Low for fixed 5-char set; grows linearly with curriculum | High (runtime upgrades, model rev, GPU/CPU variants) | Highest coordination cost |

### Decision: **A — Deterministic target comparison** for the five-character hiragana MVP

**Why not B now:** There is no loaded Japanese model today. The configured download is MNIST. Shipping “real ONNX” without a vetted JP handwriting model, license audit, startup integrity checks, and accuracy benchmarks would **repeat** the honesty failure. Unrestricted open-set recognition (esp. kanji) is explicitly deferred until measured evidence supports it.

**Why not C as the MVP default:** A staged story that keeps `ONNX_MODEL` / `make onnx-model` as a first-class path continues to imply an upgrade that does not exist. Stage ML **only after** (1) target-comparison MVP is honest and benchmarked, and (2) a later milestone proves a specific model beats fixtures on the same five characters with documented license/packaging.

**What “deterministic target comparison” means in this plan (Prompt 08 scope):**
- Assessment assumes a **known target character** from a fixed MVP set of **exactly five hiragana** (concrete glyphs chosen in starter-content work; Prompt 08 may use placeholders `あ/い/う/え/お` or whatever five the linked curriculum plan freezes—do not invent a large charset).
- Compare learner strokes to **canonical templates** (stroke count, order, normalized geometry) and return **explainable match scores** (ranking / criterion scores), never “confidence”.
- Free-board `/api/recognize` without a target either: (preferred) returns honest heuristic ranking with explicit `scoreKind`, or (acceptable) is narrowed/disabled in docs/UI until attempt-based APIs land—**do not** claim open-set ML.
- Deeper pedagogical feedback (“at most two corrections”) belongs to later target-assessment work; this plan ships the **honest engine contract + fixtures + ONNX removal**, not the full lesson UX.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| `NewONNXRecognizer` | Logs “model load not implemented; heuristic fallback path”; never calls ORT; `session` stays nil |
| `make onnx-model` | Downloads MNIST `mnist-12.onnx` → `models/handwriting.onnx` |
| `go.mod` | Requires `github.com/yalue/onnxruntime_go` solely for unused typed fields in `onnx.go` |
| Default wiring | `ONNX_MODEL` defaults to `./models/handwriting.onnx`; non-empty path always constructs `ONNXRecognizer` (heuristic), not Simple |
| Scores | Hard-coded floats like `0.95` / `0.8` presented as `Candidate.Score` |
| UI | `BoardPage.vue` shows glyph chips; raw `score` only in `title` tooltip |
| Docs | README still has optional ONNX setup; notes Prompt 08 not landed |
| docguard | Bans a few marketing phrases; does **not** yet forbid “ONNX model for advanced recognition” as an active capability claim |
| Tests | `onnx_test.go` asserts heuristic behavior through `ONNXRecognizer`; no model-load or integrity tests |
| Kanji / large set | Simple/ONNX heuristics emit 国/学/生/書/字 etc. without evidence — **must not expand** |

## Approach

1. **Lock the strategy** in code comments, README, ARCHITECTURE, DESCRIPTION, RULES: MVP recognition = deterministic target comparison for five hiragana; no unrestricted kanji; no fake ONNX upgrade.
2. **Remove the unsupported ONNX path** (preferred over “relabel and keep”): delete or gut `onnx.go` / `onnx_test.go`, remove `ONNX_MODEL` flag/env, `make onnx-model` / `mock-onnx`, drop `onnxruntime_go`, stop shipping/referencing MNIST-as-handwriting. If a stub file remains for historical reasons, it must not be loadable as a recognizer.
3. **Introduce a target-aware recognizer API** (additive): e.g. `Assess(target string, strokes []Stroke, width, height int) (Assessment, error)` alongside or replacing open-set `Recognize` for the MVP path. Keep `Candidate.Score` but add explicit `scoreKind: "match"` (JSON) or document that `score` is a **match score** only; never label confidence.
4. **Ship five-character template fixtures + evaluation harness** with pass/fail criteria (top-1 match on gold samples; reject wrong-character samples below threshold).
5. **Fallback behavior** documented and tested (empty strokes, unknown target, oversize inputs via existing `limits`).
6. **Verbose structured logging** without handwriting coordinates/images (reuse `RECOGNIZE_DEBUG` gate).
7. **Docs + docguard** updated so unsupported ONNX claims cannot regress.
8. **Roadmap** wording updated to the new milestone title.

### Startup model validation

**Decision under option A:** **No model remains** → no ONNX startup validation. Startup logs a single honest line, e.g. `INFO [main] recognizer=target_compare set=hiragana5` (exact set id TBD).

If a future milestone reintroduces a real model, require fail-closed validation before listen: file exists, size bounds, checksum/manifest match, ORT session create succeeds, smoke inference on a fixture tensor, else `FATAL` (or explicit `RECOGNIZER=heuristic` opt-out). **Do not implement that path in Prompt 08.**

## Evaluation / Benchmark Criteria (MVP)

Use fixture packs under e.g. `internal/recognize/testdata/hiragana5/`:

| Suite | Expectation |
|-------|-------------|
| Gold correct (per target) | Top match = target; match score ≥ documented threshold `T_pass` |
| Near-miss (wrong stroke order / count) | Either non-pass overall or target rank drops; must not claim “perfect” |
| Wrong character | Must **not** report pass for the requested target |
| Empty strokes | Empty candidates / non-pass; no panic |
| Latency (optional local) | p95 assess < 50ms on CI-class CPU for ≤64 strokes / existing point caps |

Publish thresholds in README as **engineering criteria**, not learner-facing “accuracy %” marketing. No claim of broad handwriting OCR.

## Fallback Behavior

| Condition | Behavior |
|-----------|----------|
| Empty stroke set | `[]` candidates or assessment `pass=false`; HTTP 200 with empty/honest payload (keep existing empty handling) |
| Unknown / unsupported target | `400` with stable code e.g. `unsupported_target` (if target enters API this plan); else ignore until attempt API |
| Open-set board Recognize (if retained) | Heuristic ranking only; UI/docs say “match score (heuristic)”; never imply model |
| Recognizer nil | Existing `503 recognizer_unavailable` |
| Limits violations | Existing `limits` codes unchanged |
| Template missing for a claimed MVP char | Fail startup or fail closed in tests — do not silently invent scores |

## Logging Contract (verbose)

| Level | Events |
|-------|--------|
| DEBUG | `[recognize.Assess]` / `[recognize.Recognize]` entry: strokeCount, pointCount, target (if any), width/height, topN; per-criterion score breakdown **without coordinates** |
| INFO | `[main] recognizer=… set=…`; `[httpapi.Recognize] result=ok|reject` with counts (existing style) |
| WARN | Unsupported target; fallback to open-set heuristic if that path remains |
| ERROR | Template load failure; unexpected assess errors (no raw stroke dumps) |

`RECOGNIZE_DEBUG=1` (non-production): optional ASCII/feature dumps remain gated as today. Never log full point arrays in default paths.

## Out of Scope

- Full guided lesson UX / attempt lifecycle APIs (later prompts)
- Rich “two corrections” pedagogy copy (later target-assessment plan)
- Training or bundling a real Japanese ONNX model
- Unrestricted kanji / large open-set OCR
- Changing WS/`boardRev` contracts except as needed for recognize response fields
- SRS, mastery, audio content

## Tasks

### Phase 0: Strategy lock & inventory
- [x] Task 1: Record strategy decision in project axioms and roadmap wording
  - Update `.ai-factory/ROADMAP.md`: replace unchecked **Honest ONNX recognition (Prompt 08)** with **Honest recognition strategy for five-character hiragana MVP (Prompt 08)** and a one-line description matching this plan’s decision (deterministic target comparison; remove fake ONNX).
  - Update `.ai-factory/RULES.md`, `.ai-factory/ARCHITECTURE.md`, `.ai-factory/DESCRIPTION.md` so recognition axioms say: MVP = deterministic target comparison for five hiragana; heuristic open-set ranking (if any) is not ML; scores are not calibrated confidence; no MNIST/ONNX upgrade claim.
  - LOGGING: none (docs only). Note in plan checklist that implement logs `INFO` when wiring changes land in Task 3.

### Phase 1: Remove unsupported ONNX path
- [x] Task 2: Delete misleading ONNX recognizer and MNIST packaging (depends on 1)
  - Remove or replace `internal/recognize/onnx.go` and `internal/recognize/onnx_test.go` so no type claims ONNX inference without loading a model.
  - Remove `ONNX_MODEL` / `-onnx_model` from `cmd/server/main.go`; always construct the honest recognizer (simple and/or target-compare).
  - Remove `Makefile` targets `onnx-model` / `mock-onnx` and MNIST URL; remove or stop referencing `models/handwriting.onnx` (delete tracked artifact if present; ensure `.gitignore` if local leftovers).
  - Drop `github.com/yalue/onnxruntime_go` from `go.mod` / `go.sum` via `go mod tidy`.
  - Grep repo for `ONNX_MODEL`, `onnx-model`, `handwriting.onnx`, `onnxruntime` and clear stale references in Docker/docs comments.
  - LOGGING: `INFO [main] recognizer=…` at startup; `WARN` only if a deprecated env `ONNX_MODEL` is set (ignore value, log once: unsupported / removed). Prefer ignore+warn over silent accept.

- [x] Task 3: Wire single honest recognizer at process start (depends on 2)
  - `cmd/server/main.go`: construct `SimpleRecognizer` and/or new `TargetCompareRecognizer`; no dual “ONNX vs simple” branch.
  - LOGGING: `INFO [main] recognizer=simple|target_compare set=hiragana5` (or equivalent); include `recognize_debug=` existing line.

### Phase 2: Target-comparison MVP engine
- [x] Task 4: Define assessment types and five-character template contract (depends on 3)
  - Extend `internal/recognize/interface.go` (or adjacent files) with target-aware API and honest score semantics, e.g. `scoreKind` / documented match score; keep JSON backward compatible where possible (`text` + `score` remain; document `score` as match score).
  - Freeze MVP character set of **five** hiragana (coordinate with starter-content plan if present; otherwise temporary placeholders documented as replaceable).
  - Store canonical templates as versioned testdata/JSON under `internal/recognize/testdata/` (stroke polylines in CSS-logical space).
  - LOGGING: `DEBUG [recognize.templates] loaded count=5 version=…` on init; `ERROR` if load fails.

- [x] Task 5: Implement deterministic compare for the five targets (depends on 4)
  - Normalize strokes (translate/scale to unit box; preserve aspect with documented policy).
  - Score stroke count match, order, and coarse shape/distance vs template; combine into overall match score with documented weights.
  - Return ranked candidates **within the five-char set** when assessing with a target, or a single assessment object `{target, pass, score, reasons[]}` if API shape prefers that—pick one contract and stick to it in handler + tests.
  - Do **not** emit kanji / unrestricted guesses from this engine.
  - LOGGING: `DEBUG [recognize.Assess] target=… strokes=… points=… pass=… score=… reasons=…`; never coordinates unless `RECOGNIZE_DEBUG`.

- [x] Task 6: Fallback + open-set honesty for existing `POST /api/recognize` (depends on 5)
  - Decide and implement one:
    - **Preferred:** Keep open-set `Recognize` as clearly labeled heuristic (`SimpleRecognizer` only), UI/docs say match score; target assess reserved for next attempt API, **or**
    - Accept optional `target` in recognize body (small additive field) for MVP demos without full attempt API.
  - Empty strokes → empty list; unsupported target → `400 unsupported_target` if target field exists.
  - LOGGING: `[httpapi.Recognize]` include `mode=heuristic|target` and `target=` when present; retain reject codes without stroke dumps.

### Phase 3: Tests & evaluation harness
- [x] Task 7: Unit/fixture tests for target comparison (depends on 5)
  - Per-character gold fixtures must pass; wrong-character fixtures must not pass; empty input; normalization stability (translated/scaled copies).
  - Table-driven tests; deterministic (no wall clock / RNG).
  - LOGGING: tests may use `t.Logf` for score breakdowns on failure only.

- [x] Task 8: Regression tests for ONNX removal and HTTP honesty (depends on 2, 6)
  - Ensure server boots without `ONNX_MODEL`; setting `ONNX_MODEL` does not select a fake model path (warn+ignore).
  - Handler tests still cover limits + happy path; if `target` added, cover unsupported target.
  - Update/remove tests that constructed `ONNXRecognizer`.
  - Expand `internal/docguard`: forbid README phrases that claim active ONNX/ML handwriting upgrade (e.g. configurable patterns for `make onnx-model` as accuracy path, “ONNX Recognizer (Optional)” as live capability). Allow historical “removed” notes.
  - LOGGING: N/A in tests beyond existing handler log assertions if any.

- [x] Task 9: Benchmark/eval documentation test or small harness (depends on 7)
  - Add `go test` eval that prints summary counts (pass/fail per glyph) under `-v`, or a tiny `internal/recognize/eval` testfile documenting thresholds `T_pass`.
  - Criteria must match Evaluation section above.
  - LOGGING: `t.Logf` summary lines only.

### Phase 4: Frontend & documentation
- [x] Task 10: UI labeling for match scores (depends on 6)
  - `web/src/pages/BoardPage.vue`: tooltip/label “match score” (not confidence); optional short helper text near Recognize results.
  - If target mode exposed, show target + pass/fail honestly.
  - Add/adjust Vitest only if pure helpers are extracted; otherwise keep Vue change minimal.
  - LOGGING: existing DEV `console.debug` for recognize errors; no score spam.

- [x] Task 11: README and operator docs checkpoint (depends on 2, 6, 8, 10)
  - Remove ONNX setup, `ONNX_MODEL` env, `make onnx-model`, dual-recognizer section; describe deterministic five-hiragana target comparison + optional heuristic free-board ranking.
  - Document evaluation criteria, fallbacks, logging prefixes, and explicit non-goals (no unrestricted kanji; no calibrated confidence).
  - Update `.ai-factory/DESCRIPTION.md` / `ARCHITECTURE.md` if Task 1 left TODOs.
  - Ensure `go test ./internal/docguard` passes.
  - LOGGING: document `[recognize.*]` / `[httpapi.Recognize]` / startup recognizer line.

## Commit Plan
- **Commit 1** (after tasks 1–3): "chore: remove fake ONNX path and lock honest recognition strategy"
- **Commit 2** (after tasks 4–6): "feat: add deterministic hiragana target comparison recognizer"
- **Commit 3** (after tasks 7–9): "test: add hiragana5 fixtures and recognition honesty regressions"
- **Commit 4** (after tasks 10–11): "docs: describe match scores and remove ONNX upgrade claims"

## Acceptance Criteria
1. No code path claims to run ONNX inference without a real load+validate implementation; MNIST download target is gone.
2. `onnxruntime_go` is not a module dependency unless a future measured model milestone reintroduces it.
3. MVP assessment for five hiragana is deterministic, fixture-tested, and explainable (match scores / reasons).
4. README, UI, RULES, and docguard agree: not AI confidence; no unsupported ONNX upgrade; no broad kanji claims.
5. Startup logs honest recognizer identity; deprecated `ONNX_MODEL` warn+ignore if set.
6. Existing recognize limits, auth, and `boardRev` gating remain intact.
7. Evaluation thresholds documented and enforced in tests for gold/wrong fixtures.
