# bridge

The pipeline's ingestion boundary: the one service that talks to 3rd party software -> [FAA](#acronyms) [SWIM](#acronyms).
It pulls raw [SFDPS](#acronyms) messages off a Solace [JMS](#acronyms) queue and forwards them, unmodified, to the Kafka topic `fixm.raw`.

```
FAA SWIM SFDPS  --JMS/Solace->  bridge  ->  Kafka: fixm.raw  ->  normalizer  ->  ...
```

## Why it exists (and why it's Java)

FAA SWIM requires JMS for SFDPS, and the mature JMS/Solace client is on the
[JVM](#acronyms), so this is the only Java service in an otherwise Go pipeline. Its job is
deliberately tiny: **retrieve and forward**. It does **no parsing or
transformation**: the raw [FIXM](#acronyms) [XML](#acronyms) goes straight to Kafka, and the `normalizer`
downstream does the heavy lifting. Keeping the SWIM-facing boundary this thin
means the protocol-specific complexity lives in one small, replaceable place.

## What it does, step by step

1. A `@JmsListener` consumes messages from the Solace queue (`SOLACE_QUEUE`).
2. It extracts the body (Text or Bytes JMS message) as a UTF-8 string.
3. It publishes that string to `fixm.raw` and **waits** for Kafka to confirm.
4. **Only then** does it acknowledge the JMS message.

## Key design decisions

- **At-least-once, never drop data.** The listener uses `CLIENT_ACKNOWLEDGE`, so
  a message stays on the Solace queue until explicitly acked, and it acks
  **only after** the Kafka write succeeds. If the Kafka send fails, it does *not*
  ack; Solace redelivers. A crash between send and ack means a duplicate is
  reprocessed, never a lost message. (Duplicates are harmless: downstream keys
  by [GUFI](#acronyms) and upserts.)
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

Loaded from a local `.env` (optional) or real env vars. In live mode the SWIM
credentials are **required and sensitive** (see `.env.example`); in dev replay
mode (`ISDEV=true`) none of them are needed.

| Variable | Default | Meaning |
|---|---|---|
| `ISDEV` | `false` | `true` replays a local seed file instead of Solace (see [Dev replay mode](#dev-replay-mode-isdevtrue)) |
| `SOLACE_HOST` | (required in live) | Solace [URI](#acronyms) incl. scheme+port (e.g. `tcps://ems2.swim.faa.gov:55443`) |
| `SOLACE_VPN` | (required in live) | Solace message VPN |
| `SOLACE_USERNAME` / `SOLACE_PASSWORD` | (required in live) | SWIM credentials |
| `SOLACE_QUEUE` | (required in live) | the SFDPS queue to consume |
| `SOLACE_CONNECTION_FACTORY` | (required in live) | [JNDI](#acronyms) connection-factory name |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `KAFKA_TOPIC_RAW` | `fixm.raw` | output topic |
| `LOG_LEVEL` | `INFO` | app log level |

Solace objects (connection factory, destinations) are resolved through **JNDI**,
using `username@vpn` as the security principal.

### Dev replay mode (`ISDEV=true`)

For local development without SWIM access, set `ISDEV=true`. The Solace beans are
not created (no credentials, no outbound connection), and a replay runner instead
streams a committed seed into `fixm.raw`, standing in for the live feed. The seed
is a [LADD](#acronyms)-clean capture of real traffic, so the whole pipeline
downstream runs on realistic data. `ISDEV` is optional and defaults to `false`,
so the real bridge runs unless you opt in.

The seed is split across several files in `seed-data/` (GitHub caps a single file
at 100 MB); the runner reads every file in that directory in sorted name order as
one continuous stream.

| Variable | Default | Meaning |
|---|---|---|
| `REPLAY_DIR` | `seed-data` | directory of separator-delimited seed files, replayed in sorted order |
| `REPLAY_DELAY_MS` | `250` | pause between records; paces the feed (lower is faster) |
| `REPLAY_LOOP` | `true` | restart from the top at the end for a continuous stream |

The downstream `filter` has the same `ISDEV` switch; set it there too, or it will
fail closed on the missing [LADD](#acronyms) list in dev. The seed is already
LADD-clean, so nothing is dropped.

## Running

**Local (from this module's directory, so `.env` resolves):**
```bash
./mvnw spring-boot:run
```

**Container / k8s:** built by a multi-stage Maven to [JRE](#acronyms) image and deployed as the
`bridge` Deployment (no Service; it only connects *out* to SWIM and Kafka). In
k8s the SWIM credentials come from the `swim` Secret; `KAFKA_BROKERS` points at
the in-cluster broker (`kafka:29092`).

## Observability

- Logs a throughput line every 1000 messages: count, MB, msg/sec, failures.
- Spring Actuator exposes `health`, `info`, `metrics`.

---

**In one line:** bridge is the thin, at-least-once JMS→Kafka ingestion edge:
SWIM's protocol complexity isolated in one place, raw FIXM forwarded to
`fixm.raw` for the rest of the pipeline to process.


## Acronyms

- **FAA**: Federal Aviation Administration
- **FIXM**: Flight Information Exchange Model
- **GUFI**: Globally Unique Flight Identifier
- **JMS**: Java Message Service
- **JNDI**: Java Naming and Directory Interface
- **JRE**: Java Runtime Environment
- **JVM**: Java Virtual Machine
- **LADD**: Limiting Aircraft Data Displayed (FAA aircraft-blocking program)
- **SFDPS**: SWIM Flight Data Publication Service
- **SWIM**: System Wide Information Management
- **URI**: Uniform Resource Identifier
- **XML**: Extensible Markup Language
