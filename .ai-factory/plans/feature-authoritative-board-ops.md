# Implementation Plan: Authoritative Board Operations

Branch: feature/authoritative-board-ops
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Authoritative board operations (revision-consistent create/undo/erase/clear/recognize)"
Rationale: Closes ordering races between WS create/delete and REST clear/recognize so the learner’s canvas, persistence, and recognition share one revision authority.

## Summary

Create, undo, erase, clear, and recognize currently span separate HTTP and WebSocket paths without shared ordering. A pending WS create can commit after REST clear (ghost stroke), eraser cannot remove pending strokes, and recognize always reads the DB snapshot (often stale vs the canvas).

Ship one authoritative board model: monotonic per-user `boardRev`, every mutate carries `opId` + `baseRev`, server applies mutations in SQLite transactions, acks/echoes return the new `boardRev`, and clients ignore stale replies. Prefer the smallest coherent protocol over compatibility wrappers for behavior that has not shipped externally.

## Decision: boardRev + baseRev + WS-first mutates

Extend the existing `opId` + ack + queue design (plan `reliable-stroke-ws-persistence`) with a scalar board revision:

1. Persist `board_rev` per user (dedicated table or column).
2. All board mutations (create, delete, clear) go through one store path that: `BEGIN IMMEDIATE` → compare `baseRev` → apply → `board_rev++` → commit → return new rev.
3. Client queue remains serial (one in-flight); each outbound mutate uses the latest known `boardRev` as `baseRev`.
4. Clear tombstones prior create `opId`s so idempotent retries cannot resurrect strokes after clear.
5. Recognize becomes revision-gated: client sends `boardRev`; server rejects mismatch; client ignores stale HTTP responses.
6. UI: disable/load clear & recognize appropriately; undo/erase work for both pending and acknowledged strokes.

### Protocol (authoritative — replace, do not dual-write)

```json
{"type":"stroke","opId":"<uuid>","baseRev":12,"stroke":{...}}
{"type":"delete","opId":"<uuid>","baseRev":12,"delete":123}
{"type":"delete","opId":"<uuid>","baseRev":12,"deleteOpId":"<create-op-uuid>"}
{"type":"clear","opId":"<uuid>","baseRev":12}

{"type":"ack","opId":"<uuid>","ok":true,"boardRev":13,"strokeId":456}
{"type":"ack","opId":"<uuid>","ok":true,"boardRev":13,"delete":123}
{"type":"ack","opId":"<uuid>","ok":true,"boardRev":13,"clear":true}
{"type":"ack","opId":"<uuid>","ok":false,"error":"stale_board","message":"...","boardRev":14}

{"type":"clear","opId":"<uuid>","boardRev":13}
```

REST (reload + recognize only as read paths; clear may remain as a thin wrapper calling the same store helper for scripts/tests, but the Vue board uses WS clear):

```json
// GET /api/strokes
{"boardRev":12,"strokes":[...]}

// POST /api/recognize
req: {"topN":10,"width":300,"height":300,"boardRev":12}
res: {"boardRev":12,"candidates":[...]}
// 409 {"error":"stale_revision","message":"...","boardRev":14}
```

Nack / HTTP codes to add: `stale_board` (WS), `stale_revision` (recognize), `op_cancelled` (create after clear/tombstone).

### Semantics

| Op | Server | Client |
|----|--------|--------|
| Create | Reject if `baseRev != board_rev` or create `opId` tombstoned; else insert + bump; idempotent active `(user_id, op_id)` still returns same stroke id **without** bumping when already active | Optimistic stroke; on ack set id + `boardRev`; on `stale_board` reconcile via reload or apply echo then retry |
| Delete by id | Idempotent delete op; bump if row deleted or already gone (still bump once per new delete `opId`) | Undo/erase when `id` known |
| Delete by `deleteOpId` | If create not yet persisted: tombstone create opId (so late create cannot insert); if persisted: delete stroke; bump | Erase/undo pending or racing creates |
| Clear | Tx: tombstone outstanding create ops, delete all strokes, bump, record clear `opId` idempotently | Drop pending queue creates, empty canvas, await clear ack; apply clear echo in other tabs |
| Recognize | Load strokes only if request `boardRev ==` store rev; else 409 | Enable only when queue empty + sync saved; ignore response if local `boardRev` moved or response `boardRev` ≠ requested |

**Strict baseRev equality** (not “≤”) keeps the serial client queue honest and forces multi-tab losers to resync rather than silently overwrite.

### Multi-tab

- Echo create/delete/clear to all same-user connections with `boardRev` (existing `sendToUser`).
- Other tabs: apply delete/clear; ignore unmatched creates (unchanged); on `stale_board` or clear echo, set local `boardRev` and empty/reconcile canvas.
- Optional later: live create fan-in remains a separate roadmap item; this plan only ensures clear/delete/rev cannot diverge.

### Error recovery

- WS nack `stale_board`: update local `boardRev` from ack; if canvas uncertain → `GET /api/strokes` and reset queue.
- Nack `op_cancelled`: drop local pending stroke; do not retry that create.
- Clear while offline: queue clear like other mutates; do not leave REST clear as a bypass that skips rev.
- Full reload: `GET /api/strokes` sets strokes + `boardRev`, `clearPending()` (unchanged durability stance).

### UI disabled / loading

- Clear: disable while clear in-flight; show Saving… via existing sync status.
- Recognize: disabled unless `syncStatus === 'saved'`, queue empty, and strokes length > 0; button loading while request outstanding; discard late responses (attempt token or matched `boardRev`).
- Undo/Erase: always allowed on visible strokes; pending → cancel/tombstone path; acknowledged → delete by id.
- Pencil may stay enabled during save (queue); block mutate enqueue when status is `error` after budget exhaustion until reload.

## Commit Plan
- **Commit 1** (after tasks 1–3): "feat: add boardRev store mutations and WS clear protocol"
- **Commit 2** (after tasks 4–6): "feat: revision-aware board client for undo erase clear recognize"
- **Commit 3** (after tasks 7–9): "test: concurrency coverage for board revision races"
- **Commit 4** (after task 10): "docs: document boardRev protocol and roadmap milestone"

## Tasks

### Phase 1: Backend revision core

- [x] Task 1: Board revision schema + transactional mutators in `internal/db`
  - Add `user_board_state(user_id PRIMARY KEY, board_rev INTEGER NOT NULL DEFAULT 0)` (or equivalent) and `stroke_op_tombstones(user_id, op_id, reason, at_rev)` (or unified `board_ops` table) via additive migration in `db.Open`.
  - Implement `GetBoardRev`, `ListStrokesWithRev`, `ApplyStrokeCreate(baseRev, opId, ...)`, `ApplyStrokeDelete(baseRev, deleteOpId, strokeID?, ...)`, `ApplyClear(baseRev, opId)` using `BEGIN IMMEDIATE`, strict `baseRev` check, rev bump, tombstones on clear / delete-by-opId.
  - Preserve create idempotency for **active** strokes; after tombstone/clear, same create `opId` must not recreate (`op_cancelled`).
  - LOGGING: DEBUG enter with `userID=`, `baseRev=`, `opId=`; INFO on apply with `boardRev=` before/after; WARN on stale/cancelled; never log coordinates.
  - Files: `internal/db/db.go`, `internal/db/db_test.go`

- [x] Task 2: WS protocol — `baseRev`, `clear`, delete-by-opId, ack `boardRev`
  - Extend message/ack types; require `baseRev` on stroke/delete/clear; handle `type:clear`; map store stale/cancelled to ack nacks; echo clear to user; include `boardRev` on success acks and echoes.
  - Soft-reject invalid/missing `baseRev` like other validation (nack when `opId` known).
  - LOGGING: `[ws.Handle]` DEBUG inbound type + `opId` + `baseRev`; INFO clear/delete/create with new `boardRev`; WARN reject `stale_board` / `op_cancelled` with codes only.
  - Files: `internal/ws/handler.go`, `internal/ws/handler_test.go`, `internal/limits` if new error codes

- [x] Task 3: REST list/recognize/clear aligned to revision
  - `GET /api/strokes` returns `{ boardRev, strokes }` (breaking shape OK — not shipped externally as a stable contract for wrappers).
  - `POST /api/recognize` requires `boardRev`; 409 `stale_revision` + current `boardRev` on mismatch; success echoes `boardRev`.
  - `POST /api/strokes/clear` calls same `ApplyClear` helper (generate server opId or accept body opId+baseRev); return `{ ok, boardRev }`. Prefer documenting Vue path as WS-primary.
  - Remove or leave unused `POST /api/strokes/delete` as thin wrapper only if tests need it; do not use from UI.
  - LOGGING: `[httpapi.ListStrokes|Recognize|ClearStrokes]` include `boardRev=`; Recognize reject logs `stale_revision`.
  - Files: `internal/httpapi/handlers.go`, related `*_test.go`, `cmd/server/main.go` if wiring needed

<!-- Commit checkpoint: tasks 1-3 -->

### Phase 2: Frontend authoritative client

- [x] Task 4: `wsClient` + types for baseRev / clear / boardRev acks
  - Track `boardRev`; stamp `baseRev` on enqueue; parse ack `boardRev` / clear echo; on `stale_board` expose callback for reload; support `deleteOpId` messages; keep max queue 32 and serial in-flight.
  - LOGGING: DEV `console.debug` `[wsClient]` for baseRev stamp, stale nack, clear ack (no PII).
  - Files: `web/src/services/wsClient.ts`, `web/tests/unit/ws-client-reliability.spec.ts`

- [x] Task 5: `strokeSync` + BoardPage undo/erase/clear/recognize UX
  - Apply clear echo (empty strokes, set rev); ignore stale acks (rev regress / cancelled op); undo/erase pending via `dropOp` + `deleteOpId` path; acknowledged via delete id; `doClear` → WS clear (not REST-only); cancel pending creates on clear; recognize gated + attempt/rev match discard; loading/disabled states.
  - LOGGING: `[BoardPage.ws]` ignore reasons include `stale_rev`, `clear_applied`, `recognize_discarded`.
  - Files: `web/src/services/strokeSync.ts`, `web/src/pages/BoardPage.vue`, unit tests under `web/tests/unit/`

- [x] Task 6: Load path + multi-tab reconciliation
  - Parse `{ boardRev, strokes }` on load; reset queue; on clear echo from other tab empty canvas; on stale force `loadStrokes()`.
  - Files: `BoardPage.vue`, contract tests if API shapes listed in `web/tests/contract/`

<!-- Commit checkpoint: tasks 4-6 -->

### Phase 3: Concurrency tests + docs

- [x] Task 7: Go concurrency / ordering tests
  - Clear then late create with pre-clear opId → no resurrection (tombstone / cancelled).
  - Create with stale `baseRev` → nack, rev unchanged.
  - Delete-by-opId before create lands → create cancelled.
  - Recognize with wrong `boardRev` → 409; matching rev → candidates from that snapshot.
  - Idempotent clear/create opIds.
  - Prefer store-level and httpapi/ws handler tests with shared store (no flaky sleeps; use deterministic sequencing).
  - Files: `internal/db/db_test.go`, `internal/ws/handler_test.go`, `internal/httpapi/recognize_handler_test.go` / clear tests

- [x] Task 8: Frontend concurrency-focused Vitest
  - Pending undo then late ack → auto-delete or ignore without resurrecting UI stroke.
  - Clear drops queue; late stroke ack ignored / does not re-add.
  - Recognize response discarded when `boardRev` moved.
  - Clear echo empties strokes.
  - Files: `web/tests/unit/stroke-isolation.spec.ts` (or new `board-revision.spec.ts`), `ws-client-reliability.spec.ts`

- [x] Task 9: End-to-end / integration verification checklist automation where cheap
  - Extended Go+Vitest suites (board_rev store races, recognize 409, `board-revision.spec.ts`).
  - Manual multi-tab smoke: two tabs same user — clear in A empties B via WS clear echo; Recognize disabled while header shows Saving…; undo pending stroke then late ack must not resurrect.

<!-- Commit checkpoint: tasks 7-9 -->

- [x] Task 10: Docs + roadmap milestone + axioms
  - Update `README.md` Drawing/WS/Recognize sections for `boardRev` / `baseRev` / clear WS message / 409 recognize; status UX for disabled recognize.
  - Add unchecked milestone to `.ai-factory/ROADMAP.md` matching Roadmap Linkage name (if still absent).
  - Update `.ai-factory/RULES.md` / `ARCHITECTURE.md` / `AGENTS.md` only where axioms/boundaries change (rev on mutates; GET strokes envelope; no dual REST clear from UI).
  - Keep recognition honesty (`docguard` green).
  - Files: `README.md`, `.ai-factory/ROADMAP.md`, `.ai-factory/RULES.md`, `.ai-factory/ARCHITECTURE.md`, `AGENTS.md` as needed

<!-- Commit checkpoint: task 10 -->

## Out of Scope
- Durable offline vault / IndexedDB (separate roadmap item)
- Cross-tab live create fan-in as authoritative merge (deletes/clear/rev only)
- ONNX honesty (Prompt 08)
- Compatibility shims that keep raw `[]` stroke list responses or REST-only clear as the UI path

## Risks & Notes
- Breaking `GET /api/strokes` response shape is intentional; update all clients/tests in the same change.
- Strict `baseRev` equality + serial queue is the intended single-tab model; multi-tab writers must resync on stale.
- Tombstones grow with ops — acceptable for personal boards; optional later GC by rev watermark.
- REST clear without hub echo will not update other tabs live — document WS clear as the product path; if REST clear remains, consider optional hub notify from httpapi (only if wiring stays clean; otherwise scripts reload).
