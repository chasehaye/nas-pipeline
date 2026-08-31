# Testing

How `nas-pipeline` is tested — the strategy, how to run tests locally, and how CI
works.

## TL;DR

```bash
make test          # unit tests for every Go module
```

CI runs the same on every push and pull request
([`.github/workflows/ci.yml`](.github/workflows/ci.yml)).

## Where tests live

Each service is its **own module**, so its tests live **inside** the module they
test — that's how Go (and Maven, and Vite) work, and it means a service ships and
versions with its own tests.

| Service | Test location | Framework |
|---|---|---|
| Go services (`api`, `normalizer`, `filter`, `cache-writer`, `ladd-admin`) | `*_test.go` next to the code | `go test` |
| `bridge` | `src/test/java/…` | JUnit (Maven) |
| `web` | `*.test.tsx` next to components *(planned)* | Vitest |

There is **no root unit-test directory** — the only thing that belongs at the
root is the cross-service **end-to-end** harness (planned; see below).

## How `go test` discovers tests

`go test ./...` scans every package in a module and runs:

- every file ending in **`_test.go`**, and within them
- every function **`func TestXxx(t *testing.T)`** (starts with `Test` + an
  uppercase letter).

So you never list tests anywhere — adding a `*_test.go` file with a `Test*`
function is enough. A test fails (and fails CI) when an assertion fires
(`t.Fatal`/`t.Error`) or the race detector finds a data race.

## Running tests

```bash
make test                       # all Go modules
cd ladd-admin && go test ./...  # one module
go test -race ./...             # with the race detector (what CI uses)
```

`-race` needs a C compiler; the CI Linux runner has it, which is why it's on in
CI but not the Windows default (`make test GOTEST_FLAGS="-race -count=1"` to opt
in locally).

## CI (`.github/workflows/ci.yml`)

A matrix over the Go modules, each running `go test -race -count=1 ./...`
independently (`fail-fast: false`, so one failure doesn't hide the others), on
every push and PR. Public repo → free minutes, and the modules run in parallel.

## The test pyramid

| Level | What it tests | Where | Backing |
|---|---|---|---|
| **Unit** | one function/package in isolation | per service, inline | none |
| **Integration** | one service against real deps | per service, build-tagged | `make up` infra (or testcontainers) |
| **End-to-end** | the whole pipeline composed | root `test/` | `docker compose` (or k3d) |

### Unit — current

- **`ladd-admin`**: `internal/crypto` (encrypt/decrypt round-trip, sign/verify),
  `internal/validate` (fail-closed checks), `cmd/server` (the full handler flow).
- **`normalizer`**: `test/parser` (FIXM parsing).
- **Good next targets** (pure functions, ideal to unit test): `flight.Parse`,
  `fixm.ParseEnvelope`, the `cache-writer` `velocity()` dead-reckoning, the api's
  `flightFromHash`, and the `ladd` blocklist.

### Integration — planned

Per service, against the **real Kafka/Redis** that `make up` provides, gated by a
build tag (`//go:build integration`) so they don't run on every `go test`.
Example: produce to `fixm.normalized`, run `filter`, assert what lands on
`fixm.filtered`.

### End-to-end — planned

A small root `test/` harness that boots the whole stack, feeds a synthetic
`fixm.raw` message, and asserts it flows through to Redis / `/flights`.

## The synthetic-input pattern (key idea)

Only **one** service needs real, sensitive credentials: `bridge` (the SWIM/JMS
connection). Everything downstream only needs Kafka topics with data in them.

So for tests **and** staging, we **fake the input edge**: a small injector
publishes a **canned `fixm.raw`** sample straight to Kafka, and the real
`normalizer → filter → cache-writer → api → web` process it. This means:

- **No real secrets** are needed to test the whole pipeline (SWIM creds and the
  CUI LADD file are replaced with a dummy `swim` secret and a small test LADD
  list).
- **Deterministic, craftable scenarios** the live feed can't give on demand — a
  LADD-blocked flight, a flight going inactive, a positionless flight, an exact
  count.

The only thing this can't verify is the live SWIM connection itself — that's
inherent, and is confirmed in prod (or against a test SWIM subscription).

## Staging (k3d rehearsal) — planned, not currently present

A local **k3d rehearsal** overlay (`deploy/k8s/overlays/dev`) was removed for now
to keep the deploy surface small — we run a single prod cluster. When a deploy
rehearsal is worth it, re-add a `dev` overlay that targets a local k3d cluster
with a **synthetic injector** + **test secrets**, to verify the Kubernetes deploy
(manifests, RBAC, networking, rollouts) without touching real credentials or the
live server.

## Environments at a glance

```
Compose + Makefile  →  server (prod)          [ future: k3d dev rehearsal
  code + unit tests     production               slots in between ]
  (this is where
   `make test` runs)
```
