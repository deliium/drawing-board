# Implementation Plan: Responsive, Bilingual, Accessible Learner UI (Prompt 14)

Branch: main
Created: 2026-09-09

## Settings
- Testing: yes
- Logging: verbose
- Docs: yes

## Roadmap Linkage
Milestone: "Responsive bilingual accessible learner UI (Prompt 14)"
Rationale: Prompt 13 shipped the guided journey on desktop-biased fixed canvases with English-only chrome, weak a11y semantics, and a stub accessibility test — this milestone makes the learner-facing SPA usable on phones/tablets, bilingual (EN/JA), and keyboard/SR/axe-verifiable without changing recognition or attempt APIs.

> Note for `$aif-roadmap` / implement: an **unchecked** milestone with this exact title was appended to `.ai-factory/ROADMAP.md` by `$aif-plan`. Do not mark it complete until implement + verify finish. Keep Prompt 08–13 entries unchanged (do not uncheck completed items).

## Summary

The SPA after Prompt 13 is functionally complete for hiragana5 practice but **not product-ready** for mixed-device learners:

- Fixed **300×300 / 280×280** canvases, BoardPage **inline styles**, no design tokens or breakpoints, double chrome (`AppShell` + BoardPage header).
- **English-only** UI, curriculum glosses, and assessment `feedback[].message`; `html lang="en"` is static; Japanese glyphs use spot `lang="ja"` only.
- Auth forms are the strongest a11y surface; practice/board lack visible focus, tool toggle semantics, live status/result announcements, and reliable Japanese fonts (system fallback only).
- `web/tests/integration/accessibility-parity.spec.ts` is a **placeholder** (`expect(true)`).

This plan delivers one coherent **mobile-first practice-notebook** visual system plus **EN/JA preference**, externalized strings, accessibility hardening, and real axe + manual smoke criteria — without gamification, decorative chrome, or scoring changes.

**Out of scope:** Changing assess criteria / `T_pass` / correction **codes** (Prompt 12); audio CDN (Prompt 17); SRS/mastery dashboards (Prompt 15–16); Playwright/Cypress (stay Vitest + jsdom + documented device smoke); collaborative board; dark mode as a product theme; new characters beyond hiragana5; persisting locale on the server (local preference only for MVP).

## Current State (authoritative)

| Area | Finding |
|------|---------|
| Global CSS | `web/src/styles/base.css` — font stack only; no tokens, focus, or breakpoints |
| Board | `BoardPage.vue` — full layout via inline styles; fixed 300×300 canvas; duplicate header inside `AppShell` |
| Practice canvases | `PracticeStageCanvas` / `ComparisonOverlay` 300×300; `StrokeOrderPlayer` 280×280; `usePracticeJourney` hardcodes 300×300 on submit |
| Resize path | `usePracticeCanvas` + `ResizeObserver` already DPR-aware — CSS pin prevents meaningful fluid sizing |
| Touch | Login ≈44–48px; board/practice controls often 4–8px padding; canvas `touch-action: none` OK |
| Viewport | `width=device-width, initial-scale=1.0`; no `dvh` / safe-area |
| i18n | No locale store/catalogs; curriculum `description.en` / `meaningEn` only; corrections English in `catalogMessage` |
| `lang` | `<html lang="en">`; per-glyph `lang="ja"` in intro/hub |
| Live regions | `JourneyStatusBanner` `role="status"`; no assertive result announcement; stage changes not announced |
| Focus | No global `:focus-visible`; auth tabs missing `aria-controls` / arrow keys |
| Reduced motion | `StrokeOrderPlayer` honors `prefers-reduced-motion` (skip to final); untested |
| Fonts | Names Noto Sans/Serif JP — **not loaded**; OS-dependent |
| A11y tests | Stub `accessibility-parity.spec.ts`; no `vitest-axe` / `@axe-core` |
| Result UX | Practice: canvas + short English summary; free-board: scores/chips — no structured SR-oriented result region |

## Design Decisions

### D1 — Practice-notebook visual system (mobile-first, not decorative)

One composition: quiet paper surface, slate ink, restrained indigo accent for primary actions — **not** purple gradients, cream+terracotta, broadsheet rules, glow, or dashboard card grids.

| Token group | Role |
|-------------|------|
| `--ink`, `--ink-muted`, `--paper`, `--paper-raised`, `--rule`, `--accent`, `--danger` | Color |
| `--space-1`…`--space-5`, `--touch-min: 44px`, `--content-max`, `--radius-sm` | Layout |
| `--canvas-size: min(92vw, min(70dvh, 420px))` (square) | Shared practice/board canvas CSS size |
| `--font-ui`, `--font-ja` | UI Latin + Japanese-capable face |

Rules:

- **Brand signal:** product name remains the strongest header text; no competing hero marketing.
- **One job per section** — practice stages keep a single focus (intro / animate / draw / result).
- **Cards only when interactive** (hub character rows, auth form). Decorative bordered panels elsewhere are out.
- Motion: intentional only (stroke-order already exists; optional 1–2 subtle page fades) — all gated by `prefers-reduced-motion`.
- BoardPage: **remove duplicate header** and migrate inline styles → scoped CSS + tokens; one shell chrome.

### D2 — Responsive breakpoints & viewport behavior

| Viewport | Behavior |
|----------|----------|
| Narrow phone (≤400px CSS) | Single column; toolbar wraps; canvas uses `--canvas-size`; nav links may shorten via `t()` keys |
| Tablet | Canvas grows within token max; hub list comfortable row height ≥ `--touch-min` |
| Desktop | Content capped by `--content-max`; canvas does not balloon past token max |

Viewport / keyboard:

- Prefer `min-height: 100dvh` on shell; avoid nested `100vh` traps.
- Safe-area: `padding` on shell header/main using `env(safe-area-inset-*)`.
- Virtual keyboard: keep primary actions (`Submit` / auth submit) in normal document flow (no fixed bottom bar that jumps under VK); when focused inputs, do not lock `body` scroll in a way that clips errors.
- Zoom: layouts must remain usable at **200%** browser zoom (no overlapping controls; no horizontal dead-end for primary tasks).

### D3 — Fluid canvas with one size source of truth

```text
CSS --canvas-size  →  component box  →  usePracticeCanvas ResizeObserver
                                   →  submit width/height from live logicalSize
```

- Add `web/src/canvas/layout.ts` (or export from `coords.ts`) with helpers to read the live CSS-logical size from a canvas element.
- Unify StrokeOrderPlayer with the same token (drop separate 280).
- Guide layer paint must use live size, not hardcoded 300.
- Submit/assess payloads use **actual** logical canvas size at submit time (never a stale constant).
- Free-board canvas follows the same token; WS stroke coords remain CSS-logical (existing invariant).

### D4 — Locale preference & `lang` for mixed content

| Concern | Decision |
|---------|----------|
| Preference | `en` \| `ja`; persist `localStorage` key `locale:v1`; default `en` (or `navigator.language` starts with `ja` → `ja`) |
| Document language | Set `document.documentElement.lang` to UI locale on bootstrap + change |
| Japanese text inside EN UI | Keep `lang="ja"` on glyphs, example words, `jaHint` |
| English text inside JA UI | Wrap EN-only pedagogy leftovers / romanization with `lang="en"` when parent is Japanese |
| Server locale | MVP: **no** `Accept-Language` required for UI chrome; preference is client-owned |
| Assessment messages | Persist API `message` as today (English catalog for storage/honesty); **display** via client map `feedback[].code` + params when available; fall back to API `message` |
| Curriculum glosses | Extend pack/API with Japanese pedagogy fields **or** client catalog keyed by character id for `description` / `meaning` / lesson title — prefer **pack fields** (`description.ja`, `example.meaningJa`, lesson `titleJa`) so content stays reviewed under `content/hiragana5/` |

Locale toggle: quiet control in `AppShell` (not a floating badge on canvas).

### D5 — Externalized strings (lightweight i18n)

Do **not** add `vue-i18n` unless a blocking need appears — prefer a thin helper:

```text
web/src/i18n/
  index.ts          # t(key, params?), setLocale, getLocale
  locales/en.ts     # or .json
  locales/ja.ts
  corrections.ts    # code → (locale, params) → string (mirrors Go catalog)
```

`useLocale()` composable + Pinia optional. Replace inline copy in:

- `AppShell`, `LoginPage`, `BoardPage`, practice pages/components, `usePracticeJourney` banners
- Progress status labels (`unseen` / `seen` / `passed` → localized)
- Auth `ERROR_COPY` map

Logging: `[i18n]` DEV debug on missing keys; never log full translation tables.

### D6 — Accessibility semantics

| Surface | Requirement |
|---------|-------------|
| Focus | Global `:focus-visible` outline in `base.css` (3:1+ against paper); never `outline: none` without replacement |
| Skip link | First focusable in `AppShell` → `#main-content` |
| Main | `<main id="main-content">` landmark |
| Auth | Keep labels; add `aria-controls` + panel ids; arrow-key tablist; visible focus |
| Board tools | Pencil / eraser as `role="radiogroup"` + `role="radio"` `aria-checked` (or segmented toggle with accessible name); color + width labeled |
| Canvas | Persist/improve `aria-label` (localized); guide layer `aria-hidden` |
| Status | `JourneyStatusBanner` + board sync status: `role="status"` `aria-live="polite"` `aria-atomic="true"` |
| Results | Dedicated **non-canvas** textual summary region (pass/fail, match score label, ≤2 corrections) with `aria-live="polite"` (use `assertive` only for hard errors); move focus to summary heading on assess complete when safe |
| Reduced motion | StrokeOrderPlayer: keep skip-to-final; listen for `change` on `matchMedia`; CSS transitions honor `prefers-reduced-motion: reduce` |
| Contrast | Text/UI ≥ 4.5:1; non-text UI components ≥ 3:1; ink on paper verified against tokens |

### D7 — Fonts (Japanese-capable, loadable)

- Self-host or vendor a **subset** of Noto Sans JP (and optional Noto Serif JP for brand wordmark only) under `web/public/fonts/` or via controlled `@font-face` with `font-display: swap`.
- `--font-ui` stack: Latin UI face + JP face; `--font-ja` for glyph-heavy nodes.
- Do not rely solely on OS fonts for hub/intro glyphs.
- Document license/attribution in README if bundling OFL fonts.

### D8 — Testing strategy

**Automated (Vitest + jsdom):**

- Add `vitest-axe` (or `@axe-core/vue` compatible helper) as devDependency.
- Replace `accessibility-parity.spec.ts` with real mounts: Login (both modes), Practice hub, Practice character (intro + **result** fixture), BoardPage (tools + status).
- Assert **zero serious/critical** axe violations on those trees (document known acceptable exceptions if any canvas false-positives).
- Unit: `t()` missing-key behavior; locale persistence; correction display map for each `Code*`; reduced-motion path in `StrokeOrderPlayer`; canvas submit size tracks layout helper.
- Existing journey/auth tests updated for localized default `en` strings (or query by role/label via `t('…')` in tests).

**Manual / smoke criteria (document in README + plan acceptance; run on ≥1 real phone):**

| Check | Pass bar |
|-------|----------|
| Keyboard | Tab through shell → practice → submit path; visible focus always; Escape cancels in-progress stroke |
| Zoom 200% | Auth + practice actions usable; no clipped primary buttons |
| Screen reader | VoiceOver or TalkBack: stage banner and result summary announced; tool radio state spoken |
| Real device | iOS Safari or Android Chrome: draw/trace, virtual keyboard on login does not hide errors permanently; canvas remains square and tappable |
| Locale | Toggle EN↔JA; chrome + corrections + hub status switch; glyphs remain `lang=ja` |

### D9 — Logging (verbose)

| Layer | Prefix | What |
|-------|--------|------|
| Locale | `[i18n]` | INFO locale set/load; WARN missing key; DEBUG key resolve (DEV) |
| Layout | `[canvas.layout]` | DEBUG logical size on resize/submit (no point dumps) |
| A11y helpers | `[a11y.announce]` | DEBUG polite/assertive text announced (DEV) |
| Shell | `[AppShell]` | DEBUG locale toggle |
| Corrections display | `[corrections.i18n]` | DEBUG code→locale map hit/fallback to API message |
| Fonts | `[fonts]` | WARN if critical face fails (if detectable) |

Production: client debug gated on `import.meta.env.DEV`; no PII; no stroke coordinates.

## Architecture touchpoints

```text
web/src/styles/base.css          # tokens, focus, reduced-motion, font-face
web/src/i18n/*                   # NEW — catalogs + t()
web/src/composables/useLocale.ts # NEW
web/src/canvas/layout.ts         # NEW — shared CSS size helpers
web/src/components/AppShell.vue  # skip link, locale toggle, main id, tokens
web/src/pages/*                  # responsive + t()
web/src/components/practice/*    # a11y, canvas size, live regions, result summary
content/hiragana5/v1/*           # optional ja pedagogy fields (+ validate)
internal/curriculum/             # decode ja fields if pack extended
internal/httpapi/curriculum.go   # project ja fields to JSON
internal/recognize/corrections.go# keep EN catalog for API persistence; optional dual later
web/tests/integration/accessibility-parity.spec.ts  # real axe
README.md, ARCHITECTURE.md, AGENTS.md
```

**Dependency rules unchanged:** practice ink stays local (no WS); assess still ≤2 `feedback` by code; honesty/docguard unchanged.

## Acceptance Scenarios

### S1 — Narrow phone practice path

1. Auth on ~360px-wide viewport; form fields ≥44px; labels associated.
2. Open `/practice` → hub rows tappable; open あ → intro readable; animate respects reduced motion if set.
3. Trace/free-write on fluid square canvas; Submit uses live logical size; result shows **textual** summary + overlay.
4. No double headers; primary actions reachable without horizontal scroll.

### S2 — Bilingual preference

1. Default EN; toggle to JA → shell, journey actions, banners, progress labels, correction **display** switch.
2. Glyphs/examples keep `lang="ja"`; `document.documentElement.lang === 'ja'`.
3. Reload preserves preference from `localStorage`.
4. API-stored English `message` still present in network payload; UI does not require re-assess to switch language.

### S3 — Accessibility

1. Keyboard-only: complete login and reach practice result summary; focus visible throughout.
2. Axe on Login / hub / result / board: no serious/critical violations.
3. Screen reader hears status “Checking…” then result pass/fail + corrections (not canvas-only).
4. `prefers-reduced-motion: reduce` → stroke order jumps to final frame; covered by unit test.

### S4 — Tablet / desktop / zoom

1. Tablet: larger canvas within token max; toolbars wrap cleanly.
2. Desktop: content max-width respected; brand name remains header-primary.
3. 200% zoom on auth + result actions: usable.

### S5 — Regression honesty

1. Match scores still labeled as match (not confidence/AI); `go test ./internal/docguard` green.
2. Free-board remains playground; practice remains attempt REST-scoped.

## Tasks

### Phase 1: Design tokens, fonts, shell layout

- [x] Task 1: Establish mobile-first tokens, Japanese-capable `@font-face`, global `:focus-visible`, reduced-motion CSS, and safe-area/`dvh` shell baseline in `web/src/styles/base.css` (+ font files under `web/public/fonts/` or equivalent). Wire `--canvas-size`. Log `[fonts]` WARN only if a load strategy can detect failure; document font license in README notes for Task 8.

  LOGGING: DEV `[fonts]` when faces register; no spam on every paint.

  Files: `web/src/styles/base.css`, `web/index.html` (preload optional), `web/public/fonts/*`

- [x] Task 2: Refactor `AppShell` (skip link, `#main-content`, tokenized chrome, locale toggle slot/control placeholder) and strip BoardPage duplicate header + migrate BoardPage from inline styles to scoped token CSS. Touch targets ≥ `--touch-min` for nav and board tools.

  LOGGING: `[AppShell]` DEBUG layout mode if useful; BoardPage keep existing WS status logs.

  Files: `web/src/components/AppShell.vue`, `web/src/pages/BoardPage.vue`, `web/src/App.vue` if needed

<!-- Commit checkpoint: tasks 1–2 -->

### Phase 2: Fluid canvas pipeline

- [x] Task 3: Shared canvas layout helper; unify practice/board/stroke-order/overlay sizes to `--canvas-size`; fix guide painting; submit width/height from live logical size in `usePracticeJourney`.

  LOGGING: `[canvas.layout]` DEBUG `{cssW,cssH,dpr}` on resync/submit (never points).

  Files: `web/src/canvas/layout.ts` (new), `web/src/canvas/coords.ts` (if extended), `PracticeStageCanvas.vue`, `StrokeOrderPlayer.vue`, `ComparisonOverlay.vue`, `usePracticeJourney.ts`, `BoardPage.vue`, `usePracticeCanvas.ts` as needed

  Depends on: 1

<!-- Commit checkpoint: task 3 -->

### Phase 3: i18n + curriculum JA fields

- [x] Task 4: Implement lightweight i18n (`t`, `useLocale`, EN/JA catalogs) and replace learner-facing chrome strings across shell, auth, board, practice hub/journey. Persist preference; set `document.documentElement.lang`; correct mixed-content `lang` attributes.

  LOGGING: `[i18n]` INFO on setLocale; WARN missing keys; DEV DEBUG resolve.

  Files: `web/src/i18n/*`, `web/src/composables/useLocale.ts`, pages/components listed above, `main.ts` bootstrap

- [x] Task 5: Externalize correction **display** by stable `feedback[].code` (params: glyph, want, got, strokeN) for EN/JA; fall back to API `message`. Extend reviewed pack + curriculum API with Japanese pedagogy fields for descriptions/meanings/lesson title (validate via `contentvalidate`). Keep Go `catalogMessage` English for persisted API messages unless a thin dual-catalog is cheaper — do **not** change codes or selection priority.

  LOGGING: `[corrections.i18n]` DEBUG map hit/fallback; Go keep existing `[recognize.corrections]` logs.

  Files: `web/src/i18n/corrections.ts`, `ComparisonOverlay.vue`, `content/hiragana5/v1/*`, `internal/curriculum/*`, `internal/httpapi/curriculum.go`, `cmd/contentvalidate` if schema changes, seed/tests

  Depends on: 4

<!-- Commit checkpoint: tasks 4–5 -->

### Phase 4: A11y semantics & result summary

- [x] Task 6: Form/tool semantics (auth tablist completeness; board tool radiogroup; labeled inputs); polite live status regions; **non-canvas textual attempt result summary** with focus management on assess complete; strengthen canvas labels; ensure contrast against tokens.

  LOGGING: `[a11y.announce]` DEV DEBUG announcement text; no PII.

  Files: `LoginPage.vue`, `BoardPage.vue`, `JourneyStatusBanner.vue`, `JourneyActions.vue`, `ComparisonOverlay.vue`, `PracticeCharacterPage.vue`, `StrokeOrderPlayer.vue` (media query listener)

  Depends on: 2, 4, 5

<!-- Commit checkpoint: task 6 -->

### Phase 5: Tests + docs

- [x] Task 7: Add `vitest-axe` (or equivalent); replace placeholder `accessibility-parity.spec.ts` with axe suites; unit tests for locale, corrections map, reduced motion, canvas layout submit size; update journey/auth tests for `t()`; document keyboard/zoom/SR/real-device smoke checklist as executable criteria in test file header comments and README.

  LOGGING: tests may assert log spies only where existing patterns do; prefer behavior assertions.

  Files: `web/package.json`, `web/tests/integration/accessibility-parity.spec.ts`, new unit specs under `web/tests/unit/`, update existing practice/auth specs

  Depends on: 3, 6

- [x] Task 8: Docs checkpoint — README (responsive/i18n/a11y operator notes + smoke checklist), `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, honesty pass (`go test ./internal/docguard`). Note Prompt 14 milestone still unchecked until verify.

  LOGGING: n/a beyond docguard.

  Files: `README.md`, `.ai-factory/ARCHITECTURE.md`, `AGENTS.md`, `.ai-factory/DESCRIPTION.md` if feature list needs a bullet

  Depends on: 7

<!-- Commit checkpoint: tasks 7–8 -->

## Commit Plan

- **Commit 1** (after tasks 1–2): `feat(web): practice-notebook tokens, fonts, and shell layout`
- **Commit 2** (after task 3): `feat(web): fluid shared canvas sizing for practice and board`
- **Commit 3** (after tasks 4–5): `feat(web): EN/JA locale preference and externalized learner copy`
- **Commit 4** (after task 6): `feat(web): a11y semantics, live status, and textual attempt summary`
- **Commit 5** (after tasks 7–8): `test+docs: axe a11y coverage and Prompt 14 operator notes`

## Acceptance Criteria

1. Learner UI is mobile-first: usable on narrow phones, tablets, and desktop with shared tokens; BoardPage has no inline-style layout and no duplicate header.
2. Practice/board canvases share one fluid square size; submit dimensions match live logical size; stroke-order size unified.
3. EN/JA preference persists; chrome + progress labels + correction **display** + pedagogy glosses switch; mixed `lang` attributes correct; `html[lang]` tracks preference.
4. Visible `:focus-visible`, skip link, labeled forms, tool toggle semantics, polite status live regions, and a non-canvas textual attempt result summary exist.
5. `prefers-reduced-motion` skips stroke animation to final frame (tested).
6. Contrast meets WCAG AA for text/UI against the token palette; Japanese-capable fonts actually load.
7. `accessibility-parity.spec.ts` runs real axe checks (no placeholder); unit coverage for i18n/corrections/layout/reduced-motion; manual smoke criteria documented and listed in acceptance.
8. Verbose DEV/server logs per D9 without stroke coordinates; docs updated; docguard green; scoring/attempt APIs unchanged in behavior aside from optional JA pedagogy fields.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Fluid canvas breaks stroke/template alignment | Single layout helper; guide + ink + overlay + submit all read live box; regression test size equality |
| JA corrections drift from Go catalog | Shared code constants; table-driven test comparing EN map to Go strings; fallback to API `message` |
| Pack JA fields need human review | Only add reviewed translations in `v1` (or bump pack version per curriculum rules); drafts stay quarantined |
| axe false positives on canvas | Scoped exclusions documented; assert textual summary exists independently |
| Font payload weight | Subset JP glyphs needed for UI chrome + hiragana5; `font-display: swap` |
| Scope creep into vue-i18n / SSR / server locale | Thin client i18n only; no Accept-Language requirement for MVP |
| Visual system becomes decorative | D1 checklist in review; no confetti, mascots, or marketing hero |

## Dependencies / Sequencing

- **Requires:** Prompt 13 journey UI + Prompt 11/12 attempt/assess contracts (on `main`).
- **Unblocks:** Prompt 15–16 mastery surfaces (reuse tokens/i18n/a11y patterns); Prompt 17 audio controls (must meet a11y/live-region patterns established here).
