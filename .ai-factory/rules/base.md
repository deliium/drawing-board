# Base Conventions

Project-specific conventions detected from the codebase. Loaded after RULES.md axioms.

## Naming

- Go packages under `internal/<name>` use short package names (`auth`, `db`, `ws`, `httpapi`, `limits`).
- Log lines use bracket prefixes: `[ws.Handle]`, `[httpapi.Recognize]`, `[auth.Login]`, `[FIX]` for hotfixes.
- JSON/API fields use camelCase (`opId`, `strokeId`, `topN`); Go structs use matching `json` tags.

## Backend

- Wire dependencies in `cmd/server`; keep business logic in `internal/*`.
- Shared numeric/string bounds live in `internal/limits` — handlers map `limits.ErrorCode` to API/WS codes.
- SQLite migrations are additive in `db.Open` / migrate helpers (e.g. nullable `op_id` + partial unique index).
- Prefer structured INFO/WARN/ERROR with `userID=` and codes; never dump stroke coordinates outside gated `RECOGNIZE_DEBUG` (non-production).

## Frontend

- Network helpers in `web/src/services/`; pages orchestrate UI.
- `apiFetch` always sends credentials and CSRF header on mutating REST calls.
- `wsClient` owns reconnect, queue (`MAX_PENDING_OPS=32`), and status callbacks; BoardPage shows Connecting / Saving / Saved / Offline / Sync error.
- Vitest covers unit, contract, and integration under `web/tests/`.

## Docs

- README is the product contract for API/WS/auth/security. When changing protocol or status UX, update README in the same change.
- When a plan forbids marketing language, grep the **entire** README before marking docs tasks done.
