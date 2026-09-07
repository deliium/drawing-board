# Implementation Plan: Japanese Training App (No Collaboration)

Branch: feature/japanese-training-no-collaboration
Created: 2026-09-07

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Summary

Reposition the product as a **personal Japanese handwriting training** app and remove collaboration so a user’s strokes are visible only to that user.

**Current state:** REST + SQLite already scope strokes by `user_id`. The privacy leak is `internal/ws` global `broadcast` to every connected client. The frontend appends every incoming WS stroke with no ownership filter.

**Approach:** Keep WebSocket as the authenticated persist + echo channel (avoids a new HTTP create-stroke API), but deliver messages only to connections belonging to the same `user_id`. Rebrand UI/docs away from “collaborative drawing board.” Do **not** build a curriculum/lesson system in this plan—training means personal practice canvas + existing recognition.

## Commit Plan
- **Commit 1** (after tasks 1–3): `feat(ws): deliver strokes only to owning user`
- **Commit 2** (after tasks 4–6): `feat(web): personal training UX and stroke isolation`
- **Commit 3** (after tasks 7–8): `docs: rebrand as Japanese training app`

## Tasks

### Phase 1: Backend stroke privacy

- [x] Task 1: Make WebSocket hub per-user instead of global broadcast
- [x] Task 2: Add cross-user WebSocket isolation tests (depends on 1)
- [x] Task 3: Verify HTTP stroke APIs remain user-scoped and document failure modes (depends on 1)

### Phase 2: Frontend personal training UX

- [x] Task 4: Harden frontend against foreign live strokes (depends on 1)
- [x] Task 5: Rebrand UI copy to Japanese handwriting training (depends on 4)
- [x] Task 6: Frontend tests for private stroke handling (depends on 4)

### Phase 3: Documentation & verification

- [x] Task 7: Rewrite docs for personal Japanese training (no collaboration) (depends on 5)
- [x] Task 8: Run full verification suite (depends on 2, 3, 6, 7)

## Completion Notes (Task 8)

**Automated verification (2026-09-07):**
- `make test` (Go packages via `./test.sh all`): passed
- `cd web && npm test`: 9 files / 12 tests passed (includes `stroke-isolation.spec.ts`)

**Manual smoke checklist:**
1. Two browsers/users logged in simultaneously: user A draws; user B must **not** see A’s live strokes. — Covered by unit/isolation tests (`TestHub_SendToUser_Isolation`, frontend ignore foreign stroke); interactive browser smoke recommended before merge.
2. Same user two tabs: echo/id assignment still works; reload restores only own strokes via `GET /api/strokes`. — Multi-tab recipient path covered by hub tests; REST scoped by `ListStrokesByUser` privacy test.
3. Recognize still returns candidates for own canvas. — No recognition path change; still loads caller strokes only.
4. Clear/undo/eraser only affect current user’s data. — Covered by `TestClearAndDelete_OnlyTouchCallerData`.

**Logging expectation:** `[ws.sendToUser] DEBUG userID=… recipients=N` stays within owning user’s connection count.
## Out of Scope
- Lesson plans, SRS decks, graded quizzes, or character curricula
- Removing authentication / going fully anonymous
- Replacing WebSocket with HTTP-only stroke create (unless Task 1 proves infeasible)
- Changing recognition model accuracy or ONNX packaging
- Docker redesign beyond copy/docs if compose already works

## Risks & Notes
- Global broadcast is the only material privacy bug; DB/REST are already correct.
- Multi-tab same user must keep receiving echoes or optimistic `id` merge breaks.
- Frontend defense-in-depth still matters if an older server is deployed against a new client (or vice versa).
