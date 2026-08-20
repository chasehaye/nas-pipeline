# filter

The pipeline's **compliance gate**. It reads normalized flights off
`fixm.normalized`, drops any aircraft on the FAA **LADD** block list, and
forwards the rest to `fixm.filtered`. If the block list is missing or stale it
**halts** rather than pass unfiltered data.

```
normalizer ──► Kafka: fixm.normalized ──►  filter  ──►  Kafka: fixm.filtered ──► cache-writer ──► …
                                        (drop LADD-listed
                                         flights, forward rest)
```

## What LADD is (and why fail-closed)

**LADD** (Limiting Aircraft Data Displayed) is an FAA program: aircraft owners
can request that their aircraft not be publicly displayed. The Industry list of
those aircraft is **CUI** and updates weekly. This service is what enforces it —
it must never publish a flight that's on the list.

Because getting this wrong is a compliance failure, the filter is **fail-closed**:
if it has no list, or the list is stale (past `LADD_MAX_AGE`), it **stops
processing entirely** instead of forwarding potentially non-compliant data. No
list, no output — never "pass everything through."

## What it does

For each flight:
1. Parse just the identifying fields (callsign, registration, GUFI).
2. If the callsign **or** registration is on the LADD set → **drop it** (not
   published).
3. Otherwise → **publish** to `fixm.filtered`, keyed by GUFI.

It's a thin, high-throughput screen: a tiny JSON parse, a set lookup, and a
forward.

## How the LADD list gets here (and updates)

The list is mounted from the **`ladd` Kubernetes Secret** at `/data/ladd/active`
(`LADD_DIR`). The filter loads the **newest-dated** `LADD_Industry_Filter_*.txt`
file and, on a timer (`LADD_CHECK_EVERY`), re-scans for a newer one — so a new
list is picked up **without a restart**.

That Secret is updated by the **`ladd-admin`** control-plane service (see
`ladd-admin/`): an operator uploads a fresh list, and the filter hot-reloads it.
The filter doesn't know or care how the Secret changed — it just reloads.

## Key design decisions

- **Fail-closed on missing/stale list.** Checked before every batch; if not
  ready, the service exits with the reason rather than forwarding.
- **Thread-safe block list.** The list is held behind an `atomic.Pointer`
  snapshot, so the concurrent workers read it lock-free while the reloader can
  atomically swap in a new one.
- **Concurrent worker pool (I/O-bound).** The per-message work is dominated by
  the Kafka publish, not CPU, so the pool can safely run **more workers than
  cores** — `FILTER_WORKERS` (default 4; raise it to drain a backlog faster).
- **At-least-once.** A batch's offset is committed only after all of it is
  published; a crash reprocesses the batch (harmless — `cache-writer` upserts by
  GUFI). A publish failure leaves the batch uncommitted.

## Configuration (environment variables)

| Variable | Default | Meaning |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `KAFKA_TOPIC_NORMALIZED` | `fixm.normalized` | input topic |
| `KAFKA_TOPIC_FILTERED` | `fixm.filtered` | output topic |
| `KAFKA_GROUP` | `filter` | consumer group |
| `LADD_DIR` | `./data/ladd/active` | dir holding the in-effect LADD file (a Secret mount in k8s) |
| `LADD_MAX_AGE` | `216h` | staleness limit from the file's publication date (9 days) |
| `LADD_CHECK_EVERY` | `1h` | how often to re-scan for a newer list |
| `FILTER_WORKERS` | `4` | concurrent workers (I/O-bound; safe above core count) |

## Running

```bash
go run ./cmd/filter
```

Container / k8s: distroless Go image, deployed as the `filter` Deployment (no
Service — pure consumer/producer). Config from the `filter-config` ConfigMap;
the LADD list from the `ladd` Secret. It won't run without a valid, current
list — that's the point.

## The LADD directory layout (local dev)

`LADD_DIR` is the `active` directory; `staging` and `archived` sit beside it. A
freshly delivered file lands in `staging`, gets promoted to `active` (newest
date wins), and superseded files move to `archived` for audit. In k8s only
`active` is mounted (from the Secret), so those promote/archive steps are a
local-dev convenience; the `ladd-admin` service handles delivery in the cluster.

---

**In one line:** filter is the fail-closed LADD compliance screen — it drops
blocked aircraft, forwards the rest, and refuses to run without a current list.
