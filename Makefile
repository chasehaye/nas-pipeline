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
	@echo -  make up         start infra (Kafka/Redis/Postgres) + create topics
	@echo -  make down       stop infra
	@echo -------------------------------------------------------------------------------------------
	@echo -  make services   run all services together (one terminal)
	@echo ------------------------------------------------------------------------------------------
	@echo -  make bridge     run the bridge service    (Spring Boot; own terminal)
	@echo -  make normalizer  run the normalizer service (own terminal)
	@echo -  make filter     run the filter service    (own terminal)
	@echo -  make cache-writer  run the redis writer  (own terminal)
	@echo ------------------------------------------------------------------------------------------
	@echo -  make test       run normalizer tests (go test ./...)
	@echo ------------------------------------------------------------------------------------------

# ---------- infra (toggle via docker compose) ----------
.PHONY: up down
up:
	docker compose up -d

down:
	docker compose down

# ---------- run all services at once, in this terminal ----------
.PHONY: services
services:
	$(MAKE) -j4 bridge normalizer filter cache-writer

# ---------- services: run each in its own terminal, any order ----------
.PHONY: bridge normalizer filter cache-writer
bridge:
	cd bridge && $(MVNW) spring-boot:run

normalizer:
	cd normalizer && go run ./cmd/normalizer

filter:
	cd filter && go run ./cmd/filter

cache-writer:
	cd cache-writer && go run ./cmd/cache-writer

# ---------- test ----------
.PHONY: test
test:
	cd normalizer && go test $(GOTEST_FLAGS) ./...
