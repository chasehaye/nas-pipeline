# platform

The shared Go module every service imports for its operational surface: structured
logging, the `/metrics`, `/healthz`, and `/readyz` endpoints, and the Kafka
dead-letter helper. Written once here instead of copy-pasted into each service.

Module: `github.com/chasehaye/nas-pipeline/platform`.

## Packages

### `observability`

The operational surface for a service.

| Function | Purpose |
|---|---|
| `InitLogging(level)` | install a JSON `slog` handler as the process default (`debug`/`info`/`warn`/`error`; anything else is `info`) |
| `Serve(ctx, addr, ...Check)` | run the ops HTTP server (`/metrics`, `/healthz`, `/readyz`) until `ctx` is cancelled, then shut down gracefully. Run it in a goroutine |
| `Live` | liveness handler: `200 ok` as long as the server can respond |
| `Ready(...Check)` | readiness handler: runs every `Check` and returns `200 ready` only if all pass, else `503` (3s timeout) |
| `MetricsHandler()` | the Prometheus `promhttp` handler (also mounted by `Serve`) |

A `Check` is `func(ctx context.Context) error` (nil means healthy), so each service
composes its own readiness from its real dependencies.

### `kafkax`

Kafka helpers shared by the consumers.

| Function / type | Purpose |
|---|---|
| `Ping(ctx, brokers)` | dial a broker to confirm Kafka is reachable |
| `ReadinessCheck(brokers)` | adapt `Ping` into a `Check` for `observability.Serve` |
| `DLQ` (`NewDLQ`, `Publish`, `Close`) | dead-letter publisher: wraps a poison message with metadata (original topic, partition, offset, stage, error class, cause, payload) as JSON and writes it to a `*.dlq` topic with `acks=all` |

## How a service uses it

Every service's `main.go` wires the same three things:

```go
observability.InitLogging(os.Getenv("LOG_LEVEL"))

// ops endpoint with a Kafka readiness check, in the background
go observability.Serve(ctx, cfg.OpsAddr, kafkax.ReadinessCheck(cfg.Brokers))

// dead-letter for poison messages
dlq := kafkax.NewDLQ(cfg.Brokers, cfg.DLQTopic)
defer dlq.Close()
```

That is the whole contract: logs go to stdout as JSON, Prometheus scrapes
`/metrics`, Kubernetes probes hit `/healthz` and `/readyz`, and unparseable
messages are dead-lettered instead of stalling a partition.

## Endpoints (served by `Serve`)

| Path | Meaning |
|---|---|
| `/metrics` | Prometheus metrics |
| `/healthz` | liveness: `200 ok` |
| `/readyz` | readiness: `200 ready` if all checks pass, else `503` |

In Kubernetes every service exposes these on port `2112`. Metric *definitions* live
in each service (its own counters and histograms); `platform` only exposes the
`/metrics` handler that publishes them.

---

**In one line:** platform is the shared operational plumbing (JSON logging, the
metrics/health/readiness endpoints, and the Kafka dead-letter helper) that every
service imports, so the boilerplate is written once.
