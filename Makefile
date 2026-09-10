# bash is on Ubuntu CI and typical local boxes; do not hardcode zsh (missing on GHA).
SHELL := /bin/bash

.PHONY: backend frontend build-web run \
	docker-build docker-build-backend docker-build-frontend \
	docker-run docker-run-dev docker-stop docker-stop-dev docker-clean \
	docker-logs docker-logs-backend docker-logs-frontend \
	docker-shell-backend docker-shell-frontend \
	test test-verbose test-web test-race test-e2e \
	check check-go check-web validate-content sync-audio verify-docs \
	coverage coverage-web security-check \
	gate-r1 gate-r2 gate-r3 gate-r4 gate-release

# --- Dev ---

backend:
	go run ./cmd/server

frontend:
	cd web && npm run dev

build-web:
	cd web && npm ci && npm run build

run:
	ADDR=:8080 STATIC_DIR=web/dist go run ./cmd/server

# Mirror pack audio into Vite/public (canonical files live under content/hiragana5/v1/audio/).
sync-audio:
	mkdir -p web/public/audio/hiragana5
	cp -f content/hiragana5/v1/audio/* web/public/audio/hiragana5/

validate-content: sync-audio
	@echo "[check] start name=validate-content"
	@diff -rq content/hiragana5/v1/audio web/public/audio/hiragana5
	go test ./internal/curriculum -count=1
	go run ./cmd/contentvalidate
	@echo "[check] ok name=validate-content"

# --- Docker ---

docker-build:
	docker compose build

docker-build-backend:
	docker build -f Dockerfile.backend -t drawing-board-backend .

docker-build-frontend:
	docker build -f Dockerfile.frontend -t drawing-board-frontend .

docker-run:
	docker compose up -d

docker-run-dev:
	docker compose -f docker-compose.dev.yml up -d

docker-stop:
	docker compose down

docker-stop-dev:
	docker compose -f docker-compose.dev.yml down

docker-logs:
	docker compose logs -f

docker-logs-backend:
	docker compose logs -f backend

docker-logs-frontend:
	docker compose logs -f frontend

docker-clean:
	docker compose down -v --rmi all
	docker system prune -f

docker-shell-backend:
	docker compose exec backend sh

docker-shell-frontend:
	docker compose exec frontend sh

# --- Quality gates (local mirrors of CI) ---

# Backward-compatible: Go unit/integration only (no race). Prefer `make check` for PR-like.
test:
	@echo "[check] start name=test (go all)"
	./test.sh all
	@echo "[check] ok name=test"

test-verbose:
	./test.sh all -v

# Go fmt + vet + golangci-lint (if installed) + build + unit/integration (no race).
check-go:
	@echo "[check] start name=check-go"
	@dirty=$$(gofmt -l .); if [ -n "$$dirty" ]; then echo "[check] fail name=check-go reason=gofmt"; echo "$$dirty"; exit 1; fi
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "[check] warn name=check-go msg=golangci-lint not installed; skipping (CI installs it)"; \
	fi
	go build ./...
	./test.sh all
	@echo "[check] ok name=check-go"

# Race detector on WS + db board + httpapi hot paths (CGO required for sqlite3).
# Override with TEST_RACE_PKGS='./...' for full sweep.
TEST_RACE_PKGS ?= ./internal/ws ./internal/db ./internal/httpapi
test-race:
	@echo "[check] start name=test-race pkgs=$(TEST_RACE_PKGS)"
	CGO_ENABLED=1 go test -race -count=1 $(TEST_RACE_PKGS)
	@echo "[check] ok name=test-race"

test-web:
	@echo "[check] start name=test-web"
	cd web && npm test
	@echo "[check] ok name=test-web"

# Typecheck + Vitest + production build.
check-web:
	@echo "[check] start name=check-web"
	cd web && npm run typecheck
	cd web && npm test
	cd web && npm run build
	@echo "[check] ok name=check-web"

# Playwright learner journeys (requires browsers installed once via npx playwright install).
test-e2e:
	@echo "[check] start name=test-e2e"
	cd web && npm run test:e2e
	@echo "[check] ok name=test-e2e"

# Go coverage HTML under coverage/ (signal only — not a vanity gate).
coverage:
	@echo "[check] start name=coverage"
	./test.sh report
	@echo "[check] ok name=coverage"

# Vitest coverage under web/coverage/.
coverage-web:
	@echo "[check] start name=coverage-web"
	cd web && npm run test:coverage
	@echo "[check] ok name=coverage-web"

# Light security scans (govulncheck + npm production audit).
security-check:
	@echo "[check] start name=security-check"
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "[check] installing govulncheck…"; \
		go run golang.org/x/vuln/cmd/govulncheck@latest ./...; \
	fi
	cd web && npm audit --omit=dev
	@echo "[check] ok name=security-check"

# PR-like local gate. Set CHECK_E2E=1 to include Playwright.
check: check-go check-web validate-content verify-docs
	@echo "[check] start name=check (PR-like)"
	@if [ "$(CHECK_E2E)" = "1" ]; then $(MAKE) test-e2e; fi
	@echo "[check] ok name=check"

# Assert README-cited make / test.sh commands exist.
verify-docs:
	@echo "[check] start name=verify-docs"
	./scripts/verify-readme-commands.sh
	@echo "[check] ok name=verify-docs"

# Release cutover gates (R1–R4) — wrap existing targets; see README “Release train”.
gate-r1:
	./scripts/gate-release.sh r1

gate-r2:
	./scripts/gate-release.sh r2

gate-r3:
	./scripts/gate-release.sh r3

gate-r4:
	./scripts/gate-release.sh r4

# Example: make gate-release RELEASE=r1
RELEASE ?= r1
gate-release:
	./scripts/gate-release.sh $(RELEASE)
