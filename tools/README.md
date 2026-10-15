# census

A long-running diagnostic that infers the **shape** of the live [SWIM](#acronyms)
feed. It tails the Kafka topic `fixm.raw` as an independent consumer and builds a model of
every path it sees, how often it occurs, its inferred value type(s), a bounded
set of distinct values, and example samples, then serves that model as a small
web UI and as JSON, persisted so it survives restarts.

```
fixm.raw  ->  census  ->  in-memory shape model  ->  web UI (/) + JSON (/shape)
             (tail, own          |
              consumer group)    +-> snapshot (census.json)
```

It is a dev/observability tool, not a pipeline stage: it only **reads**
`fixm.raw`, on its own consumer group, so it never affects [normalizer](../normalizer/README.md)
or the pipeline's offsets.

## Why it exists

[FIXM](#acronyms) is a large schema and the [FAA](#acronyms) feed drifts (new
fields appear, values change). The pipeline's `normalizer` silently ignores
anything it does not map, so schema changes are invisible there. census answers
"what is actually in the feed right now, and in what shape", which is how the
normalizer's keep-set was derived and how drift gets noticed. See
[data-model.md](../docs/data-model.md).

## What it does

For each message it walks the [XML](#acronyms) and, per path (element,
attribute, or text), records:

- occurrence count and first/last-seen,
- inferred value type(s) (string, int, float, timestamp, [UUID](#acronyms), geo,
  ...); more than one type flags a drifting or inconsistent field,
- for low-cardinality categoricals, the full set of distinct values with counts,
- a few example samples for **every** field, so even high-cardinality fields
  (routes, call signs) show what their data looks like.

## Key design decisions

- **Independent consumer.** Its own `census` group, so it observes the same
  stream without touching the pipeline's offsets.
- **Pure, testable model.** The accumulator is Kafka-free (parse a message, fold
  it in), so it is unit-tested directly; the consume loop is separate.
- **Categoricals enumerate, the rest only sample.** Continuous/unique values
  (timestamps, UUIDs, coordinates, floats) are type-counted and sampled but not
  enumerated, and a categorical that exceeds the cap is treated as free-text.
  Keeps memory bounded on a feed with millions of distinct values.
- **Atomic JSON snapshot.** Written via temp file + rename and reloaded on
  start, so a restart continues rather than resets.

## Configuration (environment variables)

| Variable | Default | Meaning |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `KAFKA_TOPIC_RAW` | `fixm.raw` | input topic |
| `KAFKA_GROUP` | `census` | consumer group (independent of the pipeline) |
| `CENSUS_SNAPSHOT` | `census.json` | JSON snapshot path |
| `CENSUS_SNAPSHOT_EVERY` | `30s` | snapshot + offset-commit interval |
| `CENSUS_HTTP_ADDR` | `:8088` | web UI (`/`, `/shape`) |
| `OPS_ADDR` | `:2116` | `/metrics`, `/healthz`, `/readyz` (next in the pipeline's local 2112-2115 sequence) |

## Running

```bash
make census        # or: cd tools && go run ./cmd/census
```

Then open the web UI (http://localhost:8088 locally). Needs Kafka reachable
(`KAFKA_BROKERS`); click a row to see a field's samples and value distribution.
(Local ops is `:2116`, continuing the pipeline's per-service local ports
(2112-2115) so it won't clash when run alongside the pipeline; in k8s every pod's
ops is `:2112`.)

Container / k8s: distroless Go image, deployed as the `census` Deployment with a
Service, exposed in prod as a `LoadBalancer` on `15003` (the 15000+ range is for
external exposure; the pod itself serves on `:8088`/`:2112`). It reads
`kafka:29092`. The snapshot volume is **ephemeral** by default (the shape
rebuilds from the feed on restart); swap it for a PVC if long-term accumulation
across restarts matters.

## Acronyms

- **FAA**: Federal Aviation Administration
- **FIXM**: Flight Information Exchange Model
- **JSON**: JavaScript Object Notation
- **SWIM**: System Wide Information Management
- **UUID**: Universally Unique Identifier
- **XML**: Extensible Markup Language
