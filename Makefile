SHELL := /usr/bin/zsh

.PHONY: dev backend frontend build-web run zinnia-build zinnia-model docker-build docker-run docker-stop docker-clean test validate-content sync-audio

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
	@diff -rq content/hiragana5/v1/audio web/public/audio/hiragana5
	go test ./internal/curriculum -count=1
	go run ./cmd/contentvalidate

# Docker commands
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

# Test commands
test:
	./test.sh all

test-verbose:
	./test.sh all -v
