# database-writer

The pipeline's **history stage**. It reads filtered flights off `fixm.filtered`
and records them in Postgres + TimescaleDB: flight metadata plus a full
**time-series of every position**, so you can replay any flight's track long
after it has landed. This is the durable counterpart to cache-writer's live
Redis view.

```
filter ──► Kafka: fixm.filtered ──►  database-writer  ──►  Postgres / TimescaleDB
                                                              airports → flights → positions
```

Same input as cache-writer, different job: cache-writer keeps the **now**
(self-expiring live keys); database-writer keeps the **history** (append-only).

## What it does

For each message, in one transaction:

1. **Parse** it — GUFI, callsign, registration, aircraft type, origin/destination,
   status, departure/arrival actual times, and the latest `enRoute` position.
2. **Upsert the airports** (origin + destination) so the flight's FKs resolve.
3. **Upsert the flight** row (one per GUFI) — filling metadata and advancing its
   lifecycle status/counters.
4. **Append a position** row (if the message carried one).

Nothing is ever deleted; history accumulates and TimescaleDB ages it out.

## Data model

Three tables, a clean dimension → fact hierarchy:

| Table | Grain | Holds |
|---|---|---|
| `airports` | one row per ICAO code | `icao_code` (PK), `name`, `lat`, `lon` |
| `flights` | **one row per GUFI** (unique) | identity + lifecycle metadata |
| `positions` | **many rows per GUFI** (hypertable) | the time-series: `time`, lat/lon/alt/heading/speed/status |

The split is the key idea: a flight's **identity** is constant (one `flights`
row), but its **position** changes constantly (many `positions` rows). Rebuild a
flight's full track by joining them on `gufi`.

- **Airport coordinates** are populated opportunistically — the feed carries the
  codes reliably but the aerodrome lat/lon only rarely, so `airports.lat/lon`
  are mostly `NULL` until backfilled from a reference dataset.
- `positions` compresses after 7 days and is dropped after 90 (TimescaleDB
  policies) — long retention stays cheap.

## Lifecycle: no status is "terminal"

FDPS flight status (`ACTIVE`, `PROPOSED`, `COMPLETED`, `CANCELLED`, `DROPPED`)
is recorded but never used to "close" a flight — because **`DROPPED` is a
coverage/airspace transition, not a landing**. Measured over real data, ~19% of
dropped flights return to `ACTIVE`, and no `DROPPED` message carries an arrival
time. So the writer stays dumb: it records status and moves on.

"Has this flight ended?" is therefore a **read-time query**, using reliable
signals only:

```sql
-- ended if: landed, or explicitly completed, or gone quiet
WHERE status = 'COMPLETED'
   OR actual_arrival_time IS NOT NULL
   OR last_seen < now() - interval '15 minutes'
```

Two counters capture the bounce-back signal, computed inside the upsert:

- `drop_count` — how many times it entered `DROPPED`,
- `reactivation_count` — how many `DROPPED → ACTIVE` transitions.

A `status_time` guard (the timestamp of the message that set the status) makes
these count real *episodes* (not every repeated `DROPPED` message) and shrugs
off out-of-order delivery. Identity fields use `COALESCE(existing, new)` so a
sparse later message never blanks out a value already known.

## Batching (why it keeps up on a spinning disk)

Postgres fsyncs the WAL on every commit. Doing that **per message** caps the
writer at the disk's fsync rate (~hundreds/sec on an HDD) and it falls behind at
peak. So the writer **batches**: up to `DB_WRITER_BATCH_SIZE` messages (or until
`DB_WRITER_FLUSH_TIMEOUT`) are written in **one transaction = one fsync**, then
the Kafka offset is committed once. That amortizes the sync ~500× and lets a
cheap HDD keep pace with the live feed.

Flights are still applied **in message order** within the batch, so ordering,
FKs, and the transition counters stay correct. At-least-once holds: the offset
is committed only after the transaction commits, and a replayed batch is safe
(upserts are idempotent, and the counters key off the already-stored status).

## Migrations

The schema is owned by this service and applied **on startup** — no manual SQL,
no `docker-entrypoint-initdb.d`. On boot (after Postgres is reachable) it:

1. ensures a `schema_migrations` table exists,
2. reads the highest applied version,
3. applies every embedded migration with a **higher** version, each atomically,
   and records it.

Migrations are **embedded in the binary** (`go:embed`), so dev and prod run the
exact same files with nothing to mount.

### Adding a migration

1. Create the next file in `internal/migrate/sql/`, named `NNNN_description.sql`
   (zero-padded, strictly increasing), e.g. `0002_add_flight_type.sql`.
2. Write **forward-only, non-destructive** SQL:
   ```sql
   ALTER TABLE flights ADD COLUMN IF NOT EXISTS flight_type text;
   ```
3. Rebuild and restart the service. It applies `0002` (and only `0002`) once,
   logs `migrate: applied 0002_add_flight_type`, and records version 2.

That's the whole workflow — drop in a higher-numbered file and it runs on the
next startup. Don't edit already-applied files (they won't re-run); ship a new
one instead. Migrations run as a superuser (needed for `CREATE EXTENSION
timescaledb`), which `naspipeline` is in both dev and k8s.

## Configuration (environment variables)

| Variable | Default | Meaning |
|---|---|---|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `KAFKA_TOPIC_FILTERED` | `fixm.filtered` | input topic |
| `KAFKA_GROUP` | `database-writer` | consumer group |
| `DATABASE_URL` | `postgres://naspipeline:changeme@localhost:5433/naspipeline?sslmode=disable` | Postgres DSN (sensitive — holds the password) |
| `DB_WRITER_BATCH_SIZE` | `500` | flush after this many messages |
| `DB_WRITER_FLUSH_TIMEOUT` | `1s` | flush after this long since the batch's first message |

> **Dev port note:** the DSN uses host port **5433**, not 5432 — a native
> PostgreSQL owns 5432 on the dev box, so compose maps the container to 5433. In
> k8s it's the normal `postgres:5432`.

## Running

```bash
go run ./cmd/database-writer
```

It waits for Postgres, applies migrations, then consumes. Container / k8s:
distroless Go image, deployed as the `database-writer` Deployment (no Service —
pure consumer/writer). Postgres is the persistent `postgres` service; the DSN
comes from a Secret (it embeds the password).

---

**In one line:** database-writer turns the filtered flight stream into a durable,
queryable history — one row per flight plus a compressed position time-series —
batching its disk writes so a cheap HDD keeps up with the live feed.
