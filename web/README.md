# web

The **frontend** — a live map of every active flight. A React + Vite single-page
app (MapLibre GL) that polls the `api` and renders each aircraft as a
heading-rotated icon. Served in production by nginx, which also proxies `/api`
to the backend.

```
api ──►  web (nginx: static app + /api proxy)  ──►  browser
                                                     (MapLibre live map)
```

## Stack

React 19 · Vite 8 · TypeScript · Tailwind CSS v4 · MapLibre GL 6 (via
react-map-gl 8) · oxlint. Basemap tiles from OpenFreeMap.

## What it does

- Polls `GET /api/flights` every 4 seconds and draws the results on the map.
- Each flight is a plane icon **rotated to its true heading**.
- A corner overlay shows the live count and last-updated time
  (`N active · <time>`), or an error if the fetch fails.

The whole app is one screen: the map fills it, the overlay floats on top.

## Structure (feature-based)

```
src/
  main.tsx                     entry
  App.tsx                      the shell: <FlightMap> + live-traffic overlay
  config/env.ts                apiBase (defaults to same-origin "/api")
  lib/apiClient.ts             sendRequest<T> — typed fetch wrapper
  features/flights/
    index.ts                   the feature's public API (FlightMap, useFlights)
    types.ts                   Flight, FlightsResponse
    api/flights.ts             fetchFlights() -> sendRequest("/flights")
    hooks/useFlights.ts        the polling hook
    components/FlightMap.tsx    the MapLibre map
```

Everything flight-related lives under `features/flights/`, and `index.ts` is its
one public surface — `App` imports from the feature, not its internals.

## Data flow

`useFlights(4000)` drives everything:
- On mount it polls immediately, then every 4 s.
- Each poll uses an **`AbortController`**, so an in-flight request is cancelled on
  unmount or interval change — no overlapping or stale responses.
- It exposes `{ flights, count, error, lastUpdated }`; `App` passes `flights`
  into `<FlightMap>` and renders the overlay.

Polling (not websockets) is deliberate: the data refreshes on a cadence, 4 s is
plenty for a live map, and polling is far simpler to reason about and operate.

## The map (`FlightMap.tsx`)

- Flights become a GeoJSON `FeatureCollection`; a symbol layer draws each one.
- The plane icon is **canvas-drawn** (a small triangle) and registered with the
  map, then rotated per-feature via `icon-rotate: ['get', 'heading']` — so every
  aircraft points along its track.
- The map is **locked to 2D** (`maxPitch=0`, no drag-rotate/touch-pitch) — it's a
  traffic view, not a flight sim.
- A **labels** toggle shows/hides basemap text.
- **`map.resize()` on load** — a fix for the map rendering blank in the built
  (nginx) image: MapLibre sometimes paints before the flex container has its
  final size, and (unlike the Vite dev server) the static build got no resize
  nudge to repaint. Calling `resize()` in `onLoad` forces it.

## Same-origin & the nginx proxy

The app fetches a **relative `/api/...`** path (`config/env.ts` defaults
`apiBase` to `/api`). Two things make that work:
- **In the built image:** `nginx.conf` proxies `/api/*` -> `api:8090` in-cluster.
- **In dev:** `vite.config.ts` proxies `/api` -> the local api.

So the browser only ever talks to `web`, there's **no API URL baked into the
bundle** (no `VITE_API_BASE` needed), and CORS is moot — one origin. The `web`
image is therefore environment-agnostic: build once, run anywhere.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `VITE_API_BASE` | `/api` | API base path. Normally unset — the same-origin proxy handles it. Set only to point at a *separate* API host. |

## Running

```bash
npm install
npm run dev        # Vite dev server (proxies /api -> localhost:8090)
npm run build      # tsc + vite build -> dist/
npm run lint       # oxlint
```

Container / k8s: multi-stage build (Node build -> nginx serving `dist/`),
deployed as the `web` Deployment **with a Service**. In prod the `web` Service is
a `LoadBalancer` on host port 15000, and the Cloudflare tunnel points the public
hostname at it. nginx serves the SPA and proxies `/api` to the backend.

---

**In one line:** web is the MapLibre live-traffic map — a feature-structured
React app that polls the api every 4 s and renders heading-rotated aircraft,
served (and API-proxied) by nginx from a single environment-agnostic image.
