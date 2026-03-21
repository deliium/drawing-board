# Quickstart: Frontend Framework Migration Validation

## Goal

Validate parity, rollout safety, and rollback readiness for the frontend migration.

## Prerequisites

- Feature branch checked out: `001-migrate-frontend-vue`
- Local backend and frontend environments runnable
- Access to baseline parity checklist for core workflows and routes

## Step 1: Baseline Capture

1. Enumerate in-scope core workflows and routes from `spec.md`.
2. Record current expected outcomes for:
   - successful completion path
   - common validation failures
   - auth/session-dependent paths
   - deep-link entry behavior

## Step 2: Parity Verification (Local)

1. Run migrated frontend implementation against existing backend.
2. Execute P1 workflows end-to-end.
3. Verify:
   - equivalent completion outcomes
   - preserved session behavior
   - equivalent user-visible error handling
   - accessible keyboard/focus behavior in critical flows

## Step 3: Contract Checks

1. Validate HTTP interactions for critical workflow endpoints.
2. Validate realtime message behavior for in-scope realtime features.
3. Confirm no contract-breaking backend changes are required.

## Step 4: Controlled Rollout Rehearsal

1. Enable migrated frontend for a limited cohort.
2. Monitor migration health indicators:
   - workflow completion
   - critical client errors
   - support incident trend
3. Compare results with rollback thresholds.

### Rollout Runbook (Operator Sequence)

1. Enable migration in cohort configuration.
2. Verify session continuity for active authenticated users.
3. Validate deep-link entry to migrated route.
4. Monitor completion/error signals for one full observation window.
5. If thresholds breach, trigger rollback workflow immediately.

## Step 5: Rollback Drill

1. Trigger rollback path using rollout controls.
2. Confirm users can return to stable frontend behavior.
3. Confirm no data restoration or manual user recovery is required.

### Rollback Runbook (Operator Sequence)

1. Disable migrated frontend runtime switch.
2. Reload active sessions into legacy frontend runtime.
3. Re-run core workflow smoke checks.
4. Log rollback cause, timestamp, and recovery confirmation.

## Exit Criteria

- P1 workflows pass parity checks.
- Contract checks pass for critical HTTP and realtime interactions.
- Controlled rollout metrics remain within thresholds.
- Rollback drill succeeds end-to-end.

## Latest Validation Run

- Frontend tests (`npm run test`) passed for contract and migration integration smoke checks.
- Frontend build (`npm run build`) succeeded with migrated runtime enabled by default and legacy fallback preserved.

