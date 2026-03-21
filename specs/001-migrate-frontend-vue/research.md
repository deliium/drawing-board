# Research: Frontend Framework Migration

## Decision 1: Parity-first migration with route-level acceptance baseline

- **Decision**: Define route-by-route and workflow-by-workflow parity baselines before changing implementation.
- **Rationale**: Baselines reduce interpretation drift and provide objective migration completion criteria.
- **Alternatives considered**:
  - Big-bang rewrite without baseline: rejected due to high regression risk.
  - Visual-only parity checks: rejected because behavior and auth/session semantics are equally critical.

## Decision 2: Preserve external backend contracts unchanged during migration

- **Decision**: Treat HTTP payloads, WebSocket message shapes, auth/session behavior, and user-facing error semantics as compatibility contracts.
- **Rationale**: The migration goal is frontend implementation replacement, not functional protocol redesign.
- **Alternatives considered**:
  - Opportunistic API cleanup during migration: rejected; couples concerns and raises rollout risk.
  - Backend compatibility shim rewrite first: rejected as unnecessary if contracts remain stable.

## Decision 3: Controlled rollout with explicit rollback thresholds

- **Decision**: Release progressively to cohorts and define pre-agreed rollback triggers tied to workflow completion and error indicators.
- **Rationale**: Limits blast radius and supports rapid recovery if production regressions appear.
- **Alternatives considered**:
  - Full rollout immediately after QA: rejected due to insufficient real-world variance coverage.
  - Manual ad hoc rollback judgment only: rejected; too slow/inconsistent for incident response.

## Decision 4: Session continuity and deep-link resilience are non-negotiable gates

- **Decision**: Require validation for active-session continuity and deep-link route behavior before expansion of rollout cohorts.
- **Rationale**: Authentication and entry-point regressions produce high-severity user-facing failures.
- **Alternatives considered**:
  - Session reset at cutover: rejected due to user disruption.
  - Deep-link support deferred to post-migration: rejected because route access parity is a core requirement.

## Decision 5: Accessibility parity validated in core flows

- **Decision**: Include keyboard navigation, focus visibility, and screen-reader semantics in parity checks for critical workflows.
- **Rationale**: Accessibility regressions are high-impact and often under-detected in pure functional testing.
- **Alternatives considered**:
  - Accessibility-only audit at end: rejected due to late discovery risk.
  - Spot checks only on new screens: rejected because migration touches existing primary flows.

## Decision 6: Migration success measured with user-centered outcomes

- **Decision**: Use completion, issue-rate, first-attempt success, and rollback-readiness metrics from spec success criteria.
- **Rationale**: Keeps decision-making aligned with business/user outcomes, not implementation internals.
- **Alternatives considered**:
  - Framework-specific technical metrics as primary gate: rejected; not directly tied to user value.
