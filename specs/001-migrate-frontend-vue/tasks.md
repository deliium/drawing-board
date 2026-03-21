# Tasks: Frontend Framework Migration

**Input**: Design documents from `/specs/001-migrate-frontend-vue/`  
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/, quickstart.md

**Tests**: Test tasks are included because migration risk is high and the plan explicitly requires contract/integration/smoke verification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create migration workspace and validation harness.

- [X] T001 Create Vue app skeleton in `web/src/main.ts`
- [X] T002 [P] Create migration app root component in `web/src/App.vue`
- [X] T003 [P] Create migration router entry in `web/src/router/index.ts`
- [X] T004 [P] Create shared migration styles entry in `web/src/styles/base.css`
- [X] T005 Configure Vite for Vue SPA in `web/vite.config.ts`
- [X] T006 Document migration run scripts and local commands in `web/package.json`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish parity baseline, rollout controls, and compatibility abstractions that block all stories.

- [X] T007 Capture route and workflow parity inventory in `specs/001-migrate-frontend-vue/contracts/frontend-parity-contract.md`
- [X] T008 [P] Create frontend API compatibility adapter in `web/src/services/apiClient.ts`
- [X] T009 [P] Create frontend websocket compatibility adapter in `web/src/services/wsClient.ts`
- [X] T010 [P] Create session context and auth guard utilities in `web/src/services/sessionContext.ts`
- [X] T011 Vue-only app bootstrap in `web/src/main.ts`
- [X] T012 [P] Add migration health metric hooks in `web/src/services/migrationHealth.ts`
- [X] T013 [P] Add contract verification tests for critical HTTP responses in `web/tests/contract/http-parity.spec.ts`
- [X] T014 [P] Add contract verification tests for websocket messages in `web/tests/contract/ws-parity.spec.ts`

**Checkpoint**: Foundational migration platform ready; user stories can proceed.

---

## Phase 3: User Story 1 - Use Core Product Flows After Migration (Priority: P1) 🎯 MVP

**Goal**: Deliver parity for primary user workflows in migrated frontend.

**Independent Test**: Run core user journeys end-to-end on migrated frontend and confirm equivalent outcomes versus baseline.

### Implementation & Tests for User Story 1

- [X] T015 [P] [US1] Implement primary layout shell in `web/src/components/AppShell.vue`
- [X] T016 [P] [US1] Implement drawing board page parity view in `web/src/pages/BoardPage.vue`
- [X] T017 [P] [US1] Implement auth and session-aware route guards in `web/src/router/guards.ts`
- [X] T018 [US1] Wire core routes to migrated pages in `web/src/router/index.ts`
- [X] T019 [US1] Implement core workflow state handling in `web/src/stores/workflowStore.ts`
- [X] T020 [US1] Integrate API and websocket adapters into core workflow screens in `web/src/pages/BoardPage.vue`
- [X] T021 [P] [US1] Add integration tests for core migrated workflows in `web/tests/integration/us1-core-workflows.spec.ts`
- [X] T022 [US1] Add deep-link behavior tests for migrated routes in `web/tests/integration/us1-deeplink.spec.ts`

**Checkpoint**: US1 is independently functional and testable as MVP.

---

## Phase 4: User Story 2 - Release Migration With Minimal Business Risk (Priority: P2)

**Goal**: Enable controlled rollout and rollback of migrated frontend.

**Independent Test**: Enable migration for limited cohort, monitor health indicators, and execute rollback rehearsal successfully.

### Implementation & Tests for User Story 2

- [X] T023 [P] [US2] Implement rollout health signals in `web/src/services/migrationHealth.ts`
- [X] T024 [US2] Vue-only entry (legacy React removed); `web/index.html` → `web/src/main.ts`
- [X] T025 [P] [US2] Implement migration health dashboard panel in `web/src/components/MigrationHealthPanel.vue`
- [X] T026 [US2] Realtime client lifecycle (`close`) in `web/src/services/wsClient.ts`
- [X] T027 [P] [US2] Add rollout cohort integration tests in `web/tests/integration/us2-rollout.spec.ts`
- [X] T028 [US2] Add rollback drill test coverage in `web/tests/integration/us2-rollback.spec.ts`
- [X] T029 [US2] Document rollout and rollback runbook steps in `specs/001-migrate-frontend-vue/quickstart.md`

**Checkpoint**: US2 is independently functional and testable for safe release control.

---

## Phase 5: User Story 3 - Reduce Ongoing Frontend Maintenance Complexity (Priority: P3)

**Goal**: Retire legacy frontend runtime dependency in production scope after parity and rollout gates.

**Independent Test**: Confirm in-scope routes run through migrated frontend and routine UI change can be delivered without legacy runtime components.

### Implementation & Tests for User Story 3

- [X] T030 [P] [US3] Migrate remaining in-scope pages to migrated frontend in `web/src/pages/`
- [X] T031 [US3] Remove legacy React app (`App.tsx` / `main.tsx` deleted)
- [X] T032 [US3] Single bootstrap: `web/src/main.ts` mounts Vue only
- [X] T033 [P] [US3] Remove React dependencies from `web/package.json` and Vite config
- [X] T034 [P] [US3] Add full-route regression test suite for migrated frontend in `web/tests/integration/us3-route-parity.spec.ts`
- [X] T035 [US3] Update parity contract final status for route retirement in `specs/001-migrate-frontend-vue/contracts/frontend-parity-contract.md`

**Checkpoint**: All user stories are independently functional; legacy runtime is retired for migration scope.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final hardening and validation across stories.

- [X] T036 [P] Validate accessibility parity (keyboard/focus/screen-reader) in `web/tests/integration/accessibility-parity.spec.ts`
- [X] T037 [P] Optimize critical rendering path and bundle impact in `web/vite.config.ts`
- [X] T038 Align feature documentation and outcomes in `specs/001-migrate-frontend-vue/spec.md`
- [X] T039 Execute full quickstart validation checklist in `specs/001-migrate-frontend-vue/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 (Setup): starts immediately.
- Phase 2 (Foundational): depends on Phase 1 completion; blocks all user stories.
- Phase 3 (US1): depends on Phase 2 completion.
- Phase 4 (US2): depends on Phase 2 completion; can run in parallel with US1 after shared conflicts are planned.
- Phase 5 (US3): depends on US1 parity completion and US2 rollout controls.
- Phase 6 (Polish): depends on completion of all targeted user stories.

### User Story Dependencies

- **US1 (P1)**: independent after Foundational.
- **US2 (P2)**: independent after Foundational, but references migrated runtime switch from US1 paths.
- **US3 (P3)**: depends on US1 parity confidence and US2 rollback readiness before retiring legacy runtime.

### Parallel Opportunities

- Setup tasks marked `[P]` can run concurrently (T002, T003, T004).
- Foundational adapter and contract-test tasks marked `[P]` can run concurrently (T008, T009, T010, T012, T013, T014).
- Within US1, UI/component and test tasks marked `[P]` can run concurrently (T015, T016, T017, T021).
- Within US2, rollout/health and tests marked `[P]` can run concurrently (T023, T025, T027).
- Within US3, migration sweep and route-parity tests marked `[P]` can run concurrently (T030, T033, T034).

---

## Parallel Example: User Story 1

```bash
Task: "T015 [US1] Implement primary layout shell in web/src/components/AppShell.vue"
Task: "T016 [US1] Implement drawing board page parity view in web/src/pages/BoardPage.vue"
Task: "T017 [US1] Implement auth and session-aware route guards in web/src/router/guards.ts"
Task: "T021 [US1] Add integration tests for core migrated workflows in web/tests/integration/us1-core-workflows.spec.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 + Phase 2.
2. Deliver Phase 3 (US1) and validate independent parity.
3. Demo migrated core workflows before expanding rollout.

### Incremental Delivery

1. Add US2 controlled rollout and rollback.
2. Expand migrated route coverage and complete US3 retirement.
3. Finish polish checks and quickstart validation.

### Suggested MVP Scope

- Phase 1
- Phase 2
- Phase 3 (US1)

---

## Notes

- All tasks follow required checklist format with task ID and concrete path.
- `[P]` marks work that can be done in parallel on separate files.
- User story labels are applied only to story phases, per workflow rules.
