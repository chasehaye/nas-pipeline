# nas-pipeline — infra, services, and tests.

PROCESSOR_DIR := processor

# go test flags. CI overrides to add -race (needs a C compiler, so it is not
# the local default on Windows): make test GOTEST_FLAGS="-race -count=1"
GOTEST_FLAGS ?= -count=1

# "Run everything in its own window" is a Windows convenience delegated to a
# native .cmd script (start is a cmd builtin, so make can't launch it directly).
ifeq ($(OS),Windows_NT)
  RUN_ALL  := cmd /c scripts\run-all.cmd
  STOP_ALL := cmd /c scripts\stop-all.cmd
else
  RUN_ALL  := echo "services-up opens a window per service on Windows only; on this OS run each service in its own terminal."
  STOP_ALL := echo "services-down is Windows-only; on this OS Ctrl-C each service's terminal."
endif

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "nas-pipeline targets:"
	@echo "  make up             start infra, then all services (1s gap each, Windows)"
	@echo "  make down           stop all services, then stop infra"
	@echo "  make services-up    start bridge -> processor -> filter, 1s gap (no infra)"
	@echo "  make services-down  stop those services, 1s gap"
	@echo "  make infra-up       start Kafka/Redis/Postgres   (infra-down to stop)"
	@echo "  make infra-down     stop Kafka/Redis/Postgres"
	@echo "  make test           run all tests (go test ./...)"

# ---------- up / down: infra + services together ----------
.PHONY: up down
up:   infra-up services-up
down: services-down infra-down

# ---------- services: start / stop all three (Windows; one window each) ----------
.PHONY: services-up services-down
services-up:
	$(RUN_ALL)

services-down:
	$(STOP_ALL)

# ---------- infra ----------
.PHONY: infra-up infra-down
infra-up:
	docker compose up -d

infra-down:
	docker compose down

# ---------- test ----------
.PHONY: test
test:
	cd $(PROCESSOR_DIR) && go test $(GOTEST_FLAGS) ./...
