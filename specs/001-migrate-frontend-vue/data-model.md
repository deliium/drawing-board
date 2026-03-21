# Data Model: Frontend Framework Migration

## 1) UserSessionContext

- **Purpose**: Represents authenticated user state required for protected workflows.
- **Fields**:
  - `userId` (string, required)
  - `sessionState` (enum: active, expired, invalid)
  - `permissionSet` (set of strings, required)
  - `lastValidatedAt` (timestamp, optional)
- **Validation Rules**:
  - Protected routes require `sessionState=active`.
  - Permission checks must remain equivalent to current production behavior.
- **State Transitions**:
  - `active -> expired` on timeout or invalidation.
  - `expired/invalid -> active` after successful re-authentication.

## 2) FrontendRouteDefinition

- **Purpose**: Represents navigable user entry points that must preserve accessibility and behavior.
- **Fields**:
  - `routeId` (string, required, unique)
  - `pathPattern` (string, required)
  - `accessLevel` (enum: public, authenticated, privileged)
  - `supportsDeepLink` (boolean, required)
  - `parityStatus` (enum: pending, in-progress, verified)
- **Validation Rules**:
  - Existing supported routes must be represented.
  - Deep-link routes must resolve without manual navigation preconditions.

## 3) WorkflowTransactionState

- **Purpose**: Represents in-progress user workflows and completion outcomes.
- **Fields**:
  - `workflowId` (string, required)
  - `stepKey` (string, required)
  - `status` (enum: not-started, in-progress, completed, failed)
  - `lastErrorCode` (string, optional)
  - `updatedAt` (timestamp, required)
- **Validation Rules**:
  - Workflow completion criteria must be equivalent to baseline behavior.
  - Failed states must surface actionable user-facing error responses.
- **State Transitions**:
  - `not-started -> in-progress -> completed`
  - `in-progress -> failed -> in-progress` (after user retry where allowed)

## 4) RolloutConfiguration

- **Purpose**: Controls progressive exposure of migrated frontend behavior.
- **Fields**:
  - `cohortId` (string, required)
  - `enabled` (boolean, required)
  - `trafficShare` (percentage 0-100, required)
  - `startAt` (timestamp, optional)
  - `rollbackEnabled` (boolean, required)
- **Validation Rules**:
  - `trafficShare` changes must be auditable.
  - Rollback path must exist whenever migrated experience is enabled.

## 5) MigrationHealthSignal

- **Purpose**: Captures quality indicators used for rollout and rollback decisions.
- **Fields**:
  - `signalName` (string, required)
  - `window` (duration, required)
  - `value` (numeric, required)
  - `threshold` (numeric, required)
  - `status` (enum: healthy, warning, breach)
- **Validation Rules**:
  - Signals must map to success criteria in spec.
  - Breach status must trigger documented operator action (pause or rollback).

## Entity Relationships

- `FrontendRouteDefinition` references required `UserSessionContext` access level.
- `WorkflowTransactionState` executes within one `FrontendRouteDefinition`.
- `RolloutConfiguration` scopes which routes/workflows use migrated behavior.
- `MigrationHealthSignal` evaluates outcomes across routes and workflows under rollout.
