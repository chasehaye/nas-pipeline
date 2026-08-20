# api

The pipeline's **read side**. A small, read-only HTTP service (Go + Gin) that
serves the live flight picture from Redis to the frontend. It writes nothing —
`cache-writer` owns Redis; the `api` only reads.

```
cache-writer ──► Redis ──►  api  ──►  web (browser)
                          (Redis → JSON)
```

## Endpoints

| Method | Path | Returns |
|---|---|---|
| `GET` | `/healthz` | `ok` if Redis is reachable (pings it) |
| `GET` | `/flights` | `{ "count": N, "flights": [ … ] }` — every active flight |
| `GET` | `/flights/:gufi` | a single flight, or `404` |

> In production the browser reaches these through the `web` nginx proxy as
> `/api/flights` etc. (nginx strips the `/api` prefix, so the api sees
> `/flights`). See **same-origin** below.

## How it reads Redis

`ListFlights` is built to pull the whole live set efficiently:
1. **`SCAN`** for keys matching the `flight:*` prefix (cursor-based, 500 at a
   time — never blocks Redis the way `KEYS` would).
2. A **pipelined `HGETALL`** — one `HGETALL` per key, issued in a **single
   round-trip** — turns them into typed `Flight` structs.

So a page render is essentially one SCAN pass plus one pipelined batch read, even
with thousands of flights. `GetFlight` is a single `HGETALL` for one GUFI.

The `Flight` JSON carries what the map needs: `gufi`, `callSign`, `status`,
`lat`/`lon`, `alt`, `heading`, `speedKt`, and timestamps.

## Middleware (the Gin stack)

Every request passes through:
- **Recovery** — a panic becomes a clean 500, not a crash.
- **Request timeout** (`REQUEST_TIMEOUT`) — each request gets a hard deadline via
  context, so a slow Redis call can't hang a connection forever.
- **gzip** — the full flight list compresses well; this cuts payload size
  noticeably.
- **CORS** — configurable via `CORS_ORIGINS` (`*` allows any). Mostly moot in
  production because the browser hits the api **same-origin** through `web`, but
  it's there for direct/dev access.

## Key design decisions

- **Read-only.** No write path exists here — a clean separation from
  `cache-writer`. The api can be scaled or restarted freely without touching the
  data.
- **Fail-fast on Redis.** At startup it pings Redis and **exits** if it's
  unreachable, rather than starting and 502-ing every request.
- **Stateless.** It holds no state of its own — just a Redis client — so it's
  trivially horizontally scalable if ever needed.

## Same-origin with `web`

The frontend calls a **relative `/api/...`** path, and `web`'s nginx proxies that
to this service in-cluster (`api:8090`). That's why the browser never contacts
`api` directly and there's **no `VITE_API_BASE` to configure** — one origin, and
nginx routes `/api` here. (See `web/` and `web/nginx.conf`.)

## Configuration (environment variables)

| Variable | Default | Meaning |
|---|---|---|
| `HTTP_ADDR` | `:8090` | listen address |
| `GO_ENV` | `dev` | environment marker (`prod` in the prod overlay) |
| `REQUEST_TIMEOUT` | `5s` | per-request deadline |
| `CORS_ORIGINS` | `*` | comma-separated allowed origins (`*` = any) |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | *(empty)* | Redis auth (sensitive) |
| `REDIS_DB` | `0` | Redis database index |
| `REDIS_KEY_PREFIX` | `flight:` | key prefix to scan |

## Running

```bash
go run ./cmd/api
```

Container / k8s: distroless Go image, deployed as the `api` Deployment **with a
Service** (`api:8090`) — because, unlike the pipeline consumers, other things
connect *to* it (the `web` nginx proxy). Config from the `api-config` ConfigMap.

---

**In one line:** api is the stateless, read-only Redis→JSON edge that feeds the
map — one SCAN + one pipelined read per request, served same-origin through
`web`.
