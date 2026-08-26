# nas-pipeline — infra, services, and tests.
#
# Topics are pre-created by `make up` (see the kafka-init service in
# docker-compose.yml), so the services have no startup ordering between them.
# Workflow: `make up` once, then run each service in its own terminal
# (any order; topics already exist). `make down` tears infra back down.

# go test flags. CI overrides to add -race (needs a C compiler, so it is not
# the local default on Windows): make test GOTEST_FLAGS="-race -count=1"
GOTEST_FLAGS ?= -count=1

# The bridge is Spring Boot: the Maven wrapper is a .cmd on Windows and a
# shell script elsewhere. Everything else (Go, Docker) is already portable.
ifeq ($(OS),Windows_NT)
  MVNW := mvnw.cmd
else
  MVNW := ./mvnw
endif

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "nas-pipeline targets:"
	@echo ------------------------------------------------------------------------------------------
	@echo -  make up         start infra (Kafka/Redis/Postgres/Prometheus/Grafana) + create topics
	@echo -  make down       stop infra
	@echo "                  Grafana http://localhost:3000 - Prometheus http://localhost:9090"
	@echo -------------------------------------------------------------------------------------------
	@echo -  make services   run all services (separate windows on Windows)
	@echo ------------------------------------------------------------------------------------------
	@echo -  make bridge     run the bridge service    (Spring Boot; own terminal)
	@echo -  make normalizer  run the normalizer service (own terminal)
	@echo -  make filter     run the filter service    (own terminal)
	@echo -  make cache-writer  run the redis writer  (own terminal)
	@echo -  make database-writer  run the postgres/timescale writer (own terminal)
	@echo -  make api         run the read API (Gin, reads Redis; own terminal)
	@echo -  make web         run the front-end map (Vite dev server, http://localhost:5173)
	@echo ------------------------------------------------------------------------------------------
	@echo -  make test       run unit tests for all Go modules (go test ./...)
	@echo ------------------------------------------------------------------------------------------

# ---------- infra (toggle via docker compose) ----------
.PHONY: up down
up:
	docker compose up -d

down:
	docker compose down

# ---------- run all services at once ----------
# Windows: one PowerShell window per service (separate, readable logs).
# Other:   parallel in this one terminal (interleaved logs, Ctrl-C stops all).
.PHONY: services
ifeq ($(OS),Windows_NT)
services:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev-services.ps1
else
services:
	$(MAKE) -j6 bridge normalizer filter cache-writer database-writer api
endif

# ---------- services: run each in its own terminal, any order ----------
.PHONY: bridge normalizer filter cache-writer database-writer api
bridge:
	cd bridge && $(MVNW) spring-boot:run

normalizer:
	cd normalizer && go run ./cmd/normalizer

filter:
	cd filter && go run ./cmd/filter

cache-writer:
	cd cache-writer && go run ./cmd/cache-writer

database-writer:
	cd database-writer && go run ./cmd/database-writer

api:
	cd api && go run ./cmd/api

# ---------- front-end (Vite dev server on :5173; run `npm install` in web/ once) ----------
.PHONY: web
web:
	cd web && npm run dev

# ---------- test ----------
.PHONY: test
test:
	cd platform && go test $(GOTEST_FLAGS) ./...
	cd ladd-admin && go test $(GOTEST_FLAGS) ./...
	cd normalizer && go test $(GOTEST_FLAGS) ./...
	cd filter && go test $(GOTEST_FLAGS) ./...
	cd cache-writer && go test $(GOTEST_FLAGS) ./...
	cd database-writer && go test $(GOTEST_FLAGS) ./...
	cd api && go test $(GOTEST_FLAGS) ./...
