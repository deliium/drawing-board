# Contract: Frontend Parity and Rollout Safety

## Purpose

Define externally observable behavior that MUST remain stable while migrating frontend implementation.

## Contract Scope

1. HTTP request/response behavior used by current frontend workflows.
2. WebSocket message semantics used by personal practice persist/echo.
3. Authentication/session continuity behavior.
4. User-visible error semantics for core workflows.
5. Route accessibility and deep-link behavior.

## Baseline Inventory (Current Scope)

- Core route `/` (authenticated personal handwriting practice workflow).
- Authentication entry flow (login/register + session restore behavior).
- Per-user WebSocket persist/echo events (`stroke`, `delete`) — not a multi-user collaborative board.
- Recognition flow (`/api/recognize`) and candidate rendering behavior.
- Clear, undo, and logout user actions with existing server-side persistence semantics.

## Required Guarantees

### G1: API Compatibility

- Existing request shapes, required fields, and response semantics used by current frontend workflows remain compatible.
- No user-facing workflow requires backend API contract changes to function after migration.

### G2: WebSocket Persist/Echo Compatibility

- Existing message types (`stroke`, `delete`) and expected user-visible outcomes remain consistent for personal practice flows (optimistic local draw, echo assigns server `id`, delete removes known local strokes).
- Messages are scoped to the authenticated user; clients must not treat unmatched inbound strokes as shared-board updates.
- Transport/session errors continue to surface actionable user-facing feedback where applicable.

### G3: Authentication and Authorization Continuity

- Users with valid sessions can continue key workflows after cutover without forced re-authentication, except when current security policy would already require it.
- Route protection and permission behavior remains equivalent to baseline.

### G4: Route and Deep-link Parity

- Every in-scope production route remains reachable.
- Existing deep links open the expected page state or an equivalent user-understandable fallback.

### G5: Error Handling Parity

- Core workflow failures provide understandable, actionable, and non-silent user feedback.
- Critical failure handling does not regress to dead-end states.

### G6: Rollout Reversibility

- Operators can disable migrated experience and restore previous stable behavior without user data restoration operations.

## Acceptance Evidence

- Baseline parity checklist for in-scope routes and workflows.
- Contract test results for critical HTTP and websocket interactions.
- Session continuity verification for authenticated journeys.
- Rollout drill logs including at least one rollback rehearsal.
- Final route retirement checklist showing migrated runtime coverage and legacy fallback deactivation approval.

## Implementation Status Snapshot

- Runtime selection now defaults to migrated frontend unless explicitly pinned to legacy mode.
- Rollback path keeps legacy mode as an immediate fallback.
- Contract smoke tests for HTTP and websocket adapters are in place in `web/tests/contract/`.
- Personal stroke isolation unit tests live in `web/tests/unit/stroke-isolation.spec.ts`.


## Out of Scope

- Introduction of new major user-facing capabilities (curriculum, SRS, quizzes).
- Redesign of backend protocol semantics unrelated to migration parity.
- Collaborative multi-user live canvas semantics (explicitly removed from product).
