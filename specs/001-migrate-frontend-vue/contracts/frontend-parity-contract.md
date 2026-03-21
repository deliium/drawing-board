# Contract: Frontend Parity and Rollout Safety

## Purpose

Define externally observable behavior that MUST remain stable while migrating frontend implementation.

## Contract Scope

1. HTTP request/response behavior used by current frontend workflows.
2. WebSocket message semantics used by current realtime features.
3. Authentication/session continuity behavior.
4. User-visible error semantics for core workflows.
5. Route accessibility and deep-link behavior.

## Baseline Inventory (Current Scope)

- Core route `/` (authenticated drawing board workflow).
- Authentication entry flow (login/register + session restore behavior).
- Realtime drawing update events (`stroke`, `delete`) over websocket.
- Recognition flow (`/api/recognize`) and candidate rendering behavior.
- Clear, undo, and logout user actions with existing server-side persistence semantics.

## Required Guarantees

### G1: API Compatibility

- Existing request shapes, required fields, and response semantics used by current frontend workflows remain compatible.
- No user-facing workflow requires backend API contract changes to function after migration.

### G2: Realtime Compatibility

- Existing message types and expected user-visible outcomes remain consistent for realtime flows.
- Realtime errors continue to surface actionable user-facing feedback.

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
- Contract test results for critical HTTP and realtime interactions.
- Session continuity verification for authenticated journeys.
- Rollout drill logs including at least one rollback rehearsal.
- Final route retirement checklist showing migrated runtime coverage and legacy fallback deactivation approval.

## Implementation Status Snapshot

- Runtime selection now defaults to migrated frontend unless explicitly pinned to legacy mode.
- Rollback path keeps legacy mode as an immediate fallback.
- Contract smoke tests for HTTP and websocket adapters are in place in `web/tests/contract/`.


## Out of Scope

- Introduction of new major user-facing capabilities.
- Redesign of backend protocol semantics unrelated to migration parity.
