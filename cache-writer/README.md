# cache-writer

The pipeline's **materialized-view stage**. It reads filtered flights off
`fixm.filtered` and maintains the live picture in Redis: **one hash per flight,
with a TTL**, so the set of keys in Redis *is* the set of aircraft currently in
the air.

```
filter ──► Kafka: fixm.filtered ──►  cache-writer  ──►  Redis (one hash per GUFI, TTL)
                                                              ▲
                                                          api reads these
```

> The Go module is internally named `redis-service`; the service/directory is
> `cache-writer`.

## What it does

For each flight message:
1. **Parse** it — GUFI, callsign, registration, status, timestamp, and the
   latest position (lat/lon/altitude) from the `enRoute` positions.
2. **Compute heading + speed** by dead reckoning from the position's track
   velocity (see below).
3. Route on status:
   - **not `ACTIVE`** (completed/cancelled) → **delete** the flight's key.
   - **no position** → skip (nothing to store) but still advance the offset.
   - **active with a position** → **upsert** the flight's hash and refresh its
     TTL.

The Redis key is `flight:<gufi>` (a hash of the flight's fields). The read `api`
serves the map from exactly these keys.

## The core idea: live keys == flights in the air

Two mechanisms keep Redis reflecting reality without any cleanup job:

- **TTL (`FLIGHT_TTL`, default 3m).** Every upsert refreshes the flight's
  expiry. A flight that stops sending updates simply **expires** on its own —
  so stale entries never accumulate. The cache is **self-cleaning**.
- **Delete on inactive.** A flight that reports a terminal status is removed
  immediately, rather than waiting for its TTL.

Between the two, `DBSIZE` tracks the count of currently-active flights, and the
map never shows ghosts.

## Dead reckoning (heading & speed)

The FIXM feed gives a **track velocity** as x/y components, not a heading. The
parser derives both:

- **heading** = `atan2(x, y)` in degrees, normalized to `0–360` (0 = North),
- **speed** = `hypot(x, y)` (knots).

These are only set when a non-zero velocity is present (`HasHeading`), so the
`api`/map can rotate the aircraft icon to its true track.

## Key design decisions

- **Latest position wins.** A message may carry several `enRoute` positions; the
  parser keeps the one with the newest `positionTime`.
- **At-least-once.** The offset is committed only *after* the Redis write
  succeeds. On a Redis error the message isn't committed and is retried; on a
  parse error or a positionless flight there's nothing to write, so it commits
  and moves on. Re-processing is harmless — upserts are idempotent.
- **Upsert is one round-trip.** `HSET` + `EXPIRE` are issued in a single Redis
  transaction pipeline.

## Scaling note (current state)

Processing is **sequential today** — one message, one Redis round-trip. That's
fine at the current volume. If it ever falls behind (Redis consumer lag on the
`redis-writer` group climbing), the right lever for a Redis-bound stage isn't
more workers, it's **pipelining** — batching many messages' commands into one
`Exec` to collapse the per-message round-trip. Not implemented yet; it's the
first thing to reach for if throughput becomes a problem.

## Configuration (environment variables)

| Variable | Default | Meaning |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `KAFKA_TOPIC_FILTERED` | `fixm.filtered` | input topic |
| `KAFKA_GROUP` | `redis-writer` | consumer group |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | *(empty)* | Redis auth (sensitive) |
| `REDIS_DB` | `0` | Redis database index |
| `REDIS_KEY_PREFIX` | `flight:` | key prefix for flight hashes |
| `FLIGHT_TTL` | `3m` | per-flight expiry, refreshed on each update |

## Running

```bash
go run ./cmd/cache-writer
```

It waits for Redis to be reachable before consuming. Container / k8s: distroless
Go image, deployed as the `cache-writer` Deployment (no Service — pure
consumer/writer). Config from the `cache-writer-config` ConfigMap; Redis is the
persistent `redis` service.

---

**In one line:** cache-writer keeps Redis as a self-expiring, live view of every
active flight — upserting positions with a TTL and deleting flights that land —
so the API can serve the map straight from memory.
