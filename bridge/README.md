# bridge

The pipeline's **ingestion boundary** — the one service that talks to FAA SWIM.
It pulls raw SFDPS messages off a Solace **JMS** queue and forwards them,
unmodified, to the Kafka topic `fixm.raw`.

```
FAA SWIM SFDPS ──JMS/Solace──►  bridge  ──►  Kafka: fixm.raw  ──►  normalizer ──► …
```

## Why it exists (and why it's Java)

FAA SWIM **requires JMS** for SFDPS, and the mature JMS/Solace client is on the
JVM — so this is the only Java service in an otherwise Go pipeline. Its job is
deliberately tiny: **retrieve and forward**. It does **no parsing or
transformation** — the raw FIXM XML goes straight to Kafka, and the `normalizer`
downstream does the heavy lifting. Keeping the SWIM-facing boundary this thin
means the protocol-specific complexity lives in one small, replaceable place.

## What it does, step by step

1. A `@JmsListener` consumes messages from the Solace queue (`SOLACE_QUEUE`).
2. It extracts the body (Text or Bytes JMS message) as a UTF-8 string.
3. It publishes that string to `fixm.raw` and **waits** for Kafka to confirm.
4. **Only then** does it acknowledge the JMS message.

## Key design decisions

- **At-least-once, never drop data.** The listener uses `CLIENT_ACKNOWLEDGE`, so
  a message stays on the Solace queue until explicitly acked — and it acks
  **only after** the Kafka write succeeds. If the Kafka send fails, it does *not*
  ack; Solace redelivers. A crash between send and ack means a duplicate is
  reprocessed, never a lost message. (Duplicates are harmless — downstream keys
  by GUFI and upserts.)
- **No partition key.** One SWIM message carries many flights with different
  GUFIs, so there's no single sensible key at this layer. The `normalizer`
  explodes the collection and re-keys per flight.
- **zstd compression.** The raw feed runs ~145 GB/day and XML compresses ~10:1,
  so this is the single highest-leverage producer setting.
- **Idempotent producer + `acks=all` + retries.** The producer dedupes its own
  retries at the broker, so an ambiguous timeout + retry can't double-write.
- **Large `max.request.size` (10 MB).** FIXM collections reach ~370 KB; this
  leaves ample headroom. A short `linger.ms` (20) lets the producer batch, which
  improves the compression ratio on payloads this size.

## Configuration (environment variables)

Loaded from a local `.env` (optional) or real env vars. SWIM credentials are
**required and sensitive** — see `.env.example`.

| Variable | Meaning |
|---|---|
| `SOLACE_HOST` | Solace URI incl. scheme+port (e.g. `tcps://ems2.swim.faa.gov:55443`) |
| `SOLACE_VPN` | Solace message VPN |
| `SOLACE_USERNAME` / `SOLACE_PASSWORD` | SWIM credentials |
| `SOLACE_QUEUE` | the SFDPS queue to consume |
| `SOLACE_CONNECTION_FACTORY` | JNDI connection-factory name |
| `KAFKA_BROKERS` | Kafka bootstrap servers (default `localhost:9092`) |
| `KAFKA_TOPIC_RAW` | output topic (default `fixm.raw`) |
| `LOG_LEVEL` | app log level (default `INFO`) |

Solace objects (connection factory, destinations) are resolved through **JNDI**,
using `username@vpn` as the security principal.

## Running

**Local (from this module's directory, so `.env` resolves):**
```bash
./mvnw spring-boot:run
```

**Container / k8s:** built by a multi-stage Maven → JRE image and deployed as the
`bridge` Deployment (no Service — it only connects *out* to SWIM and Kafka). In
k8s the SWIM credentials come from the `swim` Secret; `KAFKA_BROKERS` points at
the in-cluster broker (`kafka:29092`).

## Observability

- Logs a throughput line every 1000 messages: count, MB, msg/sec, failures.
- Spring Actuator exposes `health`, `info`, `metrics`.

## Scaling note

`concurrency` is `1` (single JMS consumer). Raising it parallelizes consumption
but gives up per-queue ordering — fine for this data, but a deliberate choice to
make when throughput demands it.

---

**In one line:** bridge is the thin, at-least-once JMS→Kafka ingestion edge —
SWIM's protocol complexity isolated in one place, raw FIXM forwarded to
`fixm.raw` for the rest of the pipeline to process.
