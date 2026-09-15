# Getting started (local dev)

How to run `nas-pipeline` locally, plus the full dev and production port maps.

## Prerequisites

Docker, Go, a JDK, and Node.

## Quick start

```bash
make up          # start infra (Kafka, Redis, Postgres) + create topics
make services    # run bridge, normalizer, filter, cache-writer, api
make web         # run the front-end
```

Local UIs: web `:5173` · API `:8090` · Grafana `:3000` · Kafka UI `:8080` · RedisInsight `:5540`.
Run `make help` for all targets.

## Running without live SWIM (local data)

Only one service needs real secrets: **`bridge`**, which pulls the live feed off
FAA SWIM over JMS/Solace. Everything downstream only needs Kafka topics with data
in them. So to run the pipeline locally you stand in for the two things `bridge`
and the real feed would otherwise provide:

- **A dummy `fixm.raw` input stream.** Without SWIM credentials, no data enters the
  pipeline. A small injector publishes canned or generated `fixm.raw` messages
  straight to Kafka, so `normalizer → filter → cache-writer → database-writer →
  api → web` process real-shaped messages with no live SWIM connection. (This is
  the same synthetic-input idea the tests use; see [testing.md](testing.md).)
- **A fake LADD list.** `filter` is **fail-closed**: it refuses to run without a
  current LADD block list. The real Industry file is CUI and must never be
  committed, so local dev uses a small, non-CUI dummy list purely to let `filter`
  start. See [SECURITY.md](SECURITY.md) for the handling rules.

Neither is a first-class `make` target yet; wiring both into a one-command local
run is tracked in [roadmap.md](roadmap.md) under Developer experience.

## Ports

### Development (local)

Infra runs in Docker Compose (`make up`); the services run on the host
(`make services` / `make web`). Compose ports below are `host → container`.

**Infra: Docker Compose** (browse at `localhost:<host port>`)

| Container | Host | Container | Purpose |
|---|---|---|---|
| kafka | 9092 | 9092 | broker (external listener) |
| kafka | - | 29092 | broker (internal listener; not published) |
| redis | 6379 | 6379 | live cache |
| postgres | 5433 | 5432 | history DB (5433 avoids a native Postgres on 5432) |
| kafka-ui | 8080 | 8080 | Kafka UI |
| redis-insight | 5540 | 5540 | Redis UI |
| pgweb | 8081 | 8081 | Postgres UI |
| prometheus | 9090 | 9090 | metrics store |
| grafana | 3000 | 3000 | dashboards |
| kafka-exporter | 9308 | 9308 | consumer-group lag |

**Services: on the host** (each service binds these on `localhost`)

| Service | Port | Purpose |
|---|---|---|
| bridge | - | none (Solace JMS → Kafka) |
| normalizer | 2112 | ops: `/metrics`, `/healthz`, `/readyz` |
| filter | 2113 | ops |
| cache-writer | 2114 | ops |
| database-writer | 2115 | ops |
| api | 8090 | REST API + ops (`/metrics`, `/healthz`, `/readyz`) |
| ladd-admin | 8092 | upload API + ops (control plane; run on demand) |
| web | 5173 | Vite dev server |

Each host service uses a distinct ops port because they share one host. In
Kubernetes every service is its own pod, so they all use `2112`.

### Production (Kubernetes)

**In-cluster**: reachable only inside the cluster (ClusterIP / container ports):

| Service | Port | Notes |
|---|---|---|
| kafka | 29092 | internal listener |
| redis | 6379 | |
| postgres | 5432 | |
| normalizer / filter / cache-writer / database-writer | 2112 | ops (probes + scraping) |
| api | 8090 | ClusterIP, **not** exposed externally |
| ladd-admin | 8092 | container/Service target port |
| web | 80 | container port |

**External**: exposed on the host/network via `LoadBalancer` (prod overlay):

| Service | External port | → target |
|---|---|---|
| web | 15000 | → 80 |
| ladd-admin | 15002 | → 8092 |

Everything else (Kafka, Redis, Postgres, the writers' ops ports, and the api)
stays cluster-internal. Monitoring (Prometheus/Grafana/kafka-exporter) is
Compose-only today; running it in-cluster is a separate step.
