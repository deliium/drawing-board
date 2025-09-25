SHELL := /usr/bin/zsh

.PHONY: dev backend frontend build-web run zinnia-build zinnia-model docker-build docker-run docker-stop docker-clean test

backend:
	go run ./cmd/server

frontend:
	cd web && npm run dev

build-web:
	cd web && npm ci && npm run build

run:
	ADDR=:8080 STATIC_DIR=web/dist go run ./cmd/server

# ONNX model setup
ONNX_MODEL_URL := https://github.com/onnx/models/raw/main/vision/classification/mnist/model/mnist-12.onnx

onnx-model:
	mkdir -p models
	curl -L "$(ONNX_MODEL_URL)" -o models/handwriting.onnx

# Mock model for testing (creates empty file)
mock-onnx:
	mkdir -p models
	touch models/handwriting.onnx

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
