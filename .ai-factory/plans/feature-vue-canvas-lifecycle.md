# Implementation Plan: Vue Canvas Lifecycle Modularization

Branch: feature/vue-canvas-lifecycle
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Vue canvas lifecycle modularization"
Rationale: Fixes broken drawing lifecycle (cleanup, resize redraw, taps/dots, DPR) without expanding into recognition honesty or offline vault work.

## Summary

`BoardPage.vue` owns pencil/eraser drawing inline. Pointer listeners are installed from a `watch([canvasRef, user])` whose cleanup ownership is opaque; resize mutates `canvas.width`/`height` (clearing the bitmap) without a guaranteed redraw; pencil commits require `points.length >= 2` so one-point taps/dots are discarded; rendering and hit-testing also skip or miss single-point strokes; backing store ignores `devicePixelRatio`; README documents Escape cancel but the board only handles Ctrl/Cmd+Z.

Deliver a **focused canvas layer**: pure coordinate/render/hit-test helpers plus a `usePracticeCanvas` composable (component only if template ownership clearly improves). Preserve mouse/touch/stylus via Pointer Events, pointer capture, existing WS sync/tools UX, and CSS-logical stored coordinates independent of display scale. No product UI redesign.

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Listener install | `setupDrawing()` in `BoardPage.vue`; `watch([canvasRef, user], …, { immediate: true })` returns cleanup from the callback — ownership is unclear vs `onScopeDispose` / explicit attach; risk of duplicate listeners or leaked capture on remount |
| Resize | `syncCanvasSize()` sets `canvas.width`/`height` from `getBoundingClientRect()`; no redraw call after bitmap wipe |
| Stroke redraw | `watch(strokes, …)` clears and redraws; skips `points.length < 2` |
| Pencil commit | `onUp` only enqueues when `points.length >= 2` — taps discarded |
| Hit-test | Segment loop `i < points.length - 1` never hits single-point strokes; dots cannot be erased even if persisted later |
| Coordinates | `clientX/Y - rect.left/top` (CSS px); no DPR; `canvas.width` == CSS px today |
| Recognize body | Sends `canvas.width` / `canvas.height` (today = CSS size; after DPR must stay **logical CSS** size matching stored stroke coords) |
| Pointers | `pointerdown` / `move` / `up` / `cancel` + `setPointerCapture`; no `pointerType` branching (correct — keep unified Pointer Events) |
| Escape | README claims cancel; only Ctrl/Cmd+Z undo is wired |
| Composables | None under `web/src/composables/`; components are `AppShell`, `MigrationHealthPanel` only |
| Tests | Vitest + jsdom; page mounts via `createApp` (see `auth-page.spec.ts`); no `@vue/test-utils` yet |

## Decision: composable + pure helpers (prefer over new presentational component)

| Option | Pros | Cons |
|--------|------|------|
| **A. `usePracticeCanvas` + pure utils** (chosen) | Clear attach/detach via effect scope; unit-test coords/render/hit without mounting BoardPage; BoardPage keeps header/tools/WS | BoardPage still hosts `<canvas>` |
| B. `<PracticeCanvas>` child component | Encapsulates template + events | Extra props/events for strokes/tool/color/ws; more UI churn than needed |
| C. Fix inline only | Smallest diff | Leaves lifecycle ownership tangled; hard to unit-test |

**Chosen A.** Extract:

1. `web/src/canvas/coords.ts` — CSS ↔ client mapping, DPR-aware backing-store sizing, recognize logical size
2. `web/src/canvas/drawStrokes.ts` — clear + paint strokes (including dots)
3. `web/src/canvas/hitTest.ts` — stroke hit-test including single-point / degenerate segments
4. `web/src/composables/usePracticeCanvas.ts` — listener lifecycle, pointer capture, in-progress stroke, resize observer/listener, Escape cancel, live preview

BoardPage keeps: auth/WS/`strokeSync`, tools UI, undo/clear/recognize gates, status header.

## Coordinate & DPR contract

1. **Stored stroke points** remain in **CSS logical pixels** relative to the canvas element’s layout box (same space as today’s `getBoundingClientRect()` mapping). Never multiply persisted points by DPR.
2. **Backing store:** `canvas.width = round(cssW * dpr)`, `canvas.height = round(cssH * dpr)` with `dpr = window.devicePixelRatio || 1` (clamp css dims so product stays within practice limits; do not exceed server `2048` on the **logical** size sent to recognize).
3. **Drawing:** `ctx.setTransform(dpr, 0, 0, dpr, 0, 0)` (or equivalent) so path code uses CSS coordinates.
4. **Recognize:** POST `width`/`height` = **logical CSS** canvas size (matching stroke space), not backing-store pixels.
5. **After any backing-store resize:** immediately redraw from current `strokes` (and in-progress preview if any).

## Pointer & stroke behavior

| Event | Behavior |
|-------|----------|
| `pointerdown` (pencil) | Capture pointer; start stroke with first point; live preview |
| `pointermove` | Append points while captured/drawing; live stroke |
| `pointerup` | Commit stroke if ≥1 point (dots allowed); release capture |
| `pointercancel` | Abort in-progress stroke without commit (same as Escape) |
| Escape | Cancel in-progress pencil stroke; clear preview; release capture if held |
| Eraser `pointerdown` | Hit-test topmost stroke (including dots); call existing remove callback |
| Sync `error` | Refuse new draw/erase (preserve today’s gate) |

Support mouse, touch, and stylus through Pointer Events only (no parallel mouse/touch listeners).

## Acceptance Criteria

1. Pointer listeners attach once per active canvas session and detach on user logout, canvas unmount, and composable dispose — no duplicate handlers after remount/HMR-style reattach.
2. Resize (window or layout) updates backing store for DPR + CSS size, then redraws all strokes (bitmap never stays blank while `strokes.length > 0`).
3. One-point pencil taps persist as strokes, render as dots, and are erasable via hit-test.
4. Stored coordinates are CSS-logical; changing DPR or zoom does not rewrite historical point values.
5. Recognize continues to send logical width/height consistent with stroke coordinates.
6. Escape and `pointercancel` cancel an in-progress stroke without enqueueing WS create.
7. Pencil/eraser/undo/clear/WS sync behavior otherwise unchanged; no board chrome redesign.
8. Unit/component tests cover lifecycle cleanup, resize redraw, coordinate mapping (incl. DPR), taps/dots, cancellation, pencil commit, eraser hit-test.
9. README Escape claim matches implementation; short note on canvas coordinates / high-DPI if operator-facing.

## Out of Scope

- Full UI / visual redesign of BoardPage chrome
- Pressure/tilt stylus properties, multi-pointer simultaneous draws
- Durable offline vault, cross-tab live create sync
- Honest ONNX recognition (Prompt 08)
- Backend `limits` / WS protocol changes (unless a test fixture needs documenting logical size only)
- Adding `@vue/test-utils` unless mount ergonomics block tests — prefer existing `createApp` + jsdom pattern first

## Logging (verbose)

Use DEV-gated `console.debug` with prefix `[practiceCanvas]` (mirror `[BoardPage.ws]`):

| Level | Events |
|-------|--------|
| DEBUG | attach/detach listeners; resize css/dpr/backing; stroke start/commit/cancel; pointerId; reason codes (`user_escape`, `pointercancel`, `duplicate_attach_prevented`) |
| INFO | (optional) first attach after login — keep quiet in prod builds |
| WARN | missing 2d context; resize skipped; capture failed |
| ERROR | unexpected throw in handlers (caught + rethrown or surfaced once) |

Never log full point arrays in production paths (counts only: `points=`).

## Failure Handling

| Failure | Behavior |
|---------|----------|
| No 2d context | WARN; no listeners; board remains non-drawing |
| Queue full on commit | Keep today’s BoardPage behavior (debug + do not add stroke / or match current early-return) |
| Resize during active stroke | Prefer cancel in-progress stroke then resize+redraw (document in composable); avoid corrupting mid-path |
| DPR / css size → backing over budget | Clamp logical size; DEBUG log clamp |

## Tasks

### Phase 1: Pure canvas utilities

- [x] Task 1: Coordinate + DPR sizing helpers
  - Implement `cssPointFromClient(clientX, clientY, rect)`, `resolveCanvasBackingSize({ cssWidth, cssHeight, dpr })`, `applyCanvasBackingStore(canvas, size)` that sets width/height only when changed and returns whether bitmap was wiped.
  - Export `logicalSizeForRecognize(canvas | size)` → CSS logical dims for recognize body.
  - LOGGING: none in pure helpers (callers log); keep functions side-effect free except apply helper.
  - Files: `web/src/canvas/coords.ts`, `web/tests/unit/canvas-coords.spec.ts`

- [x] Task 2: Stroke painting + hit-test (incl. dots)
  - `drawStrokes(ctx, strokes, opts?)`: clear; paint polylines; for single-point strokes draw a dot (arc or lineCap trick) using stroke width/color.
  - Move/fix `hitTest(point, stroke)` to include point-distance for `points.length === 1` and zero-length segments.
  - LOGGING: none in pure helpers.
  - Files: `web/src/canvas/drawStrokes.ts`, `web/src/canvas/hitTest.ts`, `web/tests/unit/canvas-draw-hit.spec.ts`

### Phase 2: Composable lifecycle

- [x] Task 3: `usePracticeCanvas` attach/detach + pointer capture
  - Inputs: `canvasRef`, `enabled` (user logged in), reactive `strokes` / `tool` / `color` / `width` / `syncStatus`, callbacks `onStrokeComplete(points)`, `onEraseAt(point)`, optional `onCancelInProgress`.
  - Own: drawing flag, in-progress points, live preview redraw, pointer capture, Escape key, window resize (and prefer `ResizeObserver` on canvas parent if cheap).
  - Guarantee single listener set via attach generation token or explicit detach-before-attach; dispose via `onScopeDispose` / `watch` cleanup that **definitely** removes listeners.
  - LOGGING: DEBUG `[practiceCanvas] attach|detach|resize|stroke_start|stroke_commit|stroke_cancel` with `dpr=`, `css=`, `backing=`, `points=`, `reason=`.
  - Files: `web/src/composables/usePracticeCanvas.ts`

- [x] Task 4: Wire BoardPage; preserve tools/WS
  - Replace inline `setupDrawing` / stroke `watch` draw loop / `syncCanvasSize` / keydown (extend with Escape) with composable + helpers.
  - Recognize uses logical size helper (not raw backing `canvas.width` after DPR).
  - Keep eraser → `removeStrokeLocally`, pencil → existing `ws.send` + `strokes` append, undo/clear unchanged.
  - LOGGING: keep `[BoardPage.ws]` for sync; canvas noise stays under `[practiceCanvas]`.
  - Files: `web/src/pages/BoardPage.vue`

### Phase 3: Tests & docs

- [x] Task 5: Lifecycle / resize / cancel / tool behavior tests
  - Unit-test composable with jsdom canvas mock / stubbed `getBoundingClientRect`, fake `devicePixelRatio`, pointer event dispatch:
    - attach → detach removes listeners (second attach does not double-fire)
    - resize wipes bitmap path → `drawStrokes` invoked with existing strokes
    - mapping: client coords → CSS points; backing = css * dpr; recognize logical ≠ backing when dpr=2
    - tap (`down`+`up` same point) commits one-point stroke
    - Escape / `pointercancel` does not call `onStrokeComplete`
    - pencil multi-point commit; eraser calls `onEraseAt` when hit-test would succeed (incl. dot)
  - Prefer testing composable/helpers over full BoardPage mount; add a thin BoardPage smoke only if wiring regresses easily.
  - LOGGING: assert debug spies optional; do not require console in CI.
  - Files: `web/tests/unit/practice-canvas.spec.ts` (and extend Task 1–2 specs as needed)

- [x] Task 6: Docs + agent map
  - README: confirm Escape cancel works; note practice canvas uses CSS-logical stroke coordinates and DPR backing store; troubleshooting blank-canvas-after-resize if relevant.
  - Update `AGENTS.md` key entry for `usePracticeCanvas` / `web/src/canvas/*`; optionally one line in `ARCHITECTURE.md` under `web/` folder structure (`composables/`, `canvas/`).
  - LOGGING: n/a for docs.
  - Files: `README.md`, `AGENTS.md`, `.ai-factory/ARCHITECTURE.md`

## Commit Plan

- **Commit 1** (tasks 1–2): `feat(web): add canvas coord, draw, and hit-test helpers`
- **Commit 2** (tasks 3–4): `feat(web): extract usePracticeCanvas and fix lifecycle/DPR/dots`
- **Commit 3** (tasks 5–6): `test(web): cover practice canvas lifecycle; document Escape and DPR`

## Risks & Notes

- Vue 3.5 `watch` **can** return cleanup; still migrate to composable + `onScopeDispose` so ownership is obvious and HMR/remount cannot double-bind.
- Changing recognize to keep logical size avoids accidental 2× raster params on retina; simple recognizer barely uses width/height but limits still apply.
- Eraser threshold for dots must use stroke width similarly to segments.
- jsdom canvas is incomplete — mock `getContext` / measure patterns carefully; pure functions carry most correctness.
- Do not log stroke coordinates (privacy / noise); counts and sizes only.
