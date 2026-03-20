<!--
Sync Impact Report
- Version change: N/A -> 1.0.0
- Modified principles: template placeholders replaced with project-specific principles
- Added sections: Technical Constraints, Delivery Workflow
- Removed sections: none
- Templates requiring updates:
  - ✅ .specify/templates/plan-template.md (no breaking mismatch)
  - ✅ .specify/templates/spec-template.md (no breaking mismatch)
  - ✅ .specify/templates/tasks-template.md (no breaking mismatch)
  - ✅ .cursor/commands/speckit.*.md (compatible with project workflow)
- Follow-up TODOs: none
-->

# Drawing Board Constitution

## Core Principles

### I. API and Realtime Contract First
Every feature that affects client behavior MUST define or update the external
contract first (HTTP payloads, WebSocket messages, auth/session behavior, and
error responses). Contract changes MUST be reflected in spec artifacts before
implementation.

### II. Data Integrity for User Drawings
All stroke mutations MUST be user-scoped, persisted consistently, and safe for
concurrent use. Features affecting storage MUST preserve backward compatibility
for existing SQLite data or provide an explicit migration path.

### III. Security and Auth by Default
New endpoints, websocket actions, and side effects MUST require explicit auth
decisions. Session/cookie behavior, CORS changes, and password handling MUST be
documented in spec/plan artifacts when touched.

### IV. Testable Changes Only
Implementation tasks MUST include automated verification proportional to risk:
unit tests for pure logic, integration tests for DB/API boundaries, and
realtime-flow verification for websocket behavior when protocol is changed.

### V. Simple, Observable, and Reversible
Prefer the smallest change that solves the requirement, with clear server logs
for critical flows (auth, persistence, recognize, websocket). When behavior
changes are risky, design MUST include rollback/fallback notes.

## Technical Constraints

- Backend runtime: Go 1.22+.
- Frontend runtime: React + TypeScript (Vite).
- Storage: SQLite as default persistent store.
- Transport: HTTP + WebSocket; JSON payloads for external interfaces.
- Avoid introducing heavy infrastructure dependencies unless justified in plan.

## Delivery Workflow

1. Create a feature spec with `/speckit.specify`.
2. Resolve ambiguity via `/speckit.clarify` when needed.
3. Generate implementation artifacts with `/speckit.plan`.
4. Break work into execution items with `/speckit.tasks`.
5. Implement using `/speckit.implement` and verify with tests/lints.
6. Update docs/contracts whenever external behavior changes.

## Governance

This constitution is the normative source for engineering decisions in this
repository. Any plan or task set that conflicts with these principles MUST be
revised before implementation.

Amendment policy:
- MAJOR for incompatible principle/governance changes.
- MINOR for new principles/mandatory sections.
- PATCH for clarifications without semantic changes.

Compliance policy:
- Each feature plan MUST include a constitution check.
- Reviews MUST validate auth boundaries, contract changes, and test coverage
  against this document.

**Version**: 1.0.0 | **Ratified**: 2026-03-20 | **Last Amended**: 2026-03-20
