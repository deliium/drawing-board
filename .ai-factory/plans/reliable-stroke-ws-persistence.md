# Implementation Plan: Reliable Stroke Persistence over WebSocket

Branch: main
Created: 2026-09-08

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Summary

`web/src/services/wsClient.ts` previously skipped sends when the socket was not open, had no reconnect or queue, and the board showed optimistic strokes as if saved. Retries after a lost echo could duplicate rows that reappeared on reload.

Delivered a **contract-first** reliability protocol: stable per-operation `opId`, server `ack` / nack, idempotent create keyed by `(user_id, op_id)`, bounded in-memory client queue (32), reconnect with exponential backoff + jitter, retry until ack/nack/exhaustion, and header status **Connecting… / Saving… / Saved / Offline — retrying… / Sync error**. Per-user `sendToUser` isolation and same-account multi-tab create-ignore behavior are preserved. No durable offline storage in this iteration—reload uses `GET /api/strokes` and clears the session queue.

## Decision: opId + ack + idempotent SQLite + bounded memory queue

Every mutating WS message carries a client-generated `opId` (UUID, ≤36). Server persists creates idempotently and replies with `type:"ack"` to the sending connection, then echoes stroke/delete to the user’s connections. Client never silently drops sends.

## Protocol (authoritative)

```json
{"type":"stroke","opId":"<uuid>","stroke":{...}}
{"type":"delete","opId":"<uuid>","delete":123}
{"type":"ack","opId":"<uuid>","ok":true,"strokeId":456}
{"type":"ack","opId":"<uuid>","ok":false,"error":"rate_limited","message":"..."}
```

## Tasks

### Phase 1: Backend

- [x] Task 1: `MaxOpIDLen` / `ValidateOpID` in `internal/limits`
- [x] Task 2: `strokes.op_id` migration + `SaveStrokeIdempotent` + list `opId`
- [x] Task 3: WS `opId` require, ack-to-sender, echo-to-user, idempotent save/delete

### Phase 2: Frontend

- [x] Task 4: Queued `wsClient` (32), reconnect backoff, ack matching, status callbacks
- [x] Task 5: `strokeSync` ack/opId merge; BoardPage status UI; REST reload clears queue
- [x] Task 6: WS-only delete from undo/eraser (no dual REST delete)

### Phase 3: Docs & verification

- [x] Task 7: README WebSocket contract + status UX
- [x] Task 8: Tests (Go idempotency/ack isolation; Vitest queue/reconnect/duplicate/reload)

## Completion Notes (Task 8)

**Automated verification (2026-09-08):**
- `go test ./internal/limits/ ./internal/db/ ./internal/ws/ ./internal/httpapi/` — passed
- `cd web && npm test` — 13 files / 36 tests passed (includes `ws-client-reliability.spec.ts`)

**Manual smoke checklist:**
1. Draw while backend down → Offline/Connecting; strokes queue; reconnect saves without duplicates.
2. Reload mid-save → only acked strokes restore via REST; pending session ops gone.
3. Two users: still no cross-user live strokes.
4. Same account two tabs: originating tab gets ack; other tab ignores unmatched creates; deletes still apply by id.

## Out of Scope
- IndexedDB / service-worker offline vault
- Cross-tab live create sync
- HTTP stroke create API
- Recognize / auth / CSRF / CORS changes

## Risks & Notes
- Legacy strokes without `op_id` remain valid (partial unique index).
- v1 intentionally loses unacked in-memory ops on full page reload.
