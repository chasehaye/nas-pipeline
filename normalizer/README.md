# normalizer

The pipeline's **transformation stage**. It reads raw FIXM XML off `fixm.raw`,
parses it into typed Go structs, and emits clean per-flight **JSON** to
`fixm.normalized` — one flight per message, keyed by GUFI.

```
bridge ──► Kafka: fixm.raw  ──►  normalizer  ──►  Kafka: fixm.normalized ──► filter ──► …
             (raw FIXM XML)      (XML → JSON,        (one JSON flight
                                  1 envelope→N flights)  per message, keyed by GUFI)
```

> The Go module is internally named `processor` (its import paths and consumer
> group use that name); the service/directory is `normalizer`.

## What it does

Each message on `fixm.raw` is a FIXM **`MessageCollection`** — an envelope
holding many `<message>` elements, each wrapping one `<flight>`. The normalizer:

1. **Parses** the envelope XML into Go structs (`fixm.ParseEnvelope`).
2. **Encodes** each flight as JSON (`fixm.EncodeOne`) — the XML tags map to clean
   JSON fields, dropping XML noise and namespacing.
3. **Publishes** each flight to `fixm.normalized`, **keyed by its GUFI**.

So it's a **1-envelope → N-flights fan-out**: one bulky XML blob in, many small,
uniform JSON records out — the format every downstream service actually works
with (`filter` reads a tiny slice of it; `cache-writer` stores it).

## The model (`internal/fixm/models.go`)

The structs capture essentially every field observed in the live feed —
flight identification, status, GUFI, departure/arrival, en-route position,
operator, flight plan, altitudes, and many rarer elements. That completeness
wasn't guessed; it was built and verified against real data with the **census**
tool below.

## Key design decisions

- **Keyed by GUFI.** Publishing each flight with its GUFI as the Kafka key means
  all updates for one flight land on the same partition, preserving per-flight
  order for everything downstream.
- **Concurrent worker pool (CPU-bound).** XML parsing is the heaviest step in the
  whole pipeline, so processing runs across a pool of workers: a batch of
  envelopes is fanned out, each worker parses + encodes + publishes its flights,
  and the batch's offset is committed only after all of it succeeds. Because the
  bottleneck is CPU, the worker count defaults to **`runtime.NumCPU()`**
  (`NORMALIZER_WORKERS`) — more workers than cores wouldn't help.
- **At-least-once.** The read offset is committed only after a batch is fully
  published; a crash reprocesses the batch (harmless — downstream keys by GUFI
  and upserts). A publish failure leaves the batch uncommitted.
- **Atomic metrics.** Envelope/byte/error counters are `atomic.Int64`, safe for
  the concurrent workers to update.

## The `census` tool (`cmd/census`)

A diagnostic that **reconstructs the shape of the FIXM feed from the data
itself** and prints it as a JSON-like nested tree — nesting, presence,
cardinality, and a real sample value for every leaf. It's how the `models.go`
structs were derived and are re-verified when the feed changes.

- Reads the `fixm.raw` partition **directly** (not via a consumer group), so it
  always starts from the beginning and **never disturbs the normalizer's
  offsets**. Safe to run repeatedly.
- Fully enumerates low-cardinality attributes (status codes, UoMs, flight
  types…); samples high-cardinality ones (callsigns) to avoid a memory blowup.

```bash
go run ./cmd/census        # print the reconstructed FIXM shape
```

## Configuration (environment variables)

| Variable | Default | Meaning |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `KAFKA_TOPIC_RAW` | `fixm.raw` | input topic |
| `KAFKA_TOPIC_NORMALIZED` | `fixm.normalized` | output topic |
| `KAFKA_GROUP` | `processor` | consumer group |
| `NORMALIZER_WORKERS` | `NumCPU` | concurrent workers (CPU-bound; keep near core count) |

## Running

```bash
go run ./cmd/normalizer
```

Container / k8s: distroless Go image, deployed as the `normalizer` Deployment
(no Service — it's a pure consumer/producer). Config comes from the
`normalizer-config` ConfigMap.

---

**In one line:** normalizer turns bulky raw FIXM XML into clean, GUFI-keyed
per-flight JSON, in parallel across CPU cores — the stage that makes the rest of
the pipeline simple.
